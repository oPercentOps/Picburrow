package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// ---- favourites / archive

func (a *App) getAssets(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	// shortcut: no paging, capped at 5000; page like the timeline if libraries get bigger favourites lists
	const own = `from assets a join libraries l on l.id = a.library_id where l.id in ` + mine + ` and a.missing_at is null and ` + notLiveClip
	switch {
	case q.Get("favorite") == "1":
		a.writeTiles(w, `select `+tileCols+` `+own+` and a.is_favorite = 1 and a.is_archived = 0
			order by a.taken_local desc, a.id desc limit 5000`, userFrom(r).ID)
	case q.Get("archived") == "1":
		a.writeTiles(w, `select `+tileCols+` `+own+` and a.is_archived = 1 order by a.taken_local desc, a.id desc limit 5000`, userFrom(r).ID)
	case q.Get("onthisday") != "":
		// today's date as the browser sees it (the server's clock may be in another time zone)
		day, err := time.Parse("2006-01-02", q.Get("onthisday"))
		if err != nil {
			httpError(w, http.StatusBadRequest, "onthisday must look like 2024-05-17")
			return
		}
		a.writeTiles(w, `select `+tileCols+` `+own+` and a.is_archived = 0 and substr(a.taken_local, 6, 5) = ?
			and a.taken_local < ? order by a.taken_local desc, a.id desc limit 5000`,
			userFrom(r).ID, day.Format("01-02"), day.Format("2006"))
	default:
		httpError(w, http.StatusBadRequest, "Ask for favorite=1, archived=1 or onthisday=<date>")
	}
}

