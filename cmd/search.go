package cmd

import (
	"context"
	"fmt"

	"github.com/busybytelab.com/rag-cli/pkg/client"
	"github.com/busybytelab.com/rag-cli/pkg/database"
	"github.com/busybytelab.com/rag-cli/pkg/embedding"
	"github.com/busybytelab.com/rag-cli/pkg/output"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search [collection-id-or-name] [query]",
	Short: "Search documents in a collection",
	Long: `Search documents in a collection using various search methods.

This command supports multiple search types:
- vector: Vector similarity search using embeddings
- text: Full-text search using PostgreSQL text search
- hybrid: Combined vector and text search
- semantic: Semantic search with filters

Reranking can be enabled with the --rerank flag for improved result accuracy.

Examples:
  # Vector search (default)
  rag-cli search my-docs-collection "machine learning algorithms"

  # Text search only
  rag-cli search my-docs-collection "machine learning" --type text

  # Hybrid search with custom weights
  rag-cli search my-docs-collection "neural networks" --type hybrid --vector-weight 0.7 --text-weight 0.3

  # Search with reranking enabled
  rag-cli search my-docs-collection "API documentation" --rerank --rerank-instruction "Focus on code examples"

  # Search with filters
  rag-cli search my-docs-collection "API documentation" --file-filter "*.md" --content-filter "authentication"

  # Show detailed scores
  rag-cli search my-docs-collection "database queries" --show-scores

  # Show document content
  rag-cli search my-docs-collection "error handling" --show-content`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		collectionID := args[0]
		query := args[1]

		// Get search options
		searchType, _ := cmd.Flags().GetString("type")
		limit, _ := cmd.Flags().GetInt("limit")
		vectorWeight, _ := cmd.Flags().GetFloat64("vector-weight")
		textWeight, _ := cmd.Flags().GetFloat64("text-weight")
		minScore, _ := cmd.Flags().GetFloat64("min-score")
		showScores, _ := cmd.Flags().GetBool("show-scores")
		showContent, _ := cmd.Flags().GetBool("show-content")
		maxDistance, _ := cmd.Flags().GetFloat64("max-distance")
		fileFilter, _ := cmd.Flags().GetString("file-filter")
		contentFilter, _ := cmd.Flags().GetString("content-filter")

		// Get reranking options
		enableReranking, _ := cmd.Flags().GetBool("rerank")
		rerankInstruction, _ := cmd.Flags().GetString("rerank-instruction")
		originalWeight, _ := cmd.Flags().GetFloat64("original-weight")
		rerankWeight, _ := cmd.Flags().GetFloat64("rerank-weight")
		rerankLimit, _ := cmd.Flags().GetInt("rerank-limit")

		// Connect to database
		db, err := database.NewConnection(&cfg.Database)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		// Create managers
		collectionMgr := database.NewCollectionManager(db)

		// Create search engine with or without reranking
		var searchEngine database.SearchEngine
		if enableReranking {
			// Create reranker
			reranker, err := client.NewReranker(cfg)
			if err != nil {
				return fmt.Errorf("failed to create reranker: %w", err)
			}
			searchEngine = database.NewSearchEngineWithReranker(db, reranker)
		} else {
			searchEngine = database.NewSearchEngine(db)
		}

		// Get collection by ID or name
		collection, err := collectionMgr.GetCollectionByIdOrName(collectionID)
		if err != nil {
			return fmt.Errorf("failed to get collection: %w", err)
		}

		output.KeyValue("Searching in collection", collection.Name)
		output.KeyValue("Query", query)
		output.KeyValue("Search type", searchType)

		// Create search options
		searchOpts := &database.SearchOptions{
			SearchType:    database.SearchType(searchType),
			VectorWeight:  vectorWeight,
			TextWeight:    textWeight,
			MinScore:      minScore,
			MaxDistance:   maxDistance,
			FileFilter:    fileFilter,
			ContentFilter: contentFilter,
		}

		// Add reranking options if enabled
		if enableReranking {
			searchOpts.EnableReranking = true
			searchOpts.RerankInstruction = rerankInstruction
			searchOpts.OriginalWeight = originalWeight
			searchOpts.RerankWeight = rerankWeight
			searchOpts.RerankLimit = rerankLimit
		}

		// Determine if we need embeddings based on search type
		var queryEmbedding []float32
		var textQuery string

		switch database.SearchType(searchType) {
		case database.SearchTypeText:
			textQuery = query
		case database.SearchTypeVector, database.SearchTypeHybrid, database.SearchTypeSemantic:
			// Create embedder for generating embeddings
			embedder, err := client.NewEmbedder(cfg)
			if err != nil {
				return fmt.Errorf("failed to create embedder: %w", err)
			}

			// Create embedding service
			embeddingService := embedding.New(embedder, &cfg.Embedding)

			// Generate embedding for query
			ctx := context.Background()
			queryEmbedding, err = embeddingService.GenerateEmbeddingForText(ctx, query)
			if err != nil {
				return fmt.Errorf("failed to generate query embedding: %w", err)
			}

			// For hybrid search, also use the original query as text
			if database.SearchType(searchType) == database.SearchTypeHybrid {
				textQuery = query
			}
		}

		// Search documents using the enhanced search
		results, err := searchEngine.SearchDocumentsWithOptions(collection.ID, queryEmbedding, textQuery, limit, searchOpts)
		if err != nil {
			return fmt.Errorf("failed to search documents: %w", err)
		}

		// Rank and filter results
		results = searchEngine.RankSearchResults(results)
		results = searchEngine.FilterSearchResults(results, minScore)

		if len(results) == 0 {
			output.Info("No documents found.")
			return nil
		}

		// Get search statistics
		stats := searchEngine.GetSearchStats(results)
		output.Success("Found %d documents:", len(results))
		if showScores {
			output.KeyValuef("Average Combined Score", "%.4f", stats["avg_combined_score"])
			output.KeyValuef("Score Range", "%.4f - %.4f", stats["min_score"], stats["max_score"])
		}
		output.Info("")

		for i, result := range results {
			output.Bold("Result %d:", i+1)
			output.KeyValue("File", result.Document.FileName)
			output.KeyValue("Path", result.Document.FilePath)
			output.KeyValuef("Chunk", "%d", result.Document.ChunkIndex)

			if showScores {
				output.KeyValuef("Vector Score", "%.4f", result.VectorScore)
				output.KeyValuef("Text Score", "%.4f", result.TextScore)
				output.KeyValuef("Combined Score", "%.4f", result.CombinedScore)
				output.KeyValuef("Rank", "%d", result.Rank)
			}

			if showContent {
				output.KeyValue("Content", result.Document.Content)
			}

			output.Info("")
		}

		return nil
	},
}

