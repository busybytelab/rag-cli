# RAG CLI Reranking Feature

## Overview

The RAG CLI now supports **reranking** as an advanced search option that improves retrieval accuracy through a two-stage process:

1. **Initial Search**: Fast vector similarity or hybrid search to retrieve candidate documents
2. **Reranking**: Precise relevance scoring using dedicated reranking models

This approach combines the speed of vector search with the accuracy of cross-encoder reranking models.

## How Reranking Works

### Two-Stage Retrieval Process

1. **Stage 1 - Dense Retrieval**: 
   - Uses embedding models (like Qwen3-Embedding) for fast similarity search
   - Retrieves a larger set of candidate documents (e.g., top 20-50)
   - Provides broad coverage but may miss some relevant documents

2. **Stage 2 - Reranking**:
   - Uses reranking models (like Qwen3-Reranker) to score query-document pairs
   - Evaluates semantic relevance more precisely than embeddings
   - Reranks and filters the candidate set for final results

### Supported Reranking Models

#### Ollama Backend
- **Qwen3-Reranker-0.6B**: Lightweight, fast reranking model
- **Qwen3-Reranker-4B**: Balanced performance and accuracy
- **Qwen3-Reranker-8B**: Highest accuracy, slower inference

#### OpenAI Backend
- **text-embedding-3-small**: Used as fallback (not a true reranker)
- **text-embedding-3-large**: Higher quality embeddings for reranking

## Configuration

### Basic Configuration

Add reranker models to your `~/.rag-cli/config.yaml`:

```yaml
# Ollama configuration
ollama:
  host: localhost
  port: 11434
  chat_model: qwen3:4b
  embedding_model: dengcao/Qwen3-Embedding-0.6B:Q8_0
  reranker_model: dengcao/Qwen3-Reranker-0.6B:Q8_0

# OpenAI configuration
openai:
  api_key: "your-openai-api-key"
  chat_model: gpt-4
  embedding_model: text-embedding-3-small
  reranker_model: text-embedding-3-small
```

### Installing Reranking Models

For Ollama, pull the reranking models:

```bash
# Install Qwen3 reranking models
ollama pull dengcao/Qwen3-Reranker-0.6B:Q8_0
ollama pull dengcao/Qwen3-Reranker-4B:Q8_0
ollama pull dengcao/Qwen3-Reranker-8B:Q8_0
```

## Usage

### Search Command with Reranking

Enable reranking in the search command with the `--rerank` flag:

```bash
# Basic search with reranking
rag-cli search my-collection "What is machine learning?" --rerank

# Advanced search with reranking options
rag-cli search my-collection "How do neural networks work?" \
  --rerank \
  --limit 20 \
  --rerank-limit 10 \
  --rerank-instruction "Focus on technical implementation details" \
  --original-weight 0.6 \
  --rerank-weight 0.4 \
  --show-scores
```

### Chat Command with Reranking

Enable reranking in the chat command for improved document retrieval:

```bash
# Start chat with reranking enabled
rag-cli chat my-collection --rerank

# Chat with custom reranking instruction
rag-cli chat my-collection --rerank --rerank-instruction "Focus on practical examples"

# Chat with specific search type and reranking
rag-cli chat my-collection --search-type hybrid --rerank
```

### Command Options

#### Search Command Reranking Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--rerank, -r` | Enable reranking for improved results | false |
| `--rerank-instruction` | Custom instruction for reranking | "Given a web search query..." |
| `--original-weight` | Weight for original search score (0.0-1.0) | 0.7 |
| `--rerank-weight` | Weight for reranking score (0.0-1.0) | 0.3 |
| `--rerank-limit` | Number of results to rerank (0 = all) | 0 |

#### Chat Command Reranking Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--rerank, -r` | Enable reranking for document retrieval | false |
| `--rerank-instruction` | Custom instruction for reranking | "" |

### Search Types with Reranking

Reranking can be applied to any search type:

```bash
# Vector search + reranking
rag-cli search my-collection "query" --type vector --rerank

# Text search + reranking  
rag-cli search my-collection "query" --type text --rerank

# Hybrid search + reranking (recommended)
rag-cli search my-collection "query" --type hybrid --rerank

# Semantic search + reranking
rag-cli search my-collection "query" --type semantic --rerank
```

