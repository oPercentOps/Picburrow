package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestGlobs(t *testing.T) {
	ex, err := compileGlobs([]string{"**/@eaDir/**", "**/.*/**", "**/*.{tif,arw}", `**/\@x/**`})
	if err != nil {
		t.Fatal(err)
	}
	for p, want := range map[string]bool{
		"/nas/@eaDir":             true,
		"/nas/@eaDir/a.jpg":       true,
		"/nas/sub/@eaDir/b/c.jpg": true,
		"/nas/.thumbnails/a.jpg":  true,
		"/nas/a.TIF":              true,
		"/nas/raw/b.arw":          true,
		"/nas/@x/1.jpg":           true,
		"/nas/a.jpg":              false,
		"/nas/eaDir/a.jpg":        false,
		"/nas/photo.tiff":         false,
		"/nas/2024/IMG_0001.HEIC": false,
	} {
		if ex(p) != want {
			t.Errorf("exclude(%q) = %v, want %v", p, !want, want)
		}
	}
}

func TestParseExifDate(t *testing.T) {
	cases := []struct {
		in, local string
		off       *int
		ok        bool
	}{
		{"2024:05:01 10:00:00", "2024-05-01 10:00:00", nil, true},
		{"2024:05:01 10:00:00.123+02:00", "2024-05-01 10:00:00", ip(120), true},
		{"2021:12:31 23:59:59-05:30", "2021-12-31 23:59:59", ip(-330), true},
		{"0000:00:00 00:00:00", "", nil, false},
		{"1904:01:01 00:00:00", "", nil, false}, // zero QuickTime date after UTC conversion
		{"1955:07:01 12:00:00", "1955-07-01 12:00:00", nil, true},
		{"garbage", "", nil, false},
	}
	for _, c := range cases {
		local, off, ok := parseExifDate(c.in)
		if local != c.local || ok != c.ok || (off == nil) != (c.off == nil) || (off != nil && *off != *c.off) {
			t.Errorf("parseExifDate(%q) = %q %v %v", c.in, local, off, ok)
		}
	}
}

func ip(i int) *int { return &i }

// TestFlow drives the real HTTP API over real files: setup, scan, timeline, thumbnails, originals,
// moved-file relink, unmounted-share protection and ownership.
func TestFlow(t *testing.T) {
	for _, tool := range []string{"exiftool", "vipsthumbnail", "ffmpeg"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s not installed", tool)
		}
	}
	lib := t.TempDir()
	mustRun(t, "ffmpeg", "-v", "error", "-f", "lavfi", "-i", "testsrc=size=640x480", "-frames:v", "1", filepath.Join(lib, "a.jpg"))
	mustRun(t, "ffmpeg", "-v", "error", "-f", "lavfi", "-i", "testsrc=size=480x640", "-frames:v", "1", filepath.Join(lib, "b.jpg"))
	mustRun(t, "exiftool", "-q", "-overwrite_original", "-DateTimeOriginal=2024:05:01 10:00:00", filepath.Join(lib, "a.jpg"))
	mustRun(t, "exiftool", "-q", "-overwrite_original", "-DateTimeOriginal=2023:12:24 18:30:00", "-OffsetTimeOriginal=+01:00", filepath.Join(lib, "b.jpg"))
	os.MkdirAll(filepath.Join(lib, "@eaDir"), 0o755)
	mustRun(t, "ffmpeg", "-v", "error", "-f", "lavfi", "-i", "testsrc=size=64x64", "-frames:v", "1", filepath.Join(lib, "@eaDir", "junk.jpg"))
	mustRun(t, "ffmpeg", "-v", "error", "-f", "lavfi", "-i", "testsrc=size=320x240:duration=2", "-pix_fmt", "yuv420p", filepath.Join(lib, "c.mp4"))

	a, srv := newTestApp(t)
	c := client(t)

	// before setup
	var st map[string]any
	get(t, c, srv.URL+"/api/status", &st)
	if st["needs_setup"] != true {
		t.Fatalf("status before setup: %v", st)
	}
	if code := post(t, c, srv.URL+"/api/setup", `{"username":"ann","password":"short","path":"`+lib+`"}`); code != 400 {
		t.Fatalf("short password accepted: %d", code)
	}
	if code := post(t, c, srv.URL+"/api/setup", `{"username":"ann","password":"correct horse","path":"/nope"}`); code != 400 {
		t.Fatalf("missing folder accepted: %d", code)
	}
	if code := post(t, c, srv.URL+"/api/setup", `{"username":"ann","password":"correct horse","path":"`+lib+`"}`); code != 200 {
		t.Fatalf("setup: %d", code)
	}
	if code := post(t, client(t), srv.URL+"/api/setup", `{"username":"eve","password":"correct horse","path":"`+lib+`"}`); code != 409 {
		t.Fatalf("second setup: %d", code)
	}
	a.scanAll() // setup also starts one in the background: either this runs it, or queues a rerun and the wait below covers it
	waitScan(t, a)

	var buckets []struct {
		Month string
		Count int
	}
	get(t, c, srv.URL+"/api/timeline/buckets", &buckets)
	months := map[string]int{}
	total := 0
	for _, b := range buckets {
		months[b.Month] = b.Count
		total += b.Count
	}
	if months["2024-05"] != 1 || months["2023-12"] != 1 || total != 3 {
		t.Fatalf("buckets = %v (want a.jpg in 2024-05, b.jpg in 2023-12, c.mp4 by file date, @eaDir skipped)", buckets)
	}
	var videoTaken string
	a.db.QueryRow("select taken_local from assets where type = 'video'").Scan(&videoTaken)
	if strings.HasPrefix(videoTaken, "1904") || videoTaken == "" {
		t.Fatalf("dateless video got taken_local %q, want its file date", videoTaken)
	}
	var tiles []tile
	get(t, c, srv.URL+"/api/timeline?month=2023-12", &tiles)
	if len(tiles) != 1 || *tiles[0].W != 480 || *tiles[0].H != 640 {
		t.Fatalf("2023-12 tiles = %+v", tiles)
	}
	bID := tiles[0].ID

	var asset assetRow
	get(t, c, srv.URL+"/api/assets/"+itoa(bID), &asset)
	if asset.TzOffsetMin == nil || *asset.TzOffsetMin != 60 || asset.Prev == nil {
		t.Fatalf("asset b = %+v", asset)
	}

	// thumbnails on demand, then cached
	resp, err := c.Get(srv.URL + "/api/assets/" + itoa(bID) + "/thumb/s")
	if err != nil || resp.StatusCode != 200 || resp.Header.Get("Content-Type") != "image/webp" {
		t.Fatalf("thumb: %v %v", err, resp)
	}
	resp.Body.Close()
	if _, err := os.Stat(a.thumbPath(bID, "l")); err != nil {
		t.Fatalf("large thumb not written: %v", err)
	}
	// worker path for the video poster
	a.wakeThumbs()
	ctx, cancel := context.WithCancel(context.Background())
	ids, _ := queryIDs(a.db, "select id from assets where type = 'video'")
	if err := a.ensureThumbs(ctx, ids[0]); err != nil {
		t.Fatalf("video thumb: %v", err)
	}
	cancel()

	// original with Range
	req, _ := http.NewRequest("GET", srv.URL+"/api/assets/"+itoa(bID)+"/original", nil)
	req.Header.Set("Range", "bytes=0-9")
	resp, err = c.Do(req)
	if err != nil || resp.StatusCode != 206 {
		t.Fatalf("range: %v %v", err, resp)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if len(body) != 10 {
		t.Fatalf("range body %d bytes", len(body))
	}

	// a moved file keeps its id (favourites/albums survive)
	a.db.Exec("update assets set is_favorite = 1 where id = ?", bID)
	os.MkdirAll(filepath.Join(lib, "2023"), 0o755)
	if err := os.Rename(filepath.Join(lib, "b.jpg"), filepath.Join(lib, "2023", "b.jpg")); err != nil {
		t.Fatal(err)
	}
	a.scanAll()
	var path string
	var fav bool
	a.db.QueryRow("select path, is_favorite from assets where id = ?", bID).Scan(&path, &fav)
	if path != filepath.Join(lib, "2023", "b.jpg") || !fav {
		t.Fatalf("after move: path %q fav %v", path, fav)
	}

	// a deleted file goes to trash
	os.Remove(filepath.Join(lib, "a.jpg"))
	a.scanAll()
	var missing int
	a.db.QueryRow("select count(*) from assets where missing_at is not null").Scan(&missing)
	if missing != 1 {
		t.Fatalf("missing after delete = %d, want 1", missing)
	}

	// an unmounted share (empty mount point) must not trash the library
	hidden := lib + ".unmounted"
	if err := os.Rename(lib, hidden); err != nil {
		t.Fatal(err)
	}
	os.Mkdir(lib, 0o755)
	a.scanAll()
	a.db.QueryRow("select count(*) from assets where missing_at is not null").Scan(&missing)
	os.Remove(lib)
	os.Rename(hidden, lib)
	if missing != 1 {
		t.Fatalf("empty mount point marked %d assets missing, want still 1", missing)
	}

	// ownership: another user sees nothing of ann's
	hash, _ := hashPassword("another pass")
	a.db.Exec("insert into users (username, password_hash) values ('bob', ?)", hash)
	bob := client(t)
	if code := post(t, bob, srv.URL+"/api/login", `{"username":"bob","password":"another pass"}`); code != 200 {
		t.Fatalf("bob login: %d", code)
	}
	for _, u := range []string{"/api/assets/" + itoa(bID), "/api/assets/" + itoa(bID) + "/original", "/api/assets/" + itoa(bID) + "/thumb/s"} {
		if r, _ := bob.Get(srv.URL + u); r.StatusCode != 404 {
			t.Errorf("bob GET %s = %d, want 404", u, r.StatusCode)
		}
	}
	if code := post(t, bob, srv.URL+"/api/libraries/1/scan", `{}`); code != 403 {
		t.Errorf("non-admin scan = %d, want 403", code)
	}
	if r, _ := client(t).Get(srv.URL + "/api/timeline/buckets"); r.StatusCode != 401 {
		t.Errorf("anonymous buckets = %d, want 401", r.StatusCode)
	}
	if code := post(t, client(t), srv.URL+"/api/login", `{"username":"ann","password":"wrong"}`); code != 401 {
		t.Errorf("wrong password = %d", code)
	}
}

