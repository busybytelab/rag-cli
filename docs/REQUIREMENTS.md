# Requirements Guide

This guide provides detailed installation instructions for all requirements needed by RAG CLI on various operating systems.

## Table of Contents

- [Go](#go)
- [PostgreSQL](#postgresql)
- [pgvector Extension](#pgvector-extension)
- [Ollama](#ollama)
- [Verification](#verification)

## Go

### macOS

#### Using Homebrew (Recommended)
```bash
brew install go
```

#### Manual Installation
1. Download from [golang.org/dl](https://golang.org/dl/)
2. Run the installer
3. Add to PATH: `export PATH=$PATH:/usr/local/go/bin`

### Linux (Ubuntu/Debian)
```bash
# Using apt
sudo apt update
sudo apt install golang-go

# Or download from golang.org/dl
wget https://go.dev/dl/go1.24.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.24.5.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

### Linux (CentOS/RHEL/Fedora)
```bash
# Using dnf/yum
sudo dnf install golang  # Fedora
sudo yum install golang  # CentOS/RHEL

# Or download from golang.org/dl
wget https://go.dev/dl/go1.24.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.24.5.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

### Windows
1. Download from [golang.org/dl](https://golang.org/dl/)
2. Run the MSI installer
3. Follow the installation wizard
4. Restart your terminal/command prompt

### Verification
```bash
go version
# Should output: go version go1.24.5 ...
```

## PostgreSQL

### macOS

#### Using Homebrew (Recommended)
```bash
# Install PostgreSQL (version 14 or later)
brew install postgresql@17

# Start PostgreSQL service
brew services start postgresql@17

# Note: On macOS with Homebrew, your system user becomes the default superuser
# Connect to PostgreSQL (your system user is the default superuser)
psql postgres

# Create database for RAG CLI
createdb ragcli

# Create database user for RAG CLI (regular user, not superuser)
createuser ragcli_admin

# Grant full access to ragcli database only
psql -d ragcli -c "GRANT ALL PRIVILEGES ON DATABASE ragcli TO ragcli_admin;"
psql -d ragcli -c "GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO ragcli_admin;"
psql -d ragcli -c "GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO ragcli_admin;"
psql -d ragcli -c "ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO ragcli_admin;"
psql -d ragcli -c "ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO ragcli_admin;"

# Verify the database was created
psql -U ragcli_admin -d ragcli -c "\l" | grep ragcli
```

#### Using Postgres.app
1. Download [Postgres.app](https://postgresapp.com/)
2. Drag to Applications folder
3. Double-click to start
4. Click "Initialize" to create a new server

### Linux (Ubuntu/Debian)
```bash
# Install PostgreSQL
sudo apt update
sudo apt install postgresql postgresql-contrib

# Start service
sudo systemctl start postgresql
sudo systemctl enable postgresql

# Create database user
sudo -u postgres createuser --superuser postgres
sudo -u postgres createdb rag_cli
```

### Linux (CentOS/RHEL/Fedora)
```bash
# Install PostgreSQL
sudo dnf install postgresql postgresql-server postgresql-contrib  # Fedora
sudo yum install postgresql postgresql-server postgresql-contrib  # CentOS/RHEL

# Initialize database
sudo postgresql-setup initdb

# Start service
sudo systemctl start postgresql
sudo systemctl enable postgresql

# Create database user
sudo -u postgres createuser --superuser postgres
sudo -u postgres createdb rag_cli
```

### Windows
1. Download from [postgresql.org/download/windows](https://www.postgresql.org/download/windows/)
2. Run the installer
3. Choose installation directory
4. Set password for postgres user
5. Keep default port (5432)
6. Complete installation

### Verification
```bash
psql --version
# Should output: psql (PostgreSQL) 14.x or later ...
```

## pgvector Extension

pgvector is an open-source PostgreSQL extension that provides vector similarity search capabilities. It supports PostgreSQL 11+ and is compatible with PostgreSQL 14+ (recommended for best performance).

### macOS

#### Using Homebrew (Recommended)
```bash
brew install pgvector
```

#### Manual Installation
```bash
# Install build dependencies (PostgreSQL 14+)
brew install postgresql@17

# Clone and build pgvector (latest stable version)
git clone --branch v0.8.0 https://github.com/pgvector/pgvector.git
cd pgvector
make
make install
```

### Linux (Ubuntu/Debian)
```bash
# Install build dependencies
sudo apt install postgresql-server-dev build-essential git

# Clone and build pgvector
git clone --branch v0.8.0 https://github.com/pgvector/pgvector.git
cd pgvector
make
sudo make install
```

### Linux (CentOS/RHEL/Fedora)
```bash
# Install build dependencies
sudo dnf install postgresql-devel gcc make git  # Fedora
sudo yum install postgresql-devel gcc make git  # CentOS/RHEL

# Clone and build pgvector
git clone --branch v0.8.0 https://github.com/pgvector/pgvector.git
cd pgvector
make
sudo make install
```

### Windows
1. Install Visual Studio Build Tools
2. Install PostgreSQL development headers
3. Follow the [pgvector Windows installation guide](https://github.com/pgvector/pgvector#windows)

### Enable Extension
After installation, enable the extension in your database:
```bash
psql -d ragcli -c "CREATE EXTENSION IF NOT EXISTS vector;"
# verify
psql -d ragcli -c "SELECT * FROM pg_extension WHERE extname = 'vector';"
```
or:
```sql
CREATE EXTENSION IF NOT EXISTS vector;
# verify
SELECT * FROM pg_extension WHERE extname = 'vector';
-- Should return a row if extension is installed
```

## Ollama

### macOS

#### Using Homebrew (Recommended)
```bash
brew install ollama
ollama serve
```

#### Manual Installation
```bash
# Download and install
curl -fsSL https://ollama.ai/install.sh | sh

# Start Ollama
ollama serve
```

### Linux
```bash
# Download and install
curl -fsSL https://ollama.ai/install.sh | sh

# Start Ollama
ollama serve
```

### Windows
1. Download from [ollama.ai/download](https://ollama.ai/download)
2. Run the installer
3. Start Ollama from the Start menu or run `ollama serve`

### Pull Required Models
```bash
# Pull embedding model
ollama pull nomic-embed-text

# Pull LLM model (optional, will be pulled automatically when needed)
ollama pull llama3.2:3b
```

### Verification
```bash
# Check if Ollama is running
curl http://localhost:11434/api/tags

# Should return a JSON response with available models
```

## Verification

After installing all components, verify your setup:

```bash
# Check Go
go version

# Check PostgreSQL
psql --version

# Check Ollama
curl http://localhost:11434/api/tags

# Test pgvector extension
psql -d rag_cli -c "CREATE EXTENSION IF NOT EXISTS vector;"
```

## Troubleshooting

### Common Issues

1. **PostgreSQL Connection Refused**
   - Ensure PostgreSQL is running: `brew services list` (macOS) or `sudo systemctl status postgresql` (Linux)
   - Check if port 5432 is available: `lsof -i :5432`

2. **pgvector Extension Not Found**
   - Verify installation: `ls /usr/local/lib/postgresql/` (should contain vector.so)
   - Restart PostgreSQL after installing pgvector

3. **Ollama Connection Refused**
   - Ensure Ollama is running: `ollama serve`
   - Check if port 11434 is available: `lsof -i :11434`

4. **Permission Issues**
   - On Linux, ensure proper PostgreSQL user permissions
   - On macOS, check Homebrew permissions

### Getting Help

- **Go**: [golang.org/doc/install](https://golang.org/doc/install)
- **PostgreSQL**: [postgresql.org/docs](https://www.postgresql.org/docs/)
- **pgvector**: [github.com/pgvector/pgvector](https://github.com/pgvector/pgvector)
- **Ollama**: [ollama.ai/docs](https://ollama.ai/docs)

## Next Steps

After completing the installation:

1. [Install RAG CLI](../README.md#installation)
2. [Configure the application](../README.md#configuration)
3. [Create your first collection](../README.md#quick-start) 