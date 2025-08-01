package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"math"

	"github.com/busybytelab.com/rag-cli/pkg/config"
	"github.com/ollama/ollama/api"
)

// New creates a new client based on the chat backend configuration
func New(cfg *config.Config) (Client, error) {
	switch cfg.ChatBackend {
	case "ollama":
		return NewOllama(&cfg.Ollama)
	case "openai":
		return NewOpenAI(&cfg.OpenAI)
	default:
		return nil, fmt.Errorf("unsupported chat_backend: %s", cfg.ChatBackend)
	}
}

// NewEmbedder creates a new embedder based on the embedding backend configuration
func NewEmbedder(cfg *config.Config) (Embedder, error) {
	// Use embedding backend if specified, otherwise fall back to chat backend
	embeddingBackend := cfg.EmbeddingBackend
	if embeddingBackend == "" {
		embeddingBackend = cfg.ChatBackend
	}

	switch embeddingBackend {
	case "ollama":
		return NewOllama(&cfg.Ollama)
	case "openai":
		return NewOpenAI(&cfg.OpenAI)
	default:
		return nil, fmt.Errorf("unsupported embedding backend: %s", embeddingBackend)
	}
}

// NewReranker creates a new reranker based on the embedding backend configuration
func NewReranker(cfg *config.Config) (Reranker, error) {
	// Use embedding backend if specified, otherwise fall back to chat backend
	embeddingBackend := cfg.EmbeddingBackend
	if embeddingBackend == "" {
		embeddingBackend = cfg.ChatBackend
	}

	switch embeddingBackend {
	case "ollama":
		client, err := NewOllama(&cfg.Ollama)
		if err != nil {
			return nil, err
		}
		// Type assertion since OllamaClient implements both Client and Reranker
		if reranker, ok := client.(Reranker); ok {
			return reranker, nil
		}
		return nil, fmt.Errorf("OllamaClient does not implement Reranker interface")
	case "openai":
		client, err := NewOpenAI(&cfg.OpenAI)
		if err != nil {
			return nil, err
		}
		// Type assertion since OpenAIClient implements both Client and Reranker
		if reranker, ok := client.(Reranker); ok {
			return reranker, nil
		}
		return nil, fmt.Errorf("OpenAIClient does not implement Reranker interface")
	default:
		return nil, fmt.Errorf("unsupported embedding backend: %s", embeddingBackend)
	}
}

// NewOllama creates a new Ollama client
func NewOllama(cfg *config.OllamaConfig) (Client, error) {
	serverURL, err := url.Parse(cfg.GetServerURL())
	if err != nil {
		return nil, fmt.Errorf("invalid server URL: %w", err)
	}

	// Create HTTP client with longer timeout for chat operations
	httpClient := &http.Client{
		Timeout: 120 * time.Second, // Increased from 30s to 120s for chat operations
	}

	client := api.NewClient(serverURL, httpClient)

	return &OllamaClient{
		serverURL: serverURL,
		config:    cfg,
		client:    client,
	}, nil
}

