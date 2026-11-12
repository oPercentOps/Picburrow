package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func (a *App) routes() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /api/status", a.getStatus)
	m.HandleFunc("POST /api/setup", a.postSetup)
	m.HandleFunc("POST /api/login", a.postLogin)
	m.HandleFunc("POST /api/logout", a.postLogout)
	m.HandleFunc("GET /api/me", a.auth(false, func(w http.ResponseWriter, r *http.Request) { writeJSON(w, userFrom(r)) }))
	m.HandleFunc("GET /api/timeline/buckets", a.auth(false, a.getBuckets))
	m.HandleFunc("GET /api/timeline", a.auth(false, a.getTimeline))
	m.HandleFunc("GET /api/assets/{id}", a.auth(false, a.getAsset))
	m.HandleFunc("GET /api/assets/{id}/thumb/{size}", a.auth(false, a.getThumb))
	m.HandleFunc("GET /api/assets/{id}/original", a.auth(false, a.getOriginal))
	m.HandleFunc("GET /api/jobs", a.auth(false, a.getJobs))
	m.HandleFunc("POST /api/libraries/{id}/scan", a.auth(true, a.postScan))
	m.HandleFunc("GET /api/assets", a.auth(false, a.getAssets))
	m.HandleFunc("PATCH /api/assets", a.auth(false, a.patchAssets))
	m.HandleFunc("GET /api/folders", a.auth(false, a.getFolders))
	m.HandleFunc("GET /api/libraries", a.auth(true, a.getLibraries))
	m.HandleFunc("PATCH /api/libraries/{id}", a.auth(true, a.patchLibrary))
	m.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) { httpError(w, http.StatusNotFound, "No such API route") })
	m.Handle("/", a.spa())
	return securityHeaders(m)
}

func securityHeaders(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer") // share-link tokens live in URLs
		w.Header().Set("X-Frame-Options", "DENY")
		h.ServeHTTP(w, r)
	})
}

// ---- helpers

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(v)
}

func httpError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func serverError(w http.ResponseWriter, err error) {
	log.Printf("error: %v", err)
	httpError(w, http.StatusInternalServerError, "Something went wrong on the server")
}

// readJSON also acts as CSRF protection: a cross-site form can't send application/json without a CORS preflight.
func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if ct, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type")); ct != "application/json" {
		httpError(w, http.StatusUnsupportedMediaType, "Expected JSON")
		return false
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(v); err != nil {
		httpError(w, http.StatusBadRequest, "Invalid JSON")
		return false
	}
	return true
}

func pathID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	return id, err == nil && id > 0
}

// ---- auth and setup

func (a *App) getStatus(w http.ResponseWriter, r *http.Request) {
	var n int
	if err := a.db.QueryRow("select count(*) from users").Scan(&n); err != nil {
		serverError(w, err)
		return
	}
	u, ok := a.currentUser(r)
	res := map[string]any{"needs_setup": n == 0}
	if ok {
		res["user"] = u
	}
	writeJSON(w, res)
}

