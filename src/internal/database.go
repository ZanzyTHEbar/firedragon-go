package internal

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/ZanzyTHEbar/firedragon-go/interfaces"
	_ "github.com/mattn/go-sqlite3"
)

// SQLiteDatabase implements the DatabaseClient interface
type SQLiteDatabase struct {
	db   *sql.DB
	path string
	mu   sync.Mutex
}

// NewSQLiteDatabase creates a new SQLite database client
func NewSQLiteDatabase(path string) (interfaces.DatabaseClient, error) {
	db, err := sql.Open("sqlite3", path)
	if (err != nil) {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Initialize tables
	if err := initializeDatabase(db); err != nil {
		db.Close()
		return nil, err
	}

	return &SQLiteDatabase{
		db:   db,
		path: path,
	}, nil
}

// Initialize database tables
func initializeDatabase(db *sql.DB) error {
	// Create imported_transactions table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS imported_transactions (
			id TEXT PRIMARY KEY,
			source TEXT NOT NULL,
			currency TEXT NOT NULL,
			amount REAL NOT NULL,
			transaction_type TEXT NOT NULL,
			description TEXT,
			timestamp TEXT NOT NULL,
			imported_at TEXT NOT NULL,
			metadata TEXT
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create imported_transactions table: %w", err)
	}

	// Create last_import table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS last_import (
			source TEXT PRIMARY KEY,
			timestamp TEXT NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create last_import table: %w", err)
	}

	return nil
}

// IsTransactionImported checks if a transaction has already been imported
func (d *SQLiteDatabase) IsTransactionImported(txID string) (bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM imported_transactions WHERE id = ?", txID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check transaction: %w", err)
	}

	return count > 0, nil
}

// MarkTransactionAsImported marks a transaction as imported with metadata
func (d *SQLiteDatabase) MarkTransactionAsImported(txID string, metadata map[string]string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.Exec(
		`INSERT INTO imported_transactions (id, imported_at, metadata) VALUES (?, ?, ?)`,
		txID,
		time.Now().UTC().Format(time.RFC3339),
		fmt.Sprintf("%v", metadata),
	)
	if err != nil {
		return fmt.Errorf("failed to mark transaction as imported: %w", err)
	}

	return nil
}

// GetLastImportTime gets the timestamp of the last import operation
func (d *SQLiteDatabase) GetLastImportTime(source string) (time.Time, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	var timestamp string
	err := d.db.QueryRow(
		"SELECT timestamp FROM last_import WHERE source = ?",
		source,
	).Scan(&timestamp)

	if err == sql.ErrNoRows {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get last import time: %w", err)
	}

	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse timestamp: %w", err)
	}

	return t, nil
}

// SetLastImportTime sets the timestamp of the last import operation
func (d *SQLiteDatabase) SetLastImportTime(source string, timestamp time.Time) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.Exec(
		`INSERT OR REPLACE INTO last_import (source, timestamp) VALUES (?, ?)`,
		source,
		timestamp.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to set last import time: %w", err)
	}

	return nil
}

// SearchSimilarTransactions finds transactions with similar metadata embeddings
func (d *SQLiteDatabase) SearchSimilarTransactions(metadata map[string]string, limit int) ([]string, error) {
    // TODO: Implement similarity search using embeddings
    // For now, return empty result since embedding search is marked as not implemented
    return []string{}, nil
}

// Close closes the database connection
func (d *SQLiteDatabase) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db != nil {
		return d.db.Close()
	}
	return nil
}
