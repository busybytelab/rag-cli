package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockDatabaseManager implements DatabaseManager for testing
type MockDatabaseManager struct {
	CollectionManager
	DocumentManager
	SearchEngine
}

func (m *MockDatabaseManager) Close() error {
	return nil
}

func (m *MockDatabaseManager) InitSchema() error {
	return nil
}

func TestSearchTypeString(t *testing.T) {
	tests := []struct {
		searchType SearchType
		expected   string
	}{
		{SearchTypeVector, "vector"},
		{SearchTypeText, "text"},
		{SearchTypeHybrid, "hybrid"},
		{SearchTypeSemantic, "semantic"},
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, string(test.searchType),
			"SearchType %s should convert to string '%s'", test.searchType, test.expected)
	}
}

func TestRankSearchResults(t *testing.T) {
	// Create test results with different scores
	results := []*SearchResult{
		{CombinedScore: 0.5, Rank: 0},
		{CombinedScore: 0.8, Rank: 0},
		{CombinedScore: 0.3, Rank: 0},
		{CombinedScore: 0.9, Rank: 0},
	}

	// Create a mock search engine for testing
	searchEngine := &SearchEngineImpl{}
	ranked := searchEngine.RankSearchResults(results)

	// Check that results are sorted by score in descending order
	expectedScores := []float64{0.9, 0.8, 0.5, 0.3}
	require.Len(t, ranked, len(expectedScores), "Should have same number of results")

	for i, result := range ranked {
		assert.Equal(t, expectedScores[i], result.CombinedScore,
			"Expected score %.1f at position %d, got %.1f", expectedScores[i], i, result.CombinedScore)
		assert.Equal(t, i+1, result.Rank,
			"Expected rank %d at position %d, got %d", i+1, i, result.Rank)
	}
}

func TestFilterSearchResults(t *testing.T) {
	results := []*SearchResult{
		{CombinedScore: 0.5},
		{CombinedScore: 0.8},
		{CombinedScore: 0.3},
		{CombinedScore: 0.9},
	}

	// Create a mock search engine for testing
	searchEngine := &SearchEngineImpl{}

	// Test filtering with minScore 0.6
	filtered := searchEngine.FilterSearchResults(results, 0.6)
	assert.Len(t, filtered, 2, "Should have 2 results above threshold 0.6")

	// Check that only high-scoring results remain
	for _, result := range filtered {
		assert.GreaterOrEqual(t, result.CombinedScore, 0.6,
			"Found result with score %.1f below threshold 0.6", result.CombinedScore)
	}

	// Test filtering with minScore 0 (should return all results)
	allResults := searchEngine.FilterSearchResults(results, 0)
	assert.Len(t, allResults, 4, "Should return all 4 results when minScore is 0")
}

func TestGetSearchStats(t *testing.T) {
	results := []*SearchResult{
		{VectorScore: 0.8, TextScore: 0.6, CombinedScore: 0.7},
		{VectorScore: 0.9, TextScore: 0.7, CombinedScore: 0.8},
		{VectorScore: 0.7, TextScore: 0.5, CombinedScore: 0.6},
	}

	// Create a mock search engine for testing
	searchEngine := &SearchEngineImpl{}
	stats := searchEngine.GetSearchStats(results)

	// Check total results
	assert.Equal(t, 3, stats["total_results"], "Should have 3 total results")

	// Check average scores (with some tolerance for floating point precision)
	avgCombined := stats["avg_combined_score"].(float64)
	expectedAvg := (0.7 + 0.8 + 0.6) / 3.0
	assert.InDelta(t, expectedAvg, avgCombined, 0.01,
		"Expected average combined score %.3f, got %.3f", expectedAvg, avgCombined)

	// Check min/max scores
	assert.Equal(t, 0.6, stats["min_score"], "Min score should be 0.6")
	assert.Equal(t, 0.8, stats["max_score"], "Max score should be 0.8")
}

func TestGetSearchStatsEmpty(t *testing.T) {
	// Create a mock search engine for testing
	searchEngine := &SearchEngineImpl{}
	stats := searchEngine.GetSearchStats([]*SearchResult{})

	// Check that empty results return zero values
	assert.Equal(t, 0, stats["total_results"], "Should have 0 total results")
	assert.Equal(t, 0.0, stats["avg_combined_score"], "Should have 0.0 average score")
	assert.Equal(t, 0.0, stats["avg_vector_score"], "Should have 0.0 average vector score")
	assert.Equal(t, 0.0, stats["avg_text_score"], "Should have 0.0 average text score")
	assert.Equal(t, 0.0, stats["min_score"], "Should have 0.0 min score")
	assert.Equal(t, 0.0, stats["max_score"], "Should have 0.0 max score")
}
