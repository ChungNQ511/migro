# 🚀 Migro - Database Migration Tool

A powerful and user-friendly database migration tool built in Go, designed to simplify PostgreSQL database schema management with support for advanced features like automatic rollback handling, missing migration recovery, and comprehensive table operations.

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

### 🛡️ Advanced Features
- **Type Safety**: Full Go type checking and error handling
- **Database Validation**: Check table and column existence before operations
- **Configuration Management**: YAML-based configuration with environment support
- **Cross-platform**: Works on any OS without shell dependencies
- **Temporary File Management**: Smart handling of missing migrations with cleanup options

## 📦 Installation

### Prerequisites
- Go 1.19+ 
- PostgreSQL database
- [Goose](https://github.com/pressly/goose) migration tool

### Install Goose
```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
```

### Build Migro
```bash
git clone https://github.com/ChungNQ511/migro.git
cd migro
go build -o migro .
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