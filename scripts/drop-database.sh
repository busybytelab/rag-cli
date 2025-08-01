#!/bin/bash

# RAG CLI Database Drop Script
# This script safely drops the existing PostgreSQL database and user for RAG CLI

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
POSTGRES_USER="postgres"

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
    echo "  -s, --postgres-user USER PostgreSQL superuser (auto-detected, can override)"
    echo "  -h, --help            Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                                    # Use defaults with auto-detected superuser"
    echo "  $0 -H localhost -n mydb -u myuser    # Custom host, database and user"
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

# Check if psql is available
if ! command -v psql &> /dev/null; then
    print_error "psql command not found. Please install PostgreSQL client tools."
    exit 1
fi

# Confirm before dropping
echo ""
print_warning "This will permanently delete the database '$DB_NAME' and user '$DB_USER'"
echo -n "Are you sure you want to continue? (yes/no): "
read -r confirm

if [ "$confirm" != "yes" ]; then
    print_status "Operation cancelled"
    exit 0
fi

print_status "Dropping database '$DB_NAME' and user '$DB_USER'..."

# Drop database if it exists
print_status "Dropping database..."
if psql -h $HOST -U "$POSTGRES_USER" -d postgres -c "SELECT 1 FROM pg_database WHERE datname = '$DB_NAME';" | grep -q 1; then
    # Terminate all connections to the database first
    psql -h $HOST -U "$POSTGRES_USER" -d postgres -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '$DB_NAME' AND pid <> pg_backend_pid();" > /dev/null 2>&1
    
    # Drop the database
    psql -h $HOST -U "$POSTGRES_USER" -d postgres -c "DROP DATABASE IF EXISTS $DB_NAME;" > /dev/null 2>&1
    print_success "Database '$DB_NAME' dropped"
else
    print_warning "Database '$DB_NAME' does not exist"
fi

# Drop user if it exists
print_status "Dropping user..."
if psql -h $HOST -U "$POSTGRES_USER" -d postgres -c "SELECT 1 FROM pg_roles WHERE rolname = '$DB_USER';" | grep -q 1; then
    psql -h $HOST -U "$POSTGRES_USER" -d postgres -c "DROP USER IF EXISTS $DB_USER;" > /dev/null 2>&1
    print_success "User '$DB_USER' dropped"
else
    print_warning "User '$DB_USER' does not exist"
fi

print_success "Database and user cleanup completed successfully!"
echo ""
print_status "You can now run the setup script to recreate them:"
echo "  ./scripts/setup-database.sh" 