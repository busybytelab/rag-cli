package reranking

import (
	"context"
	"fmt"
	"sort"

	"github.com/busybytelab.com/rag-cli/pkg/client"
	"github.com/busybytelab.com/rag-cli/pkg/database"
)

// Service provides reranking functionality for search results
type Service struct {
	reranker client.Reranker
}

// New creates a new reranking service
func New(reranker client.Reranker) *Service {
	return &Service{
		reranker: reranker,
	}
}

// RerankSearchResults reranks search results using the reranker model
func (s *Service) RerankSearchResults(ctx context.Context, query string, results []*database.SearchResult, instruction string) ([]*database.SearchResult, error) {
	if len(results) == 0 {
		return results, nil
	}

	// Extract document contents for reranking
	documents := make([]string, len(results))
	for i, result := range results {
		documents[i] = result.Document.Content
	}

	// Perform reranking
	rerankResults, err := s.reranker.Rerank(ctx, query, documents, instruction)
	if err != nil {
		return nil, fmt.Errorf("failed to rerank documents: %w", err)
	}

	// Create a map of document content to rerank result for quick lookup
	rerankMap := make(map[string]*client.RerankResult)
	for _, rr := range rerankResults {
		rerankMap[rr.Document] = &rr
	}

	// Update search results with reranking scores
	for _, result := range results {
		if rerankResult, exists := rerankMap[result.Document.Content]; exists {
			// Update the combined score to include reranking score
			// We'll use a weighted combination: 70% original score, 30% reranking score
			originalScore := result.CombinedScore
			rerankingScore := rerankResult.Score
			result.CombinedScore = 0.7*originalScore + 0.3*rerankingScore
			result.Rank = rerankResult.Rank
		}
	}

	// Sort by new combined score
	sort.Slice(results, func(i, j int) bool {
		return results[i].CombinedScore > results[j].CombinedScore
	})

	// Update ranks after sorting
	for i, result := range results {
		result.Rank = i + 1
	}

	return results, nil
}

// RerankWithOptions reranks search results with additional options
func (s *Service) RerankWithOptions(ctx context.Context, query string, results []*database.SearchResult, opts *RerankOptions) ([]*database.SearchResult, error) {
	if opts == nil {
		opts = &RerankOptions{
			Instruction:    "Given a search query, retrieve relevant passages that answer the query",
			OriginalWeight: 0.7,
			RerankWeight:   0.3,
		}
	}

	if len(results) == 0 {
		return results, nil
	}

	// Extract document contents for reranking
	documents := make([]string, len(results))
	for i, result := range results {
		documents[i] = result.Document.Content
	}

	// Perform reranking
	rerankResults, err := s.reranker.Rerank(ctx, query, documents, opts.Instruction)
	if err != nil {
		return nil, fmt.Errorf("failed to rerank documents: %w", err)
	}

	// Create a map of document content to rerank result for quick lookup
	rerankMap := make(map[string]*client.RerankResult)
	for _, rr := range rerankResults {
		rerankMap[rr.Document] = &rr
	}

	// Update search results with reranking scores
	for _, result := range results {
		if rerankResult, exists := rerankMap[result.Document.Content]; exists {
			// Update the combined score using the specified weights
			originalScore := result.CombinedScore
			rerankingScore := rerankResult.Score
			result.CombinedScore = opts.OriginalWeight*originalScore + opts.RerankWeight*rerankingScore
			result.Rank = rerankResult.Rank
		}
	}

	// Sort by new combined score
	sort.Slice(results, func(i, j int) bool {
		return results[i].CombinedScore > results[j].CombinedScore
	})

	// Update ranks after sorting
	for i, result := range results {
		result.Rank = i + 1
	}

	// Apply minimum score filter if specified
	if opts.MinScore > 0 {
		var filtered []*database.SearchResult
		for _, result := range results {
			if result.CombinedScore >= opts.MinScore {
				filtered = append(filtered, result)
			}
		}
		results = filtered
	}

	// Apply limit if specified
	if opts.Limit > 0 && len(results) > opts.Limit {
		results = results[:opts.Limit]
	}

	return results, nil
}

// RerankOptions represents options for reranking
type RerankOptions struct {
	Instruction    string  `json:"instruction"`     // Custom instruction for reranking
	OriginalWeight float64 `json:"original_weight"` // Weight for original search score (0.0-1.0)
	RerankWeight   float64 `json:"rerank_weight"`   // Weight for reranking score (0.0-1.0)
	MinScore       float64 `json:"min_score"`       // Minimum combined score threshold
	Limit          int     `json:"limit"`           // Maximum number of results to return
}
