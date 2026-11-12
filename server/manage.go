package main

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
)

// ---- favourites / archive

func (a *App) getAssets(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	// shortcut: no paging, capped at 5000; page like the timeline if libraries get bigger favourites lists
	const own = `from assets a join libraries l on l.id = a.library_id where l.owner_id = ? and a.missing_at is null`
	switch {
	case q.Get("favorite") == "1":
		a.writeTiles(w, `select `+tileCols+` `+own+` and a.is_favorite = 1 and a.is_archived = 0
			order by a.taken_local desc, a.id desc limit 5000`, userFrom(r).ID)
	case q.Get("archived") == "1":
		a.writeTiles(w, `select `+tileCols+` `+own+` and a.is_archived = 1 order by a.taken_local desc, a.id desc limit 5000`, userFrom(r).ID)
	default:
		httpError(w, http.StatusBadRequest, "Ask for favorite=1 or archived=1")
	}
}

func (a *App) patchAssets(w http.ResponseWriter, r *http.Request) {
	var in struct {
		IDs      []int64 `json:"ids"`
		Favorite *bool   `json:"favorite"`
		Archived *bool   `json:"archived"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if len(in.IDs) == 0 || len(in.IDs) > 5000 || (in.Favorite == nil && in.Archived == nil) {
		httpError(w, http.StatusBadRequest, "Send 1 to 5000 ids and a field to change")
		return
	}
	var set []string
	args := []any{}
	if in.Favorite != nil {
		set, args = append(set, "is_favorite = ?"), append(args, *in.Favorite)
	}
	if in.Archived != nil {
		set, args = append(set, "is_archived = ?"), append(args, *in.Archived)
	}
	for _, id := range in.IDs {
		args = append(args, id)
	}
	args = append(args, userFrom(r).ID)
	// the owner check is in the same statement: ids from other users' libraries are silently skipped
	res, err := a.db.Exec(`update assets set `+strings.Join(set, ", ")+`, updated_at = unixepoch()
		where id in (?`+strings.Repeat(",?", len(in.IDs)-1)+`) and library_id in (select id from libraries where owner_id = ?)`, args...)
	if err != nil {
		serverError(w, err)
		return
	}
	n, _ := res.RowsAffected()
	writeJSON(w, map[string]int64{"updated": n})
}

// ---- folders

type folder struct {
	Path  string `json:"path"`
	Name  string `json:"name"`
	Count int    `json:"count"`
	Cover *tile  `json:"cover,omitempty"`
}

// getFolders lists one directory: its subfolders (with photo counts) and the photos directly in it.
// With no path it lists the library roots.
func (a *App) getFolders(w http.ResponseWriter, r *http.Request) {
	uid := userFrom(r).ID
	libs, err := a.ownLibraries(uid)
	if err != nil {
		serverError(w, err)
		return
	}
	p := r.URL.Query().Get("path")
	if p == "" {
		out := []folder{}
		for _, l := range libs {
			for _, root := range l.Paths {
				f, err := a.folderSummary(l.ID, root)
				if err != nil {
					serverError(w, err)
					return
				}
				f.Name = root
				out = append(out, f)
			}
		}
		writeJSON(w, map[string]any{"path": "", "dirs": out, "assets": []tile{}})
		return
	}
	p = filepath.Clean(p)
	var libID int64
	var root string
	for _, l := range libs {
		for _, rt := range l.Paths {
			if underAny(p, []string{rt}) {
				libID, root = l.ID, rt
			}
		}
	}
	if libID == 0 || !filepath.IsAbs(p) {
		httpError(w, http.StatusNotFound, "That folder isn't in your libraries")
		return
	}
	// all indexed dirs below p (range scan on the assets_dir index), folded into immediate children
	prefix := strings.TrimSuffix(p, "/") + "/"
	rows, err := a.db.Query(`select dir, count(*) from assets where library_id = ? and missing_at is null
		and dir >= ? and dir < ? group by dir`, libID, prefix, strings.TrimSuffix(prefix, "/")+"0") // '0' sorts right after '/'
	if err != nil {
		serverError(w, err)
		return
	}
	counts := map[string]int{}
	for rows.Next() {
		var d string
		var n int
		if err := rows.Scan(&d, &n); err != nil {
			rows.Close()
			serverError(w, err)
			return
		}
		child := prefix + strings.SplitN(strings.TrimPrefix(d, prefix), "/", 2)[0]
		counts[child] += n
	}
	rows.Close()
	dirs := []folder{}
	for c, n := range counts {
		f := folder{Path: c, Name: filepath.Base(c), Count: n}
		covers, err := a.queryTiles(`select `+tileCols+` from assets a where a.library_id = ? and a.missing_at is null
			and (a.dir = ? or (a.dir >= ? and a.dir < ?)) and a.thumb_error is null order by a.taken_local desc limit 1`,
			libID, c, c+"/", c+"0")
		if err != nil {
			serverError(w, err)
			return
		}
		if len(covers) > 0 {
			f.Cover = &covers[0]
		}
		dirs = append(dirs, f)
	}
	sort.Slice(dirs, func(i, j int) bool { return strings.ToLower(dirs[i].Name) < strings.ToLower(dirs[j].Name) })
	assets, err := a.queryTiles(`select `+tileCols+` from assets a where a.library_id = ? and a.dir = ? and a.missing_at is null
		order by a.taken_local desc, a.id desc`, libID, p)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, map[string]any{"path": p, "root": root, "dirs": dirs, "assets": assets})
}

func (a *App) folderSummary(libID int64, dir string) (folder, error) {
	f := folder{Path: dir, Name: filepath.Base(dir)}
	if err := a.db.QueryRow(`select count(*) from assets where library_id = ? and missing_at is null
		and (dir = ? or (dir >= ? and dir < ?))`, libID, dir, dir+"/", dir+"0").Scan(&f.Count); err != nil {
		return f, err
	}
	covers, err := a.queryTiles(`select `+tileCols+` from assets a where a.library_id = ? and a.missing_at is null
		and (a.dir = ? or (a.dir >= ? and a.dir < ?)) and a.thumb_error is null order by a.taken_local desc limit 1`,
		libID, dir, dir+"/", dir+"0")
	if len(covers) > 0 {
		f.Cover = &covers[0]
	}
	return f, err
}

// ---- libraries

type library struct {
	ID            int64    `json:"id"`
	Name          string   `json:"name"`
	Paths         []string `json:"import_paths"`
	Globs         []string `json:"exclusion_globs"`
	LastScanAt    *int64   `json:"last_scan_at"`
	LastScanError *string  `json:"last_scan_error"`
	Assets        int      `json:"assets"`
	Missing       int      `json:"missing"`
}

func (a *App) ownLibraries(uid int64) ([]library, error) {
	rows, err := a.db.Query(`select l.id, l.name, l.import_paths, l.exclusion_globs, l.last_scan_at, l.last_scan_error,
		(select count(*) from assets where library_id = l.id and missing_at is null),
		(select count(*) from assets where library_id = l.id and missing_at is not null)
		from libraries l where l.owner_id = ? order by l.id`, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []library{}
	for rows.Next() {
		var l library
		var paths, globs string
		if err := rows.Scan(&l.ID, &l.Name, &paths, &globs, &l.LastScanAt, &l.LastScanError, &l.Assets, &l.Missing); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(paths), &l.Paths)
		json.Unmarshal([]byte(globs), &l.Globs)
		out = append(out, l)
	}
	return out, rows.Err()
}

func (a *App) getLibraries(w http.ResponseWriter, r *http.Request) {
	libs, err := a.ownLibraries(userFrom(r).ID)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, libs)
}

func (a *App) patchLibrary(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	var in struct {
		Name  *string   `json:"name"`
		Paths *[]string `json:"import_paths"`
		Globs *[]string `json:"exclusion_globs"`
	}
	if !ok || !readJSON(w, r, &in) {
		if !ok {
			httpError(w, http.StatusNotFound, "No such library")
		}
		return
	}
	var owner int64
	if err := a.db.QueryRow("select owner_id from libraries where id = ?", id).Scan(&owner); err != nil || owner != userFrom(r).ID {
		httpError(w, http.StatusNotFound, "No such library")
		return
	}
	// validate everything first, then write in one transaction: no half-applied settings
	var sets []string
	var args []any
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" || len(name) > 100 {
			httpError(w, http.StatusBadRequest, "Give the library a name (up to 100 characters)")
			return
		}
		sets, args = append(sets, "name = ?"), append(args, name)
	}
	if in.Paths != nil {
		var clean []string
		seen := map[string]bool{}
		for _, p := range *in.Paths {
			if p = strings.TrimSpace(p); p == "" {
				continue
			}
			if msg := checkFolder(p); msg != "" {
				httpError(w, http.StatusBadRequest, p+": "+msg)
				return
			}
			if p = filepath.Clean(p); !seen[p] {
				seen[p] = true
				clean = append(clean, p)
			}
		}
		if len(clean) == 0 {
			httpError(w, http.StatusBadRequest, "Keep at least one folder")
			return
		}
		js, _ := json.Marshal(clean)
		sets, args = append(sets, "import_paths = ?"), append(args, string(js))
	}
	if in.Globs != nil {
		globs := []string{}
		for _, g := range *in.Globs {
			if g = strings.TrimSpace(g); g != "" {
				globs = append(globs, g)
			}
		}
		if _, err := compileGlobs(globs); err != nil {
			httpError(w, http.StatusBadRequest, err.Error())
			return
		}
		js, _ := json.Marshal(globs)
		sets, args = append(sets, "exclusion_globs = ?"), append(args, string(js))
	}
	if len(sets) > 0 {
		if _, err := a.db.Exec("update libraries set "+strings.Join(sets, ", ")+", updated_at = unixepoch() where id = ?",
			append(args, id)...); err != nil {
			serverError(w, err)
			return
		}
	}
	go a.scanAll()
	a.getLibraries(w, r)
}
