package main

import (
	"database/sql"
	_ "embed"
	"fmt"
	"path/filepath"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

const schemaVersion = 1

// openDB opens (and creates on first run) the SQLite database in dataDir.
func openDB(dataDir string) (*sql.DB, error) {
	dsn := "file:" + filepath.Join(dataDir, "raspicture.db") +
		"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)" +
		"&_pragma=synchronous(NORMAL)&_pragma=cache_size(-16000)" // 16 MB page cache: 1 GB Pi budget
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(4)
	var v int
	if err := db.QueryRow("pragma user_version").Scan(&v); err != nil {
		db.Close()
		return nil, err
	}
	switch {
	case v == 0:
		if _, err := db.Exec(schemaSQL); err != nil {
			db.Close()
			return nil, fmt.Errorf("create schema: %w", err)
		}
		if _, err := db.Exec(fmt.Sprintf("pragma user_version = %d", schemaVersion)); err != nil {
			db.Close()
			return nil, err
		}
	case v > schemaVersion:
		db.Close()
		return nil, fmt.Errorf("database version %d is newer than this build (%d)", v, schemaVersion)
	}
	return db, nil
}
