package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Extensions we index. Everything else (sidecars, thumbnails dbs, documents) is skipped.
var mediaExt = map[string]bool{
	".jpg": true, ".jpeg": true, ".jpe": true, ".jfif": true, ".png": true, ".gif": true, ".webp": true,
	".heic": true, ".heif": true, ".avif": true, ".tif": true, ".tiff": true, ".bmp": true, ".jxl": true,
	".dng": true, ".cr2": true, ".cr3": true, ".nef": true, ".arw": true, ".orf": true, ".rw2": true, ".raf": true,
	".mp4": true, ".mov": true, ".m4v": true, ".3gp": true, ".mkv": true, ".webm": true, ".avi": true, ".mts": true, ".m2ts": true,
}

type scanState struct {
	mu      sync.Mutex
	running bool
	again   bool // a scan was requested while one was running
	found   int
	lastErr string
}

type existing struct {
	id         int64
	size       int64
	mtime      int64
	wasMissing bool
}

type newFile struct {
	path  string
	size  int64
	mtime int64
}

// scanLibrary indexes one library. It is an upsert keyed on path, so a crash mid-scan just resumes next time.
func (a *App) scanLibrary(libID int64) error {
	var pathsJSON, globsJSON string
	if err := a.db.QueryRow("select import_paths, exclusion_globs from libraries where id = ?", libID).
		Scan(&pathsJSON, &globsJSON); err != nil {
		return err
	}
	var roots, globs []string
	if err := json.Unmarshal([]byte(pathsJSON), &roots); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(globsJSON), &globs); err != nil {
		return err
	}
	excl, err := compileGlobs(globs)
	if err != nil {
		return err
	}

	known := map[string]existing{}
	rows, err := a.db.Query("select id, path, size, mtime, missing_at is not null from assets where library_id = ?", libID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var p string
		var e existing
		if err := rows.Scan(&e.id, &p, &e.size, &e.mtime, &e.wasMissing); err != nil {
			rows.Close()
			return err
		}
		known[p] = e
	}
	rows.Close()

	seen := map[string]bool{}
	var changed []newFile // new or modified files, metadata read in batches
	var walkErrs []string
	var offline []string // roots that look unmounted: never mark their files missing
	for _, root := range roots {
		files := 0
		rootErr := false
		err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				if p == root {
					rootErr = true
				}
				walkErrs = append(walkErrs, err.Error())
				if d != nil && d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			// patterns match the path inside the library folder, not the folders above it (BUG-002)
			if rel, _ := filepath.Rel(root, p); rel != "." && excl(rel) {
				if d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			if d.IsDir() || !d.Type().IsRegular() || !mediaExt[strings.ToLower(filepath.Ext(p))] {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return nil
			}
			seen[p] = true
			files++
			e, ok := known[p]
			if ok && e.size == info.Size() && e.mtime == info.ModTime().Unix() {
				if e.wasMissing {
					a.db.Exec("update assets set missing_at = null, updated_at = unixepoch() where id = ?", e.id)
				}
				return nil
			}
			changed = append(changed, newFile{p, info.Size(), info.ModTime().Unix()})
			a.scan.mu.Lock()
			a.scan.found++
			a.scan.mu.Unlock()
			return nil
		})
		if err != nil {
			walkErrs = append(walkErrs, err.Error())
		}
		// An unmounted share is usually an empty mount-point dir: zero files where we had some = offline.
		if rootErr || (files == 0 && hasKnownUnder(known, root)) {
			offline = append(offline, root)
			walkErrs = append(walkErrs, "folder offline or empty, files kept: "+root)
		}
	}

	// Files that were indexed but not seen this time: candidates for "moved" (relink) or "missing".
	gone := map[[32]byte][]int64{}
	var goneIDs []int64
	for p, e := range known {
		if !seen[p] && !underAny(p, offline) {
			goneIDs = append(goneIDs, e.id)
		}
	}
	if len(goneIDs) > 0 {
		for _, id := range goneIDs {
			var h []byte
			if a.db.QueryRow("select quick_hash from assets where id = ?", id).Scan(&h) == nil && len(h) == 32 {
				gone[[32]byte(h)] = append(gone[[32]byte(h)], id)
			}
		}
	}

	relinked := map[int64]bool{}
	for start := 0; start < len(changed); start += 50 {
		batch := changed[start:min(start+50, len(changed))]
		paths := make([]string, len(batch))
		for i, f := range batch {
			paths[i] = f.path
		}
		metas, err := a.exif.Read(paths)
		if err != nil {
			return err
		}
		for i, f := range batch {
			h, err := quickHash(f.path, f.size)
			if err != nil {
				walkErrs = append(walkErrs, err.Error())
				continue
			}
			m := metas[i]
			if m.Mime == "" {
				m.Mime, m.Type = mimeFromExt(f.path)
			}
			if m.TakenLocal == "" { // no EXIF date: fall back to file time
				m.TakenLocal = time.Unix(f.mtime, 0).Format("2006-01-02 15:04:05")
			}
			var relinkID int64
			if _, isKnown := known[f.path]; !isKnown {
				if ids := gone[h]; len(ids) > 0 {
					relinkID, gone[h] = ids[0], ids[1:]
					relinked[relinkID] = true
				}
			}
			if err := a.upsertAsset(libID, f, h, m, relinkID); err != nil {
				return err
			}
		}
	}

	now := time.Now().Unix()
	defer func() {
		if err := a.pairLive(libID); err != nil {
			log.Printf("live photos, library %d: %v", libID, err)
		}
	}()
	for _, id := range goneIDs {
		if !relinked[id] {
			a.db.Exec("update assets set missing_at = coalesce(missing_at, ?) where id = ?", now, id)
		}
	}
	var lastErr any
	if len(walkErrs) > 0 {
		lastErr = fmt.Sprintf("%d unreadable paths, first: %s", len(walkErrs), walkErrs[0])
	}
	_, err = a.db.Exec("update libraries set last_scan_at = ?, last_scan_error = ? where id = ?", now, lastErr, libID)
	return err
}

