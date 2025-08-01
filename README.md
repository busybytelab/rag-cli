# RAG CLI

A powerful command-line interface for building and querying RAG (Retrieval-Augmented Generation) systems using Ollama and PostgreSQL.

## Features

- **Collection Management**: Create and manage collections of documents from various folders
- **Document Indexing**: Automatically chunk and embed documents using Ollama
- **Vector Search**: Perform similarity search on indexed documents
- **Interactive Chat**: Chat with your documents using RAG
- **PostgreSQL Integration**: Store embeddings and metadata in PostgreSQL with vector support
- **Configurable**: Flexible configuration for embedding models, chunk sizes, and more

## Prerequisites

### Quick Install (macOS with Homebrew)

```bash
# Install Go
brew install go

# Install PostgreSQL (version 14 or later)
brew install postgresql@17
brew services start postgresql@17

# Install pgvector extension
brew install pgvector

# Install Ollama
brew install ollama
ollama serve
```

### Manual Installation

For detailed installation instructions on various operating systems, see [REQUIREMENTS.md](docs/REQUIREMENTS.md).

**Required Components:**
- **Go 1.24.5+**: [Install Go](https://golang.org/doc/install)
- **PostgreSQL 11+**: [Install PostgreSQL](https://www.postgresql.org/download/)
- **pgvector extension**: [Install pgvector](https://github.com/pgvector/pgvector)
- **Ollama**: [Install Ollama](https://ollama.ai/)

## Installation

### From Source

1. Clone the repository:
```bash
git clone https://github.com/busybytelab.com/rag-cli.git
cd rag-cli
```

2. Build the application:
```bash
make build
```

3. Install the binary:
```bash
make install
```

### Using Go

```bash
go install github.com/busybytelab.com/rag-cli@latest
```

## Quick Start

1. **Initialize Configuration**:
```bash
rag-cli config init
```

2. **Create a Collection**:
```bash
rag-cli collection create my-docs --description "My documentation" --folders /path/to/docs
```

3. **Index Documents**:
```bash
rag-cli index <collection-id>
```

4. **Search Documents**:
```bash
# Search by collection name
rag-cli search my-docs-collection "your search query"

# Search by collection UUID
rag-cli search 550e8400-e29b-41d4-a716-446655440000 "your search query"
```

5. **Chat with Documents**:
```bash
# Chat by collection name
rag-cli chat my-docs-collection

# Chat by collection UUID
rag-cli chat 550e8400-e29b-41d4-a716-446655440000
```

## Configuration

The application uses a YAML configuration file located at `~/.rag-cli/config.yaml`. You can manage configuration using:

```bash
# Show current configuration
rag-cli config show

# Edit configuration
rag-cli config edit
```

### Default Configuration

```yaml
# Chat backend for chat and generation operations
chat_backend: ollama

# Embedding backend for vector embeddings (defaults to chat backend if not specified)
embedding_backend: ollama

ollama:
  host: localhost
  port: 11434
  tls: false
  chat_model: qwen3:4b
  embedding_model: dengcao/Qwen3-Embedding-0.6B:Q8_0

openai:
  api_key: ""
  base_url: ""
  chat_model: gpt-4
  embedding_model: text-embedding-3-small

database:
  host: localhost
  port: 5432
  name: rag_cli
  user: postgres
  password: ""
  ssl_mode: disable

embedding:
  chunk_size: 1000
  chunk_overlap: 200
  similarity_threshold: 0.7
  max_results: 10

general:
  log_level: info
  data_dir: ~/.rag-cli/data
```

### Backend Configuration

The application supports two backends: **Ollama** and **OpenAI**. You can configure them separately:

- **`chat_backend`**: Used for chat and text generation operations
- **`embedding_backend`**: Used for generating vector embeddings (defaults to chat backend if not specified)

This allows you to use different backends for different operations. For example:

```yaml
# Use OpenAI for chat but Ollama for embeddings
chat_backend: openai
embedding_backend: ollama

# Use Ollama for both (default behavior)
chat_backend: ollama
embedding_backend: ollama  # or omit this line to use the same as chat_backend
```

## Usage

### Collection Management

```bash
# Create a new collection
rag-cli collection create my-docs --description "My documentation" --folders /path/to/docs,/path/to/other/docs

# List all collections
rag-cli collection list

# Show collection details by name
rag-cli collection show my-docs-collection

# Show collection details by UUID
rag-cli collection show 550e8400-e29b-41d4-a716-446655440000

# Delete a collection by name
rag-cli collection delete my-docs-collection --force

# Delete a collection by UUID
rag-cli collection delete 550e8400-e29b-41d4-a716-446655440000 --force
```

### Document Indexing

```bash
# Index documents in a collection by name
rag-cli index my-docs-collection

# Index documents in a collection by UUID
rag-cli index 550e8400-e29b-41d4-a716-446655440000

# Force re-indexing of all documents
rag-cli index my-docs-collection --force
```

### Search

```bash
# Search documents by collection name
rag-cli search my-docs-collection "your search query"

# Search documents by collection UUID
rag-cli search 550e8400-e29b-41d4-a716-446655440000 "your search query"

# Search with custom limit
rag-cli search my-docs-collection "your search query" --limit 20

# Show full content in results
rag-cli search my-docs-collection "your search query" --show-content
```

### Shell Completion

Enable command-line completion for faster and more convenient usage:

#### macOS (zsh)
```bash
# Install completion for zsh
rag-cli completion zsh > $(brew --prefix)/share/zsh/site-functions/_rag-cli

# Reload your shell or restart your terminal
source ~/.zshrc
```

#### Other Shells
```bash
# For bash
rag-cli completion bash > ~/.local/share/bash-completion/completions/rag-cli

# For fish
rag-cli completion fish > ~/.config/fish/completions/rag-cli.fish

# For PowerShell
rag-cli completion powershell > ~/.config/powershell/rag-cli.ps1
```

For more options and help:
```bash
rag-cli completion -h
# or
rag-cli completion bash -h
rag-cli completion zsh -h
rag-cli completion fish -h
rag-cli completion powershell -h
```

### Chat

```bash
# Start interactive chat
rag-cli chat <collection-id>

# Chat with custom context limit
rag-cli chat <collection-id> --limit 10
```

## Supported File Types

The application supports indexing of various text file types:

- **Documentation**: `.txt`, `.md`, `.rst`, `.tex`
- **Code**: `.py`, `.js`, `.ts`, `.go`, `.rs`, `.cpp`, `.c`, `.java`, `.cs`, `.php`, `.rb`, `.pl`, `.sql`
- **Configuration**: `.json`, `.xml`, `.yaml`, `.yml`, `.toml`, `.ini`, `.cfg`, `.conf`
- **Web**: `.html`, `.htm`, `.css`, `.scss`, `.sass`, `.less`
- **Data**: `.csv`, `.log`

## Architecture

### Components

- **Database Layer**: PostgreSQL with pgvector for storing embeddings and metadata
- **Embedding Service**: Ollama integration for generating embeddings
- **Chunking**: Intelligent text chunking with overlap
- **Search**: Vector similarity search using embeddings
- **Chat**: RAG-based conversation using retrieved context

### Data Flow

1. **Indexing**: Documents → Chunking → Embedding → Database Storage
2. **Search**: Query → Embedding → Vector Search → Results
3. **Chat**: Query → Embedding → Vector Search → Context → LLM Response

## Development

### Building

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Clean build artifacts
make clean
```

### Testing

```bash
# Run unit tests
make unit-test

# Run integration tests
make integration-test

# Run all tests
make test
```

### Development Setup

```bash
# Setup development environment
make dev-setup

# Run with hot reload (requires air)
make dev
```

## Troubleshooting

### Common Issues

1. **PostgreSQL Connection Error**:
   - Ensure PostgreSQL is running
   - Check database credentials in configuration
   - Verify pgvector extension is installed

2. **Ollama Connection Error**:
   - Ensure Ollama is running
   - Check Ollama host and port in configuration
   - Verify required models are pulled

3. **Embedding Generation Error**:
   - Check if embedding model is available in Ollama
   - Pull the model: `ollama pull nomic-embed-text`

### Logs

Enable verbose output for debugging:
```bash
rag-cli --verbose <command>
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For support and questions:
- Open an issue on GitHub
- Check the documentation
- Review the configuration examples
