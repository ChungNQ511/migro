# 🚀 Migro - Database Migration Tool

A powerful and user-friendly database migration tool built in Go, designed to simplify PostgreSQL database schema management with support for advanced features like automatic rollback handling, missing migration recovery, and comprehensive table operations.

## 📖 Documentation

- **[BUILD.md](BUILD.md)** - Complete build, release, and development guide
- **[Installation](#-installation)** - Multiple installation methods
- **[Quick Start](#-quick-start)** - Get started in minutes
- **[Usage Examples](#-usage)** - Comprehensive command examples

## ✨ Features

### 🔧 Core Migration Features
- **Create Migrations**: Generate timestamped migration files with enhanced templates
- **Run Migrations**: Execute migrations up to the latest version with intelligent error handling
- **Rollback Support**: Rollback specific count or all migrations with safety prompts
- **Status Tracking**: View current migration status and applied migrations
- **Missing Migration Recovery**: Automatically handle missing migration files during operations

### 🗃️ Table Management
- **Create Tables**: Generate complete table creation migrations with primary keys and timestamps
- **Add Columns**: Add single or multiple columns with full type and constraint support  
- **Delete Columns**: Remove columns with intelligent rollback that preserves original definitions
- **Read Table Schema**: Inspect table column information
- **Reset Sequences**: Automatically reset table sequences to current max values

### 💾 CRUD Operations
- **Insert Data**: Add records to tables with automatic timestamp handling
- **Update Data**: Modify existing records with automatic `updated_at` timestamps
- **Select One**: Query single records with column selection and filtering
- **Select Many**: Query multiple records with limit, ordering, and pagination
- **Soft Delete**: Safe record deletion using `deleted_at` timestamp (preserves data)
- **Query Preview**: Shows actual SQL and parameters before execution
- **Formatted Results**: Display query results in readable table format

### 🛡️ Advanced Features
- **Type Safety**: Full Go type checking and error handling
- **Database Validation**: Check table and column existence before operations
- **Configuration Management**: YAML-based configuration with environment support
- **Cross-platform**: Works on any OS without shell dependencies
- **Temporary File Management**: Smart handling of missing migrations with cleanup options

## 📦 Installation

### Prerequisites
- PostgreSQL database
- [Goose](https://github.com/pressly/goose) migration tool

### Install Goose
```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
```

## 🚀 Installation Methods

### Method 1: One-line Install (Recommended)
```bash
# Install latest release automatically
curl -sSL https://raw.githubusercontent.com/ChungNQ511/migro/main/install.sh | bash
```

### Method 2: Using Makefile (Development)
```bash
# Clone repository
git clone https://github.com/ChungNQ511/migro.git
cd migro

# Show all available commands
make help

# Install to system PATH (requires sudo)
make install

# Install to user PATH (~/.local/bin) - no sudo required
make install-user

# Build only (binary in build/ directory)
make build
```

### Method 3: Using Go Install
```bash
# Requires Go 1.19+
go install github.com/ChungNQ511/migro@latest
```

### Method 4: Manual Download
1. Go to [Releases](https://github.com/ChungNQ511/migro/releases)
2. Download binary for your platform:
   - `migro-linux-amd64` (Linux)
   - `migro-darwin-amd64` (macOS Intel)
   - `migro-darwin-arm64` (macOS Apple Silicon)
   - `migro-windows-amd64.exe` (Windows)
3. Rename to `migro` and make executable:
   ```bash
   chmod +x migro
   sudo mv migro /usr/local/bin/
   ```

### Method 5: Docker
```bash
# Build Docker image
docker build -t migro .

# Run with Docker
docker run --rm -v $(pwd):/workspace migro --help

# Using docker-compose (includes PostgreSQL)
docker-compose up -d postgres  # Start database
docker-compose run migro --help # Run migro commands
```

### Method 6: Development Setup
```bash
# Clone and build from source
git clone https://github.com/ChungNQ511/migro.git
cd migro

# Using Makefile
make deps      # Download dependencies
make build     # Build binary
make run       # Build and run

# Or using Go directly
go build -o migro .
./migro --help
```

## 🎯 Quick Start

After installation, set up your first project:

```bash
# 1. Initialize config
cp migro.example.yaml migro.yaml
# or using Makefile
make setup-example

# 2. Edit migro.yaml with your database credentials
vim migro.yaml

# 3. Test connection
migro status

# 4. Create your first migration
migro create-migration --name="init_database"

# 5. Run migrations
migro migrate
```

## ⚙️ Configuration

Create a `migro.yaml` file in your project root:

```yaml
ENV: "development"
DATABASE_DRIVER: "postgres"
DATABASE_HOST: "localhost"
DATABASE_PORT: "5432"
DATABASE_USERNAME: "your_username"
DATABASE_PASSWORD: "your_password"
DATABASE_NAME: "your_database"
TIMEOUT_SECONDS: 30
MIGRATION_DIR: "./db/migrations"
QUERY_DIR: "./db/queries"
```

The `DATABASE_CONNECTION_STRING` is automatically built from the above parameters.

**Quick Setup:**
```bash
# Copy the example config and edit with your credentials
cp migro.example.yaml migro.yaml
# Edit migro.yaml with your database settings
```

The tool automatically loads `migro.yaml` from the current directory. If you need a different config file, use the `--config` flag.

## 🚀 Usage

### Basic Commands

```bash
# Show all available commands
./migro --help

# Show migration status (auto-loads migro.yaml)
./migro status

# Create a new empty migration
./migro create-migration --name="add_user_preferences"

# Run all pending migrations
./migro migrate

# Rollback last 2 migrations
./migro rollback --count=2

# Rollback all migrations (with confirmation)
./migro rollback-all

# Use custom config file if needed
./migro status --config=production.yaml
```

### Table Operations

#### Create Table
```bash
# Create a simple table
./migro create-table \
  --table=users \
  --columns="name:varchar:not_null,email:varchar:unique,age:int:default=0"

# Create table with complex columns
./migro create-table \
  --table=products \
  --columns="name:varchar:not_null,price:decimal:check=price>0,tags:varchar:array,active:bool:default=true"
```

**Generated SQL:**
```sql
CREATE TABLE IF NOT EXISTS users(
    user_id serial primary key,
    name VARCHAR NOT NULL,
    email VARCHAR UNIQUE,
    age INTEGER DEFAULT 0,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP,
    deleted_at timestamp
);
```

#### Add Columns
```bash
# Add single column
./migro add-column \
  --table=users \
  --columns="phone:varchar"

# Add multiple columns with options
./migro add-column \
  --table=users \
  --columns="preferences:jsonb:default='{}',tags:varchar:array,is_verified:bool:default=false:not_null"
```

**Generated SQL:**
```sql
-- Up Migration
ALTER TABLE users ADD COLUMN IF NOT EXISTS phone VARCHAR;
ALTER TABLE users ADD COLUMN IF NOT EXISTS preferences JSONB DEFAULT '{}';
ALTER TABLE users ADD COLUMN IF NOT EXISTS tags VARCHAR[] DEFAULT ARRAY[]::VARCHAR[];
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_verified BOOLEAN DEFAULT false NOT NULL;

-- Down Migration (automatically generated)
ALTER TABLE users DROP COLUMN IF EXISTS phone;
ALTER TABLE users DROP COLUMN IF EXISTS preferences;
ALTER TABLE users DROP COLUMN IF EXISTS tags;
ALTER TABLE users DROP COLUMN IF EXISTS is_verified;
```

#### Delete Columns
```bash
# Delete single column
./migro delete-column \
  --table=users \
  --columns="phone"

# Delete multiple columns
./migro delete-column \
  --table=users \
  --columns="temp_field,old_status,deprecated_column"
```

**Generated SQL:**
```sql
-- Up Migration
ALTER TABLE users DROP COLUMN IF EXISTS phone;
ALTER TABLE users DROP COLUMN IF EXISTS temp_field;

-- Down Migration (with full column definitions from database)
ALTER TABLE users ADD COLUMN IF NOT EXISTS phone VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS temp_field VARCHAR(50) DEFAULT 'test';
```

### Schema Inspection

```bash
# Read table columns
./migro read-table --table=users

# Reset table sequence
./migro reset --table=users
```

## 💾 CRUD Operations

Migro includes built-in CRUD (Create, Read, Update, Delete) operations for basic data management:

### Insert Data
```bash
# Insert a single record
./migro insert \
  --table=users \
  --data="name=John Doe,email=john@example.com,age=25"

# Insert with special characters (use quotes)
./migro insert \
  --table=users \
  --data="name='John O''Brien',email=john@example.com,status=active"
```

**Example Output:**
```
🔄 Executing: INSERT INTO users (name, email, age) VALUES ($1, $2, $3) RETURNING *
📝 Values: [John Doe john@example.com 25]
✅ Insert successful!

user_id         | name           | email          | age            | created_at     
----------------|----------------|----------------|----------------|----------------
1               | John Doe       | john@example...| 25             | 2025-01-15 ...
```

### Update Data
```bash
# Update record by ID
./migro update \
  --table=users \
  --data="name=Jane Doe,age=26" \
  --where="user_id=1"

# Update by email
./migro update \
  --table=users \
  --data="status=inactive" \
  --where="email=john@example.com"
```

**Example Output:**
```
🔄 Executing: UPDATE users SET name = $1, age = $2, updated_at = $3 WHERE user_id = $4 RETURNING *
📝 Values: [Jane Doe 26 2025-01-15 14:30:45 +0000 UTC 1]
✅ Update successful!
```

### Select One Record
```bash
# Select all columns from one record
./migro select-one \
  --table=users \
  --where="user_id=1"

# Select specific columns
./migro select-one \
  --table=users \
  --columns="name,email" \
  --where="email=jane@example.com"
```

### Select Multiple Records
```bash
# Select all records (with automatic limit)
./migro select-many \
  --table=users

# Select with WHERE condition
./migro select-many \
  --table=users \
  --where="age=25" \
  --limit=50

# Select specific columns with custom limit
./migro select-many \
  --table=users \
  --columns="name,email,created_at" \
  --where="status=active" \
  --limit=20
```

**Example Output:**
```
🔄 Executing: SELECT name, email, created_at FROM users WHERE status = $1 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT 20
📝 Values: [active]
✅ Records found:

name           | email          | created_at     
---------------|----------------|----------------
Jane Doe       | jane@example...| 2025-01-15 ...
John Smith     | john.smith@... | 2025-01-15 ...

📊 Total records: 2 (showing max 20)
```

### Soft Delete
```bash
# Soft delete by ID (sets deleted_at timestamp)
./migro delete \
  --table=users \
  --where="user_id=1"

# Soft delete by condition
./migro delete \
  --table=users \
  --where="email=old@example.com"
```

**Example Output:**
```
🔄 Executing soft delete: UPDATE users SET deleted_at = $1, updated_at = $2 WHERE user_id = $3 AND deleted_at IS NULL RETURNING *
📝 Values: [2025-01-15 14:35:10 +0000 UTC 2025-01-15 14:35:10 +0000 UTC 1]
✅ Soft delete successful!
```

### CRUD Data Format

**Data Format**: Use `column=value` pairs separated by commas:
```bash
# Simple values
--data="name=John,age=25,active=true"

# Values with quotes (for strings with spaces/special chars)
--data="name='John Doe',description='A user with description'"

# Boolean and numeric values
--data="age=25,salary=50000.50,is_admin=false"
```

**WHERE Format**: Simple equality conditions:
```bash
# Numeric comparison
--where="user_id=1"

# String comparison (quotes optional for simple strings)
--where="email=john@example.com"
--where="name='John Doe'"

# Boolean comparison
--where="active=true"
```

### CRUD Features

#### Safety Features
- ✅ **Table Validation**: Checks table exists in migration files before operations
- ✅ **Soft Delete**: Delete operations set `deleted_at` timestamp (preserves data)
- ✅ **Auto Timestamps**: Updates `updated_at` automatically on modifications
- ✅ **Query Preview**: Shows actual SQL query and parameters before execution
- ✅ **Result Display**: Formats query results in readable table format

#### Automatic Columns
- 🕒 **created_at**: Auto-populated on INSERT (if column exists)
- 🕒 **updated_at**: Auto-updated on UPDATE operations
- 🗑️ **deleted_at**: Set by soft delete operations
- 🔑 **Primary Key**: Auto-incremented (typically `{table}_id`)

#### Current Limitations
- 📝 **WHERE Clauses**: Currently supports simple `column=value` conditions
- 📝 **Data Types**: Basic type inference (more complex types planned)
- 📝 **Joins**: Single table operations only

#### Future Enhancements
- 🔮 **Complex WHERE**: Support for `AND`, `OR`, `>`, `<`, `LIKE` conditions
- 🔮 **Bulk Operations**: Insert/update multiple records at once
- 🔮 **JSON Operations**: Advanced JSONB column manipulation
- 🔮 **Export/Import**: CSV/JSON data import/export functionality

## 📝 Column Type Specification

### Supported Types
```
varchar/string    → VARCHAR
int/integer       → INTEGER  
bigint           → BIGINT
bool/boolean     → BOOLEAN
float            → FLOAT
double           → DOUBLE PRECISION
decimal/numeric  → NUMERIC
text             → TEXT
json             → JSON
jsonb            → JSONB
uuid             → UUID
date             → DATE
timestamp        → TIMESTAMP
datetime         → TIMESTAMP
timestamptz      → TIMESTAMP WITH TIME ZONE
```

### Column Options
```
not_null         → NOT NULL
unique           → UNIQUE
default=value    → DEFAULT value
check=condition  → CHECK(condition)
array            → TYPE[]
```

### Examples
```bash
# String column with default
"name:varchar:not_null:default='Anonymous'"

# Integer with check constraint
"age:int:not_null:check=age>=0"

# Array column with default
"tags:varchar:array:default='{}'"

# JSON column with default object
"settings:jsonb:default='{}':not_null"

# Decimal with precision
"price:decimal:not_null:check=price>0"
```

## 🔄 Migration Workflow

### Development Workflow
1. **Create Migration**: Use `create-migration` for custom SQL or table commands for schema changes
2. **Review Generated SQL**: Check the generated migration files before applying
3. **Run Migration**: Use `migrate` to apply changes
4. **Check Status**: Use `status` to verify applied migrations
5. **Rollback if Needed**: Use `rollback` to undo changes during development

### Production Workflow
1. **Test Locally**: Run all migrations in development environment
2. **Review Changes**: Ensure rollback migrations are correct
3. **Backup Database**: Always backup before production migrations
4. **Apply Migrations**: Run `migrate` in production
5. **Verify**: Use `status` and application testing to verify success

## 🛡️ Safety Features

### Automatic Validations
- ✅ **Table Existence**: Verifies tables exist before column operations
- ✅ **Column Existence**: Checks columns exist before deletion
- ✅ **Type Validation**: Validates column types against supported list
- ✅ **Duplicate Prevention**: Prevents creating duplicate migration files

### Rollback Safety
- ✅ **Full Column Definitions**: Delete operations preserve complete column info for rollback
- ✅ **Confirmation Prompts**: `rollback-all` requires user confirmation
- ✅ **Temporary File Handling**: Smart management of missing migration files
- ✅ **Database State Checking**: Validates database state before operations

### Error Recovery
- 🔧 **Missing Migration Recovery**: Automatically creates temporary files for missing migrations
- 🔧 **Rollback Retry Logic**: Handles complex rollback scenarios with multiple attempts
- 🔧 **Cleanup Options**: Offers to clean up temporary files after operations

## 📊 Migration Status

The `status` command shows:
- ✅ Applied migrations with timestamps
- ⏳ Pending migrations 
- ❌ Missing migration files
- 🔢 Current database version

Example output:
```
📊 Current migration status:
    Applied At                  Migration
    =======================================
    2025-01-15 10:30:45 UTC -- 20250115103045_create_users_table.sql
    2025-01-15 11:15:20 UTC -- 20250115111520_add_user_preferences.sql
    Pending                 -- 20250115120000_add_user_roles.sql
```

## 🔧 Advanced Configuration

### Config File Priority
The tool looks for config files in this order:
1. `--config` flag (if specified)
2. `migro.yaml` (in current directory)
3. `migro.yml` (in current directory)
4. `config.yaml` (in current directory)
5. `config.yml` (in current directory)

### Environment Variables
You can override configuration values using environment variables:
```bash
export MIGRO_CONFIG="./production.yaml"
export DATABASE_HOST="prod-db.example.com" 
export DATABASE_PASSWORD="secure-password"
./migro migrate
```

### Multiple Environments
```bash
# Development (auto-loads migro.yaml)
./migro migrate

# Production with custom config
./migro migrate --config=production.yaml

# Staging
./migro migrate --config=staging.yaml
```

### Custom Migration Directory
```yaml
MIGRATION_DIR: "./database/migrations"
QUERY_DIR: "./database/queries"
```

## 🤝 Contributing

For detailed build and release instructions, see [BUILD.md](BUILD.md).

### Development Setup

```bash
# Clone repository
git clone https://github.com/ChungNQ511/migro.git
cd migro

# Install dependencies
make deps

# Build and test locally
make build
make test

# Run in development mode
make dev
```

### Release Process

See [BUILD.md](BUILD.md) for complete build and release documentation including:
- Multi-platform builds
- Release package creation
- Version management
- GitHub release process

### Contributing Guidelines

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📜 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Goose](https://github.com/pressly/goose) - The underlying migration engine
- [pgx](https://github.com/jackc/pgx) - PostgreSQL driver for Go
- [CLI](https://github.com/urfave/cli) - Command line interface framework
- [Viper](https://github.com/spf13/viper) - Configuration management

## 📞 Support

If you encounter any issues or have questions:
1. Check the [GitHub Issues](https://github.com/ChungNQ511/migro/issues)
2. Create a new issue with detailed description
3. Include your configuration and error logs

---

Made with ❤️ in Go | Built for PostgreSQL | Optimized for Developer Experience 