func newTestApp(t *testing.T) (*App, *httptest.Server) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "thumbs"), 0o755)
	db, err := openDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	et, err := startExifTool()
	if err != nil {
		t.Fatal(err)
	}
	a := &App{db: db, dataDir: dir, exif: et, thumbWake: make(chan struct{}, 1),
		inflight: inflight{m: map[int64]chan struct{}{}}, loginLimit: newLimiter(100), decodeSem: make(chan struct{}, 1)}
	srv := httptest.NewServer(a.routes())
	t.Cleanup(func() { srv.Close(); et.Close(); db.Close() })
	return a, srv
}

func client(t *testing.T) *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Jar: jar}
}

func get(t *testing.T, c *http.Client, url string, v any) {
	t.Helper()
	r, err := c.Get(url)
	if err != nil || r.StatusCode != 200 {
		t.Fatalf("GET %s: %v %v", url, err, r)
	}
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		t.Fatal(err)
	}
}

func post(t *testing.T, c *http.Client, url, body string) int {
	t.Helper()
	r, err := c.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	return r.StatusCode
}

func mustRun(t *testing.T, name string, args ...string) {
	t.Helper()
	if out, err := exec.Command(name, args...).CombinedOutput(); err != nil {
		t.Fatalf("%s: %v\n%s", name, err, out)
	}
}

func itoa(i int64) string { return strconv.FormatInt(i, 10) }

