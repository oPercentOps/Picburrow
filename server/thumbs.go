package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Thumbnail sizes: small fits the justified grid at 2x DPR (220 css px rows), large is the viewer preview.
var thumbSizes = map[string]string{"s": "1000x440", "l": "1920x1920"}

func (a *App) thumbPath(id int64, size string) string {
	return filepath.Join(a.dataDir, "thumbs", fmt.Sprint(id%256), fmt.Sprintf("%d-%s.webp", id, size))
}

type inflight struct {
	mu sync.Mutex
	m  map[int64]chan struct{}
}

// ensureThumbs generates both sizes for one asset once; concurrent callers wait for the first.
func (a *App) ensureThumbs(ctx context.Context, id int64) error {
	a.inflight.mu.Lock()
	if ch, ok := a.inflight.m[id]; ok {
		a.inflight.mu.Unlock()
		select {
		case <-ch:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	ch := make(chan struct{})
	a.inflight.m[id] = ch
	a.inflight.mu.Unlock()
	defer func() {
		a.inflight.mu.Lock()
		delete(a.inflight.m, id)
		a.inflight.mu.Unlock()
		close(ch)
	}()

	var path, typ string
	var done sql.NullInt64
	if err := a.db.QueryRow("select path, type, thumbs_at from assets where id = ?", id).Scan(&path, &typ, &done); err != nil {
		return err
	}
	if done.Valid {
		return nil
	}
	// one global limit for worker + on-demand: each decode can take 250 MB, the Pi has 1 GB
	select {
	case a.decodeSem <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	err := a.makeThumbs(id, path, typ)
	<-a.decodeSem
	if err != nil {
		msg := err.Error()
		if len(msg) > 300 {
			msg = msg[:300]
		}
		a.db.Exec("update assets set thumb_error = ? where id = ?", msg, id)
		return err
	}
	_, err = a.db.Exec("update assets set thumbs_at = unixepoch(), thumb_error = null where id = ?", id)
	return err
}

func (a *App) makeThumbs(id int64, src, typ string) error {
	dir := filepath.Dir(a.thumbPath(id, "s"))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if typ == "video" {
		poster := filepath.Join(dir, fmt.Sprintf("%d-poster.tmp.jpg", id))
		defer os.Remove(poster)
		// first frame after 1s (or the very first if shorter); scaled down so vips has little to decode
		if err := a.run("ffmpeg", "-v", "error", "-y", "-ss", "1", "-i", src, "-frames:v", "1",
			"-vf", "scale='min(1920,iw)':-2", poster); err != nil || !fileExists(poster) {
			if err := a.run("ffmpeg", "-v", "error", "-y", "-i", src, "-frames:v", "1",
				"-vf", "scale='min(1920,iw)':-2", poster); err != nil {
				return fmt.Errorf("video poster: %w", err)
			}
		}
		src = poster
	}
	for _, size := range []string{"l", "s"} { // large first: the small one is made from it, so the original is decoded once
		box := thumbSizes[size]
		final := a.thumbPath(id, size)
		// absolute: vipsthumbnail resolves a relative -o against the input photo's folder (BUG-003)
		tmp, err := filepath.Abs(strings.TrimSuffix(final, ".webp") + ".tmp.webp")
		if err != nil {
			return err
		}
		// vipsthumbnail decodes with shrink-on-load and auto-rotates from EXIF; strip drops metadata (GPS) from thumbs;
		// effort=0 encodes WebP ~2x faster for ~30% bigger files: CPU is the scarce resource on a Pi
		if err := a.run("vipsthumbnail", src, "--size", box, "-o", tmp+"[Q=75,strip,effort=0]"); err != nil {
			os.Remove(tmp)
			return err
		}
		if err := os.Rename(tmp, final); err != nil { // never serve a half-written file
			return err
		}
		src = final
	}
	// blur placeholder from the small thumbnail: ~16 px WebP, a few hundred bytes, stored in the row
	ph, err := filepath.Abs(filepath.Join(dir, fmt.Sprintf("%d-ph.tmp.webp", id)))
	if err != nil {
		return err
	}
	defer os.Remove(ph)
	if err := a.run("vipsthumbnail", a.thumbPath(id, "s"), "--size", "16x16", "-o", ph+"[Q=50,strip,effort=0]"); err != nil {
		return err
	}
	b, err := os.ReadFile(ph)
	if err != nil {
		return err
	}
	_, err = a.db.Exec("update assets set thumbhash = ? where id = ?", b, id)
	return err
}

// run starts a low-priority child with a hard timeout; a decoder crash or leak dies with the child.
func (a *App) run(name string, args ...string) error {
	return a.runFor(context.Background(), 3*time.Minute, nil, name, args...)
}

// watch, if set, runs beside the child (and gets a context that ends with it): the thermal guard of long jobs.
func (a *App) runFor(ctx context.Context, limit time.Duration, watch func(context.Context, *os.Process), name string, args ...string) error {
	ctx, cancel := context.WithTimeout(ctx, limit)
	defer cancel()
	argv := append([]string{"-n", "10"}, a.ionice...)
	cmd := exec.CommandContext(ctx, "nice", append(append(argv, name), args...)...)
	cmd.Env = append(os.Environ(), "VIPS_CONCURRENCY=1", "VIPS_DISC_THRESHOLD=100m")
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Start()
	if err == nil {
		if watch != nil {
			go watch(ctx, cmd.Process)
		}
		err = cmd.Wait()
	}
	if err != nil {
		var ee *exec.Error
		if errors.As(err, &ee) {
			return fmt.Errorf("%s not installed", name)
		}
		return fmt.Errorf("%s: %v: %s", name, err, strings.TrimSpace(out.String()))
	}
	return nil
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.Size() > 0
}

func (a *App) wakeThumbs() {
	select {
	case a.thumbWake <- struct{}{}:
	default:
	}
}

// thumbWorker works through assets without thumbnails, newest first, so the top of the timeline fills first.
func (a *App) thumbWorker(ctx context.Context) {
	for {
		ids, err := queryIDs(a.db, `select id from assets where thumbs_at is null and thumb_error is null and missing_at is null
			order by taken_local desc limit 20`)
		if err != nil {
			log.Printf("thumbs: %v", err)
		}
		for _, id := range ids {
			if ctx.Err() != nil {
				return
			}
			if err := a.ensureThumbs(ctx, id); err != nil {
				log.Printf("thumbs %d: %v", id, err)
			}
		}
		if len(ids) == 0 {
			select {
			case <-ctx.Done():
				return
			case <-a.thumbWake:
			case <-time.After(time.Minute):
			}
		}
	}
}
