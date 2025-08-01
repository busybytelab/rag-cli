package client

import (
	"context"
	"fmt"
	"time"

	"github.com/busybytelab.com/rag-cli/pkg/config"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// OpenAIClient represents an OpenAI API client implementation
type OpenAIClient struct {
	config *config.OpenAIConfig
	client *openai.Client
}

// NewOpenAI creates a new OpenAI client
func NewOpenAI(cfg *config.OpenAIConfig) (Client, error) {
	opts := []option.RequestOption{
		option.WithAPIKey(cfg.APIKey),
	}

	// If base URL is provided, use it (for local servers like llama-server)
	if cfg.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.BaseURL))
	}

	client := openai.NewClient(opts...)

	return &OpenAIClient{
		config: cfg,
		client: &client,
	}, nil
}

// GenerateEmbedding generates embeddings for the given text
func (c *OpenAIClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	params := openai.EmbeddingNewParams{
		Model: openai.EmbeddingModelTextEmbedding3Small,
		Input: openai.EmbeddingNewParamsInputUnion{
			OfArrayOfStrings: []string{text},
		},
	}

	response, err := c.client.Embeddings.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding: %w", err)
	}

	if len(response.Data) == 0 {
		return nil, fmt.Errorf("no embedding data returned")
	}

	// Convert []float64 to []float32
	embedding := make([]float32, len(response.Data[0].Embedding))
	for i, v := range response.Data[0].Embedding {
		embedding[i] = float32(v)
	}

	return embedding, nil
}

// Chat performs a chat completion with the specified model
func (c *OpenAIClient) Chat(ctx context.Context, model string, messages []Message, stream bool) (*ChatResponse, error) {
	if model == "" {
		model = c.config.ChatModel
	}

	// Convert our Message type to OpenAI format
	openaiMessages := make([]openai.ChatCompletionMessageParamUnion, len(messages))
	for i, msg := range messages {
		switch msg.Role {
		case "system":
			openaiMessages[i] = openai.ChatCompletionMessageParamUnion{
				OfSystem: &openai.ChatCompletionSystemMessageParam{
					Content: openai.ChatCompletionSystemMessageParamContentUnion{
						OfString: openai.String(msg.Content),
					},
				},
			}
		case "user":
			openaiMessages[i] = openai.ChatCompletionMessageParamUnion{
				OfUser: &openai.ChatCompletionUserMessageParam{
					Content: openai.ChatCompletionUserMessageParamContentUnion{
						OfString: openai.String(msg.Content),
					},
				},
			}
		case "assistant":
			openaiMessages[i] = openai.ChatCompletionMessageParamUnion{
				OfAssistant: &openai.ChatCompletionAssistantMessageParam{
					Content: openai.ChatCompletionAssistantMessageParamContentUnion{
						OfString: openai.String(msg.Content),
					},
				},
			}
		default:
			return nil, fmt.Errorf("unsupported message role: %s", msg.Role)
		}
	}

	params := openai.ChatCompletionNewParams{
		Model:    model,
		Messages: openaiMessages,
	}

	var response *openai.ChatCompletion
	var err error

	if stream {
		// Use streaming API
		stream := c.client.Chat.Completions.NewStreaming(ctx, params)
		// For now, we'll collect the first chunk only
		if stream.Next() {
			chunk := stream.Current()
			if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
				content := chunk.Choices[0].Delta.Content
				return &ChatResponse{
					Model:     model,
					CreatedAt: time.Unix(chunk.Created, 0),
					Message: Message{
						Role:    "assistant",
						Content: content,
					},
					Done: true,
				}, nil
			}
		}
		return &ChatResponse{
			Model:     model,
			CreatedAt: time.Now(),
			Message: Message{
				Role:    "assistant",
				Content: "",
			},
			Done: true,
		}, nil
	} else {
		// Use non-streaming API
		response, err = c.client.Chat.Completions.New(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to create chat completion: %w", err)
		}
	}

	// Convert OpenAI response to our generic response
	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	choice := response.Choices[0]
	content := string(choice.Message.Content)

	return &ChatResponse{
		Model:     model,
		CreatedAt: time.Unix(response.Created, 0),
		Message: Message{
			Role:    string(choice.Message.Role),
			Content: content,
		},
		Done: true,
	}, nil
}

// Rerank reranks documents using the reranker model
func (c *OpenAIClient) Rerank(ctx context.Context, query string, documents []string, instruction string) ([]RerankResult, error) {
	if len(documents) == 0 {
		return []RerankResult{}, nil
	}

	// For OpenAI, we'll use a similar approach as Ollama by generating embeddings
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

// Generate performs text generation with the specified model
func (c *OpenAIClient) Generate(ctx context.Context, model string, prompt string, options map[string]interface{}) (*GenerateResponse, error) {
	if model == "" {
		model = c.config.ChatModel
	}

	params := openai.CompletionNewParams{
		Model: openai.CompletionNewParamsModelGPT3_5TurboInstruct,
		Prompt: openai.CompletionNewParamsPromptUnion{
			OfArrayOfStrings: []string{prompt},
		},
	}

	// Apply options if provided
	if options != nil {
		if temp, ok := options["temperature"].(float64); ok {
			params.Temperature = openai.Float(temp)
		}
		if topP, ok := options["top_p"].(float64); ok {
			params.TopP = openai.Float(topP)
		}
		if maxTokens, ok := options["max_tokens"].(int); ok {
			params.MaxTokens = openai.Int(int64(maxTokens))
		}
	}

	response, err := c.client.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create completion: %w", err)
	}

	// Convert OpenAI response to our generic response
	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	choice := response.Choices[0]
	content := choice.Text

	return &GenerateResponse{
		Model:     model,
		CreatedAt: time.Unix(response.Created, 0),
		Response:  content,
		Done:      true,
	}, nil
}
