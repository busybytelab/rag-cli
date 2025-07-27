package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/lib/pq"
)

// CollectionManagerImpl implements CollectionManager interface
type CollectionManagerImpl struct {
	db *sql.DB
}

// NewCollectionManager creates a new collection manager
func NewCollectionManager(db *sql.DB) CollectionManager {
	return &CollectionManagerImpl{db: db}
}

// CreateCollection creates a new collection
func (cm *CollectionManagerImpl) CreateCollection(name, description string, folders []string) (*Collection, error) {
	query := `
		INSERT INTO collections (name, description, folders)
		VALUES ($1, $2, $3)
		RETURNING id, name, description, folders, stats, created_at, updated_at
	`

	var statsJSON string
	collection := &Collection{}

	err := cm.db.QueryRow(query, name, description, pq.Array(folders)).Scan(
		&collection.ID,
		&collection.Name,
		&collection.Description,
		pq.Array(&collection.Folders),
		&statsJSON,
		&collection.CreatedAt,
		&collection.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create collection: %w", err)
	}

	// Parse stats JSON
	if err := json.Unmarshal([]byte(statsJSON), &collection.Stats); err != nil {
		return nil, fmt.Errorf("failed to parse stats: %w", err)
	}

	return collection, nil
}

// GetCollection retrieves a collection by ID
func (cm *CollectionManagerImpl) GetCollection(id string) (*Collection, error) {
	query := `
		SELECT id, name, description, folders, stats, created_at, updated_at
		FROM collections
		WHERE id = $1
	`

	var statsJSON string
	collection := &Collection{}

	err := cm.db.QueryRow(query, id).Scan(
		&collection.ID,
		&collection.Name,
		&collection.Description,
		pq.Array(&collection.Folders),
		&statsJSON,
		&collection.CreatedAt,
		&collection.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}

	// Parse stats JSON
	if err := json.Unmarshal([]byte(statsJSON), &collection.Stats); err != nil {
		return nil, fmt.Errorf("failed to parse stats: %w", err)
	}

	return collection, nil
}

// ListCollections retrieves all collections
func (cm *CollectionManagerImpl) ListCollections() ([]*Collection, error) {
	query := `
		SELECT id, name, description, folders, stats, created_at, updated_at
		FROM collections
		ORDER BY created_at DESC
	`

	rows, err := cm.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query collections: %w", err)
	}
	defer rows.Close()

	var collections []*Collection
	for rows.Next() {
		var statsJSON string
		collection := &Collection{}

		err := rows.Scan(
			&collection.ID,
			&collection.Name,
			&collection.Description,
			pq.Array(&collection.Folders),
			&statsJSON,
			&collection.CreatedAt,
			&collection.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan collection: %w", err)
		}

		// Parse stats JSON
		if err := json.Unmarshal([]byte(statsJSON), &collection.Stats); err != nil {
			return nil, fmt.Errorf("failed to parse stats: %w", err)
		}

		collections = append(collections, collection)
	}

	return collections, nil
}

// DeleteCollection deletes a collection and all its documents
func (cm *CollectionManagerImpl) DeleteCollection(id string) error {
	query := `DELETE FROM collections WHERE id = $1`

	result, err := cm.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("collection not found")
	}

	return nil
}

// UpdateCollectionStats updates collection statistics
func (cm *CollectionManagerImpl) UpdateCollectionStats(collectionID string) error {
	query := `
		UPDATE collections 
		SET stats = (
			SELECT jsonb_build_object(
				'total_documents', COUNT(DISTINCT file_path),
				'total_chunks', COUNT(*),
				'total_size', COALESCE(SUM(length(content)), 0)
			)
			FROM documents 
			WHERE collection_id = $1
		)
		WHERE id = $1
	`

	_, err := cm.db.Exec(query, collectionID)
	if err != nil {
		return fmt.Errorf("failed to update collection stats: %w", err)
	}

	return nil
}

// isUUID checks if a string is a valid UUID format
func isUUID(str string) bool {
	uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	return uuidRegex.MatchString(strings.ToLower(str))
}

// GetCollectionByIdOrName retrieves a collection by ID (UUID) or name
// If the input looks like a UUID, it uses GetCollection directly
// Otherwise, it searches by name and handles multiple matches
func (cm *CollectionManagerImpl) GetCollectionByIdOrName(collectionIdOrName string) (*Collection, error) {
	// Check if input looks like a UUID
	if isUUID(collectionIdOrName) {
		return cm.GetCollection(collectionIdOrName)
	}

	// Search by name
	query := `
		SELECT id, name, description, folders, stats, created_at, updated_at
		FROM collections
		WHERE name = $1
		ORDER BY created_at DESC
	`

	rows, err := cm.db.Query(query, collectionIdOrName)
	if err != nil {
		return nil, fmt.Errorf("failed to query collections by name: %w", err)
	}
	defer rows.Close()

	var collections []*Collection
	for rows.Next() {
		var statsJSON string
		collection := &Collection{}

		err := rows.Scan(
			&collection.ID,
			&collection.Name,
			&collection.Description,
			pq.Array(&collection.Folders),
			&statsJSON,
			&collection.CreatedAt,
			&collection.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan collection: %w", err)
		}

		// Parse stats JSON
		if err := json.Unmarshal([]byte(statsJSON), &collection.Stats); err != nil {
			return nil, fmt.Errorf("failed to parse stats: %w", err)
		}

		collections = append(collections, collection)
	}

	if len(collections) == 0 {
		return nil, fmt.Errorf("collection not found: %s", collectionIdOrName)
	}

	if len(collections) > 1 {
		// Show warning about multiple matches
		fmt.Printf("Warning: Multiple collections found with name '%s'. Using the first match (ID: %s).\n", collectionIdOrName, collections[0].ID)
		fmt.Printf("To avoid ambiguity, use the collection ID instead.\n")
	}

	return collections[0], nil
}