func (a *App) upsertAsset(libID int64, f newFile, h [32]byte, m Meta, relinkID int64) error {
	args := []any{f.path, filepath.Dir(f.path), f.size, f.mtime, h[:], m.Type, m.Mime, nz(m.Width), nz(m.Height),
		nz(m.Orientation), nz(m.DurationMs), m.TakenLocal, m.TzOffsetMin, ns(m.Make), ns(m.Model), ns(m.Lens), m.Lat, m.Lon,
		ns(m.ContentID), m.Rating}
	if relinkID != 0 { // moved file: keep id, so favourites/albums/tags survive
		_, err := a.db.Exec(`update assets set path=?, dir=?, size=?, mtime=?, quick_hash=?, type=?, mime=?, width=?, height=?,
			orientation=?, duration_ms=?, taken_local=?, tz_offset_min=?, camera_make=?, camera_model=?, lens=?, lat=?, lon=?,
			content_id=?, rating=coalesce(rating, ?), missing_at=null, updated_at=unixepoch() where id=?`, append(args, relinkID)...)
		return err
	}
	_, err := a.db.Exec(`insert into assets (path, dir, size, mtime, quick_hash, type, mime, width, height, orientation,
			duration_ms, taken_local, tz_offset_min, camera_make, camera_model, lens, lat, lon, content_id, rating, library_id)
		values (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		on conflict (library_id, path) do update set size=excluded.size, mtime=excluded.mtime, quick_hash=excluded.quick_hash,
			type=excluded.type, mime=excluded.mime, width=excluded.width, height=excluded.height, orientation=excluded.orientation,
			duration_ms=excluded.duration_ms, taken_local=excluded.taken_local, tz_offset_min=excluded.tz_offset_min,
			camera_make=excluded.camera_make, camera_model=excluded.camera_model, lens=excluded.lens, lat=excluded.lat,
			lon=excluded.lon, content_id=excluded.content_id, rating=coalesce(assets.rating, excluded.rating),
			missing_at=null, thumbs_at=null, thumb_error=null, play_at=null, updated_at=unixepoch()`, append(args, libID)...)
	return err
}

