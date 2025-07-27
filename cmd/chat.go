package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/busybytelab.com/rag-cli/pkg/client"
	"github.com/busybytelab.com/rag-cli/pkg/config"
	"github.com/busybytelab.com/rag-cli/pkg/database"
	"github.com/busybytelab.com/rag-cli/pkg/embedding"
	"github.com/busybytelab.com/rag-cli/pkg/output"
	"github.com/spf13/cobra"
)

// getDefaultModelName returns the default model name based on the backend configuration
func getDefaultModelName(cfg *config.Config) string {
	switch cfg.Backend {
	case "ollama":
		return cfg.Ollama.Model
	case "openai":
		return cfg.OpenAI.Model
	default:
		return "unknown"
	}
}

// chatSession represents an active chat session
type chatSession struct {
	collectionID     string
	limit            int
	systemPrompt     string
	userPrompt       string
	searchQuery      string
	chatModel        string
	searchType       database.SearchType
	vectorWeight     float64
	textWeight       float64
	minScore         float64
	maxDistance      float64
	collectionMgr    database.CollectionManager
	searchEngine     database.SearchEngine
	ollamaClient     client.Client
	embeddingService *embedding.Service
	conversation     []client.Message
	reader           *bufio.Reader
}

var chatCmd = &cobra.Command{
	Use:   "chat [collection-id-or-name]",
	Short: "Start an interactive chat session with a collection",
	Long: `Start an interactive chat session with documents in a collection.

This command allows you to have a conversation with your documents using
RAG (Retrieval-Augmented Generation). The system will search for relevant
documents based on your questions and use them as context for generating responses.

The chat session supports multiple search types to find the most relevant documents:
- vector: Vector similarity search using embeddings
- text: Full-text search using PostgreSQL text search
- hybrid: Combined vector and text search (default)
- semantic: Semantic search with filters

Examples:
  # Start a chat session with a collection (uses hybrid search by default)
  rag-cli chat my-docs-collection

  # Start with a custom system prompt
  rag-cli chat my-docs-collection --system "You are a technical expert"

  # Start with a specific chat model
  rag-cli chat my-docs-collection --model llama2

  # Start with a user prompt (non-interactive)
  rag-cli chat my-docs-collection --prompt "What is machine learning?"

  # Use separate search query and user prompt
  rag-cli chat my-docs-collection --query "machine learning algorithms" --prompt "What are the key concepts?"

  # Search for API docs but ask about implementation
  rag-cli chat my-docs-collection --query "API documentation" --prompt "How do I implement authentication?"

  # Limit the number of context documents
  rag-cli chat my-docs-collection --limit 5

  # Use vector-only search
  rag-cli chat my-docs-collection --search-type vector

  # Use text-only search
  rag-cli chat my-docs-collection --search-type text

  # Use hybrid search with custom weights
  rag-cli chat my-docs-collection --search-type hybrid --vector-weight 0.8 --text-weight 0.2

  # Use semantic search with filters
  rag-cli chat my-docs-collection --search-type semantic`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		collectionID := args[0]

		// Initialize chat session
		session, err := initializeChatSession(cmd, collectionID)
		if err != nil {
			return err
		}

		// Start chat
		return session.startChat()
	},
}

