package store

import (
	"database/sql"
	"embed"
	"fmt"
	"strings"

	"github.com/pressly/goose/v3"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Open opens (creating if needed) the SQLite database at path and applies
// all pending migrations. path may be ":memory:" for a private in-memory
// database, or a "file:<name>?mode=memory&cache=shared" DSN (tests use a
// unique name per test so parallel tests don't share state). Either way, an
// in-memory DB only survives if capped to a single pooled connection.
func Open(path string) (*sql.DB, error) {
	dsn := path
	if path == ":memory:" {
		dsn = "file::memory:?cache=shared"
	}

	// foreign_keys and busy_timeout are PER-CONNECTION settings in SQLite, and
	// database/sql keeps a pool of connections. A one-shot `PRAGMA foreign_keys
	// = ON` only configures whichever single connection ran it, leaving every
	// other pooled connection with FKs OFF — so ON DELETE CASCADE would fire
	// only intermittently. modernc.org/sqlite instead applies `_pragma=` DSN
	// params to every connection it opens, so we set them there. busy_timeout
	// makes a writer wait for a held lock rather than failing immediately with
	// SQLITE_BUSY under concurrent provisioning.
	if !strings.HasPrefix(dsn, "file:") {
		dsn = "file:" + dsn
	}
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	dsn += sep + "_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"

	// On a real file database, enable WAL journaling + synchronous=NORMAL. The
	// default (DELETE) journal takes a whole-database write lock that blocks
	// readers during every write; with AFK heartbeats, two background
	// schedulers, and concurrent provisioning all writing, WAL lets reads run
	// concurrently with a writer and dramatically cuts busy_timeout stalls.
	// Skipped for in-memory DBs (tests), where WAL doesn't apply.
	if !strings.Contains(dsn, "memory") {
		dsn += "&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)"
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if strings.Contains(dsn, "memory") {
		db.SetMaxOpenConns(1)
	}

	if err := migrate(db); err != nil {
		return nil, err
	}

	return db, nil
}

func migrate(db *sql.DB) error {
	goose.SetBaseFS(migrationsFS)
	defer goose.SetBaseFS(nil)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}
