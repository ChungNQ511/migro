package migroCMD

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func MigrateUp(config *CONFIG, db *pgxpool.Pool) error {
	fmt.Println("🚀 Starting database migration up...")

	// Clear temp files tracker
	createdTempFiles = []string{}

	// First attempt - run migration normally
	output, err := executeGoose(config, MigrationScriptUp)

	if err != nil {
		// Check if it's a missing migration error
		if strings.Contains(string(output), "missing migration") && strings.Contains(string(output), "before current version") {
			fmt.Println("🔧 Found missing migration, handling...")

			// Handle missing migration ONCE
			handled := handleMissingMigrationForMigrate(config, string(output))
			if !handled {
				return fmt.Errorf("failed to handle missing migration: %w\nOutput: %s", err, string(output))
			}

			fmt.Println("✅ Created temp files, running migration again...")

			output, err = executeGoose(config, MigrationScriptUp)

			if err != nil {
				return fmt.Errorf("migration up failed even after creating temp files: %w\nOutput: %s", err, string(output))
			}
		} else {
			// Other error, not related to missing migration
			return fmt.Errorf("migration up failed: %w\nOutput: %s", err, string(output))
		}
	}

	// Success
	fmt.Println("✅ Migration up completed successfully!")
	if len(strings.TrimSpace(string(output))) > 0 {
		fmt.Println("📋 Migration output:")
		fmt.Print(string(output))
	}

	// Show migration status
	fmt.Println("\n📊 Current migration status:")
	err = showMigrationStatus(config)
	if err != nil {
		fmt.Printf("⚠️  Failed to show migration status: %v\n", err)
	}

	// Auto cleanup temp files if any were created
	if len(createdTempFiles) > 0 {
		fmt.Println("\n🧹 Auto cleanup temp migration files...")
		fmt.Printf("🔍 Tìm thấy %d temp files được tạo trong quá trình migration\n", len(createdTempFiles))
		cleanupTempFiles()
	} else {
		// Also check for any existing temp files
		fmt.Println("\n🧹 Auto cleanup temp migration files...")
		err = autoCleanupTempFiles(config)
		if err != nil {
			fmt.Printf("⚠️  Failed to cleanup temp files: %v\n", err)
		}
	}

	return nil
}

// Auto cleanup temp files without user interaction (for automated workflows)
func autoCleanupTempFiles(config *CONFIG) error {
	// Look for files with temp_ prefix in their name
	pattern := fmt.Sprintf("%s/*_temp_*.sql", config.MIGRATION_DIR)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to find temp files: %w", err)
	}

	if len(matches) == 0 {
		return nil
	}

	removedCount := 0
	for _, file := range matches {
		err := os.Remove(file)
		if err != nil {
			fmt.Printf("⚠️  Failed to remove %s: %v\n", file, err)
		} else {
			removedCount++
		}
	}

	if removedCount > 0 {
		fmt.Printf("✅ Auto cleaned up %d/%d temp files\n", removedCount, len(matches))
	}

	return nil
}