// postLookup tells a phone which of its photos are not in the index yet (an iOS Shortcut sends name + time taken).
// A photo is there if one has the same time taken to the second, or has its name in its file name within a day either
// side: the phone may give the time in another time zone, and copy tools rename (IMG_0236.HEIC -> 2023-06-15_IMG_0236.jpeg).
func (a *App) postLookup(w http.ResponseWriter, r *http.Request) {
	type photo struct {
		Name  string `json:"name"`
		Taken string `json:"taken,omitempty"` // 2023-06-15 00:06:39, the clock time on the photo
	}
	var in struct {
		Photos []photo `json:"photos"`
		// or one photo per line, names and dates in the same order: a Shortcut makes these with no loop per photo
		Names string `json:"names"`
		Dates string `json:"dates"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	lines := func(s string) []string { return strings.Split(strings.TrimSpace(strings.ReplaceAll(s, "\r", "")), "\n") }
	if in.Names != "" && in.Dates == "" {
		a.lookupFiles(w, r, lines(in.Names))
		return
	}
	if in.Names != "" || in.Dates != "" {
		names, dates := lines(in.Names), lines(in.Dates)
		if len(names) != len(dates) {
			httpError(w, http.StatusBadRequest, "names and dates need one line per photo, in the same order")
			return
		}
		for i := range names {
			in.Photos = append(in.Photos, photo{names[i], dates[i]})
		}
	}
	if len(in.Photos) == 0 || len(in.Photos) > 5000 {
		httpError(w, http.StatusBadRequest, "Send 1 to 5000 photos")
		return
	}
	const layout = "2006-01-02 15:04:05"
	missing := []photo{}
	for _, p := range in.Photos {
		t, err := time.Parse(layout, strings.TrimSpace(p.Taken))
		if err != nil { // ISO 8601, what Shortcuts gives unformatted: the clock time in this server's time zone
			if t, err = time.Parse(time.RFC3339, strings.TrimSpace(p.Taken)); err == nil {
				t = t.In(time.Local)
			} else if t, err = time.Parse("2006-01-02T15:04:05", strings.TrimSpace(p.Taken)); err != nil {
				httpError(w, http.StatusBadRequest, "Dates must look like 2024-05-31 18:30:00 or 2024-05-31T18:30:00+02:00")
				return
			}
		}
		stem := strings.TrimSuffix(filepath.Base(p.Name), filepath.Ext(p.Name)) // the copy may be a JPEG of a HEIC
		if stem == "." || stem == "/" {
			stem = ""
		}
		like := "%" + strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(stem) + "%"
		var found int
		err = a.db.QueryRow(`select 1 from assets a where a.library_id in `+mine+` and a.missing_at is null
			and a.taken_local between ? and ? and (a.taken_local = ? or (? != '' and substr(a.path, length(a.dir) + 2) like ? escape '\')) limit 1`,
			userFrom(r).ID, t.Add(-26*time.Hour).Format(layout), t.Add(26*time.Hour).Format(layout), t.Format(layout), stem, like).Scan(&found)
		if errors.Is(err, sql.ErrNoRows) {
			missing = append(missing, p)
		} else if err != nil {
			serverError(w, err)
			return
		}
	}
	writeJSON(w, map[string]any{"checked": len(in.Photos), "missing": missing})
}

// lookupFiles is the lookup by file name alone, as the phone's copy tool names them (2023-06-15_000639_IMG_0236):
// a photo is there if a file has that name, whatever its extension, letter case or duplicate suffix (-2).
func (a *App) lookupFiles(w http.ResponseWriter, r *http.Request, names []string) {
	if len(names) > 5000 {
		httpError(w, http.StatusBadRequest, "Send 1 to 5000 photos")
		return
	}
	stem := func(s string) string {
		s = strings.ToLower(filepath.Base(strings.TrimSpace(s)))
		return strings.TrimSuffix(s, filepath.Ext(s))
	}
	rows, err := a.db.Query(`select substr(a.path, length(a.dir) + 2) from assets a
		where a.library_id in `+mine+` and a.missing_at is null`, userFrom(r).ID)
	if err != nil {
		serverError(w, err)
		return
	}
	defer rows.Close()
	have := map[string]bool{}
	for rows.Next() {
		var f string
		if err := rows.Scan(&f); err != nil {
			serverError(w, err)
			return
		}
		s := stem(f)
		have[s] = true
		if i := strings.LastIndexByte(s, '-'); i > 0 && i < len(s)-1 && strings.Trim(s[i+1:], "0123456789") == "" {
			have[s[:i]] = true // IMG_0236-2.jpeg: the copy tool's second copy of IMG_0236
		}
	}
	if err := rows.Err(); err != nil {
		serverError(w, err)
		return
	}
	type file struct {
		Name string `json:"name"`
	}
	missing, checked := []file{}, 0
	for _, n := range names {
		if n = strings.TrimSpace(n); n == "" {
			continue
		}
		checked++
		if !have[stem(n)] {
			missing = append(missing, file{n})
		}
	}
	writeJSON(w, map[string]any{"checked": checked, "missing": missing})
}

// postUpload saves the photos a phone sends (multipart, field "file") into the first folder of the sender's own library,
// in a subfolder per month taken (2023-06):
// the catch-up for photos the lookup says are missing, from anywhere the server can be reached (Tailscale), without the
// NAS share. It never overwrites: a name taken by a file of another size gets -2, -3 (as copy tools do), one of the same
// size is taken as the same photo, already there.
func (a *App) postUpload(w http.ResponseWriter, r *http.Request) {
	// key only: a cross-site form can post multipart/form-data with the cookie, it can't set this header
	if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
		httpError(w, http.StatusUnauthorized, "Send an API key in the header Authorization: Bearer <key>")
		return
	}
	var pathsJSON string
	var roots []string
	err := a.db.QueryRow("select import_paths from libraries where owner_id = ? order by id limit 1", userFrom(r).ID).Scan(&pathsJSON)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		serverError(w, err)
		return
	}
	if json.Unmarshal([]byte(pathsJSON), &roots); len(roots) == 0 {
		httpError(w, http.StatusConflict, "Your library has no folder to save photos in")
		return
	}
	dir := roots[0]
	mr, err := r.MultipartReader()
	if err != nil {
		httpError(w, http.StatusBadRequest, "Send the photos as multipart/form-data, in the field file")
		return
	}
	type file struct {
		Name    string `json:"name"`              // as saved, maybe with -2
		Existed bool   `json:"existed,omitempty"` // already there, nothing written
	}
	saved := []file{}
	defer func() {
		if len(saved) > 0 {
			a.startScan() // also after an upload cut off halfway: what was saved shows up
		}
	}()
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			httpError(w, http.StatusBadRequest, "The upload was cut off")
			return
		}
		if part.FileName() == "" {
			continue
		}
		name := filepath.Base(part.FileName())
		if strings.HasPrefix(name, ".") || !mediaExt[strings.ToLower(filepath.Ext(name))] {
			httpError(w, http.StatusBadRequest, fmt.Sprintf("%s is not a photo or video Picburrow shows", name))
			return
		}
		got, existed, err := a.saveUpload(dir, name, part)
		switch {
		case errors.Is(err, errNoRoom):
			httpError(w, http.StatusInsufficientStorage, "Less than 1 GB would be left on the disk of "+dir)
			return
		case errors.Is(err, fs.ErrPermission) || errors.Is(err, syscall.EROFS):
			httpError(w, http.StatusForbidden, "Picburrow may not write to "+dir+". Give the picburrow user write access to it.")
			return
		case err != nil:
			serverError(w, err)
			return
		}
		saved = append(saved, file{got, existed})
	}
	if len(saved) == 0 {
		httpError(w, http.StatusBadRequest, "Send at least one file, in the field file")
		return
	}
	writeJSON(w, map[string]any{"saved": saved})
}

