package client

import (
	"testing"

	"github.com/busybytelab.com/rag-cli/pkg/config"
)

func TestNewEmbedder(t *testing.T) {
	// Test with Ollama as embedding backend
	cfg := &config.Config{
		ChatBackend:      "openai",
		EmbeddingBackend: "ollama",
		Ollama: config.OllamaConfig{
			Host:           "localhost",
			Port:           11434,
			ChatModel:      "qwen3:4b",
			EmbeddingModel: "dengcao/Qwen3-Embedding-0.6B:Q8_0",
		},
		OpenAI: config.OpenAIConfig{
			APIKey:         "test-key",
			ChatModel:      "gpt-4",
			EmbeddingModel: "text-embedding-3-small",
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
		ChatBackend:      "ollama",
		EmbeddingBackend: "invalid",
		Ollama: config.OllamaConfig{
			Host:           "localhost",
			Port:           11434,
			ChatModel:      "qwen3:4b",
			EmbeddingModel: "dengcao/Qwen3-Embedding-0.6B:Q8_0",
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

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        []float32
		b        []float32
		expected float32
	}{
		{
			name:     "identical vectors",
			a:        []float32{1, 2, 3},
			b:        []float32{1, 2, 3},
			expected: 1.0,
		},
		{
			name:     "orthogonal vectors",
			a:        []float32{1, 0, 0},
			b:        []float32{0, 1, 0},
			expected: 0.0,
		},
		{
			name:     "opposite vectors",
			a:        []float32{1, 2, 3},
			b:        []float32{-1, -2, -3},
			expected: -1.0,
		},
		{
			name:     "similar vectors",
			a:        []float32{1, 2, 3},
			b:        []float32{2, 4, 6},
			expected: 1.0, // Normalized vectors should be identical
		},
		{
			name:     "empty vectors",
			a:        []float32{},
			b:        []float32{},
			expected: 0.0,
		},
		{
			name:     "different lengths",
			a:        []float32{1, 2},
			b:        []float32{1, 2, 3},
			expected: 0.0,
		},
		{
			name:     "zero vectors",
			a:        []float32{0, 0, 0},
			b:        []float32{1, 2, 3},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cosineSimilarity(tt.a, tt.b)

			// Use approximate comparison for floating point values
			if abs(result-tt.expected) > 0.0001 {
				t.Errorf("cosineSimilarity(%v, %v) = %f, want %f", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestCosineSimilarityBounds(t *testing.T) {
	// Test that cosine similarity always returns values in [-1, 1]
	a := []float32{1, 2, 3, 4, 5}
	b := []float32{5, 4, 3, 2, 1}

	result := cosineSimilarity(a, b)

	if result < -1 || result > 1 {
		t.Errorf("cosineSimilarity result %f is outside valid bounds [-1, 1]", result)
	}
}

func TestCosineSimilarityPrecision(t *testing.T) {
	// Test with high precision values
	a := []float32{0.123456789, 0.987654321, 0.555555555}
	b := []float32{0.123456789, 0.987654321, 0.555555555}

	result := cosineSimilarity(a, b)

	// Should be very close to 1.0 for identical vectors
	if abs(result-1.0) > 0.000001 {
		t.Errorf("cosineSimilarity for identical vectors = %f, want 1.0", result)
	}
}

// abs returns the absolute value of a float32
func abs(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}