func init() {
	searchCmd.Flags().IntP("limit", "l", 10, "Maximum number of results to return")
	searchCmd.Flags().BoolP("show-content", "s", false, "Show full content of results")
	searchCmd.Flags().BoolP("show-scores", "", false, "Show search scores for results")
	searchCmd.Flags().StringP("type", "t", "hybrid", "Search type: vector, text, hybrid, semantic")
	searchCmd.Flags().Float64P("vector-weight", "", 0.7, "Weight for vector similarity (0.0-1.0)")
	searchCmd.Flags().Float64P("text-weight", "", 0.3, "Weight for text similarity (0.0-1.0)")
	searchCmd.Flags().Float64P("min-score", "", 0.0, "Minimum similarity score")
	searchCmd.Flags().Float64P("max-distance", "", 1.0, "Maximum vector distance")
	searchCmd.Flags().StringP("file-filter", "", "", "Filter by file name pattern")
	searchCmd.Flags().StringP("content-filter", "", "", "Filter by content text")

	// Reranking flags
	searchCmd.Flags().BoolP("rerank", "r", false, "Enable reranking for improved results")
	searchCmd.Flags().String("rerank-instruction", "Given a web search query, retrieve relevant passages that answer the query", "Custom instruction for reranking")
	searchCmd.Flags().Float64("original-weight", 0.7, "Weight for original search score (0.0-1.0)")
	searchCmd.Flags().Float64("rerank-weight", 0.3, "Weight for reranking score (0.0-1.0)")
	searchCmd.Flags().Int("rerank-limit", 0, "Number of results to rerank (0 = all)")

	rootCmd.AddCommand(searchCmd)
}
