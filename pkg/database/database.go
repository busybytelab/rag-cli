package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/busybytelab.com/rag-cli/pkg/config"
)

// DatabaseManagerImpl implements DatabaseManager interface
type DatabaseManagerImpl struct {
	db *sql.DB
}

// NewDatabaseManager creates a new database manager with all three components
func NewDatabaseManager(cfg *config.DatabaseConfig) (DatabaseManager, error) {
	dsn := cfg.GetDSN()

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Create the main database manager
	databaseManager := &DatabaseManagerImpl{
		db: db,
	}

	// Initialize the database schema
	if err := databaseManager.InitSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return databaseManager, nil
}

// Close closes the database connection
func (dm *DatabaseManagerImpl) Close() error {
	return dm.db.Close()
}

// InitSchema initializes the database schema
func (dm *DatabaseManagerImpl) InitSchema() error {
	// TODO: embedding vector size must be configurable
	queries := []string{
		`CREATE EXTENSION IF NOT EXISTS vector;`,
		`CREATE TABLE IF NOT EXISTS collections (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL UNIQUE,
			description TEXT,
			folders TEXT[] NOT NULL,
			stats JSONB DEFAULT '{"total_documents": 0, "total_chunks": 0, "total_size": 0}',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);`,
		`CREATE TABLE IF NOT EXISTS documents (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			collection_id UUID NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
			file_path TEXT NOT NULL,
			file_name VARCHAR(255) NOT NULL,
			content TEXT NOT NULL,
			chunk_index INTEGER NOT NULL DEFAULT 0,
			embedding vector(768),
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);`,
		`CREATE INDEX IF NOT EXISTS idx_documents_collection_id ON documents(collection_id);`,
		`CREATE INDEX IF NOT EXISTS idx_documents_file_path ON documents(file_path);`,
		`CREATE INDEX IF NOT EXISTS idx_documents_embedding ON documents USING hnsw (embedding vector_cosine_ops);`,
		`CREATE INDEX IF NOT EXISTS idx_documents_content_fts ON documents USING gin(to_tsvector('english', content));`,
		`CREATE INDEX IF NOT EXISTS idx_documents_file_name_fts ON documents USING gin(to_tsvector('english', file_name));`,
		`CREATE OR REPLACE FUNCTION update_updated_at_column()
		RETURNS TRIGGER AS $$
		BEGIN
			NEW.updated_at = NOW();
			RETURN NEW;
		END;
		$$ language 'plpgsql';`,
		`DROP TRIGGER IF EXISTS update_collections_updated_at ON collections;`,
		`CREATE TRIGGER update_collections_updated_at
		BEFORE UPDATE ON collections
		FOR EACH ROW
		EXECUTE FUNCTION update_updated_at_column();`,
		`DROP TRIGGER IF EXISTS update_documents_updated_at ON documents;`,
		`CREATE TRIGGER update_documents_updated_at
		BEFORE UPDATE ON documents
		FOR EACH ROW
		EXECUTE FUNCTION update_updated_at_column();`,
	}

	for _, query := range queries {
		if _, err := dm.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	return nil
}
