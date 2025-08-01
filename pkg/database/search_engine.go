package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/busybytelab.com/rag-cli/pkg/client"
	"github.com/pgvector/pgvector-go"
)

// SearchEngineImpl implements SearchEngine interface
type SearchEngineImpl struct {
	db       *sql.DB
	reranker client.Reranker
}

// NewSearchEngine creates a new search engine
func NewSearchEngine(db *sql.DB) SearchEngine {
	return &SearchEngineImpl{db: db}
}

// NewSearchEngineWithReranker creates a new search engine with reranking capability
func NewSearchEngineWithReranker(db *sql.DB, reranker client.Reranker) SearchEngine {
	return &SearchEngineImpl{
		db:       db,
		reranker: reranker,
	}
}

// SearchDocuments performs similarity search using vector similarity
func (se *SearchEngineImpl) SearchDocuments(collectionID string, embedding []float32, limit int) ([]*Document, error) {
	// Use default search options for backward compatibility
	opts := &SearchOptions{
		SearchType:   SearchTypeVector,
		VectorWeight: 1.0,
		TextWeight:   0.0,
		MinScore:     0.0,
		MaxDistance:  1.0,
	}

	results, err := se.SearchDocumentsWithOptions(collectionID, embedding, "", limit, opts)
	if err != nil {
		return nil, err
	}

	// Convert SearchResult to Document for backward compatibility
	documents := make([]*Document, len(results))
	for i, result := range results {
		documents[i] = result.Document
	}

	return documents, nil
}

// SearchDocumentsWithOptions performs advanced search with various options
func (se *SearchEngineImpl) SearchDocumentsWithOptions(collectionID string, embedding []float32, textQuery string, limit int, opts *SearchOptions) ([]*SearchResult, error) {
	if opts == nil {
		opts = &SearchOptions{
			SearchType:   SearchTypeHybrid,
			VectorWeight: 0.7,
			TextWeight:   0.3,
			MinScore:     0.0,
			MaxDistance:  1.0,
		}
	}

	var results []*SearchResult
	var err error

	switch opts.SearchType {
	case SearchTypeVector:
		results, err = se.searchVectorOnly(collectionID, embedding, limit, opts)
	case SearchTypeText:
		results, err = se.searchTextOnly(collectionID, textQuery, limit, opts)
	case SearchTypeHybrid:
		results, err = se.searchHybrid(collectionID, embedding, textQuery, limit, opts)
	case SearchTypeSemantic:
		results, err = se.searchSemantic(collectionID, embedding, textQuery, limit, opts)
	default:
		results, err = se.searchHybrid(collectionID, embedding, textQuery, limit, opts)
	}

	if err != nil {
		return nil, err
	}

	// Apply reranking if enabled and reranker is available
	if opts.EnableReranking && se.reranker != nil {
		results, err = se.applyReranking(context.Background(), textQuery, results, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to apply reranking: %w", err)
		}
	}

	return results, nil
}

// searchVectorOnly performs vector similarity search only
func (se *SearchEngineImpl) searchVectorOnly(collectionID string, embedding []float32, limit int, opts *SearchOptions) ([]*SearchResult, error) {
	query := `
		SELECT id, collection_id, file_path, file_name, content, chunk_index, embedding, metadata, created_at, updated_at,
		       1 - (embedding <=> $2) as vector_score
		FROM documents
		WHERE collection_id = $1
		  AND (embedding <=> $2) <= $3
		ORDER BY embedding <=> $2 ASC
		LIMIT $4
	`

	searchVector := pgvector.NewVector(embedding)
	maxDistance := opts.MaxDistance
	if maxDistance <= 0 {
		maxDistance = 1.0
	}

	rows, err := se.db.Query(query, collectionID, searchVector, maxDistance, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search documents: %w", err)
	}
	defer rows.Close()

	var results []*SearchResult
	for rows.Next() {
		doc := &Document{}
		var embeddingVector pgvector.Vector
		var vectorScore float64

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
			&vectorScore,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}

		// Convert pgvector.Vector back to []float32
		doc.Embedding = embeddingVector.Slice()

		result := &SearchResult{
			Document:      doc,
			VectorScore:   vectorScore,
			TextScore:     0.0,
			CombinedScore: vectorScore * opts.VectorWeight,
		}
		results = append(results, result)
	}

	return results, nil
}