## Advanced Features

### Custom Instructions

Reranking models support custom instructions to tailor relevance scoring:

```bash
# Technical focus
rag-cli search my-collection "API documentation" \
  --rerank \
  --rerank-instruction "Focus on technical implementation details and code examples"

# Academic focus
rag-cli search my-collection "research papers" \
  --rerank \
  --rerank-instruction "Prioritize academic rigor and research methodology"

# Business focus
rag-cli search my-collection "market analysis" \
  --rerank \
  --rerank-instruction "Focus on business insights and market trends"
```

### Score Weighting

Control the balance between initial search and reranking scores:

```bash
# Equal weighting
rag-cli search my-collection "query" \
  --rerank \
  --original-weight 0.5 \
  --rerank-weight 0.5

# Reranking-focused
rag-cli search my-collection "query" \
  --rerank \
  --original-weight 0.3 \
  --rerank-weight 0.7

# Search-focused
rag-cli search my-collection "query" \
  --rerank \
  --original-weight 0.8 \
  --rerank-weight 0.2
```

### Filtering and Limits

```bash
# Rerank only top 10 results from initial search
rag-cli search my-collection "query" \
  --rerank \
  --limit 50 \
  --rerank-limit 10

# Apply minimum score threshold
rag-cli search my-collection "query" \
  --rerank \
  --min-score 0.7
```

## Performance Considerations

### Model Selection

| Model | Speed | Accuracy | Memory | Use Case |
|-------|-------|----------|--------|----------|
| Qwen3-Reranker-0.6B | Fast | Good | Low | Development, testing |
| Qwen3-Reranker-4B | Medium | Better | Medium | Production, balanced |
| Qwen3-Reranker-8B | Slow | Best | High | High-accuracy applications |

### Optimization Tips

1. **Start with 0.6B model** for development and testing
2. **Use appropriate limits** to balance speed and accuracy
3. **Cache embeddings** for frequently accessed documents
4. **Batch reranking** for multiple queries when possible

## Troubleshooting

### Common Issues

1. **Model not found**: Ensure reranking models are installed in Ollama
2. **Slow performance**: Reduce rerank limits or use smaller models
3. **Memory issues**: Use quantized models (Q8_0) for lower memory usage
4. **Low accuracy**: Adjust instruction prompts or score weights

### Debugging

Enable detailed scoring to understand reranking behavior:

```bash
rag-cli search my-collection "query" \
  --rerank \
  --show-scores \
  --limit 5
```

This shows:
- Original search scores (vector/text)
- Reranking scores
- Combined scores
- Final rankings

## Best Practices

1. **Use hybrid search** as the initial search type for best coverage
2. **Start with default weights** (0.7 original, 0.3 rerank) and adjust based on results
3. **Write clear instructions** that match your use case
4. **Monitor performance** and adjust model sizes accordingly
5. **Test with your specific data** to find optimal settings

## Examples

### Technical Documentation Search

```bash
rag-cli search docs-collection "How to implement authentication?" \
  --rerank \
  --rerank-instruction "Focus on code examples and implementation details" \
  --limit 20 \
  --rerank-limit 8 \
  --show-scores
```

### Research Paper Search

```bash
rag-cli search papers-collection "machine learning algorithms" \
  --rerank \
  --rerank-instruction "Prioritize papers with experimental results and benchmarks" \
  --type semantic \
  --limit 30 \
  --rerank-limit 10
```

### Business Intelligence

```bash
rag-cli search reports-collection "market trends 2024" \
  --rerank \
  --rerank-instruction "Focus on actionable insights and business recommendations" \
  --original-weight 0.4 \
  --rerank-weight 0.6
```

### Interactive Chat with Reranking

```bash
# Start chat with reranking for better document retrieval
rag-cli chat my-collection --rerank

# Chat with technical focus
rag-cli chat my-collection --rerank --rerank-instruction "Focus on technical details and code examples"

# Chat with academic focus
rag-cli chat my-collection --rerank --rerank-instruction "Prioritize academic rigor and research methodology"
``` 