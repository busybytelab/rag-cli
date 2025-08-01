#!/bin/bash

# RAG CLI Database Setup Script
# This script automates the setup of PostgreSQL database and user for RAG CLI

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
HOST="localhost"
DB_NAME="ragcli"
DB_USER="ragcli_admin"
DB_PASSWORD=""
POSTGRES_USER="postgres"

# Detect the correct PostgreSQL superuser
detect_postgres_user() {
    if command -v brew &> /dev/null; then
        # On macOS with Homebrew, the default superuser is the system username
        POSTGRES_USER=$(whoami)
        print_status "Detected PostgreSQL superuser: $POSTGRES_USER (macOS Homebrew)"
    else
        # On Linux, try to detect the postgres user
        if id "postgres" &>/dev/null; then
            POSTGRES_USER="postgres"
        else
            # Fallback to current user
            POSTGRES_USER=$(whoami)
            print_warning "PostgreSQL superuser 'postgres' not found, using current user: $POSTGRES_USER"
        fi
    fi
}

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to show usage
show_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -H, --host HOST       PostgreSQL host (default: localhost)"
    echo "  -n, --db-name NAME     Database name (default: ragcli)"
    echo "  -u, --db-user USER     Database user (default: ragcli_admin)"
    echo "  -p, --db-password PASS Database password (will prompt if not provided)"
    echo "  -s, --postgres-user USER PostgreSQL superuser (auto-detected, can override)"
    echo "  -h, --help            Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                                    # Use defaults with auto-detected superuser"
    echo "  $0 -H localhost -n mydb -u myuser -p mypass  # Custom host, database and user"
    echo "  $0 --db-password mypass              # Set password only"
    echo "  $0 -s postgres                       # Override superuser (Linux)"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -H|--host)
            HOST="$2"
            shift 2
            ;;
        -n|--db-name)
            DB_NAME="$2"
            shift 2
            ;;
        -u|--db-user)
            DB_USER="$2"
            shift 2
            ;;
        -p|--db-password)
            DB_PASSWORD="$2"
            shift 2
            ;;
        -s|--postgres-user)
            POSTGRES_USER="$2"
            shift 2
            ;;
        -h|--help)
            show_usage
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Detect PostgreSQL superuser if not explicitly provided
if [ "$POSTGRES_USER" = "postgres" ]; then
    detect_postgres_user
fi

# Check if PostgreSQL is running
print_status "Checking PostgreSQL status..."

if command -v brew &> /dev/null; then
    # macOS with Homebrew
    if ! brew services list | grep -q "postgresql.*started"; then
        print_error "PostgreSQL is not running. Please start it first:"
        echo "  brew services start postgresql@17"
        exit 1
    fi
else
    # Linux
    if ! systemctl is-active --quiet postgresql; then
        print_error "PostgreSQL is not running. Please start it first:"
        echo "  sudo systemctl start postgresql"
        exit 1
    fi
fi

print_success "PostgreSQL is running"

# Prompt for password if not provided and user is different from current user
if [ -z "$DB_PASSWORD" ] && [ "$DB_USER" != "$(whoami)" ]; then
    echo -n "Enter password for database user '$DB_USER': "
    read -s DB_PASSWORD
    echo
fi

# Check if psql is available
if ! command -v psql &> /dev/null; then
    print_error "psql command not found. Please install PostgreSQL client tools."
    exit 1
fi

# Check if pgvector extension is installed (early warning)
print_status "Checking pgvector extension installation..."
if psql -h $HOST -U "$POSTGRES_USER" -d postgres -c "SELECT 1 FROM pg_available_extensions WHERE name = 'vector';" | grep -q 1; then
    print_success "pgvector extension is installed"
else
    print_warning "pgvector extension is not installed. You may need to install it:"
    echo "  brew install pgvector  # macOS"
    echo "  sudo apt install postgresql-vector  # Ubuntu/Debian"
    echo "  # Or build from source: https://github.com/pgvector/pgvector"
    echo ""
    print_warning "Continuing setup, but RAG CLI will not work without pgvector."
    exit 1