// quickHash = sha256(size || first 64 KiB || last 64 KiB). Cheap on SD/USB; enough to recognise a moved file.
// shortcut: equal-size files with identical heads and tails collide; switch to a full hash on match if reported.
func quickHash(path string, size int64) ([32]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return [32]byte{}, err
	}
	defer f.Close()
	h := sha256.New()
	binary.Write(h, binary.LittleEndian, size)
	const chunk = 64 << 10
	if _, err := io.CopyN(h, f, chunk); err != nil && err != io.EOF {
		return [32]byte{}, err
	}
	if size > 2*chunk {
		if _, err := f.Seek(-chunk, io.SeekEnd); err != nil {
			return [32]byte{}, err
		}
		if _, err := io.Copy(h, f); err != nil {
			return [32]byte{}, err
		}
	}
	return [32]byte(h.Sum(nil)), nil
}

// compileGlobs supports *, ?, ** and {a,b}, matched case-insensitively against a path relative to the
// library folder (e.g. DCIM/.thumbnails/a.jpg).
func compileGlobs(globs []string) (func(string) bool, error) {
	var res []*regexp.Regexp
	for _, g := range globs {
		var b strings.Builder
		b.WriteString("(?i)^")
		for i := 0; i < len(g); i++ {
			c := g[i]
			switch {
			case strings.HasPrefix(g[i:], "**/"):
				b.WriteString("(?:.*/)?")
				i += 2
			case strings.HasPrefix(g[i:], "**"):
				b.WriteString(".*")
				i++
			case c == '*':
				b.WriteString("[^/]*")
			case c == '?':
				b.WriteString("[^/]")
			case c == '{':
				b.WriteString("(?:")
			case c == '}':
				b.WriteString(")")
			case c == ',':
				b.WriteString("|")
			case c == '\\' && i+1 < len(g):
				i++
				b.WriteString(regexp.QuoteMeta(string(g[i])))
			default:
				b.WriteString(regexp.QuoteMeta(string(c)))
			}
		}
		b.WriteString("$")
		re, err := regexp.Compile(b.String())
		if err != nil {
			return nil, fmt.Errorf("exclusion pattern %q: %w", g, err)
		}
		res = append(res, re)
	}
	return func(p string) bool {
		p = filepath.ToSlash(p)
		for _, re := range res {
			// a directory pattern like **/@eaDir/** must also match the directory itself, to skip it early
			if re.MatchString(p) || re.MatchString(p+"/") {
				return true
			}
		}
		return false
	}, nil
}

func mimeFromExt(p string) (string, string) {
	switch strings.ToLower(filepath.Ext(p)) {
	case ".mp4", ".m4v":
		return "video/mp4", "video"
	case ".mov":
		return "video/quicktime", "video"
	case ".webm":
		return "video/webm", "video"
	case ".mkv":
		return "video/x-matroska", "video"
	case ".3gp", ".avi", ".mts", ".m2ts":
		return "video/octet-stream", "video"
	case ".jpg", ".jpeg", ".jpe", ".jfif":
		return "image/jpeg", "image"
	case ".png":
		return "image/png", "image"
	case ".heic", ".heif":
		return "image/heic", "image"
	}
	return "application/octet-stream", "image"
}

// scanAll scans every library and returns when done. A call that arrives during a scan queues one more
// pass instead of being dropped, so a library edited mid-scan is still picked up.
func (a *App) scanAll() {
	if a.claimScan() {
		a.runScans()
	}
}

// startScan is scanAll in the background. The scan is marked running before it returns, so the next
// /api/jobs already reports it (BUG-001: the UI briefly showed an empty, idle library after setup).
func (a *App) startScan() {
	if a.claimScan() {
		go a.runScans()
	}
}