func waitScan(t *testing.T, a *App) {
	t.Helper()
	for i := 0; i < 600; i++ {
		a.scan.mu.Lock()
		busy := a.scan.running
		a.scan.mu.Unlock()
		if !busy {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("scan did not finish in 30 s")
}

// TestScanRequestDuringScanIsNotLost: the bug where a library edited mid-scan was ignored until the nightly scan.
func TestScanRequestDuringScanIsNotLost(t *testing.T) {
	a, _ := newTestApp(t)
	a.scan.running = true // pretend a scan is in progress
	a.scanAll()           // must queue, not drop
	if !a.scan.again {
		t.Fatal("scan request during a running scan was dropped")
	}
}

func TestManage(t *testing.T) {
	for _, tool := range []string{"exiftool", "ffmpeg"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s not installed", tool)
		}
	}
	lib, extra := t.TempDir(), t.TempDir()
	for _, f := range []string{"top.jpg", "2024/trip/a.jpg", "2024/trip/b.jpg", "2024/c.jpg"} {
		os.MkdirAll(filepath.Join(lib, filepath.Dir(f)), 0o755)
		mustRun(t, "ffmpeg", "-v", "error", "-f", "lavfi", "-i", "testsrc=size=64x48", "-frames:v", "1", filepath.Join(lib, f))
	}
	mustRun(t, "ffmpeg", "-v", "error", "-f", "lavfi", "-i", "testsrc=size=64x48", "-frames:v", "1", filepath.Join(extra, "x.jpg"))
	a, srv := newTestApp(t)
	c := client(t)
	if code := post(t, c, srv.URL+"/api/setup", `{"username":"ann","password":"correct horse","path":"`+lib+`"}`); code != 200 {
		t.Fatalf("setup: %d", code)
	}
	a.scanAll()
	waitScan(t, a)

	// folders: root -> lib -> 2024 (subfolder trip + c.jpg)
	var root, y struct {
		Dirs   []folder
		Assets []tile
	}
	get(t, c, srv.URL+"/api/folders", &root)
	if len(root.Dirs) != 1 || root.Dirs[0].Count != 4 {
		t.Fatalf("root folders = %+v", root.Dirs)
	}
	get(t, c, srv.URL+"/api/folders?path="+filepath.Join(lib, "2024"), &y)
	if len(y.Dirs) != 1 || y.Dirs[0].Name != "trip" || y.Dirs[0].Count != 2 || y.Dirs[0].Cover == nil || len(y.Assets) != 1 {
		t.Fatalf("2024 folder = %+v", y)
	}
	for _, p := range []string{"/etc", lib + "/../", extra} {
		if r, _ := c.Get(srv.URL + "/api/folders?path=" + p); r.StatusCode != 404 {
			t.Errorf("folders?path=%s = %d, want 404", p, r.StatusCode)
		}
	}

	// favourite + archive, owner only
	id := y.Assets[0].ID
	if code := patch(t, c, srv.URL+"/api/assets", `{"ids":[`+itoa(id)+`],"favorite":true}`); code != 200 {
		t.Fatalf("patch favourite: %d", code)
	}
	var favs []tile
	get(t, c, srv.URL+"/api/assets?favorite=1", &favs)
	if len(favs) != 1 || favs[0].ID != id || !favs[0].Fav {
		t.Fatalf("favourites = %+v", favs)
	}
	patch(t, c, srv.URL+"/api/assets", `{"ids":[`+itoa(id)+`],"archived":true}`)
	var arch, buckets []map[string]any
	get(t, c, srv.URL+"/api/assets?archived=1", &arch)
	get(t, c, srv.URL+"/api/timeline/buckets", &buckets)
	if len(arch) != 1 || buckets[0]["count"].(float64) != 3 {
		t.Fatalf("archive %v, buckets %v: archived photo must leave the timeline", arch, buckets)
	}
	hash, _ := hashPassword("another pass")
	a.db.Exec("insert into users (username, password_hash) values ('bob', ?)", hash)
	bob := client(t)
	post(t, bob, srv.URL+"/api/login", `{"username":"bob","password":"another pass"}`)
	patch(t, bob, srv.URL+"/api/assets", `{"ids":[`+itoa(id)+`],"favorite":false,"archived":false}`)
	var fav, archived bool
	a.db.QueryRow("select is_favorite, is_archived from assets where id = ?", id).Scan(&fav, &archived)
	if !fav || !archived {
		t.Fatal("another user changed ann's photo")
	}
	if r, _ := bob.Get(srv.URL + "/api/libraries"); r.StatusCode != 403 {
		t.Errorf("non-admin libraries = %d", r.StatusCode)
	}

	// library settings: validation, then a new folder gets scanned
	for body, want := range map[string]int{
		`{"import_paths":["/does/not/exist"]}`:        400,
		`{"import_paths":["relative/dir"]}`:           400,
		`{"import_paths":[" "]}`:                      400,
		`{"exclusion_globs":["**/{a"]}`:               400,
		`{"name":"  "}`:                               400,
		`{"name":"half","exclusion_globs":["**/{a"]}`: 400,
	} {
		if code := patch(t, c, srv.URL+"/api/libraries/1", body); code != want {
			t.Errorf("PATCH library %s = %d, want %d", body, code, want)
		}
	}
	var name string
	a.db.QueryRow("select name from libraries where id = 1").Scan(&name)
	if name == "half" {
		t.Fatal("a rejected PATCH still saved its valid fields")
	}
	if code := patch(t, c, srv.URL+"/api/libraries/1", `{"name":"Family","import_paths":["`+lib+`","`+extra+`","`+lib+`/"]}`); code != 200 {
		t.Fatalf("PATCH library: %d", code)
	}
	a.scanAll()
	waitScan(t, a)
	var libs []library
	get(t, c, srv.URL+"/api/libraries", &libs)
	if len(libs) != 1 || libs[0].Name != "Family" || len(libs[0].Paths) != 2 || libs[0].Assets != 5 {
		t.Fatalf("libraries = %+v (want deduped paths and the extra folder scanned)", libs)
	}
}

func patch(t *testing.T, c *http.Client, url, body string) int {
	t.Helper()
	req, _ := http.NewRequest("PATCH", url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	return r.StatusCode
}

// BUG-001: right after setup (or a library change) /api/jobs reported "idle, nothing to do" until the scan
// goroutine got going, so the UI showed "No photos here yet" and no indexing banner. startScan must mark the
// scan as running before it returns, so the very next /api/jobs sees it.
func TestStartScanIsVisibleImmediately(t *testing.T) {
	a, _ := newTestApp(t)
	a.startScan()
	a.scan.mu.Lock()
	running := a.scan.running
	a.scan.mu.Unlock()
	if !running {
		t.Fatal("startScan returned before the scan was marked running")
	}
	waitScan(t, a)
}

// BUG-002: exclusion patterns were matched against the full path, so the default "skip hidden folders"
// pattern also matched a library that itself lives under a hidden folder, and every file was skipped.
func TestLibraryUnderHiddenFolder(t *testing.T) {
	if _, err := exec.LookPath("exiftool"); err != nil {
		t.Skip("exiftool not installed")
	}
	lib := filepath.Join(t.TempDir(), ".local", "photos")
	os.MkdirAll(filepath.Join(lib, ".thumbnails"), 0o755)
	os.WriteFile(filepath.Join(lib, "a.jpg"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(lib, ".thumbnails", "skip.jpg"), []byte("x"), 0o644)
	a, srv := newTestApp(t)
	if code := post(t, client(t), srv.URL+"/api/setup", `{"username":"ann","password":"correct horse","path":"`+lib+`"}`); code != 200 {
		t.Fatalf("setup: %d", code)
	}
	waitScan(t, a)
	var paths []string
	rows, _ := a.db.Query("select path from assets")
	for rows.Next() {
		var p string
		rows.Scan(&p)
		paths = append(paths, p)
	}
	rows.Close()
	if len(paths) != 1 || filepath.Base(paths[0]) != "a.jpg" {
		t.Fatalf("indexed %v, want only a.jpg (hidden subfolder skipped, hidden parent of the library ignored)", paths)
	}
}

// BUG-003: with a relative --data-dir (the README's own `--data-dir ./data`), every thumbnail failed:
// vipsthumbnail resolves a relative -o path against the input photo's folder, not the working directory.
func TestRelativeDataDir(t *testing.T) {
	for _, tool := range []string{"vipsthumbnail", "ffmpeg", "exiftool"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s not installed", tool)
		}
	}
	work, lib := t.TempDir(), t.TempDir()
	mustRun(t, "ffmpeg", "-v", "error", "-f", "lavfi", "-i", "testsrc=size=64x48", "-frames:v", "1", filepath.Join(lib, "a.jpg"))
	t.Chdir(work)
	os.MkdirAll(filepath.Join("data", "thumbs"), 0o755)
	db, err := openDB("data")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	a := &App{db: db, dataDir: "data", inflight: inflight{m: map[int64]chan struct{}{}}, decodeSem: make(chan struct{}, 1)}
	db.Exec("insert into users (username, password_hash, is_admin) values ('ann', 'x', 1)")
	db.Exec(`insert into libraries (owner_id, name, import_paths) values (1, 'l', ?)`, `["`+lib+`"]`)
	db.Exec(`insert into assets (library_id, path, dir, size, mtime, quick_hash, type, mime) values (1, ?, ?, 1, 1, x'00', 'image', 'image/jpeg')`,
		filepath.Join(lib, "a.jpg"), lib)
	if err := a.ensureThumbs(context.Background(), 1); err != nil {
		t.Fatalf("thumbnail with a relative data dir: %v", err)
	}
	for _, s := range []string{"s", "l"} {
		if !fileExists(a.thumbPath(1, s)) {
			t.Errorf("%s thumbnail not in the data dir", s)
		}
	}
	if entries, _ := os.ReadDir(lib); len(entries) != 1 {
		t.Errorf("thumbnail files leaked into the photo folder: %d entries", len(entries))
	}
}

// S01-N2: guessing passwords is limited to 10 tries per minute per address.
func TestLoginRateLimit(t *testing.T) {
	a, srv := newTestApp(t)
	a.loginLimit = newLimiter(10)
	c := client(t)
	for i := 1; i <= 10; i++ {
		if code := post(t, c, srv.URL+"/api/login", `{"username":"x","password":"y"}`); code != 401 {
			t.Fatalf("attempt %d = %d, want 401", i, code)
		}
	}
	if code := post(t, c, srv.URL+"/api/login", `{"username":"x","password":"y"}`); code != 429 {
		t.Fatalf("11th attempt = %d, want 429", code)
	}
}

// HEIC (iPhone default) goes through the same thumbnail pipeline. testdata/gradient.heic is a generated
// 640x480 gradient (pillow-heif), not a camera file.
func TestHEICThumbnail(t *testing.T) {
	if out, err := exec.Command("vips", "-l").Output(); err != nil || !strings.Contains(string(out), "heifload") {
		t.Skip("vips without HEIF support")
	}
	lib := t.TempDir()
	src, _ := os.ReadFile("testdata/gradient.heic")
	os.WriteFile(filepath.Join(lib, "IMG_0001.HEIC"), src, 0o644)
	a, srv := newTestApp(t)
	if code := post(t, client(t), srv.URL+"/api/setup", `{"username":"ann","password":"correct horse","path":"`+lib+`"}`); code != 200 {
		t.Fatalf("setup: %d", code)
	}
	waitScan(t, a)
	var w, h int
	var mime string
	a.db.QueryRow("select width, height, mime from assets").Scan(&w, &h, &mime)
	if w != 640 || h != 480 || mime != "image/heic" {
		t.Fatalf("HEIC metadata = %dx%d %s", w, h, mime)
	}
	if err := a.ensureThumbs(context.Background(), 1); err != nil {
		t.Fatalf("HEIC thumbnail: %v", err)
	}
	if !fileExists(a.thumbPath(1, "l")) || !fileExists(a.thumbPath(1, "s")) {
		t.Fatal("HEIC thumbnails not written")
	}
}

func TestAlbumsLinksSearch(t *testing.T) {
	for _, tool := range []string{"exiftool", "ffmpeg", "vipsthumbnail"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s not installed", tool)
		}
	}
	lib := t.TempDir()
	for i, f := range []string{"beach_100%.jpg", "city.jpg", "forest.jpg"} {
		p := filepath.Join(lib, f)
		mustRun(t, "ffmpeg", "-v", "error", "-f", "lavfi", "-i", "testsrc=size=64x48", "-frames:v", "1", p)
		mustRun(t, "exiftool", "-q", "-overwrite_original", fmt.Sprintf("-DateTimeOriginal=2024:0%d:15 10:00:00", i+1),
			"-Make=Google", fmt.Sprintf("-Model=Pixel %d", i+6), p)
	}
	a, srv := newTestApp(t)
	ann := client(t)
	if code := post(t, ann, srv.URL+"/api/setup", `{"username":"ann","password":"correct horse","path":"`+lib+`"}`); code != 200 {
		t.Fatalf("setup: %d", code)
	}
	waitScan(t, a)
	ids := map[string]int64{}
	rows, _ := a.db.Query("select id, path from assets")
	for rows.Next() {
		var id int64
		var p string
		rows.Scan(&id, &p)
		ids[filepath.Base(p)] = id
	}
	rows.Close()
	beach, city, forest := itoa(ids["beach_100%.jpg"]), itoa(ids["city.jpg"]), itoa(ids["forest.jpg"])

	// placeholder made with the thumbnails
	if err := a.ensureThumbs(context.Background(), ids["city.jpg"]); err != nil {
		t.Fatal(err)
	}
	var ph []byte
	a.db.QueryRow("select thumbhash from assets where id = ?", ids["city.jpg"]).Scan(&ph)
	if len(ph) < 20 || len(ph) > 2000 || string(ph[8:12]) != "WEBP" {
		t.Fatalf("placeholder = %d bytes, want a tiny WebP", len(ph))
	}

	// another user, with a photo of their own
	hash, _ := hashPassword("another pass")
	a.db.Exec("insert into users (username, password_hash) values ('bob', ?)", hash)
	a.db.Exec(`insert into libraries (owner_id, name, import_paths) values (2, 'bob', '[]')`)
	a.db.Exec(`insert into assets (library_id, path, dir, size, mtime, quick_hash, type, mime, taken_local) values (2, '/bob/x.jpg', '/bob', 1, 1, x'00', 'image', 'image/jpeg', '2024-01-01 00:00:00')`)
	var bobAsset int64
	a.db.QueryRow("select id from assets where library_id = 2").Scan(&bobAsset)
	bob := client(t)
	post(t, bob, srv.URL+"/api/login", `{"username":"bob","password":"another pass"}`)

	// albums: create with photos; someone else's photo is skipped
	if code := post(t, ann, srv.URL+"/api/albums", `{"name":"  Summer  ","ids":[`+beach+`,`+city+`,`+itoa(bobAsset)+`]}`); code != 200 {
		t.Fatalf("create album: %d", code)
	}
	if code := post(t, ann, srv.URL+"/api/albums", `{"name":"   "}`); code != 400 {
		t.Fatalf("blank album name: %d", code)
	}
	var albums []album
	get(t, ann, srv.URL+"/api/albums", &albums)
	if len(albums) != 1 || albums[0].Name != "Summer" || albums[0].Count != 2 || albums[0].Cover == nil {
		t.Fatalf("albums = %+v", albums)
	}
	al := itoa(albums[0].ID)
	post(t, ann, srv.URL+"/api/albums/"+al+"/assets", `{"ids":[`+forest+`]}`)
	if code := del(t, ann, srv.URL+"/api/albums/"+al+"/assets", `{"ids":[`+city+`]}`); code != 200 {
		t.Fatalf("remove from album: %d", code)
	}
	var one album
	get(t, ann, srv.URL+"/api/albums/"+al, &one)
	if len(one.Assets) != 2 {
		t.Fatalf("album assets = %d, want 2 (beach + forest)", len(one.Assets))
	}
	for _, u := range []string{"/api/albums/" + al} {
		if r, _ := bob.Get(srv.URL + u); r.StatusCode != 404 {
			t.Errorf("bob GET %s = %d", u, r.StatusCode)
		}
	}
	if code := post(t, bob, srv.URL+"/api/albums/"+al+"/assets", `{"ids":[`+itoa(bobAsset)+`]}`); code != 404 {
		t.Errorf("bob adding to ann's album = %d", code)
	}
	if code := post(t, bob, srv.URL+"/api/albums/"+al+"/links", `{}`); code != 404 {
		t.Errorf("bob sharing ann's album = %d", code)
	}

	// open link without downloads
	if code := post(t, ann, srv.URL+"/api/albums/"+al+"/links", `{"allow_download":false}`); code != 200 {
		t.Fatalf("create link: %d", code)
	}
	get(t, ann, srv.URL+"/api/albums/"+al, &one)
	tok := one.Links[0].Token
	if len(tok) < 40 {
		t.Fatalf("token too short: %q", tok)
	}
	guest := client(t)
	var shared struct {
		Name   string
		Locked bool
		Assets []tile
	}
	get(t, guest, srv.URL+"/api/s/"+tok, &shared)
	if shared.Name != "Summer" || shared.Locked || len(shared.Assets) != 2 {
		t.Fatalf("shared = %+v", shared)
	}
	for u, want := range map[string]int{
		"/api/s/" + tok + "/assets/" + beach + "/thumb/s":  200,
		"/api/s/" + tok + "/assets/" + beach:               200,
		"/api/s/" + tok + "/assets/" + beach + "/original": 403, // downloads off
		"/api/s/" + tok + "/assets/" + city + "/thumb/s":   404, // removed from the album
		"/api/s/" + tok + "/assets/" + itoa(bobAsset):      404,
		"/api/s/not-a-token":                               410,
		"/api/assets/" + beach:                             401, // the link is not a session
	} {
		if r, _ := guest.Get(srv.URL + u); r.StatusCode != want {
			t.Errorf("guest GET %s = %d, want %d", u, r.StatusCode, want)
		}
	}
	var sa map[string]any
	get(t, guest, srv.URL+"/api/s/"+tok+"/assets/"+beach, &sa)
	if _, leaks := sa["path"]; leaks || sa["lat"] != nil {
		t.Errorf("shared asset leaks owner details: %v", sa)
	}

	// password link: locked until unlocked, per browser
	post(t, ann, srv.URL+"/api/albums/"+al+"/links", `{"password":"sunny","allow_download":true}`)
	get(t, ann, srv.URL+"/api/albums/"+al, &one)
	ptok := one.Links[1].Token
	shared.Assets = nil // decoding into a reused struct keeps fields the next response leaves out
	get(t, guest, srv.URL+"/api/s/"+ptok, &shared)
	if !shared.Locked || shared.Assets != nil {
		t.Fatalf("password link before unlock = %+v", shared)
	}
	if r, _ := guest.Get(srv.URL + "/api/s/" + ptok + "/assets/" + beach + "/thumb/s"); r.StatusCode != 401 {
		t.Errorf("locked thumb = %d", r.StatusCode)
	}
	if code := post(t, guest, srv.URL+"/api/s/"+ptok+"/unlock", `{"password":"rainy"}`); code != 401 {
		t.Errorf("wrong link password = %d", code)
	}
	if code := post(t, guest, srv.URL+"/api/s/"+ptok+"/unlock", `{"password":"sunny"}`); code != 204 {
		t.Fatalf("unlock = %d", code)
	}
	get(t, guest, srv.URL+"/api/s/"+ptok, &shared)
	if shared.Locked || len(shared.Assets) != 2 {
		t.Fatalf("after unlock = %+v", shared)
	}
	if r, _ := guest.Get(srv.URL + "/api/s/" + ptok + "/assets/" + beach + "/original"); r.StatusCode != 200 {
		t.Errorf("download on an allow_download link = %d", r.StatusCode)
	}
	// unlocking one link doesn't open the other link of the same album... it's open anyway; but another browser stays locked
	if r, _ := client(t).Get(srv.URL + "/api/s/" + ptok + "/assets/" + beach + "/thumb/s"); r.StatusCode != 401 {
		t.Errorf("fresh browser on a password link = %d", r.StatusCode)
	}
	// expiry and revoke
	if code := post(t, ann, srv.URL+"/api/albums/"+al+"/links", `{"expires_at":1}`); code != 400 {
		t.Errorf("expiry in the past = %d", code)
	}
	a.db.Exec("update shared_links set expires_at = unixepoch() - 1 where token = ?", ptok)
	if r, _ := guest.Get(srv.URL + "/api/s/" + ptok); r.StatusCode != 410 {
		t.Errorf("expired link = %d", r.StatusCode)
	}
	if code := del(t, bob, srv.URL+"/api/links/"+itoa(one.Links[0].ID), ``); code != 404 {
		t.Errorf("bob revoking ann's link = %d", code)
	}
	if code := del(t, ann, srv.URL+"/api/links/"+itoa(one.Links[0].ID), ``); code != 204 {
		t.Errorf("revoke = %d", code)
	}
	if r, _ := guest.Get(srv.URL + "/api/s/" + tok); r.StatusCode != 410 {
		t.Errorf("revoked link = %d", r.StatusCode)
	}

	// search
	for q, want := range map[string][]string{
		"q=beach":                       {"beach_100%.jpg"},
		"q=%25":                         {"beach_100%.jpg"}, // a literal %, not a wildcard
		"q=_":                           {"beach_100%.jpg"}, // a literal _, not "any one character"
		"q=pixel%208":                   {"forest.jpg"},
		"from=2024-02-01&to=2024-02-15": {"city.jpg"}, // inclusive end day
		"q=CITY&type=video":             {},
		"":                              {"beach_100%.jpg", "city.jpg", "forest.jpg"},
	} {
		var res []tile
		get(t, ann, srv.URL+"/api/search?"+q, &res)
		var got []string
		for _, x := range res {
			for n, id := range ids {
				if id == x.ID {
					got = append(got, n)
				}
			}
		}
		sort.Strings(got)
		if fmt.Sprint(got) != fmt.Sprint(want) && !(len(got) == 0 && len(want) == 0) {
			t.Errorf("search %q = %v, want %v", q, got, want)
		}
	}
	var bobRes []tile
	get(t, bob, srv.URL+"/api/search?q=beach", &bobRes)
	if len(bobRes) != 0 {
		t.Errorf("bob finds ann's photos: %v", bobRes)
	}
	if r, _ := ann.Get(srv.URL + "/api/search?from=yesterday"); r.StatusCode != 400 {
		t.Errorf("bad date = %d", r.StatusCode)
	}

	// deleting the album keeps the photos
	if code := del(t, ann, srv.URL+"/api/albums/"+al, ``); code != 204 {
		t.Fatalf("delete album: %d", code)
	}
	var n int
	a.db.QueryRow("select count(*) from assets where library_id = 1").Scan(&n)
	if n != 3 {
		t.Fatalf("photos after deleting the album = %d", n)
	}
}

func del(t *testing.T, c *http.Client, url, body string) int {
	t.Helper()
	req, _ := http.NewRequest("DELETE", url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	return r.StatusCode
}

// A database made by the first release (schema version 1) upgrades in place and keeps its rows.
func TestMigrateFromV1(t *testing.T) {
	dir := t.TempDir()
	v1 := strings.NewReplacer(
		"  content_id    text,                                -- Apple live-photo pairing id (still and clip share it)\n", "",
		"create index assets_content  on assets(content_id) where content_id is not null;\n", "",
		"  play_at       integer,                             -- videos: null = not yet checked for a playback copy (play/<id>.mp4)\n", "",
		"create index assets_play     on assets(id) where type = 'video' and play_at is null;\n", "").Replace(schemaSQL)
	if v1 == schemaSQL {
		t.Fatal("couldn't rebuild the v1 schema from schema.sql")
	}
	old, err := sql.Open("sqlite", "file:"+filepath.Join(dir, "picburrow.db"))
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{v1, "pragma user_version = 1",
		"insert into users (username, password_hash) values ('ann', 'x')",
		`insert into libraries (owner_id, name) values (1, 'l')`,
		`insert into assets (library_id, path, dir, size, mtime, quick_hash, type, mime, is_favorite) values (1, '/p/a.jpg', '/p', 1, 1, x'00', 'image', 'image/jpeg', 1)`} {
		if _, err := old.Exec(q); err != nil {
			t.Fatalf("%v: %s", err, q[:min(len(q), 60)])
		}
	}
	old.Close()
	db, err := openDB(dir)
	if err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	defer db.Close()
	// a backup of the old database was taken first, and it still opens at version 1 with the data
	baks, _ := filepath.Glob(filepath.Join(dir, "picburrow.db.v1-*.bak"))
	if len(baks) != 1 {
		t.Fatalf("backups before upgrade = %v, want one v1 backup", baks)
	}
	bak, _ := sql.Open("sqlite", "file:"+baks[0])
	var bv, bn int
	bak.QueryRow("pragma user_version").Scan(&bv)
	bak.QueryRow("select count(*) from assets").Scan(&bn)
	bak.Close()
	if bv != 1 || bn != 1 {
		t.Fatalf("backup: version %d, %d assets", bv, bn)
	}
	var v int
	var fav bool
	var cid *string
	db.QueryRow("pragma user_version").Scan(&v)
	if err := db.QueryRow("select is_favorite, content_id from assets").Scan(&fav, &cid); err != nil || v != schemaVersion || !fav {
		t.Fatalf("after upgrade: version %d, fav %v, err %v", v, fav, err)
	}
	var member int
	if db.QueryRow("select count(*) from library_members where library_id = 1 and user_id = 1").Scan(&member); member != 1 {
		t.Fatal("after upgrade, the library's owner isn't a member of it: they'd see no photos")
	}
}

func TestLiveRatingsTrashUsers(t *testing.T) {
	for _, tool := range []string{"exiftool", "ffmpeg", "vipsthumbnail"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s not installed", tool)
		}
	}
	lib, bobLib := t.TempDir(), t.TempDir()
	still := func(name string) {
		mustRun(t, "ffmpeg", "-v", "error", "-f", "lavfi", "-i", "testsrc=size=64x48", "-frames:v", "1", filepath.Join(lib, name))
	}
	clip := func(name string, secs int) {
		mustRun(t, "ffmpeg", "-v", "error", "-f", "lavfi", "-i", fmt.Sprintf("testsrc=size=64x48:duration=%d", secs), "-pix_fmt", "yuv420p", filepath.Join(lib, name))
	}
	still("IMG_0001.jpg")
	clip("IMG_0001.MOV", 2) // live photo pair
	still("IMG_0002.jpg")
	clip("IMG_0002.mp4", 8) // same name but a real video: not a live photo
	still("rated.jpg")
	mustRun(t, "exiftool", "-q", "-overwrite_original", "-XMP-xmp:Rating=4", filepath.Join(lib, "rated.jpg"))
	mustRun(t, "ffmpeg", "-v", "error", "-f", "lavfi", "-i", "testsrc=size=64x48", "-frames:v", "1", filepath.Join(bobLib, "bob.jpg"))

	a, srv := newTestApp(t)
	ann := client(t)
	post(t, ann, srv.URL+"/api/setup", `{"username":"ann","password":"correct horse","path":"`+lib+`"}`)
	waitScan(t, a)
	id := func(name string) int64 {
		var x int64
		a.db.QueryRow("select id from assets where path = ?", filepath.Join(lib, name)).Scan(&x)
		return x
	}

	// live photos
	var liveID *int64
	a.db.QueryRow("select live_video_id from assets where id = ?", id("IMG_0001.jpg")).Scan(&liveID)
	if liveID == nil || *liveID != id("IMG_0001.MOV") {
		t.Fatalf("IMG_0001.jpg not paired with its clip: %v", liveID)
	}
	a.db.QueryRow("select live_video_id from assets where id = ?", id("IMG_0002.jpg")).Scan(&liveID)
	if liveID != nil {
		t.Fatal("an 8 s video was taken for a live photo")
	}
	var buckets []map[string]any
	get(t, ann, srv.URL+"/api/timeline/buckets", &buckets)
	var month []tile
	get(t, ann, srv.URL+"/api/timeline?month="+buckets[0]["month"].(string), &month)
	var sawClip, sawLive bool
	for _, x := range month {
		sawClip = sawClip || x.ID == id("IMG_0001.MOV")
		sawLive = sawLive || (x.ID == id("IMG_0001.jpg") && x.Live != nil)
	}
	if sawClip || !sawLive || len(month) != 4 {
		t.Fatalf("timeline: %d items, clip shown %v, still marked live %v", len(month), sawClip, sawLive)
	}
	var folder struct{ Assets []tile }
	get(t, ann, srv.URL+"/api/folders?path="+lib, &folder)
	if len(folder.Assets) != 4 {
		t.Fatalf("folder shows %d items, want 4 (clip hidden)", len(folder.Assets))
	}
	// the clip plays through a share link of an album holding the still
	post(t, ann, srv.URL+"/api/albums", `{"name":"L","ids":[`+itoa(id("IMG_0001.jpg"))+`]}`)
	post(t, ann, srv.URL+"/api/albums/1/links", `{"allow_download":true}`)
	var al album
	get(t, ann, srv.URL+"/api/albums/1", &al)
	if r, _ := client(t).Get(srv.URL + "/api/s/" + al.Links[0].Token + "/assets/" + itoa(id("IMG_0001.MOV")) + "/original"); r.StatusCode != 200 {
		t.Errorf("live clip through the share link = %d", r.StatusCode)
	}
	if r, _ := client(t).Get(srv.URL + "/api/s/" + al.Links[0].Token + "/assets/" + itoa(id("IMG_0002.mp4")) + "/original"); r.StatusCode != 404 {
		t.Errorf("unrelated video through the share link = %d", r.StatusCode)
	}

	// ratings: read from the file, set in the app, the app wins on rescan
	var rating *int
	a.db.QueryRow("select rating from assets where id = ?", id("rated.jpg")).Scan(&rating)
	if rating == nil || *rating != 4 {
		t.Fatalf("rating from XMP = %v, want 4", rating)
	}
	patch(t, ann, srv.URL+"/api/assets", `{"ids":[`+itoa(id("rated.jpg"))+`],"rating":2}`)
	os.Chtimes(filepath.Join(lib, "rated.jpg"), time.Now(), time.Now().Add(time.Hour)) // changed on disk: re-read
	a.scanAll()
	a.db.QueryRow("select rating from assets where id = ?", id("rated.jpg")).Scan(&rating)
	if rating == nil || *rating != 2 {
		t.Fatalf("rating after rescan = %v, want the app's 2", rating)
	}
	var found []tile
	get(t, ann, srv.URL+"/api/search?rating=2", &found)
	if len(found) != 1 {
		t.Fatalf("search rating>=2 = %d results", len(found))
	}
	if code := patch(t, ann, srv.URL+"/api/assets", `{"ids":[1],"rating":6}`); code != 400 {
		t.Errorf("rating 6 = %d", code)
	}
	patch(t, ann, srv.URL+"/api/assets", `{"ids":[`+itoa(id("rated.jpg"))+`],"rating":0}`)
	a.db.QueryRow("select rating from assets where id = ?", id("rated.jpg")).Scan(&rating)
	if rating != nil {
		t.Fatalf("rating 0 should clear, got %v", *rating)
	}

	// a live clip that disappears unpairs; trash lists missing files, purge after 30 days, never deletes files
	os.Remove(filepath.Join(lib, "IMG_0001.MOV"))
	os.Remove(filepath.Join(lib, "IMG_0002.jpg"))
	a.scanAll()
	a.db.QueryRow("select live_video_id from assets where id = ?", id("IMG_0001.jpg")).Scan(&liveID)
	if liveID != nil {
		t.Fatal("still paired with a missing clip")
	}
	var trash []tile
	get(t, ann, srv.URL+"/api/trash", &trash)
	if len(trash) != 2 {
		t.Fatalf("trash = %d, want 2", len(trash))
	}
	a.db.Exec("update assets set missing_at = unixepoch() - 31*86400 where path like '%IMG_0002.jpg'")
	a.purgeTrash()
	get(t, ann, srv.URL+"/api/trash", &trash)
	if len(trash) != 1 {
		t.Fatalf("after the 30-day purge trash = %d, want 1", len(trash))
	}
	if code := del(t, ann, srv.URL+"/api/trash", ``); code != 200 {
		t.Fatalf("empty trash = %d", code)
	}
	get(t, ann, srv.URL+"/api/trash", &trash)
	if len(trash) != 0 {
		t.Fatalf("trash after emptying = %d", len(trash))
	}
	if _, err := os.Stat(filepath.Join(lib, "IMG_0001.jpg")); err != nil {
		t.Fatal("a photo file was touched")
	}

	// family accounts
	for body, want := range map[string]int{
		`{"username":"bob","password":"short","path":"` + bobLib + `"}`:             400,
		`{"username":"bob","password":"long enough","path":"/nope"}`:                400,
		`{"username":"ann","password":"long enough","path":"` + bobLib + `"}`:       409,
		`{"username":"two words","password":"long enough","path":"` + bobLib + `"}`: 400,
		`{"username":"bob","password":"long enough","path":"` + bobLib + `"}`:       200,
	} {
		if code := post(t, ann, srv.URL+"/api/users", body); code != want {
			t.Errorf("POST /api/users %s = %d, want %d", body, code, want)
		}
	}
	waitScan(t, a)
	bob := client(t)
	if code := post(t, bob, srv.URL+"/api/login", `{"username":"bob","password":"long enough"}`); code != 200 {
		t.Fatalf("bob login: %d", code)
	}
	get(t, bob, srv.URL+"/api/timeline/buckets", &buckets)
	if len(buckets) != 1 || buckets[0]["count"].(float64) != 1 {
		t.Fatalf("bob's timeline = %v, want only his one photo", buckets)
	}
	if code := post(t, bob, srv.URL+"/api/users", `{"username":"eve","password":"long enough","path":"`+bobLib+`"}`); code != 403 {
		t.Errorf("non-admin creating users = %d", code)
	}
	var users []userRow
	get(t, ann, srv.URL+"/api/users", &users)
	bobID := itoa(users[1].ID)
	if code := patch(t, ann, srv.URL+"/api/users/"+bobID, `{"password":"brand new pass"}`); code != 204 {
		t.Fatalf("reset password = %d", code)
	}
	if r, _ := bob.Get(srv.URL + "/api/me"); r.StatusCode != 401 {
		t.Errorf("bob still signed in after a password reset: %d", r.StatusCode)
	}
	if code := post(t, client(t), srv.URL+"/api/login", `{"username":"bob","password":"brand new pass"}`); code != 200 {
		t.Errorf("login with the new password = %d", code)
	}
	if code := del(t, ann, srv.URL+"/api/users/"+itoa(users[0].ID), ``); code != 400 {
		t.Errorf("deleting yourself = %d", code)
	}
	if code := del(t, ann, srv.URL+"/api/users/"+bobID, ``); code != 204 {
		t.Fatalf("delete bob = %d", code)
	}
	var n int
	a.db.QueryRow("select count(*) from assets a join libraries l on l.id = a.library_id where l.owner_id = ?", users[1].ID).Scan(&n)
	if _, err := os.Stat(filepath.Join(bobLib, "bob.jpg")); n != 0 || err != nil {
		t.Fatalf("after deleting bob: %d indexed photos left, file err %v", n, err)
	}
}

// Live pairing by Apple's content id (real iPhones), which generated test files can't carry on the still.
func TestPairLiveByContentID(t *testing.T) {
	a, _ := newTestApp(t)
	a.db.Exec("insert into users (username, password_hash) values ('ann', 'x')")
	a.db.Exec(`insert into libraries (owner_id, name) values (1, 'l')`)
	ins := `insert into assets (library_id, path, dir, size, mtime, quick_hash, type, mime, duration_ms, content_id) values (1, ?, '/p', 1, 1, x'00', ?, 'x', ?, ?)`
	a.db.Exec(ins, "/p/IMG_E0001.HEIC", "image", nil, "ABC") // edited copy, different name, same id
	a.db.Exec(ins, "/p/IMG_0001.MOV", "video", 3000, "ABC")
	a.db.Exec(ins, "/p/other.jpg", "image", nil, "XYZ")
	if err := a.pairLive(1); err != nil {
		t.Fatal(err)
	}
	var live *int64
	a.db.QueryRow("select live_video_id from assets where path = '/p/IMG_E0001.HEIC'").Scan(&live)
	if live == nil || *live != 2 {
		t.Fatalf("paired by content id = %v, want 2", live)
	}
	a.db.QueryRow("select live_video_id from assets where path = '/p/other.jpg'").Scan(&live)
	if live != nil {
		t.Fatal("unrelated photo paired")
	}
}

func TestMountSource(t *testing.T) {
	mi := `22 1 0:21 / /proc rw - proc proc rw
25 1 179:2 / / rw,noatime shared:1 - ext4 /dev/mmcblk0p2 rw
31 25 8:1 / /mnt/ssd rw,relatime shared:9 - ext4 /dev/sda1 rw
40 25 0:45 / /mnt/nas rw - cifs //nas/photos rw`
	for _, c := range []struct {
		maj, min uint32
		want     string
	}{{179, 2, "/dev/mmcblk0p2"}, {8, 1, "/dev/sda1"}, {0, 45, "//nas/photos"}, {9, 9, ""}} {
		if got := mountSource(mi, c.maj, c.min); got != c.want {
			t.Errorf("mountSource(%d:%d) = %q, want %q", c.maj, c.min, got, c.want)
		}
	}
	if w := sdCardWarning(t.TempDir()); w != "" { // the build machine has no SD card
		t.Errorf("unexpected warning on this machine: %s", w)
	}
}

func TestCaptureDetails(t *testing.T) {
	for _, tool := range []string{"exiftool", "ffmpeg"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s not installed", tool)
		}
	}
	lib := t.TempDir()
	p := filepath.Join(lib, "shot.jpg")
	mustRun(t, "ffmpeg", "-v", "error", "-f", "lavfi", "-i", "testsrc=size=64x48", "-frames:v", "1", p)
	mustRun(t, "exiftool", "-q", "-overwrite_original", "-FNumber=1.8", "-ExposureTime=1/120", "-ISO=100", "-FocalLength=6.9", "-Flash#=16", p) // # = numeric code (16: off, did not fire)
	a, srv := newTestApp(t)
	ann := client(t)
	post(t, ann, srv.URL+"/api/setup", `{"username":"ann","password":"correct horse","path":"`+lib+`"}`)
	waitScan(t, a)
	var d map[string]string
	get(t, ann, srv.URL+"/api/assets/1/details", &d)
	if d["FNumber"] != "1.8" || d["ExposureTime"] != "1/120" || d["ISO"] != "100" || !strings.HasPrefix(d["FocalLength"], "6.9") || d["Flash"] == "" {
		t.Fatalf("details = %v", d)
	}
	if r, _ := client(t).Get(srv.URL + "/api/assets/1/details"); r.StatusCode != 401 {
		t.Errorf("details without a session = %d", r.StatusCode)
	}
}

