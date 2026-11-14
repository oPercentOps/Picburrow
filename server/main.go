package main

import (
	"context"
	"database/sql"
	"embed"
	"flag"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"time"
)

//go:embed all:dist
var webFS embed.FS

type App struct {
	db         *sql.DB
	dataDir    string
	exif       *exifTool
	scan       scanState
	inflight   inflight
	thumbWake  chan struct{}
	loginLimit *limiter
	ionice     []string
	decodeSem  chan struct{}
	playWant   atomic.Int64                       // video to convert next: the last one played without a playback copy
	playStop   atomic.Pointer[context.CancelFunc] // stops the conversion that is running
}

func main() {
	dataDir := flag.String("data-dir", envOr("PICBURROW_DATA", "/var/lib/picburrow"), "database and thumbnails")
	listen := flag.String("listen", envOr("PICBURROW_LISTEN", ":8080"), "address to serve on")
	workers := flag.Int("workers", 1, "parallel thumbnail decodes (1 for a 1 GB Pi; 2-4 with more RAM)")
	flag.Parse()

	if abs, err := filepath.Abs(*dataDir); err == nil {
		*dataDir = abs // log and child processes see where data really is
	}
	if err := os.MkdirAll(filepath.Join(*dataDir, "thumbs"), 0o755); err != nil {
		log.Fatal(err)
	}
	db, err := openDB(*dataDir)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	et, err := startExifTool()
	if err != nil {
		log.Fatal(err)
	}
	defer et.Close()

	a := &App{db: db, dataDir: *dataDir, exif: et, thumbWake: make(chan struct{}, 1),
		inflight: inflight{m: map[int64]chan struct{}{}}, loginLimit: newLimiter(10), decodeSem: make(chan struct{}, max(*workers, 1))}
	if _, err := exec.LookPath("ionice"); err == nil {
		a.ionice = []string{"ionice", "-c3"} // idle I/O class: never slow the NAS for other users
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go a.thumbWorker(ctx)
	go a.playWorker(ctx)
	go a.housekeeping(ctx)

	srv := &http.Server{Addr: *listen, Handler: a.routes(), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(sctx)
	}()
	log.Printf("serving on %s, data in %s", *listen, *dataDir)
	if w := sdCardWarning(*dataDir); w != "" {
		log.Print("warning: " + w)
	}
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

// housekeeping: scan at start and nightly at 03:00 local; every 15 minutes in between, scan again only if a
// folder changed, so photos a sync tool drops in show up soon; daily cleanup of expired sessions.
func (a *App) housekeeping(ctx context.Context) {
	a.scanAll()
	quick := time.NewTicker(15 * time.Minute)
	defer quick.Stop()
	next3 := func() time.Time {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, now.Location())
		if !next.After(now) {
			next = next.AddDate(0, 0, 1)
		}
		return next
	}
	next := next3() // kept across quick checks: one that runs past 03:00 must not skip the night's work
	for {
		select {
		case <-ctx.Done():
			return
		case <-quick.C:
			if a.foldersChanged() {
				a.scanAll()
			}
		case <-time.After(time.Until(next)):
			a.db.Exec("delete from sessions where expires_at < unixepoch()")
			a.scanAll()
			a.purgeTrash()
			next = next3()
		}
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
