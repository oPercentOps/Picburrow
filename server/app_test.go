package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
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