var errNoRoom = errors.New("disk nearly full")

// saveUpload writes src to a hidden file in dir (the scan skips it), then renames it to <month taken>/name, or name-2...
// if taken. saved is that path inside dir: 2023-06/IMG_0236.HEIC.
func (a *App) saveUpload(dir, name string, src io.Reader) (saved string, existed bool, err error) {
	var st unix.Statfs_t
	if err := unix.Statfs(dir, &st); err != nil {
		return "", false, err
	}
	room := int64(st.Bavail*uint64(st.Bsize)) - 1<<30 // keep 1 GB: the disk is the NAS's, not only ours
	tmp := filepath.Join(dir, ".picburrow-upload-"+randomToken()[:16])
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o666) // the umask decides, like any other copy
	if err != nil {
		return "", false, err
	}
	defer os.Remove(tmp) // fails harmlessly once renamed
	n, err := io.Copy(f, io.LimitReader(src, max(room, 0)+1))
	if err == nil && n > room {
		err = errNoRoom
	}
	if err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return "", false, err
	}
	// the month it was taken: from the file's own date, else from a name that starts with it (2023-06-15_..., as the
	// Shortcut names them), else this month
	month := time.Now().Format("2006-01")
	if _, err := time.Parse("2006-01", name[:min(7, len(name))]); err == nil {
		month = name[:7]
	}
	if m, err := a.exif.Read([]string{tmp}); err == nil && len(m[0].TakenLocal) >= 7 {
		month = m[0].TakenLocal[:7]
	}
	dir = filepath.Join(dir, month)
	if err := os.MkdirAll(dir, 0o777); err != nil {
		return "", false, err
	}
	ext := filepath.Ext(name)
	for i := 1; ; i++ {
		cand := name
		if i > 1 {
			cand = fmt.Sprintf("%s-%d%s", strings.TrimSuffix(name, ext), i, ext)
		}
		p := filepath.Join(dir, cand)
		if info, err := os.Stat(p); err == nil {
			if info.Size() == n {
				return month + "/" + cand, true, nil
			}
			continue
		}
		// claim the name first: O_EXCL, so two uploads of the same name at once can't both take it
		res, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o666)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return "", false, err
		}
		res.Close()
		if err := os.Rename(tmp, p); err != nil {
			os.Remove(p)
			return "", false, err
		}
		return month + "/" + cand, false, nil
	}
}