// searchTextOnly performs full-text search only
func (se *SearchEngineImpl) searchTextOnly(collectionID string, textQuery string, limit int, opts *SearchOptions) ([]*SearchResult, error) {
	if textQuery == "" {
		return nil, fmt.Errorf("text query is required for text search")
	}

	// Build the text search query
	searchQuery := fmt.Sprintf("to_tsquery('english', '%s')", strings.ReplaceAll(textQuery, " ", " & "))

	query := `
		SELECT id, collection_id, file_path, file_name, content, chunk_index, embedding, metadata, created_at, updated_at,
		       ts_rank(to_tsvector('english', content), %s) as text_score
		FROM documents
		WHERE collection_id = $1
		  AND to_tsvector('english', content) @@ %s
		ORDER BY text_score DESC
		LIMIT $2
	`

	query = fmt.Sprintf(query, searchQuery, searchQuery)

	rows, err := se.db.Query(query, collectionID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search documents: %w", err)
	}
	defer rows.Close()

	var results []*SearchResult
	for rows.Next() {
		doc := &Document{}
		var embeddingVector pgvector.Vector
		var textScore float64

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
			&textScore,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}

		// Convert pgvector.Vector back to []float32
		doc.Embedding = embeddingVector.Slice()

		result := &SearchResult{
			Document:      doc,
			VectorScore:   0.0,
			TextScore:     textScore,
			CombinedScore: textScore * opts.TextWeight,
		}
		results = append(results, result)
	}

	return results, nil
}

// searchHybrid performs combined vector and text search
func (se *SearchEngineImpl) searchHybrid(collectionID string, embedding []float32, textQuery string, limit int, opts *SearchOptions) ([]*SearchResult, error) {
	// Normalize weights
	totalWeight := opts.VectorWeight + opts.TextWeight
	if totalWeight == 0 {
		opts.VectorWeight = 0.7
		opts.TextWeight = 0.3
		totalWeight = 1.0
	}

	vectorWeight := opts.VectorWeight / totalWeight
	textWeight := opts.TextWeight / totalWeight

	// Build the query based on available inputs
	var query string
	var args []interface{}

	if embedding != nil && textQuery != "" {
		// Both vector and text search
		searchQuery := fmt.Sprintf("to_tsquery('english', '%s')", strings.ReplaceAll(textQuery, " ", " & "))
		query = `
			SELECT id, collection_id, file_path, file_name, content, chunk_index, embedding, metadata, created_at, updated_at,
			       1 - (embedding <=> $2) as vector_score,
			       ts_rank(to_tsvector('english', content), %s) as text_score,
			       ($5 * (1 - (embedding <=> $2))) + ($6 * ts_rank(to_tsvector('english', content), %s)) as combined_score
			FROM documents
			WHERE collection_id = $1
			  AND (embedding <=> $2) <= $3
			  AND to_tsvector('english', content) @@ %s
			ORDER BY combined_score DESC
			LIMIT $4
		`
		query = fmt.Sprintf(query, searchQuery, searchQuery, searchQuery)
		searchVector := pgvector.NewVector(embedding)
		maxDistance := opts.MaxDistance
		if maxDistance <= 0 {
			maxDistance = 1.0
		}
		args = []interface{}{collectionID, searchVector, maxDistance, limit, vectorWeight, textWeight}
	} else if embedding != nil {
		// Vector search only
		return se.searchVectorOnly(collectionID, embedding, limit, opts)
	} else if textQuery != "" {
		// Text search only
		return se.searchTextOnly(collectionID, textQuery, limit, opts)
	} else {
		return nil, fmt.Errorf("either embedding or text query must be provided")
	}

	rows, err := se.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search documents: %w", err)
	}
	defer rows.Close()

	var results []*SearchResult
	for rows.Next() {
		doc := &Document{}
		var embeddingVector pgvector.Vector
		var vectorScore, textScore, combinedScore float64

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
			&vectorScore,
			&textScore,
			&combinedScore,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}

		// Convert pgvector.Vector back to []float32
		doc.Embedding = embeddingVector.Slice()

		result := &SearchResult{
			Document:      doc,
			VectorScore:   vectorScore,
			TextScore:     textScore,
			CombinedScore: combinedScore,
		}
		results = append(results, result)
	}

	return results, nil
}

// searchSemantic performs semantic search with additional filters
func (se *SearchEngineImpl) searchSemantic(collectionID string, embedding []float32, textQuery string, limit int, opts *SearchOptions) ([]*SearchResult, error) {
	// Build additional filters
	var filters []string
	var args []interface{}
	argIndex := 1

	// Base collection filter
	filters = append(filters, fmt.Sprintf("collection_id = $%d", argIndex))
	args = append(args, collectionID)
	argIndex++

	// File name filter
	if opts.FileFilter != "" {
		filters = append(filters, fmt.Sprintf("file_name ILIKE $%d", argIndex))
		args = append(args, "%"+opts.FileFilter+"%")
		argIndex++
	}

	// Content filter
	if opts.ContentFilter != "" {
		filters = append(filters, fmt.Sprintf("content ILIKE $%d", argIndex))
		args = append(args, "%"+opts.ContentFilter+"%")
		argIndex++
	}

	// Build the WHERE clause
	whereClause := strings.Join(filters, " AND ")

	// Build the query
	query := fmt.Sprintf(`
		SELECT id, collection_id, file_path, file_name, content, chunk_index, embedding, metadata, created_at, updated_at,
		       1 - (embedding <=> $%d) as vector_score
		FROM documents
		WHERE %s
		  AND (embedding <=> $%d) <= $%d
		ORDER BY embedding <=> $%d ASC
		LIMIT $%d
	`, argIndex, whereClause, argIndex, argIndex+1, argIndex, argIndex+2)

	searchVector := pgvector.NewVector(embedding)
	maxDistance := opts.MaxDistance
	if maxDistance <= 0 {
		maxDistance = 1.0
	}
	args = append(args, searchVector, maxDistance, limit)

	rows, err := se.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search documents: %w", err)
	}
	defer rows.Close()

	var results []*SearchResult
	for rows.Next() {
		doc := &Document{}
		var embeddingVector pgvector.Vector
		var vectorScore float64

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
			&vectorScore,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}

		// Convert pgvector.Vector back to []float32
		doc.Embedding = embeddingVector.Slice()

		result := &SearchResult{
			Document:      doc,
			VectorScore:   vectorScore,
			TextScore:     0.0,
			CombinedScore: vectorScore * opts.VectorWeight,
		}
		results = append(results, result)
	}

	return results, nil
}