func (a *App) postSetup(w http.ResponseWriter, r *http.Request) {
	var in struct{ Username, Password, Path string }
	if !readJSON(w, r, &in) {
		return
	}
	in.Username = strings.TrimSpace(in.Username)
	if !validUsername(in.Username) {
		httpError(w, http.StatusBadRequest, "Pick a username without spaces")
		return
	}
	if len(in.Password) < 8 {
		httpError(w, http.StatusBadRequest, "Use at least 8 characters for the password")
		return
	}
	if msg := checkFolder(in.Path); msg != "" {
		httpError(w, http.StatusBadRequest, msg)
		return
	}
	hash, err := hashPassword(in.Password)
	if err != nil {
		serverError(w, err)
		return
	}
	tx, err := a.db.Begin()
	if err != nil {
		serverError(w, err)
		return
	}
	defer tx.Rollback()
	// "where not exists" makes setup one-shot even if two browsers race it
	res, err := tx.Exec(`insert into users (username, password_hash, is_admin) select ?, ?, 1
		where not exists (select 1 from users)`, in.Username, hash)
	if err != nil {
		serverError(w, err)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		httpError(w, http.StatusConflict, "Setup is already done. Sign in instead.")
		return
	}
	uid, _ := res.LastInsertId()
	paths, _ := json.Marshal([]string{filepath.Clean(in.Path)})
	if _, err := tx.Exec("insert into libraries (owner_id, name, import_paths) values (?, ?, ?)",
		uid, filepath.Base(filepath.Clean(in.Path)), string(paths)); err != nil {
		serverError(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		serverError(w, err)
		return
	}
	if err := a.startSession(w, r, uid); err != nil {
		serverError(w, err)
		return
	}
	go a.scanAll()
	writeJSON(w, User{ID: uid, Username: in.Username, IsAdmin: true})
}

// checkFolder returns a user-facing problem with an import path, or "".
func checkFolder(p string) string {
	if !filepath.IsAbs(p) {
		return "Use a full path that starts with /"
	}
	st, err := os.Stat(p)
	if err != nil {
		return "That folder doesn't exist or the server can't reach it"
	}
	if !st.IsDir() {
		return "That's a file, not a folder"
	}
	f, err := os.Open(p)
	if err != nil {
		return "The server isn't allowed to read that folder"
	}
	f.Close()
	return ""
}

func (a *App) postLogin(w http.ResponseWriter, r *http.Request) {
	if !a.loginLimit.allow(r) {
		httpError(w, http.StatusTooManyRequests, "Too many attempts. Wait a minute and try again.")
		return
	}
	var in struct{ Username, Password string }
	if !readJSON(w, r, &in) {
		return
	}
	var u User
	var hash string
	err := a.db.QueryRow("select id, username, is_admin, password_hash from users where username = ?",
		strings.TrimSpace(in.Username)).Scan(&u.ID, &u.Username, &u.IsAdmin, &hash)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		serverError(w, err)
		return
	}
	if !checkPassword(hash, in.Password) {
		httpError(w, http.StatusUnauthorized, "Wrong username or password")
		return
	}
	if err := a.startSession(w, r, u.ID); err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, u)
}

func (a *App) postLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		a.db.Exec("delete from sessions where token_hash = ?", hashToken(c.Value))
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	w.WriteHeader(http.StatusNoContent)
}

// ---- timeline

// visible = assets the user owns that belong on the timeline.
const visible = `from assets a join libraries l on l.id = a.library_id
	where l.owner_id = ? and a.missing_at is null and a.is_archived = 0`

func (a *App) getBuckets(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.Query(`select substr(a.taken_local, 1, 7) as m, count(*) `+visible+` group by m order by m desc`, userFrom(r).ID)
	if err != nil {
		serverError(w, err)
		return
	}
	defer rows.Close()
	type bucket struct {
		Month string `json:"month"`
		Count int    `json:"count"`
	}
	out := []bucket{}
	for rows.Next() {
		var b bucket
		if err := rows.Scan(&b.Month, &b.Count); err != nil {
			serverError(w, err)
			return
		}
		out = append(out, b)
	}
	writeJSON(w, out)
}

type tile struct {
	ID       int64  `json:"id"`
	W        *int   `json:"w"`
	H        *int   `json:"h"`
	Type     string `json:"type"`
	Taken    string `json:"taken"`
	Fav      bool   `json:"fav"`
	Duration *int   `json:"duration_ms,omitempty"`
	V        *int64 `json:"v"` // thumbs version for cache busting; null = not generated yet
	Err      bool   `json:"err,omitempty"`
}

func (a *App) getTimeline(w http.ResponseWriter, r *http.Request) {
	month := r.URL.Query().Get("month")
	t, err := time.Parse("2006-01", month)
	if err != nil {
		httpError(w, http.StatusBadRequest, "month must look like 2024-05")
		return
	}
	next := t.AddDate(0, 1, 0).Format("2006-01")
	a.writeTiles(w, `select `+tileCols+` `+visible+` and a.taken_local >= ? and a.taken_local < ?
		order by a.taken_local desc, a.id desc`, userFrom(r).ID, month, next)
}

const tileCols = `a.id, a.width, a.height, a.type, a.taken_local, a.is_favorite, a.duration_ms, a.thumbs_at, a.thumb_error is not null`

