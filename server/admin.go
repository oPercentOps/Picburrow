package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ---- trash: photos whose file disappeared from disk. Files are never touched; only the index entry goes.

const trashDays = 30

func (a *App) getTrash(w http.ResponseWriter, r *http.Request) {
	a.writeTiles(w, `select `+tileCols+` from assets a join libraries l on l.id = a.library_id
		where l.id in `+mine+` and a.missing_at is not null order by a.missing_at desc, a.id desc limit 5000`, userFrom(r).ID)
}

// deleteTrash forgets missing photos now instead of after 30 days (their albums and favourites go with them).
func (a *App) deleteTrash(w http.ResponseWriter, r *http.Request) {
	ids, err := queryIDs(a.db, `select a.id from assets a join libraries l on l.id = a.library_id
		where l.id in `+mine+` and a.missing_at is not null`, userFrom(r).ID)
	if err != nil {
		serverError(w, err)
		return
	}
	n, err := a.forget(ids)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, map[string]int{"removed": n})
}

// forget deletes index rows and their thumbnails. Never the photo files.
func (a *App) forget(ids []int64) (int, error) {
	n := 0
	for _, id := range ids {
		res, err := a.db.Exec("delete from assets where id = ? and missing_at is not null", id) // only ever missing ones
		if err != nil {
			return n, err
		}
		if k, _ := res.RowsAffected(); k == 1 {
			n++
			for _, s := range []string{"s", "l"} {
				os.Remove(a.thumbPath(id, s))
			}
			os.Remove(a.playPath(id))
		}
	}
	return n, nil
}

// purgeTrash runs daily: photos missing for more than 30 days leave the index.
func (a *App) purgeTrash() {
	ids, err := queryIDs(a.db, "select id from assets where missing_at < ?", time.Now().Add(-trashDays*24*time.Hour).Unix())
	if err == nil && len(ids) > 0 {
		var n int
		n, err = a.forget(ids)
		log.Printf("trash: forgot %d photos missing for over %d days", n, trashDays)
	}
	if err != nil {
		log.Printf("trash purge: %v", err)
	}
}

// ---- users (admins only)

type userRow struct {
	User
	Created int64 `json:"created_at"`
	Assets  int   `json:"assets"`
}

func (a *App) getUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.Query(`select u.id, u.username, u.is_admin, u.created_at,
		(select count(*) from assets x where x.library_id in (select library_id from library_members where user_id = u.id) and x.missing_at is null)
		from users u order by u.id`)
	if err != nil {
		serverError(w, err)
		return
	}
	defer rows.Close()
	out := []userRow{}
	for rows.Next() {
		var u userRow
		if err := rows.Scan(&u.ID, &u.Username, &u.IsAdmin, &u.Created, &u.Assets); err != nil {
			serverError(w, err)
			return
		}
		out = append(out, u)
	}
	writeJSON(w, out)
}

// postUser adds a family member with their own photo folder, or sharing one of the admin's libraries (library_id):
// same photos, favourites, ratings and archive, nothing indexed twice.
func (a *App) postUser(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
		IsAdmin  bool   `json:"is_admin"`
		Path     string `json:"path"`
		Library  int64  `json:"library_id"`
	}
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
	if in.Library != 0 {
		var n int
		a.db.QueryRow("select count(*) from library_members where library_id = ? and user_id = ?", in.Library, userFrom(r).ID).Scan(&n)
		if n == 0 {
			httpError(w, http.StatusNotFound, "No such library")
			return
		}
	} else if msg := checkFolder(in.Path); msg != "" {
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
	res, err := tx.Exec("insert into users (username, password_hash, is_admin) values (?, ?, ?)", in.Username, hash, in.IsAdmin)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			httpError(w, http.StatusConflict, "That username is taken")
			return
		}
		serverError(w, err)
		return
	}
	uid, _ := res.LastInsertId()
	lib := in.Library
	if lib == 0 {
		paths, _ := json.Marshal([]string{filepath.Clean(in.Path)})
		res, err := tx.Exec("insert into libraries (owner_id, name, import_paths) values (?, ?, ?)", uid, filepath.Base(filepath.Clean(in.Path)), string(paths))
		if err != nil {
			serverError(w, err)
			return
		}
		lib, _ = res.LastInsertId()
	}
	if _, err := tx.Exec("insert into library_members (library_id, user_id) values (?, ?)", lib, uid); err != nil {
		serverError(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		serverError(w, err)
		return
	}
	if in.Library == 0 {
		a.startScan() // a shared library is already indexed
	}
	a.getUsers(w, r)
}

func (a *App) patchUser(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	var in struct {
		Password string `json:"password"`
	}
	if !ok || !readJSON(w, r, &in) {
		if !ok {
			httpError(w, http.StatusNotFound, "No such user")
		}
		return
	}
	if len(in.Password) < 8 {
		httpError(w, http.StatusBadRequest, "Use at least 8 characters for the password")
		return
	}
	hash, err := hashPassword(in.Password)
	if err != nil {
		serverError(w, err)
		return
	}
	res, err := a.db.Exec("update users set password_hash = ?, updated_at = unixepoch() where id = ?", hash, id)
	if n, _ := res.RowsAffected(); err != nil || n == 0 {
		httpError(w, http.StatusNotFound, "No such user")
		return
	}
	a.db.Exec("delete from sessions where user_id = ? and user_id != ?", id, userFrom(r).ID) // sign them out everywhere
	w.WriteHeader(http.StatusNoContent)
}

// deleteUser removes an account, its albums and links, and the index of libraries nobody else shares; a library shared
// with others passes to one of them, so the family keeps its photos, favourites and ratings. Photo files stay on disk.
func (a *App) deleteUser(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		httpError(w, http.StatusNotFound, "No such user")
		return
	}
	if id == userFrom(r).ID {
		httpError(w, http.StatusBadRequest, "You can't delete your own account")
		return
	}
	if _, err := a.db.Exec(`update libraries set owner_id = (select min(m.user_id) from library_members m
		where m.library_id = libraries.id and m.user_id != ?1) where owner_id = ?1
		and exists (select 1 from library_members m where m.library_id = libraries.id and m.user_id != ?1)`, id); err != nil {
		serverError(w, err)
		return
	}
	ids, _ := queryIDs(a.db, "select x.id from assets x join libraries l on l.id = x.library_id where l.owner_id = ?", id)
	res, err := a.db.Exec("delete from users where id = ?", id) // cascades: sessions, libraries -> assets, albums -> links
	if err != nil {
		serverError(w, err)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		httpError(w, http.StatusNotFound, "No such user")
		return
	}
	for _, x := range ids {
		for _, s := range []string{"s", "l"} {
			os.Remove(a.thumbPath(x, s))
		}
		os.Remove(a.playPath(x))
	}
	w.WriteHeader(http.StatusNoContent)
}