func TestCheckFolderNamesTheBlockingFolder(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root passes through every folder")
	}
	home := filepath.Join(t.TempDir(), "home") // like Ubuntu's private /home/<user> (mode 750, another owner)
	photos := filepath.Join(home, "Pictures", "Phone")
	if err := os.MkdirAll(photos, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := checkFolder(photos); got != "" {
		t.Fatalf("readable folder: %q", got)
	}
	if err := os.Chmod(home, 0o600); err != nil { // no x: can't pass through
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(home, 0o755) })
	got := checkFolder(photos)
	if !strings.Contains(got, "isn't allowed through "+home+" ") {
		t.Fatalf("got %q, want it to name %s", got, home)
	}
	if got := checkFolder(filepath.Join(filepath.Dir(home), "nope")); !strings.Contains(got, "doesn't exist") {
		t.Fatalf("missing folder: %q", got)
	}
}

func TestSlideshow(t *testing.T) {
	a, srv := newTestApp(t)
	ann := client(t)
	post(t, ann, srv.URL+"/api/setup", `{"username":"ann","password":"correct horse","path":"`+t.TempDir()+`"}`)
	waitScan(t, a)
	add := func(lib int, name, typ, extra string) {
		a.db.Exec(`insert into assets (library_id, path, dir, size, mtime, quick_hash, type, mime, taken_local, thumbs_at)
			values (?, ?, '/p', 1, 1, x'00', ?, 'image/jpeg', '2024-01-01 00:00:00', 1)`, lib, "/p/"+name, typ)
		if extra != "" {
			a.db.Exec("update assets set "+extra+" where path = ?", "/p/"+name)
		}
	}
	for i := 0; i < 60; i++ {
		add(1, fmt.Sprintf("ok%02d.jpg", i), "image", "")
	}
	add(1, "clip.mp4", "video", "")
	add(1, "archived.jpg", "image", "is_archived = 1")
	add(1, "gone.jpg", "image", "missing_at = 1")
	add(1, "broken.jpg", "image", "thumb_error = 'bad'")
	add(1, "noprev.jpg", "image", "thumbs_at = null")
	hash, _ := hashPassword("another pass")
	a.db.Exec("insert into users (username, password_hash) values ('bob', ?)", hash)
	a.db.Exec(`insert into libraries (owner_id, name, import_paths) values (2, 'bob', '[]')`)
	add(2, "bobs.jpg", "image", "")

	names := map[int64]string{}
	rows, _ := a.db.Query("select id, path from assets")
	for rows.Next() {
		var id int64
		var p string
		rows.Scan(&id, &p)
		names[id] = filepath.Base(p)
	}
	rows.Close()
	seen := map[string]bool{}
	for round := 0; round < 4; round++ {
		var got []tile
		get(t, ann, srv.URL+"/api/slideshow", &got)
		if len(got) != 50 {
			t.Fatalf("round %d: %d photos, want a batch of 50", round, len(got))
		}
		for _, p := range got {
			n := names[p.ID]
			if !strings.HasPrefix(n, "ok") {
				t.Fatalf("slideshow showed %s", n)
			}
			seen[n] = true
		}
	}
	if len(seen) <= 50 {
		t.Errorf("4 batches covered only %d of 60 photos: not random", len(seen))
	}
	if r, _ := http.Get(srv.URL + "/api/slideshow"); r.StatusCode != 401 {
		t.Errorf("signed out: %d, want 401", r.StatusCode)
	}
}

