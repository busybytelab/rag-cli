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
	db               *sql.DB
	migrationManager *MigrationManager
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
		db:               db,
		migrationManager: NewMigrationManager(db),
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

// InitSchema initializes the database schema using migrations
func (dm *DatabaseManagerImpl) InitSchema() error {
	// Run all pending migrations
	if err := dm.migrationManager.Migrate(-1); err != nil {
		return fmt.Errorf("failed to run database migrations: %w", err)
	}
	return nil
}

// GetEmbeddingDimensions gets the embedding dimensions for a collection
func (dm *DatabaseManagerImpl) GetEmbeddingDimensions(collectionID string) (int, error) {
	return dm.migrationManager.GetEmbeddingDimensions(collectionID)
}

// SetEmbeddingDimensions sets the embedding dimensions for a collection
func (dm *DatabaseManagerImpl) SetEmbeddingDimensions(collectionID string, dimensions int, modelName string) error {
	return dm.migrationManager.SetEmbeddingDimensions(collectionID, dimensions, modelName)
}

// GetMigrationVersion gets the current migration version
func (dm *DatabaseManagerImpl) GetMigrationVersion() (int, error) {
	return dm.migrationManager.GetCurrentVersion()
}

// RunMigrations runs migrations to a target version
func (dm *DatabaseManagerImpl) RunMigrations(targetVersion int) error {
	return dm.migrationManager.Migrate(targetVersion)
}

// GetTotalMigrations returns the total number of available migrations
func (dm *DatabaseManagerImpl) GetTotalMigrations() int {
	return len(dm.migrationManager.migrations)
}