// initializeChatSession sets up the chat session with all necessary components
func initializeChatSession(cmd *cobra.Command, collectionID string) (*chatSession, error) {
	limit, _ := cmd.Flags().GetInt("limit")
	systemPrompt, _ := cmd.Flags().GetString("system")
	userPrompt, _ := cmd.Flags().GetString("prompt")
	searchQuery, _ := cmd.Flags().GetString("query")
	chatModel, _ := cmd.Flags().GetString("model")
	searchTypeStr, _ := cmd.Flags().GetString("search-type")
	vectorWeight, _ := cmd.Flags().GetFloat64("vector-weight")
	textWeight, _ := cmd.Flags().GetFloat64("text-weight")
	minScore, _ := cmd.Flags().GetFloat64("min-score")
	maxDistance, _ := cmd.Flags().GetFloat64("max-distance")

	// Parse search type
	searchType := database.SearchType(searchTypeStr)
	if searchType == "" {
		searchType = database.SearchTypeHybrid // Default to hybrid
	}

	// Connect to database
	db, err := database.NewConnection(&cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Create managers
	collectionMgr := database.NewCollectionManager(db)
	searchEngine := database.NewSearchEngine(db)

	// Get collection by ID or name
	collection, err := collectionMgr.GetCollectionByIdOrName(collectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}

	// Create embedder for generating embeddings
	embedder, err := client.NewEmbedder(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	// Create client for chat operations
	chatClient, err := client.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat client: %w", err)
	}

	// Create embedding service
	embeddingService := embedding.New(embedder, &cfg.Embedding)

	session := &chatSession{
		collectionID:     collection.ID,
		limit:            limit,
		systemPrompt:     systemPrompt,
		userPrompt:       userPrompt,
		searchQuery:      searchQuery,
		chatModel:        chatModel,
		searchType:       searchType,
		vectorWeight:     vectorWeight,
		textWeight:       textWeight,
		minScore:         minScore,
		maxDistance:      maxDistance,
		collectionMgr:    collectionMgr,
		searchEngine:     searchEngine,
		ollamaClient:     chatClient,
		embeddingService: embeddingService,
		conversation:     make([]client.Message, 0),
		reader:           bufio.NewReader(os.Stdin),
	}

	output.Success("Starting chat session with collection: %s", collection.Name)
	output.KeyValue("Collection", collection.Name)
	output.KeyValue("Chat Backend", cfg.Backend)
	output.KeyValue("Embedding Backend", cfg.EmbeddingBackend)
	if chatModel != "" {
		output.KeyValue("Chat Model", chatModel)
	} else {
		output.KeyValue("Chat Model", getDefaultModelName(cfg))
	}
	if searchQuery != "" {
		output.KeyValue("Search Query", searchQuery)
	}
	output.KeyValue("Search Type", string(searchType))
	if searchType == database.SearchTypeHybrid {
		output.KeyValuef("Vector Weight", "%.1f", vectorWeight)
		output.KeyValuef("Text Weight", "%.1f", textWeight)
	}

	// Show different messages based on whether this is interactive or non-interactive
	if userPrompt != "" {
		output.KeyValue("User Prompt", userPrompt)
	} else {
		output.Info("Type 'quit' or 'exit' to end the session")
	}
	output.Info("")

	return session, nil
}

// startChat begins the interactive chat loop
func (s *chatSession) startChat() error {
	// Check if this is a non-interactive session (has initial user prompt)
	hasInitialPrompt := s.userPrompt != ""

	for {
		if err := s.processUserInput(); err != nil {
			// If this was a non-interactive session and we've processed the prompt, exit gracefully
			if hasInitialPrompt && s.userPrompt == "" {
				return nil
			}
			return err
		}

		// If this was a non-interactive session and we've processed the prompt, exit
		if hasInitialPrompt && s.userPrompt == "" {
			return nil
		}
	}
}

// processUserInput handles a single user input and generates a response
func (s *chatSession) processUserInput() error {
	var input string

	if s.userPrompt != "" {
		// Use the provided user prompt as input directly
		input = s.userPrompt
		// Clear the user prompt after using it to prevent infinite loop
		s.userPrompt = ""
	} else {
		// Wait for user input
		output.Print("You: ")
		userInput, err := s.reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(userInput)
		if input == "" {
			return nil
		}

		if input == "quit" || input == "exit" {
			output.Info("Goodbye!")
			return fmt.Errorf("chat session ended")
		}
	}

	if err := s.generateAndDisplayResponse(input); err != nil {
		output.Error("Failed to generate response: %v", err)
		return nil // Continue chat loop even if there's an error
	}

	return nil
}

// generateAndDisplayResponse generates a response for the user input and displays it
func (s *chatSession) generateAndDisplayResponse(userInput string) error {
	// Determine what to use for search embedding
	searchText := userInput
	if s.searchQuery != "" {
		searchText = s.searchQuery
	}

	// Generate embedding for search query
	ctx := context.Background()
	queryEmbedding, err := s.embeddingService.GenerateEmbeddingForText(ctx, searchText)
	if err != nil {
		return fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Use configured search options
	searchOpts := &database.SearchOptions{
		SearchType:   s.searchType,
		VectorWeight: s.vectorWeight,
		TextWeight:   s.textWeight,
		MinScore:     s.minScore,
		MaxDistance:  s.maxDistance,
	}

	// Search for relevant documents using the search text
	results, err := s.searchEngine.SearchDocumentsWithOptions(s.collectionID, queryEmbedding, searchText, s.limit, searchOpts)
	if err != nil {
		return fmt.Errorf("failed to search documents: %w", err)
	}

	// Convert SearchResult to Document for backward compatibility
	documents := make([]*database.Document, len(results))
	for i, result := range results {
		documents[i] = result.Document
	}

	// Build context from documents
	contextStr := buildContextFromDocuments(documents)

	// Create system message with context
	systemMessage := s.buildSystemMessage(contextStr)

	// Prepare messages for chat
	messages := s.prepareMessages(systemMessage, userInput)

	// Get response from LLM
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second) // 3 minute timeout for chat
	defer cancel()

	response, err := s.ollamaClient.Chat(ctx, s.chatModel, messages, false)
	if err != nil {
		output.Info("This might be due to a timeout. Try reducing the context limit with -l flag.")
		return fmt.Errorf("failed to get response: %w", err)
	}

	// Add to conversation history
	s.conversation = append(s.conversation, client.Message{Role: "user", Content: userInput})
	s.conversation = append(s.conversation, client.Message{Role: "assistant", Content: response.Message.Content})

	// Display response
	output.Info("Assistant: %s", response.Message.Content)
	output.Info("")

	return nil
}

// buildSystemMessage creates the system message with context and custom prompt
func (s *chatSession) buildSystemMessage(contextStr string) string {
	baseSystemPrompt := `You are a helpful assistant that answers questions based on the provided context. 
Use the following context to answer the user's question. If the context doesn't contain relevant information, 
say so but try to be helpful.

Context:
%s

Answer the user's question based on the context above.`

	if s.systemPrompt != "" {
		// Append custom system prompt to the base prompt
		baseSystemPrompt = fmt.Sprintf(`%s

%s`, baseSystemPrompt, s.systemPrompt)
	}

	return fmt.Sprintf(baseSystemPrompt, contextStr)
}

// prepareMessages creates the message array for the LLM
func (s *chatSession) prepareMessages(systemMessage, userInput string) []client.Message {
	messages := []client.Message{
		{Role: "system", Content: systemMessage},
	}

	// Add conversation history
	messages = append(messages, s.conversation...)

	// Add user message
	messages = append(messages, client.Message{Role: "user", Content: userInput})

	return messages
}

// buildContextFromDocuments builds context string from search results
func buildContextFromDocuments(documents []*database.Document) string {
	if len(documents) == 0 {
		return "No relevant documents found."
	}

	var contextParts []string
	for i, doc := range documents {
		contextParts = append(contextParts, fmt.Sprintf("Document %d (from %s):\n%s", i+1, doc.FileName, doc.Content))
	}

	return strings.Join(contextParts, "\n\n")
}

func init() {
	chatCmd.Flags().IntP("limit", "l", 5, "Maximum number of documents to use as context")
	chatCmd.Flags().String("system", "", "Custom system prompt to append to the default assistant behavior")
	chatCmd.Flags().String("prompt", "", "Custom user prompt to use as input directly (instead of waiting for user input)")
	chatCmd.Flags().String("query", "", "Search query to use for document retrieval (separate from user prompt)")
	chatCmd.Flags().StringP("model", "m", "", "Override the default chat model (e.g., 'llama2', 'mistral', 'codellama')")
	chatCmd.Flags().StringP("search-type", "t", "hybrid", "Search type: vector, text, hybrid, semantic")
	chatCmd.Flags().Float64P("vector-weight", "", 0.7, "Weight for vector similarity (0.0-1.0)")
	chatCmd.Flags().Float64P("text-weight", "", 0.3, "Weight for text similarity (0.0-1.0)")
	chatCmd.Flags().Float64P("min-score", "", 0.1, "Minimum similarity score")
	chatCmd.Flags().Float64P("max-distance", "", 0.8, "Maximum vector distance")
	rootCmd.AddCommand(chatCmd)
}