func (a *App) queryTiles(q string, args ...any) ([]tile, error) {
	rows, err := a.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []tile{}
	for rows.Next() {
		var t tile
		if err := rows.Scan(&t.ID, &t.W, &t.H, &t.Type, &t.Taken, &t.Fav, &t.Duration, &t.V, &t.Err); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// writeTiles runs a query selecting tileCols and writes the tiles as JSON.
func (a *App) writeTiles(w http.ResponseWriter, q string, args ...any) {
	out, err := a.queryTiles(q, args...)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, out)
}

// ---- assets

type assetRow struct {
	tile
	Path        string   `json:"path"`
	Dir         string   `json:"dir"`
	Name        string   `json:"name"`
	Size        int64    `json:"size"`
	Mime        string   `json:"mime"`
	TzOffsetMin *int     `json:"tz_offset_min"`
	Make        *string  `json:"camera_make"`
	Model       *string  `json:"camera_model"`
	Lens        *string  `json:"lens"`
	Lat         *float64 `json:"lat"`
	Lon         *float64 `json:"lon"`
	Prev        *int64   `json:"prev_id"` // newer
	Next        *int64   `json:"next_id"` // older
	libraryID   int64
}

func (a *App) loadAsset(userID, id int64) (*assetRow, error) {
	var x assetRow
	err := a.db.QueryRow(`select a.id, a.width, a.height, a.type, a.taken_local, a.is_favorite, a.duration_ms, a.thumbs_at,
		a.thumb_error is not null, a.path, a.dir, a.size, a.mime, a.tz_offset_min, a.camera_make, a.camera_model, a.lens,
		a.lat, a.lon, a.library_id
		from assets a join libraries l on l.id = a.library_id where a.id = ? and l.owner_id = ?`, id, userID).
		Scan(&x.ID, &x.W, &x.H, &x.Type, &x.Taken, &x.Fav, &x.Duration, &x.V, &x.Err, &x.Path, &x.Dir, &x.Size, &x.Mime,
			&x.TzOffsetMin, &x.Make, &x.Model, &x.Lens, &x.Lat, &x.Lon, &x.libraryID)
	if err != nil {
		return nil, err
	}
	x.Name = filepath.Base(x.Path)
	return &x, nil
}

func (a *App) getAsset(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		httpError(w, http.StatusNotFound, "No such photo")
		return
	}
	uid := userFrom(r).ID
	x, err := a.loadAsset(uid, id)
	if errors.Is(err, sql.ErrNoRows) {
		httpError(w, http.StatusNotFound, "No such photo")
		return
	} else if err != nil {
		serverError(w, err)
		return
	}
	// neighbours in timeline order (taken_local desc, id desc)
	a.db.QueryRow(`select a.id `+visible+` and (a.taken_local > ? or (a.taken_local = ? and a.id > ?))
		order by a.taken_local asc, a.id asc limit 1`, uid, x.Taken, x.Taken, x.ID).Scan(&x.Prev)
	a.db.QueryRow(`select a.id `+visible+` and (a.taken_local < ? or (a.taken_local = ? and a.id < ?))
		order by a.taken_local desc, a.id desc limit 1`, uid, x.Taken, x.Taken, x.ID).Scan(&x.Next)
	writeJSON(w, x)
}

func (a *App) getThumb(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	size := r.PathValue("size")
	if _, known := thumbSizes[size]; !ok || !known {
		httpError(w, http.StatusNotFound, "No such thumbnail")
		return
	}
	x, err := a.loadAsset(userFrom(r).ID, id)
	if errors.Is(err, sql.ErrNoRows) {
		httpError(w, http.StatusNotFound, "No such photo")
		return
	} else if err != nil {
		serverError(w, err)
		return
	}
	if x.V == nil && !x.Err {
		if err := a.ensureThumbs(r.Context(), id); err != nil { // on demand: jumps the worker's queue
			httpError(w, http.StatusUnprocessableEntity, "Can't make a preview of this file")
			return
		}
	} else if x.Err {
		httpError(w, http.StatusUnprocessableEntity, "Can't make a preview of this file")
		return
	}
	f, err := os.Open(a.thumbPath(id, size))
	if err != nil {
		a.db.Exec("update assets set thumbs_at = null where id = ?", id) // thumbnail cache was wiped: regenerate
		httpError(w, http.StatusNotFound, "Preview not ready, try again")
		return
	}
	defer f.Close()
	st, _ := f.Stat()
	if r.URL.Query().Get("v") != "" { // versioned URL: content never changes
		w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "private, no-cache")
	}
	w.Header().Set("Content-Type", "image/webp")
	http.ServeContent(w, r, "", st.ModTime(), f)
}