// Behind a proxy on the same machine, wrong passwords are counted per visitor, not for everyone at once.
func TestLoginRateLimitBehindProxy(t *testing.T) {
	a, srv := newTestApp(t)
	a.loginLimit = newLimiter(3)
	try := func(xff string) int {
		req, _ := http.NewRequest("POST", srv.URL+"/api/login", strings.NewReader(`{"username":"x","password":"y"}`))
		req.Header.Set("Content-Type", "application/json")
		if xff != "" {
			req.Header.Set("X-Forwarded-For", xff)
		}
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		r.Body.Close()
		return r.StatusCode
	}
	for i := 0; i < 3; i++ {
		try("6.6.6.6, 203.0.113.5") // the proxy appends the address it saw; the left part is whatever the visitor sent
	}
	if code := try("203.0.113.5"); code != 429 {
		t.Fatalf("4th attempt from the same visitor = %d, want 429", code)
	}
	if code := try("198.51.100.7"); code != 401 {
		t.Fatalf("another visitor = %d, want 401 (not locked out by someone else)", code)
	}
	if code := try("9.9.9.9, 198.51.100.7"); code != 401 {
		t.Fatalf("faking an earlier hop must not matter: %d", code)
	}
}

// A family shares one photo folder: a second person joins the admin's library instead of indexing it again.
func TestSharedLibrary(t *testing.T) {
	a, srv := newTestApp(t)
	ann := client(t)
	post(t, ann, srv.URL+"/api/setup", `{"username":"ann","password":"correct horse","path":"`+t.TempDir()+`"}`)
	waitScan(t, a)
	for i := 0; i < 3; i++ {
		a.db.Exec(`insert into assets (library_id, path, dir, size, mtime, quick_hash, type, mime, taken_local, thumbs_at)
			values (1, ?, '/p', 1, 1, x'00', 'image', 'image/jpeg', '2024-01-01 00:00:00', 1)`, fmt.Sprintf("/p/%d.jpg", i))
	}
	if code := post(t, ann, srv.URL+"/api/users", `{"username":"bob","password":"another pass","is_admin":true,"library_id":1}`); code != 200 {
		t.Fatalf("add bob sharing library 1: %d", code)
	}
	var libs, assets int
	a.db.QueryRow("select count(*) from libraries").Scan(&libs)
	a.db.QueryRow("select count(*) from assets").Scan(&assets)
	if libs != 1 || assets != 3 {
		t.Fatalf("sharing made %d libraries and %d assets, want 1 and 3 (nothing indexed twice)", libs, assets)
	}
	if code := post(t, ann, srv.URL+"/api/users", `{"username":"eve","password":"another pass","library_id":99}`); code != 404 {
		t.Fatalf("sharing a library you can't see = %d, want 404", code)
	}
	bob := client(t)
	post(t, bob, srv.URL+"/api/login", `{"username":"bob","password":"another pass"}`)
	var month []tile
	get(t, bob, srv.URL+"/api/timeline?month=2024-01", &month)
	if len(month) != 3 {
		t.Fatalf("bob sees %d photos, want the 3 shared ones", len(month))
	}
	// favourites are the family's: bob's heart shows for ann
	patch(t, bob, srv.URL+"/api/assets", `{"ids":[`+itoa(month[0].ID)+`],"favorite":true}`)
	var favs []tile
	get(t, ann, srv.URL+"/api/assets?favorite=1", &favs)
	if len(favs) != 1 {
		t.Fatalf("ann sees %d favourites, want bob's 1", len(favs))
	}
	var mine []map[string]any
	get(t, bob, srv.URL+"/api/libraries", &mine)
	if len(mine) != 0 {
		t.Fatalf("bob can edit %d libraries, want 0 (only the owner edits)", len(mine))
	}
	if code := patch(t, bob, srv.URL+"/api/libraries/1", `{"name":"x"}`); code != 403 && code != 404 {
		t.Fatalf("bob renaming ann's library = %d, want refused", code)
	}
	// deleting the owner hands the library to bob: photos and favourites stay
	if code := del(t, bob, srv.URL+"/api/users/1", ""); code != 204 {
		t.Fatalf("delete ann: %d", code)
	}
	var owner int64
	a.db.QueryRow("select count(*) from assets").Scan(&assets)
	a.db.QueryRow("select owner_id from libraries where id = 1").Scan(&owner)
	if assets != 3 || owner != 2 {
		t.Fatalf("after deleting the owner: %d assets, owner %d; want 3 and bob (2)", assets, owner)
	}
	get(t, bob, srv.URL+"/api/assets?favorite=1", &favs)
	if len(favs) != 1 {
		t.Fatalf("bob lost the family favourite")
	}
}

