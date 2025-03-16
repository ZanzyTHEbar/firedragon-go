package internal

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteDatabase implements the DatabaseClient interface
type SQLiteDatabase struct {
	db *sql.DB
}

// NewSQLiteDatabase creates a new SQLite database connection
func NewSQLiteDatabase(path string) (*SQLiteDatabase, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create tables if they don't exist
	if err := initializeDatabase(db); err != nil {
		db.Close()
		return nil, err
	}

	return &SQLiteDatabase{db: db}, nil
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
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM imported_transactions WHERE id = ?", txID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if transaction exists: %w", err)
	}

	return count > 0, nil
}

// MarkTransactionAsImported marks a transaction as imported
func (d *SQLiteDatabase) MarkTransactionAsImported(txID string, metadata map[string]string) error {
	// Check if transaction exists first
	exists, err := d.IsTransactionImported(txID)
	if err != nil {
		return err
	}

	if exists {
		// Transaction already imported
		return nil
	}

	// Convert metadata to JSON if not nil
	var metadataJSON string
	if metadata != nil {
		metadataBytes, err := JSONMarshal(metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		metadataJSON = string(metadataBytes)
	}

	// Insert transaction record
	_, err = d.db.Exec(`
		INSERT INTO imported_transactions 
		(id, source, currency, amount, transaction_type, description, timestamp, imported_at, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, txID, metadata["source"], metadata["currency"], metadata["amount"], metadata["type"],
		metadata["description"], metadata["timestamp"], time.Now().Format(time.RFC3339), metadataJSON)

	if err != nil {
		return fmt.Errorf("failed to mark transaction as imported: %w", err)
	}

	return nil
}

// GetLastImportTime gets the timestamp of the last import operation
func (d *SQLiteDatabase) GetLastImportTime(source string) (time.Time, error) {
	var timeStr string
	err := d.db.QueryRow("SELECT timestamp FROM last_import WHERE source = ?", source).Scan(&timeStr)

	if err != nil {
		if err == sql.ErrNoRows {
			// No previous import, return zero time
			return time.Time{}, nil
		}
		return time.Time{}, fmt.Errorf("failed to get last import time: %w", err)
	}

	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse timestamp: %w", err)
	}

	return t, nil
}

// SetLastImportTime sets the timestamp of the last import operation
func (d *SQLiteDatabase) SetLastImportTime(source string, timestamp time.Time) error {
	timeStr := timestamp.Format(time.RFC3339)

	_, err := d.db.Exec(`
		INSERT OR REPLACE INTO last_import (source, timestamp) VALUES (?, ?)
	`, source, timeStr)

	if err != nil {
		return fmt.Errorf("failed to set last import time: %w", err)
	}

	return nil
}

// Close closes the database connection
func (d *SQLiteDatabase) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}
