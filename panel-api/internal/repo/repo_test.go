package repo

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/store"
)

// newTestDB gives each test its own uniquely-named in-memory SQLite database
// so tests can run without leaking state between each other.
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()

	name := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", name)

	db, err := store.Open(dsn)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	return db
}
