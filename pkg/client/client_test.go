package client

import (
	"testing"

	"github.com/busybytelab.com/rag-cli/pkg/config"
)

func TestNewEmbedder(t *testing.T) {
	// Test with Ollama as embedding backend
	cfg := &config.Config{
		Backend:          "openai",
		EmbeddingBackend: "ollama",
		Ollama: config.OllamaConfig{
			Host:       "localhost",
			Port:       11434,
			Model:      "llama3.2:3b",
			EmbedModel: "nomic-embed-text",
		},
		OpenAI: config.OpenAIConfig{
			APIKey:     "test-key",
			Model:      "gpt-4",
			EmbedModel: "text-embedding-3-small",
		},
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			Name:     "testdb",
			User:     "testuser",
			Password: "",
			SSLMode:  "disable",
		},
	}

	embedder, err := NewEmbedder(cfg)
	if err != nil {
		t.Fatalf("Failed to create embedder: %v", err)
	}

	// Verify it's an Ollama client (since embedding backend is ollama)
	if _, ok := embedder.(*OllamaClient); !ok {
		t.Error("Expected embedder to be OllamaClient when embedding_backend is ollama")
	}

	// Test with OpenAI as embedding backend
	cfg.EmbeddingBackend = "openai"
	embedder, err = NewEmbedder(cfg)
	if err != nil {
		t.Fatalf("Failed to create embedder: %v", err)
	}

	// Verify it's an OpenAI client
	if _, ok := embedder.(*OpenAIClient); !ok {
		t.Error("Expected embedder to be OpenAIClient when embedding_backend is openai")
	}

	// Test fallback to main backend when embedding backend is not specified
	cfg.EmbeddingBackend = ""
	embedder, err = NewEmbedder(cfg)
	if err != nil {
		t.Fatalf("Failed to create embedder: %v", err)
	}

	// Should fall back to OpenAI (main backend)
	if _, ok := embedder.(*OpenAIClient); !ok {
		t.Error("Expected embedder to fall back to OpenAIClient when embedding_backend is empty")
	}
}

func TestNewEmbedderInvalidBackend(t *testing.T) {
	cfg := &config.Config{
		Backend:          "ollama",
		EmbeddingBackend: "invalid",
		Ollama: config.OllamaConfig{
			Host:       "localhost",
			Port:       11434,
			Model:      "llama3.2:3b",
			EmbedModel: "nomic-embed-text",
		},
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			Name:     "testdb",
			User:     "testuser",
			Password: "",
			SSLMode:  "disable",
		},
	}

	_, err := NewEmbedder(cfg)
	if err == nil {
		t.Error("Expected error for invalid embedding backend")
	}
}