// Handle missing migration specifically for migrate up operation
// Returns true if handled successfully, false otherwise
// @param config *CONFIG
// @param errorOutput string
// @return bool
func handleMissingMigrationForMigrate(config *CONFIG, errorOutput string) bool {
	fmt.Println("🔍 Analyzing missing migration error...")

	// First try to extract directly from error output (most accurate)
	version, migrationName := extractMissingMigrationInfo(config.MIGRATION_DIR, errorOutput)

	var missingVersions []int64
	var migrationNames []string

	if version > 0 {
		fmt.Printf("🎯 Extracted from error: version=%d, name=%s\n", version, migrationName)
		missingVersions = []int64{version}
		migrationNames = []string{migrationName}
	} else {
		// Fallback to database query
		fmt.Println("⚠️  Could not extract from error, trying database query...")
		dbVersions, err := findMissingMigrations(config)
		if err != nil {
			fmt.Printf("❌ Database query also failed: %v\n", err)
			return false
		}

		if len(dbVersions) == 0 {
			fmt.Println("❌ No missing migrations found")
			return false
		}

		missingVersions = dbVersions
		// Use default names for database-sourced versions
		migrationNames = make([]string, len(dbVersions))
		for i := range migrationNames {
			migrationNames[i] = "temp_migration"
		}
	}

	fmt.Printf("🎯 Found %d missing migration(s): %v\n", len(missingVersions), missingVersions)

	// Check if migration files exist before creating temp files
	var existingFiles []string
	var needTempFiles []int64
	var needTempNames []string

	for i, version := range missingVersions {
		migrationName := migrationNames[i]

		// Check if actual migration file exists
		actualFileName := fmt.Sprintf("db/migrations/%d_%s.sql", version, migrationName)
		if _, err := os.Stat(actualFileName); err == nil {
			existingFiles = append(existingFiles, actualFileName)
		} else {
			// File doesn't exist, need temp file
			needTempFiles = append(needTempFiles, version)
			needTempNames = append(needTempNames, migrationName)
		}
	}

	// If some files exist, suggest rollback instead
	if len(existingFiles) > 0 {
		fmt.Printf("🚨 MIGRATION FILES ĐÃ TỒN TẠI:\n")
		for _, file := range existingFiles {
			fmt.Printf("   📁 %s\n", file)
		}

		// Calculate rollback count
		rollbackCount, err := calculateRollbackCount(config, missingVersions)
		if err != nil {
			fmt.Printf("⚠️  Không thể tính rollback count: %v\n", err)
			fmt.Println("💡 Gợi ý: Bạn cần ROLLBACK về version missing thay vì migrate up!")
			fmt.Println("   Chạy: go run cmd/cmd.go rollback [count]")
		} else {
			fmt.Printf("💡 GỢI Ý ROLLBACK:\n")
			fmt.Printf("   Cần rollback %d migration(s) để về version missing\n", rollbackCount)
			fmt.Printf("   Chạy: go run cmd/cmd.go rollback %d\n", rollbackCount)
		}

		return false // Don't create temp files, user should rollback instead
	}

	// If no files exist, create temp files as before
	if len(needTempFiles) == 0 {
		fmt.Println("✅ Tất cả missing migrations đã có file")
		return true
	}

	fmt.Printf("📝 Tạo temp files cho %d missing migration(s)...\n", len(needTempFiles))

	// Create temp files for missing migrations (only if not already created)
	createdCount := 0
	for i, version := range needTempFiles {
		migrationName := needTempNames[i]

		// Use the actual migration name if available
		var tempName string
		if migrationName != "temp_migration" {
			tempName = fmt.Sprintf("temp_%s", migrationName)
		} else {
			tempName = "temp_migration"
		}

		// Check if temp file already exists
		tempFileName := fmt.Sprintf("db/migrations/%d_%s.sql", version, tempName)
		if _, err := os.Stat(tempFileName); err == nil {
			fmt.Printf("⏭️  Temp file already exists: %s\n", tempFileName)
			continue // Skip if already exists
		}

		tempFile := createTempMigration(config.MIGRATION_DIR, version, tempName)
		if tempFile == "" {
			fmt.Printf("❌ Failed to create temp migration file for version %d\n", version)
			return false
		}

		fmt.Printf("📝 Created temp file: %s\n", tempFile)
		createdTempFiles = append(createdTempFiles, tempFile)
		createdCount++
	}

	if createdCount == 0 {
		fmt.Println("⚠️  No new temp files created (all already exist)")
		return false // Might be stuck in loop
	}

	fmt.Printf("✅ Successfully created %d temp migration files\n", createdCount)
	return true
}

// Find missing migrations by comparing database versions with local files
// @param config *CONFIG
// @return []int64, error
func findMissingMigrations(config *CONFIG) ([]int64, error) {
	ctx := context.Background()

	// Get all migration versions from database
	dbVersions, err := getDatabaseMigrationVersions(config, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database versions: %w", err)
	}

	// Get all local migration file versions
	localVersions, err := getLocalMigrationVersions(config.MIGRATION_DIR)
	if err != nil {
		return nil, fmt.Errorf("failed to get local versions: %w", err)
	}

	// Find versions that exist in database but not in local files
	var missingVersions []int64
	for _, dbVersion := range dbVersions {
		found := false
		for _, localVersion := range localVersions {
			if dbVersion == localVersion {
				found = true
				break
			}
		}
		if !found {
			missingVersions = append(missingVersions, dbVersion)
		}
	}

	return missingVersions, nil
}

// Get migration versions from database using goose
func getDatabaseMigrationVersions(config *CONFIG, ctx context.Context) ([]int64, error) {
	output, err := executeGoose(config, MigrationScriptStatus)

	if err != nil {
		return nil, fmt.Errorf("goose status failed: %w\nOutput: %s", err, string(output))
	}

	return parseVersionsFromStatus(string(output)), nil
}

// Parse versions from goose status output
// @param output string
// @return []int64
func parseVersionsFromStatus(output string) []int64 {
	var versions []int64
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Look for lines with timestamp and migration version
		// Format: "2025-07-07 17:17:59 UTC -- 20250707100918_create_table.sql"
		re := regexp.MustCompile(`(\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}\s+\w+\s+--\s+)?(\d{14})_.*\.sql`)
		matches := re.FindStringSubmatch(line)

		if len(matches) > 2 {
			version, err := strconv.ParseInt(matches[2], 10, 64)
			if err == nil {
				versions = append(versions, version)
			}
		}
	}

	return versions
}

