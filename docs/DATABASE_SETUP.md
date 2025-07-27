# Database Setup Guide

This guide covers setting up PostgreSQL for use with the RAG CLI tool, including database creation, user setup, permissions, and troubleshooting common issues.

## Table of Contents

- [Database and User Setup](#database-and-user-setup)
- [Permissions Configuration](#permissions-configuration)
- [SSL Configuration](#ssl-configuration)
- [Troubleshooting](#troubleshooting)
- [Configuration Examples](#configuration-examples)

## Database and User Setup

### 1. Connect to PostgreSQL

```bash
# Connect as the default postgres user
psql postgres
```

### 2. Create Database

```sql
-- Create the database for RAG CLI
CREATE DATABASE ragcli;
```

### 3. Create User

```sql
-- Create a dedicated user for RAG CLI
CREATE USER ragcli_admin WITH PASSWORD 'your_secure_password';

-- Grant database privileges
GRANT ALL PRIVILEGES ON DATABASE ragcli TO ragcli_admin;
```

### 4. Connect to the New Database

```sql
-- Connect to the ragcli database
\c ragcli

-- Grant schema permissions
GRANT ALL ON SCHEMA public TO ragcli_admin;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO ragcli_admin;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO ragcli_admin;

-- Set default privileges for future objects
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO ragcli_admin;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO ragcli_admin;
```

### 5. Verify Setup

```bash
# Test connection with the new user
psql -U ragcli_admin -d ragcli -c "SELECT 1;"
```

## Permissions Configuration

### Required Permissions

The `ragcli_admin` user needs the following permissions:

- **Database Access**: Full access to the `ragcli` database
- **Schema Access**: Full access to the `public` schema
- **Table Creation**: Ability to create, modify, and delete tables
- **Extension Installation**: Ability to install the `vector` extension

### Permission Commands

```sql
-- Grant all privileges on database
GRANT ALL PRIVILEGES ON DATABASE ragcli TO ragcli_admin;

-- Grant schema permissions
GRANT ALL ON SCHEMA public TO ragcli_admin;

-- Grant permissions on existing objects
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO ragcli_admin;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO ragcli_admin;

-- Set default privileges for future objects
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO ragcli_admin;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO ragcli_admin;

-- Grant extension creation permission (if needed)
GRANT CREATE ON DATABASE ragcli TO ragcli_admin;
```

## SSL Configuration

### Understanding SSL Modes

The RAG CLI supports different SSL modes for database connections:

- **`disable`**: No SSL (not recommended for production)
- **`allow`**: Try non-SSL first, fall back to SSL
- **`prefer`**: Try SSL first, fall back to non-SSL (default)
- **`require`**: Require SSL connection
- **`verify-ca`**: Require SSL and verify CA certificate
- **`verify-full`**: Require SSL and verify CA certificate + hostname


### Common SSL Issues

#### Issue: "SSL is not enabled on the server"

**Cause**: The Go PostgreSQL driver is trying to negotiate SSL even when SSL is disabled on the server.

**Solution**: Update your config file to use `prefer` mode instead of `disable`:

```yaml
# ~/.rag-cli/config.yaml
database:
  ssl_mode: prefer
```

#### Issue: "unrecognized configuration parameter sslcertmode"

**Cause**: The `sslcertmode` parameter is not supported by the Go PostgreSQL driver.

**Solution**: Remove `sslcertmode` from the connection string and use standard SSL modes.

## Troubleshooting

### Connection Issues

#### 1. "Connection refused"

```bash
# Check if PostgreSQL is running
brew services list | grep postgresql  # macOS
sudo systemctl status postgresql      # Linux

# Start PostgreSQL if not running
brew services start postgresql@17     # macOS
sudo systemctl start postgresql       # Linux
```

#### 2. "Database does not exist"

```bash
# Create the database
psql postgres -c "CREATE DATABASE ragcli;"
```

#### 3. "User does not exist"

```bash
# Create the user
psql postgres -c "CREATE USER ragcli_admin WITH PASSWORD 'your_password';"
psql postgres -c "GRANT ALL PRIVILEGES ON DATABASE ragcli TO ragcli_admin;"
```

#### 4. "Permission denied for schema public"

```bash
# Grant schema permissions
psql postgres -c "GRANT ALL ON SCHEMA public TO ragcli_admin;"
```

### Configuration Issues

#### 1. Check PostgreSQL Configuration

```bash
# Check SSL configuration
grep -i ssl /opt/homebrew/var/postgresql@17/postgresql.conf

# Check authentication configuration
grep -v "^#" /opt/homebrew/var/postgresql@17/pg_hba.conf | grep -v "^$"
```

#### 2. Test Connection with psql

```bash
# Test basic connection
psql -U ragcli_admin -d ragcli -c "SELECT 1;"

# Test with specific SSL mode
psql "host=localhost port=5432 dbname=ragcli user=ragcli_admin sslmode=prefer"
```

### Performance Issues

#### 1. Connection Pool Settings

The RAG CLI uses the following connection pool settings:

- **Max Open Connections**: 25
- **Max Idle Connections**: 25
- **Connection Lifetime**: 5 minutes

#### 2. Vector Extension

Ensure the `vector` extension is available:

```sql
-- Check if vector extension is available
SELECT * FROM pg_available_extensions WHERE name = 'vector';

-- Install vector extension (if needed)
CREATE EXTENSION IF NOT EXISTS vector;
```

## Configuration Examples

### Basic Configuration

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

### Environment Variables

You can override configuration using environment variables:
```bash
# Set environment variables for database connection
export RAGCLI_DB_HOST=localhost
export RAGCLI_DB_PORT=5432
export RAGCLI_DB_NAME=ragcli
export RAGCLI_DB_USER=ragcli_admin
export RAGCLI_DB_PASSWORD=your_secure_password
export RAGCLI_DB_SSL_MODE=prefer
```

### Docker Configuration

If using PostgreSQL in Docker:

```bash
# Run PostgreSQL container
docker run -d \
  --name postgres-rag \
  -e POSTGRES_DB=ragcli \
  -e POSTGRES_USER=ragcli_admin \
  -e POSTGRES_PASSWORD=your_secure_password \
  -p 5432:5432 \
  postgres:17

# Wait for container to start
sleep 10

# Grant permissions
docker exec -it postgres-rag psql -U ragcli_admin -d ragcli -c "
GRANT ALL ON SCHEMA public TO ragcli_admin;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO ragcli_admin;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO ragcli_admin;
"
```

## Security Best Practices

### 1. Use Strong Passwords

```sql
-- Create user with strong password
CREATE USER ragcli_admin WITH PASSWORD 'complex_password_here';
```

### 2. Limit Network Access

```bash
# Configure pg_hba.conf to limit access
# Only allow local connections
host    ragcli    ragcli_admin    127.0.0.1/32    md5
host    ragcli    ragcli_admin    ::1/128         md5
```

### 3. Use SSL in Production

Configure SSL for production environments in your config file:
```yaml
# ~/.rag-cli/config.yaml
database:
  ssl_mode: require  # or verify-full for maximum security
```

### 4. Regular Backups

```bash
# Create backup script
#!/bin/bash
pg_dump -U ragcli_admin -d ragcli > backup_$(date +%Y%m%d_%H%M%S).sql
```

## Next Steps

After completing the database setup:

1. **Validate Configuration**: Run `rag-cli config validate` to verify everything works
2. **Create Collections**: Use `rag-cli collection create` to create your first collection
3. **Index Documents**: Use `rag-cli index` to index your documents
4. **Search and Chat**: Use `rag-cli search` and `rag-cli chat` to interact with your documents

For more information, see the [main README](../README.md) and other documentation in the `docs/` folder. 