fi

print_status "Setting up database '$DB_NAME' with user '$DB_USER'..."

# Create database if it doesn't exist
print_status "Creating database..."
if psql -h $HOST -U "$POSTGRES_USER" -d postgres -c "SELECT 1 FROM pg_database WHERE datname = '$DB_NAME';" | grep -q 1; then
    print_warning "Database '$DB_NAME' already exists"
else
    psql -h $HOST -U "$POSTGRES_USER" -d postgres -c "CREATE DATABASE $DB_NAME;" > /dev/null 2>&1
    print_success "Database '$DB_NAME' created"
fi

# Create user if it doesn't exist
print_status "Creating user..."
if psql -h $HOST -U "$POSTGRES_USER" -d postgres -c "SELECT 1 FROM pg_roles WHERE rolname = '$DB_USER';" | grep -q 1; then
    print_warning "User '$DB_USER' already exists"
    # Update password only if provided
    if [ -n "$DB_PASSWORD" ]; then
        psql -h $HOST -U "$POSTGRES_USER" -d postgres -c "ALTER USER $DB_USER WITH PASSWORD '$DB_PASSWORD';" > /dev/null 2>&1
        print_success "Password updated for user '$DB_USER'"
    else
        print_success "User '$DB_USER' already exists (no password change needed)"
    fi
else
    if [ -n "$DB_PASSWORD" ]; then
        psql -h $HOST -U "$POSTGRES_USER" -d postgres -c "CREATE USER $DB_USER WITH PASSWORD '$DB_PASSWORD';" > /dev/null 2>&1
    else
        psql -h $HOST -U "$POSTGRES_USER" -d postgres -c "CREATE USER $DB_USER;" > /dev/null 2>&1
    fi
    print_success "User '$DB_USER' created"
fi

# Grant database privileges
print_status "Granting database privileges..."
psql -h $HOST -U "$POSTGRES_USER" -d postgres -c "GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $DB_USER;" > /dev/null 2>&1
psql -h $HOST -U "$POSTGRES_USER" -d postgres -c "GRANT CREATE ON DATABASE $DB_NAME TO $DB_USER;" > /dev/null 2>&1

# Connect to the database and grant schema privileges
print_status "Granting schema privileges..."
# First, transfer ownership of the public schema
psql -h $HOST -U "$POSTGRES_USER" -d "$DB_NAME" -c "ALTER SCHEMA public OWNER TO $DB_USER;" > /dev/null 2>&1

# Then grant all other privileges
psql -h $HOST -U "$POSTGRES_USER" -d "$DB_NAME" -c "
GRANT ALL ON SCHEMA public TO $DB_USER;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO $DB_USER;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO $DB_USER;
GRANT ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA public TO $DB_USER;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO $DB_USER;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO $DB_USER;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON FUNCTIONS TO $DB_USER;
" > /dev/null 2>&1

# Grant extension creation privileges (required for vector extension)
print_status "Granting extension privileges..."

psql -h $HOST -U "$POSTGRES_USER" -d "$DB_NAME" -c "GRANT CREATE ON SCHEMA public TO $DB_USER;" > /dev/null 2>&1

# Create vector extension as superuser (required for RAG CLI)
print_status "Creating vector extension..."
# Check if extension is already created
if psql -h $HOST -U "$POSTGRES_USER" -d "$DB_NAME" -c "SELECT 1 FROM pg_extension WHERE extname = 'vector';" | grep -q 1; then
    print_warning "Vector extension already exists"
else
    print_status "Creating vector extension in database '$DB_NAME'..."
    psql -h $HOST -U "$POSTGRES_USER" -d "$DB_NAME" -c "CREATE EXTENSION vector;" > /dev/null 2>&1
    if [ $? -eq 0 ]; then
        print_success "Vector extension created successfully"
    else
        print_error "Failed to create vector extension"
        print_status "Trying to create extension with verbose output..."
        psql -h $HOST -U "$POSTGRES_USER" -d "$DB_NAME" -c "CREATE EXTENSION vector;"
        exit 1
    fi
fi

