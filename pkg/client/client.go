package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

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
	req := &api.EmbeddingRequest{
		Model:  c.config.EmbeddingModel,
		Prompt: text,
	}

	resp, err := c.client.Embeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Convert []float64 to []float32
	embedding := make([]float32, len(resp.Embedding))
	for i, v := range resp.Embedding {
		embedding[i] = float32(v)
	}

	return embedding, nil
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