// Generate SQLC code from database
// @param config *CONFIG
// @return error
func GenerateSQLC(config *CONFIG) error {
	// Check if sqlc.yaml exists, create if not
	sqlcConfigPath := filepath.Join(config.MIGRATION_DIR, "sqlc.yaml")
	if _, err := os.Stat(sqlcConfigPath); os.IsNotExist(err) {
		fmt.Println("📝 sqlc.yaml not found, creating default configuration...")
		err := createDefaultSQLCConfig(config)
		if err != nil {
			return fmt.Errorf("failed to create sqlc.yaml: %w", err)
		}
		fmt.Printf("✅ Created %s\n", sqlcConfigPath)
	}

	// Check if sqlc command exists
	if !commandExists("sqlc") {
		return fmt.Errorf("❌ sqlc command not found. Install it first:\n" +
			"   # Using Go:\n" +
			"   go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest\n" +
			"   \n" +
			"   # Using Homebrew (macOS):\n" +
			"   brew install sqlc\n" +
			"   \n" +
			"   # Or download from: https://docs.sqlc.dev/en/latest/overview/install.html")
	}

	// Run sqlc generate
	fmt.Println("🔄 Generating SQLC code...")
	cmd := exec.Command("sqlc", "generate", "-f", "sqlc.yaml")
	cmd.Dir = config.MIGRATION_DIR
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to generate SQLC code: %w\n"+
			"💡 Make sure:\n"+
			"   1. Your database is running and migrated\n"+
			"   2. sqlc.yaml configuration is correct\n"+
			"   3. Query directory exists with .sql files", err)
	}

	fmt.Println("✅ SQLC code generated successfully!")
	return nil
}

// Initialize SQLC configuration
// @param config *CONFIG
// @return error
func InitSQLC(config *CONFIG) error {
	sqlcConfigPath := filepath.Join(config.MIGRATION_DIR, "sqlc.yaml")

	// Check if sqlc.yaml already exists
	if _, err := os.Stat(sqlcConfigPath); err == nil {
		fmt.Printf("⚠️  sqlc.yaml already exists at: %s\n", sqlcConfigPath)
		fmt.Print("Do you want to overwrite it? (y/N): ")

		var response string
		fmt.Scanln(&response)

		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("❌ Cancelled. sqlc.yaml not modified.")
			return nil
		}
	}

	err := createDefaultSQLCConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create sqlc.yaml: %w", err)
	}

	fmt.Printf("✅ Created sqlc.yaml at: %s\n", sqlcConfigPath)
	fmt.Println("📝 Next steps:")
	fmt.Println("   1. Create queries in your query directory")
	fmt.Println("   2. Run: migro sqlc")
	fmt.Println("   3. Check generated Go code")

	return nil
}

// Create default SQLC configuration file
// @param config *CONFIG
// @return error
func createDefaultSQLCConfig(config *CONFIG) error {
	// Ensure migration directory exists
	if err := os.MkdirAll(config.MIGRATION_DIR, 0755); err != nil {
		return fmt.Errorf("failed to create migration directory: %w", err)
	}

	// Ensure query directory exists
	queryDir := config.QUERY_DIR
	if queryDir == "" {
		queryDir = filepath.Join(config.MIGRATION_DIR, "queries")
	}
	if err := os.MkdirAll(queryDir, 0755); err != nil {
		return fmt.Errorf("failed to create query directory: %w", err)
	}

	// Create relative paths
	relativeQueryDir, err := filepath.Rel(config.MIGRATION_DIR, queryDir)
	if err != nil {
		relativeQueryDir = queryDir
	}

	// Default SQLC configuration
	sqlcConfig := fmt.Sprintf(`version: "2"
sql:
  - engine: "postgresql"
    queries: "%s"
    schema: "."
    gen:
      go:
        package: "db"
        out: "../internal/db"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_prepared_queries: false
        emit_interface: true
        emit_empty_slices: true
        emit_all_enum_values: true
        overrides:
          - db_type: "timestamptz"
            go_type: "time.Time"
          - db_type: "uuid"
            go_type: "github.com/google/uuid.UUID"
`, relativeQueryDir)

	// Write sqlc.yaml
	sqlcConfigPath := filepath.Join(config.MIGRATION_DIR, "sqlc.yaml")
	err = os.WriteFile(sqlcConfigPath, []byte(sqlcConfig), 0644)
	if err != nil {
		return fmt.Errorf("failed to write sqlc.yaml: %w", err)
	}

	// Create example query file if queries directory is empty
	exampleQueryPath := filepath.Join(queryDir, "example.sql")
	if _, err := os.Stat(exampleQueryPath); os.IsNotExist(err) {
		exampleQuery := `-- Example query file
-- name: GetUser :one
SELECT * FROM users WHERE user_id = $1 AND deleted_at IS NULL;

-- name: ListUsers :many
SELECT * FROM users WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT $1;

-- name: CreateUser :one
INSERT INTO users (name, email) VALUES ($1, $2) RETURNING *;

-- name: UpdateUser :one
UPDATE users SET name = $1, email = $2, updated_at = CURRENT_TIMESTAMP 
WHERE user_id = $3 AND deleted_at IS NULL RETURNING *;

-- name: DeleteUser :exec
UPDATE users SET deleted_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
WHERE user_id = $1 AND deleted_at IS NULL;
`
		err = os.WriteFile(exampleQueryPath, []byte(exampleQuery), 0644)
		if err != nil {
			fmt.Printf("⚠️  Warning: Could not create example query file: %v\n", err)
		} else {
			fmt.Printf("📝 Created example query file: %s\n", exampleQueryPath)
		}
	}

	return nil
}

// Check if a command exists in PATH
// @param cmd string
// @return bool
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
