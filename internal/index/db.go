package index

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"
)

// DB wraps an embedded SQLite database connection.
type DB struct {
	dbPath string
	sqlDB  *sql.DB
	mu     sync.RWMutex
}

// Open initializes or opens the SQLite database at dbPath.
func Open(dbPath string) (*DB, error) {
	if dbPath != ":memory:" {
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory for sqlite db: %w", err)
		}
	}

	dsn := dbPath
	if dbPath != ":memory:" {
		dsn = fmt.Sprintf("%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)", dbPath)
	}

	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Single writer / connection pool settings for embedded usage
	sqlDB.SetMaxOpenConns(1)

	db := &DB{
		dbPath: dbPath,
		sqlDB:  sqlDB,
	}

	if err := db.initSchema(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return db, nil
}

// Close closes the underlying SQLite database.
func (d *DB) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.sqlDB != nil {
		return d.sqlDB.Close()
	}
	return nil
}

// SqlDB returns the raw *sql.DB instance.
func (d *DB) SqlDB() *sql.DB {
	return d.sqlDB
}