// UpdateCollection updates a collection's name and description
func (cm *CollectionManagerImpl) UpdateCollection(id string, name *string, description *string) (*Collection, error) {
	// Check if any fields are being updated
	if name == nil && description == nil {
		return nil, fmt.Errorf("no fields to update")
	}

	// Build dynamic query based on which fields are being updated
	var query string
	var args []interface{}
	argIndex := 1

	// Start with the base query
	query = `UPDATE collections SET updated_at = NOW()`

	// Add name update if provided
	if name != nil {
		query += fmt.Sprintf(", name = $%d", argIndex+1)
		args = append(args, *name)
		argIndex++
	}

	// Add description update if provided
	if description != nil {
		query += fmt.Sprintf(", description = $%d", argIndex+1)
		args = append(args, *description)
		argIndex++
	}

	// Add WHERE clause and RETURNING
	query += fmt.Sprintf(" WHERE id = $%d", argIndex+1)
	query += " RETURNING id, name, description, folders, stats, created_at, updated_at"
	args = append(args, id)

	var statsJSON string
	collection := &Collection{}

	err := cm.db.QueryRow(query, args...).Scan(
		&collection.ID,
		&collection.Name,
		&collection.Description,
		pq.Array(&collection.Folders),
		&statsJSON,
		&collection.CreatedAt,
		&collection.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update collection: %w", err)
	}

	// Parse stats JSON
	if err := json.Unmarshal([]byte(statsJSON), &collection.Stats); err != nil {
		return nil, fmt.Errorf("failed to parse stats: %w", err)
	}

	return collection, nil
}

// AddFolderToCollection adds a folder to a collection
func (cm *CollectionManagerImpl) AddFolderToCollection(id, folder string) (*Collection, error) {
	// First get the current collection to check if folder already exists
	collection, err := cm.GetCollection(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}

	// Check if folder already exists
	for _, existingFolder := range collection.Folders {
		if existingFolder == folder {
			return nil, fmt.Errorf("folder '%s' already exists in collection", folder)
		}
	}

	// Add the new folder
	newFolders := append(collection.Folders, folder)

	query := `
		UPDATE collections 
		SET folders = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, description, folders, stats, created_at, updated_at
	`

	var statsJSON string
	updatedCollection := &Collection{}

	err = cm.db.QueryRow(query, id, pq.Array(newFolders)).Scan(
		&updatedCollection.ID,
		&updatedCollection.Name,
		&updatedCollection.Description,
		pq.Array(&updatedCollection.Folders),
		&statsJSON,
		&updatedCollection.CreatedAt,
		&updatedCollection.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to add folder to collection: %w", err)
	}

	// Parse stats JSON
	if err := json.Unmarshal([]byte(statsJSON), &updatedCollection.Stats); err != nil {
		return nil, fmt.Errorf("failed to parse stats: %w", err)
	}

	return updatedCollection, nil
}

// RemoveFolderFromCollection removes a folder from a collection and deletes associated documents
func (cm *CollectionManagerImpl) RemoveFolderFromCollection(id, folder string) (*Collection, error) {
	// First get the current collection to check if folder exists
	collection, err := cm.GetCollection(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}

	// Check if folder exists
	folderExists := false
	var newFolders []string
	for _, existingFolder := range collection.Folders {
		if existingFolder == folder {
			folderExists = true
		} else {
			newFolders = append(newFolders, existingFolder)
		}
	}

	if !folderExists {
		return nil, fmt.Errorf("folder '%s' does not exist in collection", folder)
	}

	// Delete documents from the folder
	documentMgr := NewDocumentManager(cm.db)
	err = documentMgr.DeleteDocumentsByFolder(id, folder)
	if err != nil {
		return nil, fmt.Errorf("failed to delete documents from folder: %w", err)
	}

	// Update collection folders
	query := `
		UPDATE collections 
		SET folders = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, description, folders, stats, created_at, updated_at
	`

	var statsJSON string
	updatedCollection := &Collection{}

	err = cm.db.QueryRow(query, id, pq.Array(newFolders)).Scan(
		&updatedCollection.ID,
		&updatedCollection.Name,
		&updatedCollection.Description,
		pq.Array(&updatedCollection.Folders),
		&statsJSON,
		&updatedCollection.CreatedAt,
		&updatedCollection.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to remove folder from collection: %w", err)
	}

	// Parse stats JSON
	if err := json.Unmarshal([]byte(statsJSON), &updatedCollection.Stats); err != nil {
		return nil, fmt.Errorf("failed to parse stats: %w", err)
	}

	return updatedCollection, nil
}
