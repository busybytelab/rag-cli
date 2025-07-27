# Embedding Backend Configuration

## Overview

The RAG CLI supports separate backend configuration for embeddings and chat operations. This allows you to use different backends for different operations, providing flexibility in your RAG setup.

## Configuration

### Basic Configuration

In your `~/.rag-cli/config.yaml` file, you can now specify:

```yaml
# Main backend for chat and generation operations
backend: ollama

# Embedding backend for vector embeddings
embedding_backend: ollama  # defaults to backend if not specified
```

### Use Cases

#### 1. Use Ollama for Embeddings, OpenAI for Chat

```yaml
backend: openai
embedding_backend: ollama

ollama:
  host: localhost
  port: 11434
  model: llama3.2:3b
  embed_model: nomic-embed-text

openai:
  api_key: "your-openai-api-key"
  model: gpt-4
  embed_model: text-embedding-3-small
```

This configuration:
- Uses Ollama for generating embeddings (indexing and search)
- Uses OpenAI for chat operations
- Useful when you want fast local embeddings but powerful cloud-based chat

#### 2. Use OpenAI for Both

```yaml
backend: openai
embedding_backend: openai

openai:
  api_key: "your-openai-api-key"
  model: gpt-4
  embed_model: text-embedding-3-small
```

#### 3. Use Ollama for Both (Default)

```yaml
backend: ollama
# embedding_backend: ollama  # optional, defaults to backend

ollama:
  host: localhost
  port: 11434
  model: llama3.2:3b
  embed_model: nomic-embed-text
```

## How It Works

### Commands That Use Embeddings

The following commands use the `embedding_backend`:

- **`rag-cli index`**: Generates embeddings for document chunks
- **`rag-cli search`**: Generates embeddings for search queries (vector/hybrid search)
- **`rag-cli chat`**: Generates embeddings for user queries

### Commands That Use Chat

The following commands use the main `backend`:

- **`rag-cli chat`**: Generates responses using the chat model

### Implementation Details

- When `embedding_backend` is not specified, it defaults to the main `backend`
- Both backends are validated independently
- The system creates separate client instances for embeddings and chat when needed
- If both backends are the same, the system optimizes by reusing the same client instance

## Examples

### Scenario 1: Local Development with Cloud Chat

```bash
# Configure for local embeddings, cloud chat
cat > ~/.rag-cli/config.yaml << EOF
backend: openai
embedding_backend: ollama

ollama:
  host: localhost
  port: 11434
  model: llama3.2:3b
  embed_model: nomic-embed-text

openai:
  api_key: "your-openai-api-key"
  model: gpt-4
  embed_model: text-embedding-3-small

database:
  host: localhost
  port: 5432
  name: ragcli
  user: ragcli_admin
  password: "your-password"
  ssl_mode: disable

embedding:
  chunk_size: 1000
  chunk_overlap: 200
  similarity_threshold: 0.7
  max_results: 10

general:
  log_level: info
  data_dir: ~/.rag-cli/data
EOF

# Index documents using Ollama embeddings
rag-cli index my-collection

# Chat using OpenAI
rag-cli chat my-collection
```

### Scenario 2: Production with OpenAI

```bash
# Configure for production OpenAI usage
cat > ~/.rag-cli/config.yaml << EOF
backend: openai
embedding_backend: openai

openai:
  api_key: "your-openai-api-key"
  model: gpt-4
  embed_model: text-embedding-3-small

database:
  host: your-db-host
  port: 5432
  name: ragcli
  user: ragcli_admin
  password: "your-secure-password"
  ssl_mode: require

embedding:
  chunk_size: 1000
  chunk_overlap: 200
  similarity_threshold: 0.7
  max_results: 10

general:
  log_level: info
  data_dir: ~/.rag-cli/data
EOF
```

## Validation

The configuration system validates both backends:

```bash
# Check your configuration
rag-cli config show

# This will show both backend and embedding_backend settings
```

## Migration

If you have an existing configuration, the system will automatically:

1. Use your existing `backend` setting
2. Set `embedding_backend` to the same value as `backend`
3. Maintain backward compatibility

No manual migration is required. 