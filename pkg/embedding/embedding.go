package embedding

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/busybytelab.com/rag-cli/pkg/client"
	"github.com/busybytelab.com/rag-cli/pkg/config"
)

// Service represents the embedding service
type Service struct {
	embedder client.Embedder
	config   *config.EmbeddingConfig
}

// Chunk represents a text chunk with its metadata
type Chunk struct {
	Content   string            `json:"content"`
	Index     int               `json:"index"`
	Metadata  map[string]string `json:"metadata"`
	Embedding []float32         `json:"embedding,omitempty"`
}

// New creates a new embedding service
func New(embedder client.Embedder, config *config.EmbeddingConfig) *Service {
	return &Service{
		embedder: embedder,
		config:   config,
	}
}

// ChunkText splits text into chunks based on configuration
func (s *Service) ChunkText(text string, metadata map[string]string) ([]*Chunk, error) {
	if metadata == nil {
		metadata = make(map[string]string)
	}

	// Clean and normalize text
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("empty text provided")
	}

	// Split text into sentences first
	sentences := s.splitIntoSentences(text)

	var chunks []*Chunk
	var currentChunk strings.Builder
	currentLength := 0
	chunkIndex := 0

	for _, sentence := range sentences {
		sentenceLength := len(sentence)

		// If adding this sentence would exceed chunk size, finalize current chunk
		if currentLength+sentenceLength > s.config.ChunkSize && currentLength > 0 {
			chunk := &Chunk{
				Content:  strings.TrimSpace(currentChunk.String()),
				Index:    chunkIndex,
				Metadata: copyMetadata(metadata),
			}
			chunks = append(chunks, chunk)

			// Start new chunk with overlap
			overlapText := s.getOverlapText(currentChunk.String(), s.config.ChunkOverlap)
			currentChunk.Reset()
			currentChunk.WriteString(overlapText)
			currentLength = len(overlapText)
			chunkIndex++
		}

		currentChunk.WriteString(sentence)
		currentLength += sentenceLength
	}

	// Add the last chunk if it has content
	if currentLength > 0 {
		chunk := &Chunk{
			Content:  strings.TrimSpace(currentChunk.String()),
			Index:    chunkIndex,
			Metadata: copyMetadata(metadata),
		}
		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// GenerateEmbeddings generates embeddings for all chunks
func (s *Service) GenerateEmbeddings(ctx context.Context, chunks []*Chunk) error {
	for i, chunk := range chunks {
		embedding, err := s.embedder.GenerateEmbedding(ctx, chunk.Content)
		if err != nil {
			return fmt.Errorf("failed to generate embedding for chunk %d: %w", i, err)
		}
		chunk.Embedding = embedding
	}
	return nil
}

// GenerateEmbeddingForText generates embedding for a single text
func (s *Service) GenerateEmbeddingForText(ctx context.Context, text string) ([]float32, error) {
	return s.embedder.GenerateEmbedding(ctx, text)
}

// splitIntoSentences splits text into sentences
func (s *Service) splitIntoSentences(text string) []string {
	// Simple sentence splitting - can be improved with NLP libraries
	var sentences []string
	var current strings.Builder

	for _, char := range text {
		current.WriteRune(char)

		// Check for sentence endings
		if char == '.' || char == '!' || char == '?' {
			// Look ahead to see if it's really the end of a sentence
			nextChar := ' '
			if len(text) > current.Len() {
				nextChar = rune(text[current.Len()])
			}

			// If next character is whitespace or end of text, it's likely end of sentence
			if unicode.IsSpace(nextChar) || current.Len() == len(text) {
				sentence := strings.TrimSpace(current.String())
				if sentence != "" {
					sentences = append(sentences, sentence)
				}
				current.Reset()
			}
		}
	}

	// Add any remaining text
	remaining := strings.TrimSpace(current.String())
	if remaining != "" {
		sentences = append(sentences, remaining)
	}

	return sentences
}

// getOverlapText gets the last N characters from text for overlap
func (s *Service) getOverlapText(text string, overlapSize int) string {
	if overlapSize <= 0 || len(text) <= overlapSize {
		return ""
	}

	// Find the last sentence boundary within the overlap
	overlapText := text[len(text)-overlapSize:]

	// Try to find a sentence boundary
	for i := 0; i < len(overlapText); i++ {
		if overlapText[i] == '.' || overlapText[i] == '!' || overlapText[i] == '?' {
			// Check if next character is whitespace
			if i+1 < len(overlapText) && unicode.IsSpace(rune(overlapText[i+1])) {
				return strings.TrimSpace(overlapText[i+1:])
			}
		}
	}

	// If no sentence boundary found, return the overlap text
	return strings.TrimSpace(overlapText)
}

// copyMetadata creates a copy of metadata map
func copyMetadata(metadata map[string]string) map[string]string {
	if metadata == nil {
		return make(map[string]string)
	}

	copied := make(map[string]string, len(metadata))
	for k, v := range metadata {
		copied[k] = v
	}
	return copied
}
