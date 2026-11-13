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

// migrations[i] takes a database from version i+1 to i+2. Fresh installs get schema.sql at the latest version.
var migrations = []string{
	`alter table assets add column content_id text;
	 create index assets_content on assets(content_id) where content_id is not null;`,
}

var schemaVersion = len(migrations) + 1

// openDB opens (and creates on first run) the SQLite database in dataDir.
func openDB(dataDir string) (*sql.DB, error) {
	dsn := "file:" + filepath.Join(dataDir, "homeroll.db") +
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
	default:
		for ; v < schemaVersion; v++ { // one transaction per step: a failed upgrade leaves the old version intact
			tx, err := db.Begin()
			if err == nil {
				if _, err = tx.Exec(migrations[v-1]); err == nil {
					if _, err = tx.Exec(fmt.Sprintf("pragma user_version = %d", v+1)); err == nil {
						err = tx.Commit()
					}
				}
				if err != nil {
					tx.Rollback()
				}
			}
			if err != nil {
				db.Close()
				return nil, fmt.Errorf("upgrade database to version %d: %w", v+1, err)
			}
		}
	}
	return db, nil
}
