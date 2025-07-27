# RAG CLI Documentation

This directory contains comprehensive documentation for the RAG CLI tool.

## Available Documentation

### 📚 [Database Setup Guide](DATABASE_SETUP.md)
Complete guide for setting up PostgreSQL database and user configuration, including:
- Database and user creation
- Permissions configuration
- SSL configuration and troubleshooting
- Security best practices
- Configuration examples

### 📋 [Requirements](REQUIREMENTS.md)
System requirements and dependencies for running RAG CLI.

## Quick Start

### 1. Database Setup

**Option A: Automated Setup (Recommended)**
```bash
# Run the automated database setup script
./scripts/setup-database.sh

```

### 2. Validate Configuration
```bash
# Test your database and Ollama connections
rag-cli config validate
```

### 3. Create Your First Collection
```bash
# Create a collection with your documents
rag-cli collection create my-docs -d "My documentation" -f ./docs

# Index the documents (by name)
rag-cli index my-docs-collection

# Index the documents (by UUID)
rag-cli index 550e8400-e29b-41d4-a716-446655440000

# Search your documents (by name)
rag-cli search my-docs-collection "how to use the API"

# Search your documents (by UUID)
rag-cli search 550e8400-e29b-41d4-a716-446655440000 "how to use the API"

# Chat with your documents (by name)
rag-cli chat my-docs-collection

# Chat with your documents (by UUID)
rag-cli chat 550e8400-e29b-41d4-a716-446655440000
```

## Common Issues

### SSL Connection Issues
If you encounter SSL-related errors like "SSL is not enabled on the server":

1. **Update your config file** to use `prefer` SSL mode:
   ```yaml
   # ~/.rag-cli/config.yaml
   database:
     ssl_mode: prefer
   ```

2. **Check PostgreSQL configuration**:
   ```bash
   grep -i ssl /opt/homebrew/var/postgresql@17/postgresql.conf
   ```

3. **Verify authentication settings**:
   ```bash
   grep -v "^#" /opt/homebrew/var/postgresql@17/pg_hba.conf | grep -v "^$"
   ```

### Permission Issues
If you get "permission denied for schema public":

```bash
# Grant schema permissions
psql postgres -c "GRANT ALL ON SCHEMA public TO ragcli_admin;"
```

### Database Connection Issues
If you can't connect to the database:

1. **Check if PostgreSQL is running**:
   ```bash
   brew services list | grep postgresql  # macOS
   sudo systemctl status postgresql      # Linux
   ```

2. **Verify database exists**:
   ```bash
   psql postgres -c "SELECT datname FROM pg_database WHERE datname = 'ragcli';"
   ```

3. **Test connection manually**:
   ```bash
   psql -U ragcli_admin -d ragcli -c "SELECT 1;"
   ```

## Configuration

### Configuration File Location
- **Default**: `~/.rag-cli/config.yaml`
- **Custom**: Use `--config` flag or `--config-name` for different environments

### Environment Variables
You can override configuration using environment variables:
```bash
export RAGCLI_DB_HOST=localhost
export RAGCLI_DB_PORT=5432
export RAGCLI_DB_NAME=ragcli
export RAGCLI_DB_USER=ragcli_admin
export RAGCLI_DB_PASSWORD=your_password
export RAGCLI_DB_SSL_MODE=prefer
```

### Database Configuration
Database settings should be configured in your config file:
```yaml
# ~/.rag-cli/config.yaml
database:
  host: localhost
  port: 5432
  name: ragcli
  user: ragcli_admin
  password: your_secure_password
  ssl_mode: prefer
```

### Ollama Configuration
Ollama settings can be overridden via command line flags or configured in the config file:
```bash
# Command line override
rag-cli --ollama-host=localhost --ollama-port=11434 collection list

# Or configure in config file
# ~/.rag-cli/config.yaml
ollama:
  host: localhost
  port: 11434
```

## Security Considerations

### Production Environments
- Use strong passwords for database users
- Enable SSL with `require` or `verify-full` mode in config
- Limit network access in `pg_hba.conf`
- Regular database backups
- Monitor connection logs

### Development Environments
- Use `prefer` SSL mode for flexibility
- Local connections only
- Simple password requirements

## Getting Help

### Troubleshooting Steps
1. **Check configuration**: `rag-cli config show`
2. **Validate connections**: `rag-cli config validate`
3. **Check logs**: Look for error messages in command output
4. **Verify PostgreSQL**: Test with `psql` directly
5. **Review documentation**: Check relevant sections in `DATABASE_SETUP.md`

### Common Commands
```bash
# Show current configuration
rag-cli config show

# Validate all connections
rag-cli config validate

# Edit configuration
rag-cli config edit

# List collections
rag-cli collection list

# Show collection details
rag-cli collection show <collection-id>

# Search documents
rag-cli search <collection-id> "query"

# Chat with documents
rag-cli chat <collection-id>
```

## Contributing

When adding new features or making changes:

1. **Update documentation** in the appropriate files
2. **Add examples** for new commands
3. **Include troubleshooting** steps for common issues
4. **Test the setup script** with your changes
5. **Update this README** if adding new documentation files

## Support

For issues not covered in this documentation:

1. Check the [main project README](../README.md)
2. Review the [requirements documentation](REQUIREMENTS.md)
3. Test with the [database setup script](../scripts/setup-database.sh)
4. Check PostgreSQL and Ollama documentation for external issues 