// A video browsers stutter on gets an H.264 playback copy, served by /play; a plain H.264 one is left alone.
func TestPlaybackCopy(t *testing.T) {
	asIs := probe{}
	json.Unmarshal([]byte(`{"streams":[{"codec_name":"h264","pix_fmt":"yuv420p","width":1920,"height":1080}],
		"format":{"format_name":"mov,mp4,m4a,3gp,3g2,mj2","bit_rate":"17000000"}}`), &asIs)
	hevc, uhd, tenBit, heavy, avi := asIs, asIs, asIs, asIs, asIs
	hevc.Streams = append(hevc.Streams[:0:0], hevc.Streams...)
	hevc.Streams[0].Codec = "hevc"
	uhd.Streams = append(uhd.Streams[:0:0], uhd.Streams...)
	uhd.Streams[0].Width, uhd.Streams[0].Height = 2160, 3840
	tenBit.Streams = append(tenBit.Streams[:0:0], tenBit.Streams...)
	tenBit.Streams[0].PixFmt = "yuv420p10le"
	heavy.Format.BitRate = "60000000"
	avi.Format.Name = "avi"
	if !asIs.playsAsIs() || hevc.playsAsIs() || uhd.playsAsIs() || tenBit.playsAsIs() || heavy.playsAsIs() || avi.playsAsIs() {
		t.Fatal("playsAsIs picks the wrong videos")
	}

	for _, tool := range []string{"ffmpeg", "ffprobe"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s not installed", tool)
		}
	}
	dir := t.TempDir()
	db, err := openDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	a := &App{db: db, dataDir: dir}
	old, plain := filepath.Join(dir, "old.avi"), filepath.Join(dir, "plain.mp4")
	mustRun(t, "ffmpeg", "-v", "error", "-f", "lavfi", "-i", "testsrc=size=1920x1080:duration=1", "-c:v", "mpeg4", old)
	mustRun(t, "ffmpeg", "-v", "error", "-f", "lavfi", "-i", "testsrc=size=320x240:duration=1", "-pix_fmt", "yuv420p", plain)
	if err := a.makePlay(context.Background(), 1, old); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=codec_name,width,height", "-of", "csv=p=0", a.playPath(1)).Output()
	if err != nil || strings.TrimSpace(string(out)) != "h264,1280,720" {
		t.Fatalf("playback copy is %q (%v), want h264,1280,720", out, err)
	}
	os.WriteFile(a.playPath(2), []byte("stale"), 0o644)
	if err := a.makePlay(context.Background(), 2, plain); err != nil || fileExists(a.playPath(2)) {
		t.Fatalf("plain H.264 video: err %v, copy exists %v", err, fileExists(a.playPath(2)))
	}

	// /play serves the copy once the row says it was checked, the original before that
	w := httptest.NewRecorder()
	a.servePlay(w, httptest.NewRequest("GET", "/x", nil), &assetRow{tile: tile{ID: 1}, playAt: sql.NullInt64{Int64: 1, Valid: true}})
	if w.Code != 200 || w.Header().Get("Content-Type") != "video/mp4" || w.Body.Len() == 0 {
		t.Fatalf("play: %d %s", w.Code, w.Header().Get("Content-Type"))
	}
}

