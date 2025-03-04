package internal

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"./embedding"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

// SQLiteDatabase implements the DatabaseClient interface
type SQLiteDatabase struct {
	db              *sql.DB
	ctx             context.Context
	cancel          context.CancelFunc
	embeddingService embedding.EmbeddingService
}

func NewSQLiteDatabase(path string, embeddingService embedding.EmbeddingService) (*SQLiteDatabase, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Open database using libsql driver
	db, err := sql.Open("libsql", path)
	if (err != nil) {
		cancel()
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		cancel()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	sqlite := &SQLiteDatabase{
		db:              db,
		ctx:             ctx,
		cancel:          cancel,
		embeddingService: embeddingService,
	}

	// Initialize database schema
	if err := sqlite.initializeDatabase(); err != nil {
		sqlite.Close()
		return nil, err
	}

	return sqlite, nil
}

// Initialize database tables with vector support
func (d *SQLiteDatabase) initializeDatabase() error {
	// Create imported_transactions table with vector support
	_, err := d.db.ExecContext(d.ctx, `
		CREATE TABLE IF NOT EXISTS imported_transactions (
			id TEXT PRIMARY KEY,
			source TEXT NOT NULL,
			currency TEXT NOT NULL,
			amount REAL NOT NULL,
			transaction_type TEXT NOT NULL,
			description TEXT,
			timestamp TEXT NOT NULL,
			imported_at TEXT NOT NULL,
			metadata TEXT,
			embedding F32_BLOB(384)
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create imported_transactions table: %w", err)
	}

	// Create last_import table
	_, err = d.db.ExecContext(d.ctx, `
		CREATE TABLE IF NOT EXISTS last_import (
			source TEXT PRIMARY KEY,
			timestamp TEXT NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create last_import table: %w", err)
	}

	// Create vector index for semantic search using cosine similarity
	_, err = d.db.ExecContext(d.ctx, `
		CREATE INDEX IF NOT EXISTS idx_transaction_embeddings 
		ON imported_transactions (libsql_vector_idx(embedding, 'metric=cosine'));
	`)
	if err != nil {
		return fmt.Errorf("failed to create vector index: %w", err)
	}

	return nil
}

// IsTransactionImported checks if a transaction has already been imported
func (d *SQLiteDatabase) IsTransactionImported(txID string) (bool, error) {
	var count int
	err := d.db.QueryRowContext(d.ctx, "SELECT COUNT(*) FROM imported_transactions WHERE id = ?", txID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if transaction exists: %w", err)
	}
	return count > 0, nil
}

// MarkTransactionAsImported marks a transaction as imported and stores its embedding
func (d *SQLiteDatabase) MarkTransactionAsImported(txID string, metadata map[string]string) error {
	// Convert metadata to JSON
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Generate embedding from transaction metadata
	embedding, err := embedding.MetadataToEmbedding(d.embeddingService, metadata)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}
	
	// Convert embedding to JSON string for vector function
	embeddingJSON, err := json.Marshal(embedding)
	if err != nil {
		return fmt.Errorf("failed to marshal embedding: %w", err)
	}

	// Begin transaction
	tx, err := d.db.BeginTx(d.ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert transaction record with embedding using vector function
	_, err = tx.ExecContext(d.ctx, `
		INSERT INTO imported_transactions (
			id, source, currency, amount, transaction_type,
			description, timestamp, imported_at, metadata, embedding
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, vector(?))`,
		txID,
		metadata["source"],
		metadata["currency"],
		metadata["amount"],
		metadata["type"],
		metadata["description"],
		metadata["timestamp"],
		time.Now().UTC().Format(time.RFC3339),
		string(metadataJSON),
		string(embeddingJSON))
	if err != nil {
		return fmt.Errorf("failed to insert transaction: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// SearchSimilarTransactions finds transactions with similar metadata embeddings
func (d *SQLiteDatabase) SearchSimilarTransactions(metadata map[string]string, limit int) ([]string, error) {
	// Generate embedding for search
	searchEmbedding, err := embedding.MetadataToEmbedding(d.embeddingService, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to generate search embedding: %w", err)
	}

	embeddingJSON, err := json.Marshal(searchEmbedding)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embedding: %w", err)
	}
	
	// Use vector_top_k function to find similar transactions
	rows, err := d.db.QueryContext(d.ctx, `
		SELECT t.id, t.metadata 
		FROM vector_top_k('idx_transaction_embeddings', vector(?), ?) as v
		JOIN imported_transactions t
		ON t.rowid = v.id
		ORDER BY v.distance ASC`,
		string(embeddingJSON), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search similar transactions: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var (
			id       string
			metadata string
		)
		if err := rows.Scan(&id, &metadata); err != nil {
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating results: %w", err)
	}

	return ids, nil
}

// GetLastImportTime gets the timestamp of the last import operation
func (d *SQLiteDatabase) GetLastImportTime(source string) (time.Time, error) {
	var timestamp string
	err := d.db.QueryRowContext(d.ctx, "SELECT timestamp FROM last_import WHERE source = ?", source).Scan(&timestamp)
	if err == sql.ErrNoRows {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get last import time: %w", err)
	}

	return time.Parse(time.RFC3339, timestamp)
}

// SetLastImportTime sets the timestamp of the last import operation
func (d *SQLiteDatabase) SetLastImportTime(source string, timestamp time.Time) error {
	_, err := d.db.ExecContext(d.ctx, `
		INSERT INTO last_import (source, timestamp)
		VALUES (?, ?)
		ON CONFLICT(source) DO UPDATE SET timestamp = ?`,
		source, timestamp.UTC().Format(time.RFC3339), timestamp.UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("failed to update last import time: %w", err)
	}

	return nil
}

// Close closes the database connection and cancels the context
func (d *SQLiteDatabase) Close() error {
	d.cancel()
	if err := d.db.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}
	return nil
}
