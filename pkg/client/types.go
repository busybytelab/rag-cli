package client

import (
	"context"
	"net/url"
	"time"

	"github.com/busybytelab.com/rag-cli/pkg/config"
	"github.com/ollama/ollama/api"
)

type (
	// Embedder represents an interface for embedding text
	Embedder interface {
		GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
	}

	// Reranker represents an interface for reranking search results
	Reranker interface {
		Rerank(ctx context.Context, query string, documents []string, instruction string) ([]RerankResult, error)
	}

	// RerankResult represents a reranked document with relevance score
	RerankResult struct {
		Document string  `json:"document"`
		Score    float64 `json:"score"`
		Rank     int     `json:"rank"`
	}

	// Client represents a generic LLM API client interface
	Client interface {
		Embedder
		// TODO: Chat method should be used instead of Generate
		Chat(ctx context.Context, model string, messages []Message, stream bool) (*ChatResponse, error)
		// TODO: remove
		Generate(ctx context.Context, model string, prompt string, options map[string]interface{}) (*GenerateResponse, error)
	}

	// OllamaClient represents an Ollama API client implementation
	OllamaClient struct {
		serverURL *url.URL
		config    *config.OllamaConfig
		client    *api.Client
	}

	// Message represents a chat message
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	// ChatResponse represents a chat completion response
	ChatResponse struct {
		Model     string    `json:"model"`
		CreatedAt time.Time `json:"created_at"`
		Message   Message   `json:"message"`
		Done      bool      `json:"done"`
	}

	// GenerateResponse represents a text generation response
	GenerateResponse struct {
		Model     string    `json:"model"`
		CreatedAt time.Time `json:"created_at"`
		Response  string    `json:"response"`
		Done      bool      `json:"done"`
	}

	// EmbeddingResponse represents an embedding response
	EmbeddingResponse struct {
		Embeddings [][]float64 `json:"embeddings"`
	}
)
