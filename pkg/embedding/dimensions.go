package embedding

import (
	"fmt"
	"strings"
)

// ModelDimensions maps embedding model names to their dimensions
var ModelDimensions = map[string]int{
	// Ollama models
	"nomic-embed-text":                  768,
	"nomic-embed-text-v2":               768,
	"all-minilm":                        384,
	"all-MiniLM-L6-v2":                  384,
	"all-MiniLM-L12-v2":                 384,
	"all-mpnet-base-v2":                 768,
	"all-MiniLM-L6-v2-fp16":             384,
	"dengcao/Qwen3-Embedding-0.6B:Q8_0": 1024,
	"Qwen3-Embedding-0.6B":              1024,
	"qwen3-embedding":                   1024,

	// OpenAI models
	"text-embedding-3-small": 1536,
	"text-embedding-3-large": 3072,
	"text-embedding-ada-002": 1536,

	// Cohere models
	"embed-english-v3.0":      1024,
	"embed-multilingual-v3.0": 1024,

	// HuggingFace models
	"sentence-transformers/all-MiniLM-L6-v2":                      384,
	"sentence-transformers/all-mpnet-base-v2":                     768,
	"sentence-transformers/paraphrase-multilingual-MiniLM-L12-v2": 384,
}

// GetModelDimensions returns the dimensions for a given model name
func GetModelDimensions(modelName string) (int, error) {
	// Try exact match first
	if dimensions, exists := ModelDimensions[modelName]; exists {
		return dimensions, nil
	}

	// Try case-insensitive match
	modelNameLower := strings.ToLower(modelName)
	for name, dimensions := range ModelDimensions {
		if strings.ToLower(name) == modelNameLower {
			return dimensions, nil
		}
	}

	// Try partial matches for common patterns
	if strings.Contains(strings.ToLower(modelName), "nomic") {
		return 768, nil
	}
	if strings.Contains(strings.ToLower(modelName), "minilm") {
		return 384, nil
	}
	if strings.Contains(strings.ToLower(modelName), "mpnet") {
		return 768, nil
	}
	if strings.Contains(strings.ToLower(modelName), "text-embedding-3-large") {
		return 3072, nil
	}
	if strings.Contains(strings.ToLower(modelName), "text-embedding-3-small") {
		return 1536, nil
	}
	if strings.Contains(strings.ToLower(modelName), "text-embedding-ada") {
		return 1536, nil
	}
	if strings.Contains(strings.ToLower(modelName), "qwen") {
		return 1024, nil
	}

	return 0, fmt.Errorf("unknown embedding model: %s. Please specify dimensions manually in config", modelName)
}

// ValidateDimensions validates that the provided dimensions match the model
func ValidateDimensions(modelName string, dimensions int) error {
	expectedDimensions, err := GetModelDimensions(modelName)
	if err != nil {
		// If we can't determine the expected dimensions, just return the error
		return err
	}

	if expectedDimensions != dimensions {
		return fmt.Errorf("model %s expects %d dimensions, but %d were provided", modelName, expectedDimensions, dimensions)
	}

	return nil
}
