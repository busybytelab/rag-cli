package database

import (
	"database/sql"
	"fmt"
	"log"
)

// Migration represents a database migration
type Migration struct {
	Version     int
	Description string
	Up          func(*sql.Tx) error
	Down        func(*sql.Tx) error
}

// MigrationManager handles database migrations
type MigrationManager struct {
	db         *sql.DB
	migrations []Migration
}

// NewMigrationManager creates a new migration manager
func NewMigrationManager(db *sql.DB) *MigrationManager {
	mm := &MigrationManager{
		db:         db,
		migrations: []Migration{},
	}
	mm.registerMigrations()
	return mm
}

// registerMigrations registers all available migrations
func (mm *MigrationManager) registerMigrations() {
	mm.migrations = []Migration{
		{
			Version:     1,
			Description: "Create complete schema with configurable embedding dimensions",
			Up:          mm.migration001CreateCompleteSchema,
			Down:        mm.migration001CreateCompleteSchemaDown,
		},
	}
}

// GetCurrentVersion gets the current migration version
func (mm *MigrationManager) GetCurrentVersion() (int, error) {
	// Check if migrations table exists
	var exists bool
	err := mm.db.QueryRow(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'migrations'
		);
	`).Scan(&exists)
	if err != nil {
		return 0, fmt.Errorf("failed to check migrations table: %w", err)
	}

	if !exists {
		return 0, nil
	}

	var version int
	err = mm.db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM migrations").Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("failed to get current migration version: %w", err)
	}

	return version, nil
}

// Migrate runs all pending migrations
func (mm *MigrationManager) Migrate(targetVersion int) error {
	currentVersion, err := mm.GetCurrentVersion()
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	// Create migrations table if it doesn't exist
	if currentVersion == 0 {
		if err := mm.createMigrationsTable(); err != nil {
			return fmt.Errorf("failed to create migrations table: %w", err)
		}
	}

	if targetVersion == -1 {
		targetVersion = len(mm.migrations)
	}

	if targetVersion < currentVersion {
		return fmt.Errorf("downgrading migrations is not supported")
	}

	if targetVersion == currentVersion {
		log.Printf("Database is already at version %d", currentVersion)
		return nil
	}

	// Run pending migrations
	for i := currentVersion; i < targetVersion; i++ {
		if i >= len(mm.migrations) {
			break
		}

		migration := mm.migrations[i]
		log.Printf("Running migration %d: %s", migration.Version, migration.Description)

		tx, err := mm.db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration %d: %w", migration.Version, err)
		}

		if err := migration.Up(tx); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to run migration %d: %w", migration.Version, err)
		}

		// Record the migration
		_, err = tx.Exec("INSERT INTO migrations (version, description, applied_at) VALUES ($1, $2, NOW())",
			migration.Version, migration.Description)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %d: %w", migration.Version, err)
		}

		log.Printf("Migration %d completed successfully", migration.Version)
	}

	return nil
}

// createMigrationsTable creates the migrations tracking table
func (mm *MigrationManager) createMigrationsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS migrations (
			id SERIAL PRIMARY KEY,
			version INTEGER NOT NULL UNIQUE,
			description TEXT NOT NULL,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`
	_, err := mm.db.Exec(query)
	return err
}

// migration001CreateCompleteSchema creates the complete schema with configurable dimensions
func (mm *MigrationManager) migration001CreateCompleteSchema(tx *sql.Tx) error {
	queries := []string{
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
			embedding vector(1024),
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);`,
		`CREATE TABLE IF NOT EXISTS embedding_config (
			id SERIAL PRIMARY KEY,
			collection_id UUID NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
			dimensions INTEGER NOT NULL,
			model_name VARCHAR(255) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(collection_id)
		);`,

		`CREATE INDEX IF NOT EXISTS idx_documents_collection_id ON documents(collection_id);`,
		`CREATE INDEX IF NOT EXISTS idx_documents_file_path ON documents(file_path);`,
		`CREATE INDEX IF NOT EXISTS idx_documents_embedding_hnsw ON documents USING hnsw (embedding vector_cosine_ops);`,

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
		if _, err := tx.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	return nil
}

// migration001CreateCompleteSchemaDown drops the complete schema
func (mm *MigrationManager) migration001CreateCompleteSchemaDown(tx *sql.Tx) error {
	queries := []string{
		`DROP TABLE IF EXISTS documents CASCADE;`,
		`DROP TABLE IF EXISTS collections CASCADE;`,
		`DROP TABLE IF EXISTS embedding_config CASCADE;`,

		`DROP FUNCTION IF EXISTS update_updated_at_column CASCADE;`,
	}

	for _, query := range queries {
		if _, err := tx.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	return nil
}

// GetEmbeddingDimensions gets the embedding dimensions for a collection
func (mm *MigrationManager) GetEmbeddingDimensions(collectionID string) (int, error) {
	var dimensions int
	err := mm.db.QueryRow(`
		SELECT dimensions FROM embedding_config 
		WHERE collection_id = $1
	`, collectionID).Scan(&dimensions)

	if err == sql.ErrNoRows {
		// Return default dimensions if no config found
		return 768, nil
	}

	if err != nil {
		return 0, fmt.Errorf("failed to get embedding dimensions: %w", err)
	}

	return dimensions, nil
}

// SetEmbeddingDimensions sets the embedding dimensions for a collection
func (mm *MigrationManager) SetEmbeddingDimensions(collectionID string, dimensions int, modelName string) error {
	query := `
		INSERT INTO embedding_config (collection_id, dimensions, model_name, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (collection_id) 
		DO UPDATE SET 
			dimensions = EXCLUDED.dimensions,
			model_name = EXCLUDED.model_name,
			updated_at = NOW()
	`
	_, err := mm.db.Exec(query, collectionID, dimensions, modelName)
	return err
}