// GenerateEmbedding generates embeddings for the given text
func (c *OllamaClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	request := &api.EmbeddingRequest{
		Model:  c.config.EmbeddingModel,
		Prompt: text,
	}

	response, err := c.client.Embeddings(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Convert []float64 to []float32
	embedding := make([]float32, len(response.Embedding))
	for i, v := range response.Embedding {
		embedding[i] = float32(v)
	}

	return embedding, nil
}

// Rerank reranks documents using the reranker model
func (c *OllamaClient) Rerank(ctx context.Context, query string, documents []string, instruction string) ([]RerankResult, error) {
	if len(documents) == 0 {
		return []RerankResult{}, nil
	}

	// For Ollama, we'll use a simple approach by generating embeddings for query and documents
	// and computing cosine similarity as a reranking score
	queryEmbedding, err := c.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	var results []RerankResult
	for i, doc := range documents {
		docEmbedding, err := c.GenerateEmbedding(ctx, doc)
		if err != nil {
			return nil, fmt.Errorf("failed to generate document embedding: %w", err)
		}

		// Compute cosine similarity
		score := cosineSimilarity(queryEmbedding, docEmbedding)

		results = append(results, RerankResult{
			Document: doc,
			Score:    float64(score),
			Rank:     i + 1,
		})
	}

	// Sort by score in descending order
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[i].Score < results[j].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// Update ranks after sorting
	for i := range results {
		results[i].Rank = i + 1
	}

	return results, nil
}

// cosineSimilarity computes the cosine similarity between two vectors
// Returns a value between -1 and 1, where 1 indicates identical vectors
// Based on the formula: cos(θ) = (A·B) / (||A|| * ||B||)
func cosineSimilarity(a, b []float32) float32 {
	// Check for valid input
	if len(a) == 0 || len(b) == 0 {
		return 0
	}

	if len(a) != len(b) {
		return 0
	}

	var dotProduct float64
	var normASquared float64
	var normBSquared float64

	// Use float64 for intermediate calculations to avoid precision loss
	for i := 0; i < len(a); i++ {
		ai := float64(a[i])
		bi := float64(b[i])
		dotProduct += ai * bi
		normASquared += ai * ai
		normBSquared += bi * bi
	}

	// Check for zero vectors to avoid division by zero
	if normASquared == 0 || normBSquared == 0 {
		return 0
	}

	// Compute the cosine similarity
	normA := math.Sqrt(normASquared)
	normB := math.Sqrt(normBSquared)
	similarity := dotProduct / (normA * normB)

	// Ensure the result is within valid bounds [-1, 1]
	// This handles floating point precision issues
	if similarity > 1 {
		similarity = 1
	} else if similarity < -1 {
		similarity = -1
	}

	return float32(similarity)
}

// Chat performs a chat completion with the specified model
func (c *OllamaClient) Chat(ctx context.Context, model string, messages []Message, stream bool) (*ChatResponse, error) {
	if model == "" {
		model = c.config.ChatModel
	}

	// Convert our Message type to Ollama api.Message type
	ollamaMessages := make([]api.Message, len(messages))
	for i, msg := range messages {
		ollamaMessages[i] = api.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	req := &api.ChatRequest{
		Model:    model,
		Messages: ollamaMessages,
		Stream:   &stream,
	}

	var resp *api.ChatResponse
	err := c.client.Chat(ctx, req, func(response api.ChatResponse) error {
		resp = &response
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to chat: %w", err)
	}

	// Convert Ollama response to our generic response
	return &ChatResponse{
		Model:     resp.Model,
		CreatedAt: resp.CreatedAt,
		Message: Message{
			Role:    resp.Message.Role,
			Content: resp.Message.Content,
		},
		Done: resp.Done,
	}, nil
}

// Generate performs text generation with the specified model
func (c *OllamaClient) Generate(ctx context.Context, model string, prompt string, options map[string]interface{}) (*GenerateResponse, error) {
	if model == "" {
		model = c.config.ChatModel
	}

	req := &api.GenerateRequest{
		Model:  model,
		Prompt: prompt,
	}

	// Apply options if provided
	if options != nil {
		req.Options = make(map[string]interface{})
		if temp, ok := options["temperature"].(float64); ok {
			req.Options["temperature"] = temp
		}
		if topP, ok := options["top_p"].(float64); ok {
			req.Options["top_p"] = topP
		}
		if maxTokens, ok := options["max_tokens"].(int); ok {
			req.Options["num_predict"] = maxTokens
		}
	}

	var resp *api.GenerateResponse
	err := c.client.Generate(ctx, req, func(response api.GenerateResponse) error {
		resp = &response
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate: %w", err)
	}

	// Convert Ollama response to our generic response
	return &GenerateResponse{
		Model:     resp.Model,
		CreatedAt: resp.CreatedAt,
		Response:  resp.Response,
		Done:      resp.Done,
	}, nil
}
