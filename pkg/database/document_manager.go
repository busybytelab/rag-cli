package database

import (
	"database/sql"
	"fmt"

	"github.com/pgvector/pgvector-go"
)

// DocumentManagerImpl implements DocumentManager interface
type DocumentManagerImpl struct {
	db *sql.DB
}

// NewDocumentManager creates a new document manager
func NewDocumentManager(db *sql.DB) DocumentManager {
	return &DocumentManagerImpl{db: db}
}

// InsertDocument inserts a new document
func (dm *DocumentManagerImpl) InsertDocument(doc *Document) error {
	query := `
		INSERT INTO documents (collection_id, file_path, file_name, content, chunk_index, embedding, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`

	// Convert embedding to vector type
	embeddingVector := pgvector.NewVector(doc.Embedding)

	err := dm.db.QueryRow(query, doc.CollectionID, doc.FilePath, doc.FileName, doc.Content, doc.ChunkIndex, embeddingVector, doc.Metadata, doc.CreatedAt, doc.UpdatedAt).Scan(
		&doc.ID,
		&doc.CreatedAt,
		&doc.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert document: %w", err)
	}

	return nil
}

// DeleteDocumentsByPath deletes all documents with a specific file path
func (dm *DocumentManagerImpl) DeleteDocumentsByPath(collectionID, filePath string) error {
	query := `DELETE FROM documents WHERE collection_id = $1 AND file_path = $2`

	_, err := dm.db.Exec(query, collectionID, filePath)
	if err != nil {
		return fmt.Errorf("failed to delete documents: %w", err)
	}

	return nil
}

// DeleteDocumentsByFolder deletes all documents from a specific folder in a collection
func (dm *DocumentManagerImpl) DeleteDocumentsByFolder(collectionID, folder string) error {
	query := `DELETE FROM documents WHERE collection_id = $1 AND file_path LIKE $2`

	// Use LIKE with wildcard to match folder path
	folderPattern := folder + "/%"

	_, err := dm.db.Exec(query, collectionID, folderPattern)
	if err != nil {
		return fmt.Errorf("failed to delete documents from folder: %w", err)
	}

	return nil
}

// ListDocumentsByFolder lists documents from a specific folder in a collection
func (dm *DocumentManagerImpl) ListDocumentsByFolder(collectionID, folder string, limit, offset int) ([]*Document, error) {
	query := `
		SELECT id, collection_id, file_path, file_name, content, chunk_index, embedding, metadata, created_at, updated_at
		FROM documents 
		WHERE collection_id = $1 AND file_path LIKE $2
		ORDER BY file_path ASC
		LIMIT $3 OFFSET $4
	`

	// Use LIKE with wildcard to match folder path
	folderPattern := folder + "/%"

	rows, err := dm.db.Query(query, collectionID, folderPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query documents: %w", err)
	}
	defer rows.Close()

	var documents []*Document
	for rows.Next() {
		doc := &Document{}
		var embeddingVector pgvector.Vector

		err := rows.Scan(
			&doc.ID,
			&doc.CollectionID,
			&doc.FilePath,
			&doc.FileName,
			&doc.Content,
			&doc.ChunkIndex,
			&embeddingVector,
			&doc.Metadata,
			&doc.CreatedAt,
			&doc.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}

		// Convert vector back to float32 slice
		doc.Embedding = embeddingVector.Slice()

		documents = append(documents, doc)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over documents: %w", err)
	}

	return documents, nil
}

// ListDocumentsByFolderWithFilter lists documents from a specific folder in a collection with file pattern filtering
func (dm *DocumentManagerImpl) ListDocumentsByFolderWithFilter(collectionID, folder, fileFilter string, limit, offset int) ([]*Document, error) {
	var query string
	var args []interface{}

	if fileFilter != "" {
		query = `
			SELECT id, collection_id, file_path, file_name, content, chunk_index, embedding, metadata, created_at, updated_at
			FROM documents 
			WHERE collection_id = $1 AND file_path LIKE $2 AND file_name LIKE $3
			ORDER BY file_path ASC
			LIMIT $4 OFFSET $5
		`
		// Use LIKE with wildcard to match folder path
		folderPattern := folder + "/%"
		args = []interface{}{collectionID, folderPattern, fileFilter, limit, offset}
	} else {
		query = `
			SELECT id, collection_id, file_path, file_name, content, chunk_index, embedding, metadata, created_at, updated_at
			FROM documents 
			WHERE collection_id = $1 AND file_path LIKE $2
			ORDER BY file_path ASC
			LIMIT $3 OFFSET $4
		`
		// Use LIKE with wildcard to match folder path
		folderPattern := folder + "/%"
		args = []interface{}{collectionID, folderPattern, limit, offset}
	}

	rows, err := dm.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query documents: %w", err)
	}
	defer rows.Close()

	var documents []*Document
	for rows.Next() {
		doc := &Document{}
		var embeddingVector pgvector.Vector

		err := rows.Scan(
			&doc.ID,
			&doc.CollectionID,
			&doc.FilePath,
			&doc.FileName,
			&doc.Content,
			&doc.ChunkIndex,
			&embeddingVector,
			&doc.Metadata,
			&doc.CreatedAt,
			&doc.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}

		// Convert vector back to float32 slice
		doc.Embedding = embeddingVector.Slice()

		documents = append(documents, doc)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over documents: %w", err)
	}

	return documents, nil
}

// GetDocumentByID retrieves a document by its ID
func (dm *DocumentManagerImpl) GetDocumentByID(documentID string) (*Document, error) {
	query := `
		SELECT id, collection_id, file_path, file_name, content, chunk_index, embedding, metadata, created_at, updated_at
		FROM documents 
		WHERE id = $1
	`

	var doc Document
	var embeddingVector pgvector.Vector

	err := dm.db.QueryRow(query, documentID).Scan(
		&doc.ID,
		&doc.CollectionID,
		&doc.FilePath,
		&doc.FileName,
		&doc.Content,
		&doc.ChunkIndex,
		&embeddingVector,
		&doc.Metadata,
		&doc.CreatedAt,
		&doc.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	// Convert vector back to float32 slice
	doc.Embedding = embeddingVector.Slice()

	return &doc, nil
}

// DeleteDocumentByID deletes a document by its ID
func (dm *DocumentManagerImpl) DeleteDocumentByID(documentID string) error {
	query := `DELETE FROM documents WHERE id = $1`

	result, err := dm.db.Exec(query, documentID)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("document with ID '%s' not found", documentID)
	}

	return nil
}

// GetDocumentByPathAndIndex retrieves a document by collection ID, file path, and chunk index
func (dm *DocumentManagerImpl) GetDocumentByPathAndIndex(collectionID, filePath string, chunkIndex int) (*Document, error) {
	query := `
		SELECT id, collection_id, file_path, file_name, content, chunk_index, embedding, metadata, created_at, updated_at
		FROM documents 
		WHERE collection_id = $1 AND file_path = $2 AND chunk_index = $3
	`

	var doc Document
	var embeddingVector pgvector.Vector

	err := dm.db.QueryRow(query, collectionID, filePath, chunkIndex).Scan(
		&doc.ID,
		&doc.CollectionID,
		&doc.FilePath,
		&doc.FileName,
		&doc.Content,
		&doc.ChunkIndex,
		&embeddingVector,
		&doc.Metadata,
		&doc.CreatedAt,
		&doc.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	// Convert vector back to float32 slice
	doc.Embedding = embeddingVector.Slice()

	return &doc, nil
}
