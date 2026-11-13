package main

import (
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type album struct {
	ID      int64   `json:"id"`
	Name    string  `json:"name"`
	Count   int     `json:"count"`
	Cover   *tile   `json:"cover,omitempty"`
	Shared  bool    `json:"shared"`
	Updated int64   `json:"updated_at"`
	Assets  []tile  `json:"assets,omitempty"`
	Links   []*link `json:"links,omitempty"`
}

type link struct {
	ID            int64  `json:"id"`
	Token         string `json:"token"`
	HasPassword   bool   `json:"has_password"`
	AllowDownload bool   `json:"allow_download"`
	ExpiresAt     *int64 `json:"expires_at"`
	Created       int64  `json:"created_at"`
}

// ownAlbum returns the album id if the user owns it.
func (a *App) ownAlbum(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, ok := pathID(r)
	var owner int64
	if !ok || a.db.QueryRow("select owner_id from albums where id = ?", id).Scan(&owner) != nil || owner != userFrom(r).ID {
		httpError(w, http.StatusNotFound, "No such album")
		return 0, false
	}
	return id, true
}

func validAlbumName(s string) (string, bool) {
	s = strings.TrimSpace(s)
	return s, s != "" && len(s) <= 200
}

func (a *App) getAlbums(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.Query(`select al.id, al.name, al.updated_at, al.cover_asset_id,
		(select count(*) from album_assets aa join assets x on x.id = aa.asset_id where aa.album_id = al.id and x.missing_at is null),
		exists(select 1 from shared_links where album_id = al.id)
		from albums al where al.owner_id = ? order by al.updated_at desc`, userFrom(r).ID)
	if err != nil {
		serverError(w, err)
		return
	}
	out := []album{}
	covers := map[int]*int64{}
	for rows.Next() {
		var al album
		var cover *int64
		if err := rows.Scan(&al.ID, &al.Name, &al.Updated, &cover, &al.Count, &al.Shared); err != nil {
			rows.Close()
			serverError(w, err)
			return
		}
		covers[len(out)] = cover
		out = append(out, al)
	}
	rows.Close()
	for i := range out {
		if c, err := a.albumCover(out[i].ID, covers[i]); err == nil {
			out[i].Cover = c
		}
	}
	writeJSON(w, out)
}

// albumCover: the chosen cover if still present, else the newest photo in the album.
func (a *App) albumCover(albumID int64, chosen *int64) (*tile, error) {
	ts, err := a.queryTiles(`select `+tileCols+` from album_assets aa join assets a on a.id = aa.asset_id
		where aa.album_id = ? and a.missing_at is null and a.thumb_error is null
		order by (a.id = ?) desc, a.taken_local desc limit 1`, albumID, chosen)
	if err != nil || len(ts) == 0 {
		return nil, err
	}
	return &ts[0], nil
}

func (a *App) postAlbum(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string  `json:"name"`
		IDs  []int64 `json:"ids"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	name, ok := validAlbumName(in.Name)
	if !ok {
		httpError(w, http.StatusBadRequest, "Give the album a name (up to 200 characters)")
		return
	}
	res, err := a.db.Exec("insert into albums (owner_id, name) values (?, ?)", userFrom(r).ID, name)
	if err != nil {
		serverError(w, err)
		return
	}
	id, _ := res.LastInsertId()
	if _, err := a.addToAlbum(userFrom(r).ID, id, in.IDs); err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, album{ID: id, Name: name})
}

// addToAlbum adds the user's own assets only; others are skipped. Returns how many were added.
func (a *App) addToAlbum(uid, albumID int64, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	if len(ids) > 5000 {
		return 0, fmt.Errorf("too many ids")
	}
	args := []any{albumID}
	for _, id := range ids {
		args = append(args, id)
	}
	args = append(args, uid)
	res, err := a.db.Exec(`insert or ignore into album_assets (album_id, asset_id)
		select ?, a.id from assets a join libraries l on l.id = a.library_id
		where a.id in (?`+strings.Repeat(",?", len(ids)-1)+`) and l.id in `+mine, args...)
	if err != nil {
		return 0, err
	}
	a.db.Exec("update albums set updated_at = unixepoch() where id = ?", albumID)
	return res.RowsAffected()
}

func (a *App) getAlbum(w http.ResponseWriter, r *http.Request) {
	id, ok := a.ownAlbum(w, r)
	if !ok {
		return
	}
	var al album
	a.db.QueryRow("select id, name, updated_at from albums where id = ?", id).Scan(&al.ID, &al.Name, &al.Updated)
	var err error
	if al.Assets, err = a.albumTiles(id); err != nil {
		serverError(w, err)
		return
	}
	al.Count = len(al.Assets)
	if al.Links, err = a.albumLinks(id); err != nil {
		serverError(w, err)
		return
	}
	al.Shared = len(al.Links) > 0
	writeJSON(w, al)
}

func (a *App) albumTiles(id int64) ([]tile, error) {
	return a.queryTiles(`select `+tileCols+` from album_assets aa join assets a on a.id = aa.asset_id
		where aa.album_id = ? and a.missing_at is null order by a.taken_local desc, a.id desc`, id)
}

func (a *App) albumLinks(id int64) ([]*link, error) {
	rows, err := a.db.Query(`select id, token, password_hash is not null, allow_download, expires_at, created_at
		from shared_links where album_id = ? order by id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*link{}
	for rows.Next() {
		l := &link{}
		if err := rows.Scan(&l.ID, &l.Token, &l.HasPassword, &l.AllowDownload, &l.ExpiresAt, &l.Created); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (a *App) patchAlbum(w http.ResponseWriter, r *http.Request) {
	id, ok := a.ownAlbum(w, r)
	if !ok {
		return
	}
	var in struct {
		Name  *string `json:"name"`
		Cover *int64  `json:"cover_id"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if in.Name != nil {
		name, ok := validAlbumName(*in.Name)
		if !ok {
			httpError(w, http.StatusBadRequest, "Give the album a name (up to 200 characters)")
			return
		}
		a.db.Exec("update albums set name = ?, updated_at = unixepoch() where id = ?", name, id)
	}
	if in.Cover != nil { // only a photo that is in this album
		a.db.Exec(`update albums set cover_asset_id = ? where id = ? and exists
			(select 1 from album_assets where album_id = ? and asset_id = ?)`, *in.Cover, id, id, *in.Cover)
	}
	a.getAlbum(w, r)
}

func (a *App) deleteAlbum(w http.ResponseWriter, r *http.Request) {
	id, ok := a.ownAlbum(w, r)
	if !ok {
		return
	}
	// photos stay on disk and in the library: only the album and its share links go
	if _, err := a.db.Exec("delete from albums where id = ?", id); err != nil {
		serverError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) albumAssets(add bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := a.ownAlbum(w, r)
		if !ok {
			return
		}
		var in struct {
			IDs []int64 `json:"ids"`
		}
		if !readJSON(w, r, &in) {
			return
		}
		if len(in.IDs) == 0 || len(in.IDs) > 5000 {
			httpError(w, http.StatusBadRequest, "Send 1 to 5000 ids")
			return
		}
		var n int64
		var err error
		if add {
			n, err = a.addToAlbum(userFrom(r).ID, id, in.IDs)
		} else {
			args := []any{id}
			for _, x := range in.IDs {
				args = append(args, x)
			}
			var res sql.Result
			res, err = a.db.Exec(`delete from album_assets where album_id = ? and asset_id in (?`+strings.Repeat(",?", len(in.IDs)-1)+`)`, args...)
			if err == nil {
				n, _ = res.RowsAffected()
				a.db.Exec("update albums set updated_at = unixepoch() where id = ?", id)
			}
		}
		if err != nil {
			serverError(w, err)
			return
		}
		writeJSON(w, map[string]int64{"changed": n})
	}
}

// ---- share links

func (a *App) postLink(w http.ResponseWriter, r *http.Request) {
	id, ok := a.ownAlbum(w, r)
	if !ok {
		return
	}
	var in struct {
		Password      string `json:"password"`
		AllowDownload bool   `json:"allow_download"`
		ExpiresAt     *int64 `json:"expires_at"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if in.ExpiresAt != nil && *in.ExpiresAt <= time.Now().Unix() {
		httpError(w, http.StatusBadRequest, "Pick an expiry date in the future")
		return
	}
	var hash any
	if in.Password != "" {
		if len(in.Password) < 4 {
			httpError(w, http.StatusBadRequest, "Use at least 4 characters for the link password")
			return
		}
		h, err := hashPassword(in.Password)
		if err != nil {
			serverError(w, err)
			return
		}
		hash = h
	}
	if _, err := a.db.Exec(`insert into shared_links (album_id, token, password_hash, allow_download, expires_at) values (?, ?, ?, ?, ?)`,
		id, randomToken(), hash, in.AllowDownload, in.ExpiresAt); err != nil {
		serverError(w, err)
		return
	}
	a.getAlbum(w, r)
}

func (a *App) deleteLink(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	res, err := a.db.Exec(`delete from shared_links where id = ? and album_id in (select id from albums where owner_id = ?)`, id, userFrom(r).ID)
	if err != nil {
		serverError(w, err)
		return
	}
	if n, _ := res.RowsAffected(); !ok || n == 0 {
		httpError(w, http.StatusNotFound, "No such link")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type sharedLink struct {
	id, albumID   int64
	token         string
	passwordHash  sql.NullString
	allowDownload bool
}

var errLinkGone = errors.New("gone")

// resolveLink finds a link by token; expired links are gone. needsPassword is true when locked for this browser.
func (a *App) resolveLink(r *http.Request) (*sharedLink, bool, error) {
	l := &sharedLink{}
	var expires sql.NullInt64
	err := a.db.QueryRow(`select id, album_id, token, password_hash, allow_download, expires_at from shared_links where token = ?`,
		r.PathValue("token")).Scan(&l.id, &l.albumID, &l.token, &l.passwordHash, &l.allowDownload, &expires)
	if err != nil {
		return nil, false, errLinkGone
	}
	if expires.Valid && expires.Int64 <= time.Now().Unix() {
		return nil, false, errLinkGone
	}
	if !l.passwordHash.Valid {
		return l, false, nil
	}
	c, err := r.Cookie(linkCookie(l.id))
	return l, err != nil || subtle.ConstantTimeCompare([]byte(c.Value), []byte(l.unlockValue())) != 1, nil
}

func linkCookie(id int64) string { return fmt.Sprintf("rg_link_%d", id) }

// unlockValue changes when the password changes or the link is recreated, so old unlocks stop working.
func (l *sharedLink) unlockValue() string {
	h := sha256.Sum256([]byte(l.token + "\x00" + l.passwordHash.String))
	return hex.EncodeToString(h[:])
}

func linkError(w http.ResponseWriter, err error) {
	httpError(w, http.StatusGone, "This link has expired or was turned off")
}

func (a *App) getShared(w http.ResponseWriter, r *http.Request) {
	l, locked, err := a.resolveLink(r)
	if err != nil {
		linkError(w, err)
		return
	}
	var name string
	a.db.QueryRow("select name from albums where id = ?", l.albumID).Scan(&name)
	if locked {
		writeJSON(w, map[string]any{"name": name, "locked": true})
		return
	}
	ts, err := a.albumTiles(l.albumID)
	if err != nil {
		serverError(w, err)
		return
	}
	for i := range ts {
		ts[i].Fav = false // the owner's favourites are not the visitor's business
	}
	writeJSON(w, map[string]any{"name": name, "locked": false, "allow_download": l.allowDownload, "assets": ts})
}

func (a *App) postUnlock(w http.ResponseWriter, r *http.Request) {
	if !a.loginLimit.allow(r) {
		httpError(w, http.StatusTooManyRequests, "Too many attempts. Wait a minute and try again.")
		return
	}
	l, _, err := a.resolveLink(r)
	if err != nil {
		linkError(w, err)
		return
	}
	var in struct{ Password string }
	if !readJSON(w, r, &in) {
		return
	}
	if !l.passwordHash.Valid || !checkPassword(l.passwordHash.String, in.Password) {
		httpError(w, http.StatusUnauthorized, "Wrong password")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: linkCookie(l.id), Value: l.unlockValue(), Path: "/", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, Secure: r.TLS != nil, MaxAge: 30 * 24 * 3600})
	w.WriteHeader(http.StatusNoContent)
}

// sharedAsset loads an asset only if it is in the link's album and the link is open for this browser.
func (a *App) sharedAsset(w http.ResponseWriter, r *http.Request) (*sharedLink, *assetRow, bool) {
	l, locked, err := a.resolveLink(r)
	if err != nil {
		linkError(w, err)
		return nil, nil, false
	}
	if locked {
		httpError(w, http.StatusUnauthorized, "This album needs a password")
		return nil, nil, false
	}
	id, ok := pathID(r)
	var owner int64
	if ok {
		// the photo itself, or the live-photo clip of a photo in the album
		err = a.db.QueryRow(`select al.owner_id from album_assets aa join albums al on al.id = aa.album_id
			join assets s on s.id = aa.asset_id where aa.album_id = ? and (aa.asset_id = ? or s.live_video_id = ?) limit 1`,
			l.albumID, id, id).Scan(&owner)
	}
	if !ok || err != nil {
		httpError(w, http.StatusNotFound, "No such photo")
		return nil, nil, false
	}
	x, err := a.loadAsset(owner, id)
	if err != nil {
		httpError(w, http.StatusNotFound, "No such photo")
		return nil, nil, false
	}
	return l, x, true
}

func (a *App) getSharedAsset(w http.ResponseWriter, r *http.Request) {
	l, x, ok := a.sharedAsset(w, r)
	if !ok {
		return
	}
	// visitors get what they need to view it, not where it lives or where it was taken
	writeJSON(w, map[string]any{"id": x.ID, "w": x.W, "h": x.H, "type": x.Type, "taken": x.Taken, "v": x.V, "err": x.Err,
		"mime": x.Mime, "name": x.Name, "allow_download": l.allowDownload, "live": x.Live})
}

func (a *App) getSharedThumb(w http.ResponseWriter, r *http.Request) {
	if _, x, ok := a.sharedAsset(w, r); ok {
		a.serveThumb(w, r, x)
	}
}

func (a *App) getSharedOriginal(w http.ResponseWriter, r *http.Request) {
	l, x, ok := a.sharedAsset(w, r)
	if !ok {
		return
	}
	if !l.allowDownload {
		httpError(w, http.StatusForbidden, "Downloads are turned off for this album")
		return
	}
	a.serveOriginal(w, r, x, r.URL.Query().Get("download") == "1")
}

// ---- search

func (a *App) getSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	where := []string{"l.id in " + mine, "a.missing_at is null", "a.is_archived = 0", notLiveClip}
	args := []any{userFrom(r).ID}
	if text := strings.TrimSpace(q.Get("q")); text != "" {
		like := "%" + strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(text) + "%"
		where = append(where, `(a.path like ? escape '\' or coalesce(a.camera_make, '') || ' ' || coalesce(a.camera_model, '') like ? escape '\'
			or coalesce(a.lens, '') like ? escape '\')`)
		args = append(args, like, like, like)
	}
	for param, cond := range map[string]string{"from": "a.taken_local >= ?", "to": "a.taken_local < ?"} {
		v := q.Get(param)
		if v == "" {
			continue
		}
		d, err := time.Parse("2006-01-02", v)
		if err != nil {
			httpError(w, http.StatusBadRequest, param+" must look like 2024-05-31")
			return
		}
		if param == "to" {
			d = d.AddDate(0, 0, 1) // inclusive end day
		}
		where, args = append(where, cond), append(args, d.Format("2006-01-02"))
	}
	switch q.Get("type") {
	case "image", "video":
		where, args = append(where, "a.type = ?"), append(args, q.Get("type"))
	case "":
	default:
		httpError(w, http.StatusBadRequest, "type must be image or video")
		return
	}
	if q.Get("favorite") == "1" {
		where = append(where, "a.is_favorite = 1")
	}
	if v := q.Get("rating"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 5 {
			httpError(w, http.StatusBadRequest, "rating must be 1 to 5")
			return
		}
		where, args = append(where, "a.rating >= ?"), append(args, n)
	}
	// shortcut: LIKE over every row and a 2000-result cap; fine for ~100k photos in SQLite, FTS5 if not
	a.writeTiles(w, `select `+tileCols+` from assets a join libraries l on l.id = a.library_id where `+
		strings.Join(where, " and ")+` order by a.taken_local desc, a.id desc limit 2000`, args...)
}
