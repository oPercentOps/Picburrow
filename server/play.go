package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// Playback copies: browsers get the original file when any phone or laptop decodes it in hardware
// (H.264, 8-bit, up to 1080p). Anything else (iPhone HEVC, 4K, 10-bit HDR, AVI/MKV...) freezes on the picture
// while the sound goes on, so the Pi makes a 720p H.264 copy in the background and the viewer plays that.

func (a *App) playPath(id int64) string {
	return filepath.Join(a.dataDir, "play", fmt.Sprint(id%256), fmt.Sprintf("%d.mp4", id))
}

type probe struct {
	Streams []struct {
		Codec  string `json:"codec_name"`
		PixFmt string `json:"pix_fmt"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
	} `json:"streams"`
	Format struct {
		Name    string `json:"format_name"`
		BitRate string `json:"bit_rate"`
	} `json:"format"`
}

// playsAsIs: the original is something every browser decodes smoothly, and light enough for the Pi 3's network.
func (p probe) playsAsIs() bool {
	if len(p.Streams) == 0 {
		return true // no picture to fix (or unreadable): leave it alone
	}
	s := p.Streams[0]
	rate, _ := strconv.Atoi(p.Format.BitRate)
	return s.Codec == "h264" && (s.PixFmt == "yuv420p" || s.PixFmt == "yuvj420p") &&
		max(s.Width, s.Height) <= 1920 && rate <= 25_000_000 && strings.Contains(p.Format.Name, "mp4")
}

// makePlay checks one video and, if browsers would struggle with it, writes its playback copy.
func (a *App) makePlay(ctx context.Context, id int64, src string) error {
	out, err := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries",
		"stream=codec_name,pix_fmt,width,height:format=format_name,bit_rate", "-of", "json", src).Output()
	if err != nil {
		return fmt.Errorf("ffprobe: %w", err)
	}
	var p probe
	if err := json.Unmarshal(out, &p); err != nil {
		return fmt.Errorf("ffprobe: %w", err)
	}
	final := a.playPath(id)
	if p.playsAsIs() {
		os.Remove(final) // a copy of an earlier version of the file
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(final), 0o755); err != nil {
		return err
	}
	tmp := strings.TrimSuffix(final, ".mp4") + ".tmp.mp4"
	defer os.Remove(tmp)
	// 2 threads each way: half the Pi stays free for browsing and a 4K decode stays near 300 MB.
	// 720p/30 veryfast is what a Pi 3 encodes at a bearable speed; ffmpeg applies the rotation itself.
	if err := a.runFor(ctx, 12*time.Hour, keepCool, "ffmpeg", "-v", "error", "-y", "-threads", "2", "-i", src,
		"-map", "0:v:0", "-map", "0:a:0?", "-sn", "-dn", "-fpsmax", "30",
		"-vf", "scale='min(1280,iw)':'min(1280,ih)':force_original_aspect_ratio=decrease:force_divisible_by=2",
		"-c:v", "libx264", "-preset", "veryfast", "-crf", "26", "-pix_fmt", "yuv420p", "-threads", "2",
		"-c:a", "aac", "-b:a", "128k", "-ac", "2", "-movflags", "+faststart", tmp); err != nil {
		return err
	}
	return os.Rename(tmp, final) // never serve a half-written file
}

// A Pi 3 without a fan reaches 82 °C converting video and the firmware then slows the whole board down.
var (
	thermalZone = "/sys/class/thermal/thermal_zone0/temp" // millidegrees
	thermalTick = 10 * time.Second
)

const pauseAbove, resumeBelow = 75_000, 65_000

// keepCool freezes the conversion while the CPU is too hot and lets it go on once it has cooled down.
func keepCool(ctx context.Context, p *os.Process) {
	paused := false
	for {
		select {
		case <-ctx.Done(): // the child is gone or being killed (a kill also ends a frozen one)
			return
		case <-time.After(thermalTick):
		}
		b, err := os.ReadFile(thermalZone)
		temp, err2 := strconv.Atoi(strings.TrimSpace(string(b)))
		if err != nil || err2 != nil {
			return // no sensor on this machine
		}
		if !paused && temp >= pauseAbove {
			paused = p.Signal(syscall.SIGSTOP) == nil
		} else if paused && temp <= resumeBelow {
			paused = p.Signal(syscall.SIGCONT) != nil
		}
	}
}

// The admin's off switch is a file in the data folder: no schema change for one yes/no.
func (a *App) copiesOffFile() string { return filepath.Join(a.dataDir, "video-copies-off") }

func (a *App) patchJobs(w http.ResponseWriter, r *http.Request) {
	var in struct {
		VideoCopies bool `json:"video_copies"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	var err error
	if in.VideoCopies {
		if err = os.Remove(a.copiesOffFile()); os.IsNotExist(err) {
			err = nil
		}
	} else if err = os.WriteFile(a.copiesOffFile(), []byte("off\n"), 0o644); err == nil {
		if stop := a.playStop.Load(); stop != nil {
			(*stop)() // the video being converted is redone when the switch goes back on
		}
	}
	if err != nil {
		serverError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// playWorker makes playback copies one at a time, the video someone just tried to watch first, then newest first, and only while no thumbnail is waiting:
// previews matter more and a 1 GB Pi shouldn't decode a photo and a 4K video together for long.
func (a *App) playWorker(ctx context.Context) {
	wait := func(d time.Duration) bool {
		select {
		case <-ctx.Done():
			return false
		case <-time.After(d):
			return true
		}
	}
	for _, tool := range []string{"ffmpeg", "ffprobe"} {
		if _, err := exec.LookPath(tool); err != nil {
			log.Printf("no playback copies of videos: %s not installed", tool)
			return
		}
	}
	for {
		var id int64
		var src string
		var thumbsTodo int
		a.db.QueryRow("select count(*) from assets where thumbs_at is null and thumb_error is null and missing_at is null").Scan(&thumbsTodo)
		if fileExists(a.copiesOffFile()) || thumbsTodo > 0 || a.db.QueryRow(`select id, path from assets where type = 'video' and play_at is null and missing_at is null
			order by id = ? desc, taken_local desc limit 1`, a.playWant.Load()).Scan(&id, &src) != nil {
			if !wait(time.Minute) {
				return
			}
			continue
		}
		var fs unix.Statfs_t
		if unix.Statfs(a.dataDir, &fs) == nil && fs.Bavail*uint64(fs.Bsize) < 2<<30 {
			log.Printf("playback copies paused: less than 2 GB free in %s", a.dataDir)
			if !wait(time.Hour) {
				return
			}
			continue
		}
		job, stop := context.WithCancel(ctx)
		a.playStop.Store(&stop)
		err := a.makePlay(job, id, src)
		stopped := job.Err() != nil
		stop()
		if ctx.Err() != nil {
			return // shutting down: this video is redone next start
		}
		if stopped {
			continue // switched off by the admin
		}
		if err != nil { // not retried: the viewer falls back to the original
			log.Printf("playback copy %d: %v", id, err)
		}
		a.db.Exec("update assets set play_at = unixepoch() where id = ?", id)
	}
}

// servePlay streams the playback copy when there is one, else the original.
func (a *App) servePlay(w http.ResponseWriter, r *http.Request, x *assetRow) {
	if x.playAt.Valid {
		if f, err := os.Open(a.playPath(x.ID)); err == nil {
			defer f.Close()
			if st, err := f.Stat(); err == nil {
				w.Header().Set("Content-Type", "video/mp4")
				w.Header().Set("Cache-Control", "private, max-age=3600")
				http.ServeContent(w, r, "", st.ModTime(), f)
				return
			}
		}
	}
	if !x.playAt.Valid {
		a.playWant.Store(x.ID) // someone is watching it: convert it next
	}
	a.serveOriginal(w, r, x, false)
}

func (a *App) getPlay(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	var x *assetRow
	var err error
	if ok {
		x, err = a.loadAsset(userFrom(r).ID, id)
	}
	if !ok || err != nil {
		httpError(w, http.StatusNotFound, "No such photo")
		return
	}
	a.servePlay(w, r, x)
}

func (a *App) getSharedPlay(w http.ResponseWriter, r *http.Request) {
	l, x, ok := a.sharedAsset(w, r)
	if !ok {
		return
	}
	if !l.allowDownload {
		httpError(w, http.StatusForbidden, "Downloads are turned off for this album")
		return
	}
	a.servePlay(w, r, x)
}
