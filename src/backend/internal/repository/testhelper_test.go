package repository_test

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://portfolio:portfolio_dev@localhost:5432/portfolio?sslmode=disable"
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("skipping integration test: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Skipf("skipping integration test: %v", err)
	}
	// Clean tables for test isolation
	for _, table := range []string{"transactions", "stocks", "prices_cache"} {
		if _, err := db.Exec(fmt.Sprintf("DELETE FROM %s", table)); err != nil {
			// Table might not exist if migrations haven't run
			t.Skipf("skipping: table %s not ready: %v", table, err)
		}
	}
	return db
}
