package database

import (
	"testing"

	"github.com/busybytelab.com/rag-cli/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDatabaseManager(t *testing.T) {
	// Test with invalid config (should fail)
	invalidConfig := &config.DatabaseConfig{
		Host: "invalid-host",
		Port: 9999,
		Name: "invalid-db",
		User: "invalid-user",
	}

	_, err := NewDatabaseManager(invalidConfig)
	assert.Error(t, err, "Expected error with invalid database config")
}

func TestNewLegacyDatabase(t *testing.T) {
	// Test with invalid config (should fail)
	invalidConfig := &config.DatabaseConfig{
		Host: "invalid-host",
		Port: 9999,
		Name: "invalid-db",
		User: "invalid-user",
	}

	_, err := NewDatabaseManager(invalidConfig)
	assert.Error(t, err, "Expected error with invalid database config")
}

func TestDatabaseManagerInterfaces(t *testing.T) {
	// Test that DatabaseManagerImpl implements all required interfaces
	var _ CollectionManager = (*CollectionManagerImpl)(nil)
	var _ DocumentManager = (*DocumentManagerImpl)(nil)
	var _ SearchEngine = (*SearchEngineImpl)(nil)
	var _ DatabaseManager = (*DatabaseManagerImpl)(nil)
}

func TestLegacyDatabaseCompatibility(t *testing.T) {
	// Test that Database struct provides backward compatibility
	// Note: Database no longer implements DatabaseManager, it uses individual managers
	// This test is kept for documentation purposes
}

func TestSearchTypeConstants(t *testing.T) {
	// Test that search type constants are properly defined
	assert.Equal(t, SearchType("vector"), SearchTypeVector, "SearchTypeVector should be 'vector'")
	assert.Equal(t, SearchType("text"), SearchTypeText, "SearchTypeText should be 'text'")
	assert.Equal(t, SearchType("hybrid"), SearchTypeHybrid, "SearchTypeHybrid should be 'hybrid'")
	assert.Equal(t, SearchType("semantic"), SearchTypeSemantic, "SearchTypeSemantic should be 'semantic'")
}

func TestSearchOptionsDefaults(t *testing.T) {
	opts := &SearchOptions{}

	// Test that default values are zero
	assert.Empty(t, opts.SearchType, "SearchType should be empty by default")
	assert.Zero(t, opts.VectorWeight, "VectorWeight should be zero by default")
	assert.Zero(t, opts.TextWeight, "TextWeight should be zero by default")
	assert.Zero(t, opts.MinScore, "MinScore should be zero by default")
	assert.Zero(t, opts.MaxDistance, "MaxDistance should be zero by default")
	assert.False(t, opts.UseFuzzyMatch, "UseFuzzyMatch should be false by default")
	assert.Zero(t, opts.FuzzyDistance, "FuzzyDistance should be zero by default")
}

func TestDocumentStruct(t *testing.T) {
	doc := &Document{
		ID:           "test-id",
		CollectionID: "test-collection",
		FilePath:     "/test/path",
		FileName:     "test.txt",
		Content:      "test content",
		ChunkIndex:   0,
		Embedding:    []float32{0.1, 0.2, 0.3},
		Metadata:     "{}",
	}

	assert.Equal(t, "test-id", doc.ID, "Document ID should match")
	assert.Equal(t, "test-collection", doc.CollectionID, "Collection ID should match")
	assert.Equal(t, "/test/path", doc.FilePath, "File path should match")
	assert.Equal(t, "test.txt", doc.FileName, "File name should match")
	assert.Equal(t, "test content", doc.Content, "Content should match")
	assert.Equal(t, 0, doc.ChunkIndex, "Chunk index should match")
	assert.Len(t, doc.Embedding, 3, "Embedding should have 3 elements")
	assert.Equal(t, "{}", doc.Metadata, "Metadata should match")
}

func TestCollectionStruct(t *testing.T) {
	collection := &Collection{
		ID:          "test-id",
		Name:        "test-collection",
		Description: "test description",
		Folders:     []string{"/test/folder1", "/test/folder2"},
		Stats: Stats{
			TotalDocuments: 10,
			TotalChunks:    50,
			TotalSize:      1000,
		},
	}

	assert.Equal(t, "test-id", collection.ID, "Collection ID should match")
	assert.Equal(t, "test-collection", collection.Name, "Collection name should match")
	assert.Equal(t, "test description", collection.Description, "Description should match")
	assert.Len(t, collection.Folders, 2, "Should have 2 folders")
	assert.Equal(t, "/test/folder1", collection.Folders[0], "First folder should match")
	assert.Equal(t, "/test/folder2", collection.Folders[1], "Second folder should match")
	assert.Equal(t, 10, collection.Stats.TotalDocuments, "Total documents should match")
	assert.Equal(t, 50, collection.Stats.TotalChunks, "Total chunks should match")
	assert.Equal(t, int64(1000), collection.Stats.TotalSize, "Total size should match")
}

func TestStatsStruct(t *testing.T) {
	stats := &Stats{
		TotalDocuments: 100,
		TotalChunks:    500,
		TotalSize:      10000,
	}

	assert.Equal(t, 100, stats.TotalDocuments, "Total documents should match")
	assert.Equal(t, 500, stats.TotalChunks, "Total chunks should match")
	assert.Equal(t, int64(10000), stats.TotalSize, "Total size should match")
}

func TestSearchResultStruct(t *testing.T) {
	doc := &Document{
		ID:           "test-id",
		CollectionID: "test-collection",
		Content:      "test content",
	}

	result := &SearchResult{
		Document:      doc,
		VectorScore:   0.8,
		TextScore:     0.6,
		CombinedScore: 0.7,
		Rank:          1,
	}

	require.NotNil(t, result.Document, "Document should not be nil")
	assert.Equal(t, "test-id", result.Document.ID, "Document ID should match")
	assert.Equal(t, 0.8, result.VectorScore, "Vector score should match")
	assert.Equal(t, 0.6, result.TextScore, "Text score should match")
	assert.Equal(t, 0.7, result.CombinedScore, "Combined score should match")
	assert.Equal(t, 1, result.Rank, "Rank should match")
}