func (a *App) patchAssets(w http.ResponseWriter, r *http.Request) {
	var in struct {
		IDs      []int64 `json:"ids"`
		Favorite *bool   `json:"favorite"`
		Archived *bool   `json:"archived"`
		Rating   *int    `json:"rating"` // 0 clears
		Rotate   int     `json:"rotate"` // 90, 180 or 270 degrees clockwise, added to the turn it already has
	}
	if !readJSON(w, r, &in) {
		return
	}
	if in.Rating != nil && (*in.Rating < 0 || *in.Rating > 5) {
		httpError(w, http.StatusBadRequest, "Rating is 0 to 5 stars")
		return
	}
	if in.Rotate != 0 && in.Rotate != 90 && in.Rotate != 180 && in.Rotate != 270 {
		httpError(w, http.StatusBadRequest, "Rotate by 90, 180 or 270 degrees")
		return
	}
	if len(in.IDs) == 0 || len(in.IDs) > 5000 || (in.Favorite == nil && in.Archived == nil && in.Rating == nil && in.Rotate == 0) {
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
	if in.Rating != nil {
		set, args = append(set, "rating = nullif(?, 0)"), append(args, *in.Rating)
	}
	if in.Rotate != 0 { // photos only: a turned poster over an unturned video would mislead. Previews are remade turned.
		set = append(set, "rotation = case when type = 'image' then (rotation + ?) % 360 else rotation end",
			"thumbs_at = case when type = 'image' then null else thumbs_at end")
		args = append(args, in.Rotate)
		defer a.wakeThumbs()
	}
	for _, id := range in.IDs {
		args = append(args, id)
	}
	args = append(args, userFrom(r).ID)
	// the access check is in the same statement: ids from libraries the user doesn't see are silently skipped
	res, err := a.db.Exec(`update assets set `+strings.Join(set, ", ")+`, updated_at = unixepoch()
		where id in (?`+strings.Repeat(",?", len(in.IDs)-1)+`) and library_id in `+mine, args...)
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
	libs, err := a.ownLibraries(uid, true) // shared libraries' folders too
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
				out = append(out, f) // Name is the folder's own name; Path keeps the full root
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
	rows, err := a.db.Query(`select a.dir, count(*) from assets a where a.library_id = ? and a.missing_at is null
		and a.dir >= ? and a.dir < ? and `+notLiveClip+` group by a.dir`, libID, prefix, strings.TrimSuffix(prefix, "/")+"0") // '0' sorts right after '/'
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
		and `+notLiveClip+` order by a.taken_local desc, a.id desc`, libID, p)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, map[string]any{"path": p, "root": root, "dirs": dirs, "assets": assets})
}

func (a *App) folderSummary(libID int64, dir string) (folder, error) {
	f := folder{Path: dir, Name: filepath.Base(dir)}
	if err := a.db.QueryRow(`select count(*) from assets a where a.library_id = ? and a.missing_at is null
		and (a.dir = ? or (a.dir >= ? and a.dir < ?)) and `+notLiveClip, libID, dir, dir+"/", dir+"0").Scan(&f.Count); err != nil {
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
	Photos        int      `json:"photos"`
	Videos        int      `json:"videos"` // live-photo clips not counted: they show as part of their photo
	Missing       int      `json:"missing"`
}

func (a *App) ownLibraries(uid int64, shared bool) ([]library, error) {
	who := "l.owner_id = ?"
	if shared {
		who = "l.id in " + mine
	}
	rows, err := a.db.Query(`select l.id, l.name, l.import_paths, l.exclusion_globs, l.last_scan_at, l.last_scan_error,
		(select count(*) from assets where library_id = l.id and missing_at is null and type = 'image'),
		(select count(*) from assets a where a.library_id = l.id and a.missing_at is null and a.type = 'video' and `+notLiveClip+`),
		(select count(*) from assets where library_id = l.id and missing_at is not null)
		from libraries l where `+who+` order by l.id`, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []library{}
	for rows.Next() {
		var l library
		var paths, globs string
		if err := rows.Scan(&l.ID, &l.Name, &paths, &globs, &l.LastScanAt, &l.LastScanError, &l.Photos, &l.Videos, &l.Missing); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(paths), &l.Paths)
		json.Unmarshal([]byte(globs), &l.Globs)
		out = append(out, l)
	}
	return out, rows.Err()
}

func (a *App) getLibraries(w http.ResponseWriter, r *http.Request) {
	libs, err := a.ownLibraries(userFrom(r).ID, false) // only the owner edits a library
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
	a.startScan()
	a.getLibraries(w, r)
}
