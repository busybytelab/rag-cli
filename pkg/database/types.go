package database

import (
	"time"
)

// CollectionManager defines operations for managing collections
type CollectionManager interface {
	// Collection CRUD operations
	CreateCollection(name, description string, folders []string) (*Collection, error)
	GetCollection(id string) (*Collection, error)
	GetCollectionByIdOrName(collectionIdOrName string) (*Collection, error)
	ListCollections() ([]*Collection, error)
	DeleteCollection(id string) error
	UpdateCollectionStats(collectionID string) error

	// Collection editing operations
	UpdateCollection(id string, name *string, description *string) (*Collection, error)
	AddFolderToCollection(id, folder string) (*Collection, error)
	RemoveFolderFromCollection(id, folder string) (*Collection, error)
}

// DocumentManager defines operations for managing documents
type DocumentManager interface {
	// Document operations
	InsertDocument(doc *Document) error
	DeleteDocumentsByPath(collectionID, filePath string) error
	DeleteDocumentsByFolder(collectionID, folder string) error
	DeleteDocumentByID(documentID string) error
	ListDocumentsByFolder(collectionID, folder string, limit, offset int) ([]*Document, error)
	ListDocumentsByFolderWithFilter(collectionID, folder, fileFilter string, limit, offset int) ([]*Document, error)
	GetDocumentByID(documentID string) (*Document, error)
	GetDocumentByPathAndIndex(collectionID, filePath string, chunkIndex int) (*Document, error)
}

// SearchEngine defines operations for searching documents
type SearchEngine interface {
	// Search operations
	SearchDocuments(collectionID string, embedding []float32, limit int) ([]*Document, error)
	SearchDocumentsWithOptions(collectionID string, embedding []float32, textQuery string, limit int, opts *SearchOptions) ([]*SearchResult, error)

	// Search result processing
	RankSearchResults(results []*SearchResult) []*SearchResult
	FilterSearchResults(results []*SearchResult, minScore float64) []*SearchResult
	GetSearchStats(results []*SearchResult) map[string]interface{}
}

// DatabaseManager manages database connection and schema
type DatabaseManager interface {
	// Connection management
	Close() error
	InitSchema() error

	// Embedding dimension management
	GetEmbeddingDimensions(collectionID string) (int, error)
	SetEmbeddingDimensions(collectionID string, dimensions int, modelName string) error

	// Migration management
	GetMigrationVersion() (int, error)
	RunMigrations(targetVersion int) error
	GetTotalMigrations() int
}

// Common types used across interfaces
type SearchType string

const (
	SearchTypeVector   SearchType = "vector"   // Vector similarity search only
	SearchTypeText     SearchType = "text"     // Full-text search only
	SearchTypeHybrid   SearchType = "hybrid"   // Combined vector and text search
	SearchTypeSemantic SearchType = "semantic" // Semantic search (vector with filters)
)

// SearchOptions represents search configuration options
type SearchOptions struct {
	SearchType    SearchType `json:"search_type"`
	VectorWeight  float64    `json:"vector_weight"`   // Weight for vector similarity (0.0-1.0)
	TextWeight    float64    `json:"text_weight"`     // Weight for text similarity (0.0-1.0)
	MinScore      float64    `json:"min_score"`       // Minimum similarity score
	MaxDistance   float64    `json:"max_distance"`    // Maximum vector distance
	FileFilter    string     `json:"file_filter"`     // File name pattern filter
	ContentFilter string     `json:"content_filter"`  // Content text filter
	UseFuzzyMatch bool       `json:"use_fuzzy_match"` // Enable fuzzy text matching
	FuzzyDistance int        `json:"fuzzy_distance"`  // Levenshtein distance for fuzzy matching
}

// SearchResult represents a search result with scoring information
type SearchResult struct {
	Document      *Document `json:"document"`
	VectorScore   float64   `json:"vector_score"`   // Vector similarity score (0-1, higher is better)
	TextScore     float64   `json:"text_score"`     // Text search score (0-1, higher is better)
	CombinedScore float64   `json:"combined_score"` // Combined weighted score
	Rank          int       `json:"rank"`           // Result rank
}

// Document represents a document in the database
type Document struct {
	ID           string    `json:"id"`
	CollectionID string    `json:"collection_id"`
	FilePath     string    `json:"file_path"`
	FileName     string    `json:"file_name"`
	Content      string    `json:"content"`
	ChunkIndex   int       `json:"chunk_index"`
	Embedding    []float32 `json:"embedding"`
	Metadata     string    `json:"metadata"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Collection represents a collection in the database
type Collection struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Folders     []string  `json:"folders"`
	Stats       Stats     `json:"stats"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Stats represents collection statistics
type Stats struct {
	TotalDocuments int   `json:"total_documents"`
	TotalChunks    int   `json:"total_chunks"`
	TotalSize      int64 `json:"total_size"`
}