// claimScan marks a scan as running; if one already is, it queues another pass and returns false.
func (a *App) claimScan() bool {
	a.scan.mu.Lock()
	defer a.scan.mu.Unlock()
	if a.scan.running {
		a.scan.again = true
		return false
	}
	a.scan.running, a.scan.found, a.scan.lastErr = true, 0, ""
	return true
}

func (a *App) runScans() {
	defer a.wakeThumbs()
	for {
		ids, err := queryIDs(a.db, "select id from libraries")
		if err != nil {
			log.Printf("scan: %v", err)
		}
		for _, id := range ids {
			if err := a.scanLibrary(id); err != nil {
				log.Printf("scan library %d: %v", id, err)
				a.scan.mu.Lock()
				a.scan.lastErr = err.Error()
				a.scan.mu.Unlock()
			}
		}
		a.scan.mu.Lock()
		if !a.scan.again {
			a.scan.running = false
			a.scan.mu.Unlock()
			return
		}
		a.scan.again = false
		a.scan.mu.Unlock()
	}
}

func queryIDs(db *sql.DB, q string, args ...any) ([]int64, error) {
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func nz(i int) any {
	if i == 0 {
		return nil
	}
	return i
}

func ns(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func hasKnownUnder(known map[string]existing, root string) bool {
	for p := range known {
		if underAny(p, []string{root}) {
			return true
		}
	}
	return false
}

func underAny(p string, roots []string) bool {
	for _, r := range roots {
		if rel, err := filepath.Rel(r, p); err == nil && rel != ".." && !strings.HasPrefix(rel, "../") {
			return true
		}
	}
	return false
}

// pairLive links each still to its live-photo clip: same Apple content id, or else the same name in the same
// folder (IMG_1234.HEIC + IMG_1234.MOV) with a clip of 5 s or less. Pairs whose clip went missing are undone.
func (a *App) pairLive(libID int64) error {
	if _, err := a.db.Exec(`update assets set live_video_id = null where library_id = ? and live_video_id in
		(select id from assets where missing_at is not null)`, libID); err != nil {
		return err
	}
	type clip struct {
		id  int64
		key string
	}
	byContent, byName := map[string]int64{}, map[string]int64{}
	rows, err := a.db.Query(`select id, path, coalesce(content_id, ''), coalesce(duration_ms, 0) from assets
		where library_id = ? and type = 'video' and missing_at is null`, libID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id int64
		var p, cid string
		var dur int
		if err := rows.Scan(&id, &p, &cid, &dur); err != nil {
			rows.Close()
			return err
		}
		if cid != "" {
			byContent[cid] = id
		}
		if dur > 0 && dur <= 5000 {
			byName[stemKey(p)] = id
		}
	}
	rows.Close()
	if len(byContent)+len(byName) == 0 {
		return nil
	}
	type pair struct{ still, clip int64 }
	var pairs []pair
	rows, err = a.db.Query(`select id, path, coalesce(content_id, '') from assets
		where library_id = ? and type = 'image' and missing_at is null and live_video_id is null`, libID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id int64
		var p, cid string
		if err := rows.Scan(&id, &p, &cid); err != nil {
			rows.Close()
			return err
		}
		if v, ok := byContent[cid]; ok && cid != "" {
			pairs = append(pairs, pair{id, v})
		} else if v, ok := byName[stemKey(p)]; ok {
			pairs = append(pairs, pair{id, v})
		}
	}
	rows.Close()
	for _, pr := range pairs {
		if _, err := a.db.Exec("update assets set live_video_id = ? where id = ?", pr.clip, pr.still); err != nil {
			return err
		}
	}
	return nil
}

// stemKey: folder + file name without extension, case-insensitive (IMG_1234.HEIC and IMG_1234.mov match).
func stemKey(p string) string {
	return strings.ToLower(strings.TrimSuffix(p, filepath.Ext(p)))
}

// notLiveClip hides clips that belong to a live photo: they are played from their still.
const notLiveClip = `a.id not in (select live_video_id from assets where live_video_id is not null)`