// applyReranking applies reranking to search results
func (se *SearchEngineImpl) applyReranking(ctx context.Context, textQuery string, results []*SearchResult, opts *SearchOptions) ([]*SearchResult, error) {
	if se.reranker == nil {
		return results, fmt.Errorf("reranker not initialized")
	}

	// Extract document contents for reranking
	documents := make([]string, len(results))
	for i, result := range results {
		documents[i] = result.Document.Content
	}

	// Use default instruction if not provided
	instruction := opts.RerankInstruction
	if instruction == "" {
		instruction = "Given a web search query, retrieve relevant passages that answer the query"
	}

	// Call reranking service
	rerankResults, err := se.reranker.Rerank(ctx, textQuery, documents, instruction)
	if err != nil {
		return nil, fmt.Errorf("reranking failed: %w", err)
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

			// Use default weights if not specified
			originalWeight := opts.OriginalWeight
			rerankWeight := opts.RerankWeight
			if originalWeight == 0 && rerankWeight == 0 {
				originalWeight = 0.7
				rerankWeight = 0.3
			}

			result.CombinedScore = originalWeight*originalScore + rerankWeight*rerankingScore
			result.Rank = rerankResult.Rank
		}
	}

	// Sort by new combined score
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[i].CombinedScore < results[j].CombinedScore {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// Update ranks after sorting
	for i, result := range results {
		result.Rank = i + 1
	}

	// Apply minimum score filter if specified
	if opts.MinScore > 0 {
		var filtered []*SearchResult
		for _, result := range results {
			if result.CombinedScore >= opts.MinScore {
				filtered = append(filtered, result)
			}
		}
		results = filtered
	}

	// Apply rerank limit if specified
	if opts.RerankLimit > 0 && len(results) > opts.RerankLimit {
		results = results[:opts.RerankLimit]
	}

	return results, nil
}

// RankSearchResults ranks search results by combined score and assigns ranks
func (se *SearchEngineImpl) RankSearchResults(results []*SearchResult) []*SearchResult {
	// Sort by combined score in descending order
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[i].CombinedScore < results[j].CombinedScore {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// Assign ranks
	for i, result := range results {
		result.Rank = i + 1
	}

	return results
}

// FilterSearchResults filters search results based on minimum score threshold
func (se *SearchEngineImpl) FilterSearchResults(results []*SearchResult, minScore float64) []*SearchResult {
	if minScore <= 0 {
		return results
	}

	var filtered []*SearchResult
	for _, result := range results {
		if result.CombinedScore >= minScore {
			filtered = append(filtered, result)
		}
	}

	return filtered
}

// GetSearchStats returns statistics about the search results
func (se *SearchEngineImpl) GetSearchStats(results []*SearchResult) map[string]interface{} {
	if len(results) == 0 {
		return map[string]interface{}{
			"total_results":      0,
			"avg_vector_score":   0.0,
			"avg_text_score":     0.0,
			"avg_combined_score": 0.0,
			"min_score":          0.0,
			"max_score":          0.0,
		}
	}

	var totalVectorScore, totalTextScore, totalCombinedScore float64
	minScore := results[0].CombinedScore
	maxScore := results[0].CombinedScore

	for _, result := range results {
		totalVectorScore += result.VectorScore
		totalTextScore += result.TextScore
		totalCombinedScore += result.CombinedScore

		if result.CombinedScore < minScore {
			minScore = result.CombinedScore
		}
		if result.CombinedScore > maxScore {
			maxScore = result.CombinedScore
		}
	}

	count := float64(len(results))
	return map[string]interface{}{
		"total_results":      len(results),
		"avg_vector_score":   totalVectorScore / count,
		"avg_text_score":     totalTextScore / count,
		"avg_combined_score": totalCombinedScore / count,
		"min_score":          minScore,
		"max_score":          maxScore,
	}
}