// The thermal guard freezes a conversion while the CPU is hot and lets it go on once it has cooled.
func TestKeepCool(t *testing.T) {
	zone := filepath.Join(t.TempDir(), "temp")
	defer func(z string, d time.Duration) { thermalZone, thermalTick = z, d }(thermalZone, thermalTick)
	thermalZone, thermalTick = zone, 10*time.Millisecond
	os.WriteFile(zone, []byte("82000\n"), 0o644)
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Skip(err)
	}
	defer cmd.Process.Kill()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go keepCool(ctx, cmd.Process)
	state := func() string { // third field of /proc/<pid>/stat: T = stopped
		b, _ := os.ReadFile(fmt.Sprintf("/proc/%d/stat", cmd.Process.Pid))
		f := strings.Fields(string(b))
		if len(f) < 3 {
			t.Skip("no /proc")
		}
		return f[2]
	}
	waitFor := func(want bool, what string) {
		for i := 0; i < 200; i++ {
			if (state() == "T") == want {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatalf("conversion not %s (state %s)", what, state())
	}
	waitFor(true, "paused at 82 °C")
	os.WriteFile(zone, []byte("70000\n"), 0o644) // between the two limits: stays paused
	time.Sleep(50 * time.Millisecond)
	waitFor(true, "still paused at 70 °C")
	os.WriteFile(zone, []byte("64000\n"), 0o644)
	waitFor(false, "resumed at 64 °C")
}

// The admin switch stops the running conversion, keeps the worker idle, and comes back on.
func TestVideoCopiesSwitch(t *testing.T) {
	a := &App{dataDir: t.TempDir()}
	stopped := false
	stop := context.CancelFunc(func() { stopped = true })
	a.playStop.Store(&stop)
	set := func(body string) int {
		r := httptest.NewRequest("PATCH", "/api/jobs", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		a.patchJobs(w, r)
		return w.Code
	}
	if set(`{"video_copies":false}`) != 204 || !fileExists(a.copiesOffFile()) || !stopped {
		t.Fatalf("off: file %v, running conversion stopped %v", fileExists(a.copiesOffFile()), stopped)
	}
	if set(`{"video_copies":true}`) != 204 || set(`{"video_copies":true}`) != 204 || fileExists(a.copiesOffFile()) {
		t.Fatal("on: the switch file is still there")
	}
}
