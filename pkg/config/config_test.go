package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "rag-cli-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Store original home directory
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	// Set HOME to temp directory for this test
	os.Setenv("HOME", tempDir)

	// Test with empty config name
	config, err := LoadConfig("")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Check default values
	if config.ChatBackend != "ollama" {
		t.Errorf("Expected chat_backend to be 'ollama', got '%s'", config.ChatBackend)
	}

	if config.EmbeddingBackend != "ollama" {
		t.Errorf("Expected embedding_backend to be 'ollama', got '%s'", config.EmbeddingBackend)
	}

	if config.Ollama.Host != "localhost" {
		t.Errorf("Expected Ollama host to be 'localhost', got '%s'", config.Ollama.Host)
	}

	if config.Ollama.Port != 11434 {
		t.Errorf("Expected Ollama port to be 11434, got %d", config.Ollama.Port)
	}

	if config.Database.Host != "localhost" {
		t.Errorf("Expected database host to be 'localhost', got '%s'", config.Database.Host)
	}

	if config.Database.Port != 5432 {
		t.Errorf("Expected database port to be 5432, got %d", config.Database.Port)
	}

	if config.Embedding.ChunkSize != 1000 {
		t.Errorf("Expected chunk size to be 1000, got %d", config.Embedding.ChunkSize)
	}

	if config.Embedding.Dimensions != 1024 {
		t.Errorf("Expected embedding dimensions to be 1024, got %d", config.Embedding.Dimensions)
	}
}

func TestEmbeddingBackendFallback(t *testing.T) {
	// Test that embedding backend falls back to main backend when not specified
	config := &Config{
		ChatBackend: "openai",
		OpenAI: OpenAIConfig{
			APIKey:         "test-key",
			ChatModel:      "gpt-4",
			EmbeddingModel: "text-embedding-3-small",
		},
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			Name:     "testdb",
			User:     "testuser",
			Password: "",
			SSLMode:  "disable",
		},
		Embedding: EmbeddingConfig{
			ChunkSize:           1000,
			ChunkOverlap:        200,
			SimilarityThreshold: 0.7,
			MaxResults:          10,
			Dimensions:          1536, // text-embedding-3-small dimensions
		},
	}

	// Validate should set embedding backend to main backend
	if err := config.Validate(); err != nil {
		t.Fatalf("Config validation failed: %v", err)
	}

	if config.EmbeddingBackend != "openai" {
		t.Errorf("Expected embedding_backend to fall back to 'openai', got '%s'", config.EmbeddingBackend)
	}
}

func TestEmbeddingBackendValidation(t *testing.T) {
	// Test invalid embedding backend
	config := &Config{
		ChatBackend:      "ollama",
		EmbeddingBackend: "invalid",
		Ollama: OllamaConfig{
			Host:           "localhost",
			Port:           11434,
			ChatModel:      "qwen3:4b",
			EmbeddingModel: "dengcao/Qwen3-Embedding-0.6B:Q8_0",
		},
		Embedding: EmbeddingConfig{
			ChunkSize:           1000,
			ChunkOverlap:        200,
			SimilarityThreshold: 0.7,
			MaxResults:          10,
			Dimensions:          1024, // dengcao/Qwen3-Embedding-0.6B:Q8_0 dimensions
		},
	}

	if err := config.Validate(); err == nil {
		t.Error("Expected validation to fail with invalid embedding backend")
	}
}

func TestGetServerURL(t *testing.T) {
	config := &OllamaConfig{
		Host: "localhost",
		Port: 11434,
		TLS:  false,
	}

	url := config.GetServerURL()
	expected := "http://localhost:11434"
	if url != expected {
		t.Errorf("Expected URL '%s', got '%s'", expected, url)
	}

	// Test with TLS
	config.TLS = true
	url = config.GetServerURL()
	expected = "https://localhost:11434"
	if url != expected {
		t.Errorf("Expected URL '%s', got '%s'", expected, url)
	}
}

func TestGetDSN(t *testing.T) {
	config := &DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		Name:     "testdb",
		User:     "testuser",
		Password: "testpass",
		SSLMode:  "disable",
	}

	dsn := config.GetDSN()
	expected := "host=localhost port=5432 dbname=testdb user=testuser password=testpass sslmode=disable"
	if dsn != expected {
		t.Errorf("Expected DSN '%s', got '%s'", expected, dsn)
	}
}