# Grant usage permissions on the vector extension to the user
print_status "Granting vector extension usage permissions..."
psql -h $HOST -U "$POSTGRES_USER" -d "$DB_NAME" -c "GRANT USAGE ON SCHEMA public TO $DB_USER;" > /dev/null 2>&1
psql -h $HOST -U "$POSTGRES_USER" -d "$DB_NAME" -c "GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO $DB_USER;" > /dev/null 2>&1
psql -h $HOST -U "$POSTGRES_USER" -d "$DB_NAME" -c "ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT EXECUTE ON FUNCTIONS TO $DB_USER;" > /dev/null 2>&1
print_success "Vector extension permissions granted"

# Test connection
print_status "Testing connection..."
if psql -h $HOST -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1;" > /dev/null 2>&1; then
    print_success "Connection test successful"
else
    print_error "Connection test failed"
    exit 1
fi

# Test extension creation capability
print_status "Testing extension creation capability..."
if psql -h $HOST -U "$DB_USER" -d "$DB_NAME" -c "SELECT has_database_privilege('$DB_NAME', 'CREATE');" | grep -q t; then
    print_success "User has CREATE privilege on database"
else
    print_warning "User may not have sufficient privileges for extension creation"
fi

# Verify schema ownership
print_status "Verifying schema ownership..."
if psql -h $HOST -U "$POSTGRES_USER" -d "$DB_NAME" -c "SELECT nspowner::regrole FROM pg_namespace WHERE nspname = 'public';" | grep -q "$DB_USER"; then
    print_success "Public schema ownership transferred to $DB_USER"
else
    print_error "Failed to transfer public schema ownership to $DB_USER"
    exit 1
fi

# Test table creation capability
print_status "Testing table creation capability..."
if psql -h $HOST -U "$DB_USER" -d "$DB_NAME" -c "CREATE TABLE test_permissions (id SERIAL PRIMARY KEY); DROP TABLE test_permissions;" > /dev/null 2>&1; then
    print_success "User can create tables in public schema"
else
    print_error "User cannot create tables in public schema"
    exit 1
fi

# Test vector extension usage capability
print_status "Testing vector extension usage capability..."
if psql -h $HOST -U "$DB_USER" -d "$DB_NAME" -c "SELECT vector_dims('[1,2,3]'::vector);" > /dev/null 2>&1; then
    print_success "User can use vector extension functions"
else
    print_warning "User cannot use vector extension functions - this may cause issues with RAG CLI"
fi



# Create configuration example
print_status "Creating configuration example..."
CONFIG_EXAMPLE="# RAG CLI Database Configuration Example
# Copy this to ~/.rag-cli/config.yaml

database:
  host: $HOST
  port: 5432
  name: $DB_NAME
  user: $DB_USER
  password: $DB_PASSWORD
  ssl_mode: prefer

ollama:
  host: $HOST
  port: 11434
  tls: false
  chat_model: qwen3:4b
  embedding_model: dengcao/Qwen3-Embedding-0.6B:Q8_0

embedding:
  chunk_size: 1000
  chunk_overlap: 200
  similarity_threshold: 0.7
  max_results: 10

general:
  log_level: info
  data_dir: ~/.rag-cli/data"

echo "$CONFIG_EXAMPLE" > ragcli-config-example.yaml

print_success "Database setup completed successfully!"
echo ""
print_status "Next steps:"
echo "1. Copy the configuration example:"
echo "   cp ragcli-config-example.yaml ~/.rag-cli/config.yaml"
echo ""
echo "2. Validate the configuration:"
echo "   rag-cli config validate"
echo ""
echo "3. Create your first collection:"
echo "   rag-cli collection create my-docs -d \"My documentation\" -f ./docs"
echo ""
print_warning "Remember to secure your database password and use SSL in production environments."
echo ""
print_status "Note: This setup requires the pgvector extension for vector similarity search."
echo "If you encounter extension-related errors, ensure pgvector is installed:"
echo "  brew install pgvector  # macOS"
echo "  sudo apt install postgresql-vector  # Ubuntu/Debian" 