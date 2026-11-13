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
		"create index assets_content  on assets(content_id) where content_id is not null;\n", "").Replace(schemaSQL)
	if v1 == schemaSQL {
		t.Fatal("couldn't rebuild the v1 schema from schema.sql")
	}
	old, err := sql.Open("sqlite", "file:"+filepath.Join(dir, "raspicture.db"))
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
	var v int
	var fav bool
	var cid *string
	db.QueryRow("pragma user_version").Scan(&v)
	if err := db.QueryRow("select is_favorite, content_id from assets").Scan(&fav, &cid); err != nil || v != schemaVersion || !fav {
		t.Fatalf("after upgrade: version %d, fav %v, err %v", v, fav, err)
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