func (a *App) getOriginal(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		httpError(w, http.StatusNotFound, "No such photo")
		return
	}
	x, err := a.loadAsset(userFrom(r).ID, id)
	if errors.Is(err, sql.ErrNoRows) {
		httpError(w, http.StatusNotFound, "No such photo")
		return
	} else if err != nil {
		serverError(w, err)
		return
	}
	a.serveOriginal(w, r, x, r.URL.Query().Get("download") == "1")
}

// serveOriginal streams the file (Range support for video seeking), only if it still resolves inside an import path.
func (a *App) serveOriginal(w http.ResponseWriter, r *http.Request, x *assetRow, download bool) {
	var pathsJSON string
	var roots []string
	if err := a.db.QueryRow("select import_paths from libraries where id = ?", x.libraryID).Scan(&pathsJSON); err != nil {
		serverError(w, err)
		return
	}
	json.Unmarshal([]byte(pathsJSON), &roots)
	real, err := filepath.EvalSymlinks(x.Path)
	if err != nil {
		httpError(w, http.StatusNotFound, "The file is no longer on disk")
		return
	}
	inside := false
	for _, root := range roots {
		if rr, err := filepath.EvalSymlinks(root); err == nil && underAny(real, []string{rr}) {
			inside = true
		}
	}
	if !inside { // symlink escape or library paths changed
		httpError(w, http.StatusForbidden, "That file is outside the library folders")
		return
	}
	f, err := os.Open(real)
	if err != nil {
		httpError(w, http.StatusNotFound, "The file is no longer on disk")
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || st.IsDir() {
		httpError(w, http.StatusNotFound, "The file is no longer on disk")
		return
	}
	disp := "inline"
	if download {
		disp = "attachment"
	}
	w.Header().Set("Content-Disposition", mime.FormatMediaType(disp, map[string]string{"filename": x.Name}))
	w.Header().Set("Content-Type", x.Mime)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	http.ServeContent(w, r, "", st.ModTime(), f)
}

// ---- jobs

func (a *App) getJobs(w http.ResponseWriter, r *http.Request) {
	var done, todo, errs int
	a.db.QueryRow(`select count(*) filter (where thumbs_at is not null), count(*) filter (where thumbs_at is null and thumb_error is null),
		count(*) filter (where thumb_error is not null) from assets where missing_at is null`).Scan(&done, &todo, &errs)
	a.scan.mu.Lock()
	res := map[string]any{"scanning": a.scan.running, "scan_found": a.scan.found, "scan_error": a.scan.lastErr,
		"thumbs_done": done, "thumbs_todo": todo, "thumb_errors": errs}
	a.scan.mu.Unlock()
	if userFrom(r).IsAdmin { // file paths are admin-only detail
		type fileErr struct {
			Path  string `json:"path"`
			Error string `json:"error"`
		}
		errs := []fileErr{}
		rows, err := a.db.Query("select path, thumb_error from assets where thumb_error is not null and missing_at is null order by id desc limit 50")
		if err == nil {
			for rows.Next() {
				var e fileErr
				if rows.Scan(&e.Path, &e.Error) == nil {
					errs = append(errs, e)
				}
			}
			rows.Close()
		}
		res["errors"] = errs
	}
	writeJSON(w, res)
}

func (a *App) postScan(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	var n int
	if !ok || a.db.QueryRow("select count(*) from libraries where id = ?", id).Scan(&n) != nil || n == 0 {
		httpError(w, http.StatusNotFound, "No such library")
		return
	}
	go a.scanAll()
	w.WriteHeader(http.StatusAccepted)
}

// ---- SPA

func (a *App) spa() http.Handler {
	sub, _ := fs.Sub(webFS, "dist")
	files := http.FileServerFS(sub)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if p == "" {
			p = "index.html"
		}
		if st, err := fs.Stat(sub, p); err != nil || st.IsDir() {
			// client-side route: serve the app shell
			index, err := fs.ReadFile(sub, "index.html")
			if err != nil {
				http.Error(w, "Web UI not built. Run: make web", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-cache")
			w.Write(index)
			return
		}
		if strings.HasPrefix(p, "_app/immutable/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		files.ServeHTTP(w, r)
	})
}
