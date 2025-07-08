package main

// import (
// 	"bufio"
// 	"context"
// 	"fmt"
// 	"os"
// 	"os/exec"
// 	"path/filepath"
// 	"regexp"
// 	"strconv"
// 	"strings"
// 	"time"

// 	"github.com/jackc/pgx/v5/pgxpool"
// )

// // private name scopes

// var createdTempFiles []string // Track temp files for cleanup

// // Perform single rollback with missing migration handling

// // Cleanup all temporary migration files (independent command)
// func CleanupAllTempFiles() error {
// 	// Look for files with temp_ prefix in their name
// 	pattern := "db/migrations/*_temp_*.sql"
// 	matches, err := filepath.Glob(pattern)
// 	if err != nil {
// 		return fmt.Errorf("failed to find temp files: %w", err)
// 	}

// 	if len(matches) == 0 {
// 		fmt.Println("📭 Không tìm thấy temp migration files nào")
// 		return nil
// 	}

// 	fmt.Printf("🔍 Tìm thấy %d temp migration files:\n", len(matches))
// 	for _, file := range matches {
// 		fmt.Printf("   - %s\n", file)
// 	}

// 	fmt.Print("\n❓ Bạn có muốn xóa tất cả temp files này không? (y/N): ")
// 	reader := bufio.NewReader(os.Stdin)
// 	confirm, _ := reader.ReadString('\n')
// 	confirm = strings.TrimSpace(strings.ToLower(confirm))

// 	if confirm != "y" && confirm != "yes" {
// 		fmt.Println("❌ Hủy cleanup temp files")
// 		return nil
// 	}

// 	removedCount := 0
// 	for _, file := range matches {
// 		err := os.Remove(file)
// 		if err != nil {
// 			fmt.Printf("⚠️  Failed to remove %s: %v\n", file, err)
// 		} else {
// 			fmt.Printf("🗑️  Removed: %s\n", file)
// 			removedCount++
// 		}
// 	}

// 	fmt.Printf("✅ Đã xóa %d/%d temp files\n", removedCount, len(matches))
// 	return nil
// }

// type MigrationScript string

// const (
// 	MigrationScriptUp   MigrationScript = "up"
// 	MigrationScriptDown MigrationScript = "down"
// )

// // run migration
// func runMigrationScript(config util.Config, db *pgxpool.Pool, script MigrationScript) ([]byte, error) {
// 	ctx := context.Background()

// 	cmd := exec.CommandContext(ctx, "goose",
// 		"-dir", "db/migrations",
// 		config.DATABASE_DRIVER,
// 		fmt.Sprintf("%s://%s:%s@%s:%s/%s?sslmode=disable",
// 			config.DATABASE_DRIVER,
// 			config.DATABASE_USERNAME,
// 			config.DATABASE_PASSWORD,
// 			config.DATABASE_HOST,
// 			config.DATABASE_PORT,
// 			config.DATABASE_NAME),
// 		string(script))

// 	return cmd.CombinedOutput()
// }

// // Migrate up and auto cleanup temp files
// // @param config: util.Config
// // @param db: *pgxpool.Pool

// 	// Success
// 	fmt.Println("✅ Migration up completed successfully!")
// 	if len(strings.TrimSpace(string(output))) > 0 {
// 		fmt.Println("📋 Migration output:")
// 		fmt.Print(string(output))
// 	}

// 	// Show migration status
// 	fmt.Println("\n📊 Current migration status:")
// 	err = showMigrationStatus(config)
// 	if err != nil {
// 		fmt.Printf("⚠️  Failed to show migration status: %v\n", err)
// 	}

// 	// Auto cleanup temp files if any were created
// 	if len(createdTempFiles) > 0 {
// 		fmt.Println("\n🧹 Auto cleanup temp migration files...")
// 		fmt.Printf("🔍 Tìm thấy %d temp files được tạo trong quá trình migration\n", len(createdTempFiles))
// 		cleanupTempFiles()
// 	} else {
// 		// Also check for any existing temp files
// 		fmt.Println("\n🧹 Auto cleanup temp migration files...")
// 		err = autoCleanupTempFiles()
// 		if err != nil {
// 			fmt.Printf("⚠️  Failed to cleanup temp files: %v\n", err)
// 		}
// 	}

// 	return nil
// }

// // Handle missing migration specifically for migrate up operation
// // Returns true if handled successfully, false otherwise
// func handleMissingMigrationForMigrate(config util.Config, errorOutput string) bool {
// 	fmt.Println("🔍 Analyzing missing migration error...")

// 	// First try to extract directly from error output (most accurate)
// 	version, migrationName := extractMissingMigrationInfo(errorOutput)

// 	var missingVersions []int64
// 	var migrationNames []string

// 	if version > 0 {
// 		fmt.Printf("🎯 Extracted from error: version=%d, name=%s\n", version, migrationName)
// 		missingVersions = []int64{version}
// 		migrationNames = []string{migrationName}
// 	} else {
// 		// Fallback to database query
// 		fmt.Println("⚠️  Could not extract from error, trying database query...")
// 		dbVersions, err := FindMissingMigrations(config)
// 		if err != nil {
// 			fmt.Printf("❌ Database query also failed: %v\n", err)
// 			return false
// 		}

// 		if len(dbVersions) == 0 {
// 			fmt.Println("❌ No missing migrations found")
// 			return false
// 		}

// 		missingVersions = dbVersions
// 		// Use default names for database-sourced versions
// 		migrationNames = make([]string, len(dbVersions))
// 		for i := range migrationNames {
// 			migrationNames[i] = "temp_migration"
// 		}
// 	}

// 	fmt.Printf("🎯 Found %d missing migration(s): %v\n", len(missingVersions), missingVersions)

// 	// Check if migration files exist before creating temp files
// 	var existingFiles []string
// 	var needTempFiles []int64
// 	var needTempNames []string

// 	for i, version := range missingVersions {
// 		migrationName := migrationNames[i]

// 		// Check if actual migration file exists
// 		actualFileName := fmt.Sprintf("db/migrations/%d_%s.sql", version, migrationName)
// 		if _, err := os.Stat(actualFileName); err == nil {
// 			existingFiles = append(existingFiles, actualFileName)
// 		} else {
// 			// File doesn't exist, need temp file
// 			needTempFiles = append(needTempFiles, version)
// 			needTempNames = append(needTempNames, migrationName)
// 		}
// 	}

// 	// If some files exist, suggest rollback instead
// 	if len(existingFiles) > 0 {
// 		fmt.Printf("🚨 MIGRATION FILES ĐÃ TỒN TẠI:\n")
// 		for _, file := range existingFiles {
// 			fmt.Printf("   📁 %s\n", file)
// 		}

// 		// Calculate rollback count
// 		rollbackCount, err := calculateRollbackCount(config, missingVersions)
// 		if err != nil {
// 			fmt.Printf("⚠️  Không thể tính rollback count: %v\n", err)
// 			fmt.Println("💡 Gợi ý: Bạn cần ROLLBACK về version missing thay vì migrate up!")
// 			fmt.Println("   Chạy: go run cmd/cmd.go rollback [count]")
// 		} else {
// 			fmt.Printf("💡 GỢI Ý ROLLBACK:\n")
// 			fmt.Printf("   Cần rollback %d migration(s) để về version missing\n", rollbackCount)
// 			fmt.Printf("   Chạy: go run cmd/cmd.go rollback %d\n", rollbackCount)
// 		}

// 		return false // Don't create temp files, user should rollback instead
// 	}

// 	// If no files exist, create temp files as before
// 	if len(needTempFiles) == 0 {
// 		fmt.Println("✅ Tất cả missing migrations đã có file")
// 		return true
// 	}

// 	fmt.Printf("📝 Tạo temp files cho %d missing migration(s)...\n", len(needTempFiles))

// 	// Create temp files for missing migrations (only if not already created)
// 	createdCount := 0
// 	for i, version := range needTempFiles {
// 		migrationName := needTempNames[i]

// 		// Use the actual migration name if available
// 		var tempName string
// 		if migrationName != "temp_migration" {
// 			tempName = fmt.Sprintf("temp_%s", migrationName)
// 		} else {
// 			tempName = "temp_migration"
// 		}

// 		// Check if temp file already exists
// 		tempFileName := fmt.Sprintf("db/migrations/%d_%s.sql", version, tempName)
// 		if _, err := os.Stat(tempFileName); err == nil {
// 			fmt.Printf("⏭️  Temp file already exists: %s\n", tempFileName)
// 			continue // Skip if already exists
// 		}

// 		tempFile := createTempMigration(version, tempName)
// 		if tempFile == "" {
// 			fmt.Printf("❌ Failed to create temp migration file for version %d\n", version)
// 			return false
// 		}

// 		fmt.Printf("📝 Created temp file: %s\n", tempFile)
// 		createdTempFiles = append(createdTempFiles, tempFile)
// 		createdCount++
// 	}

// 	if createdCount == 0 {
// 		fmt.Println("⚠️  No new temp files created (all already exist)")
// 		return false // Might be stuck in loop
// 	}

// 	fmt.Printf("✅ Successfully created %d temp migration files\n", createdCount)
// 	return true
// }

// // Calculate how many migrations need to be rolled back to reach the missing version
// func calculateRollbackCount(config util.Config, missingVersions []int64) (int, error) {
// 	// Get current applied migrations
// 	ctx := context.Background()
// 	cmd := exec.CommandContext(ctx, "goose",
// 		"-dir", "db/migrations",
// 		config.DATABASE_DRIVER,
// 		fmt.Sprintf("%s://%s:%s@%s:%s/%s?sslmode=disable",
// 			config.DATABASE_DRIVER,
// 			config.DATABASE_USERNAME,
// 			config.DATABASE_PASSWORD,
// 			config.DATABASE_HOST,
// 			config.DATABASE_PORT,
// 			config.DATABASE_NAME),
// 		"status")

// 	output, err := cmd.CombinedOutput()
// 	if err != nil {
// 		return 0, fmt.Errorf("failed to get migration status: %w", err)
// 	}

// 	// Parse applied migration versions
// 	appliedVersions := parseVersionsFromStatus(string(output))
// 	if len(appliedVersions) == 0 {
// 		return 0, fmt.Errorf("no applied migrations found")
// 	}

// 	// Find the earliest missing version
// 	earliestMissing := missingVersions[0]
// 	for _, version := range missingVersions {
// 		if version < earliestMissing {
// 			earliestMissing = version
// 		}
// 	}

// 	// Count how many applied migrations are after the earliest missing version
// 	rollbackCount := 0
// 	for _, appliedVersion := range appliedVersions {
// 		if appliedVersion > earliestMissing {
// 			rollbackCount++
// 		}
// 	}

// 	return rollbackCount, nil
// }

// // Find missing migrations by comparing database versions with local files
// func FindMissingMigrations(config util.Config) ([]int64, error) {
// 	ctx := context.Background()

// 	// Get all migration versions from database
// 	dbVersions, err := getDatabaseMigrationVersions(config, ctx)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get database versions: %w", err)
// 	}

// 	// Get all local migration file versions
// 	localVersions, err := getLocalMigrationVersions()
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get local versions: %w", err)
// 	}

// 	// Find versions that exist in database but not in local files
// 	var missingVersions []int64
// 	for _, dbVersion := range dbVersions {
// 		found := false
// 		for _, localVersion := range localVersions {
// 			if dbVersion == localVersion {
// 				found = true
// 				break
// 			}
// 		}
// 		if !found {
// 			missingVersions = append(missingVersions, dbVersion)
// 		}
// 	}

// 	return missingVersions, nil
// }

// // Get migration versions from database using goose
// func getDatabaseMigrationVersions(config util.Config, ctx context.Context) ([]int64, error) {
// 	cmd := exec.CommandContext(ctx, "goose",
// 		"-dir", "db/migrations",
// 		config.DATABASE_DRIVER,
// 		fmt.Sprintf("%s://%s:%s@%s:%s/%s?sslmode=disable",
// 			config.DATABASE_DRIVER,
// 			config.DATABASE_USERNAME,
// 			config.DATABASE_PASSWORD,
// 			config.DATABASE_HOST,
// 			config.DATABASE_PORT,
// 			config.DATABASE_NAME),
// 		"status")

// 	output, err := cmd.CombinedOutput()
// 	if err != nil {
// 		return nil, fmt.Errorf("goose status failed: %w\nOutput: %s", err, string(output))
// 	}

// 	return parseVersionsFromStatus(string(output)), nil
// }

// // Get migration versions from local files
// func getLocalMigrationVersions() ([]int64, error) {
// 	pattern := "db/migrations/[0-9]*.sql"
// 	matches, err := filepath.Glob(pattern)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to glob migration files: %w", err)
// 	}

// 	var versions []int64
// 	re := regexp.MustCompile(`(\d{14})_.*\.sql`)

// 	for _, file := range matches {
// 		filename := filepath.Base(file)
// 		matches := re.FindStringSubmatch(filename)
// 		if len(matches) > 1 {
// 			version, err := strconv.ParseInt(matches[1], 10, 64)
// 			if err == nil {
// 				versions = append(versions, version)
// 			}
// 		}
// 	}

// 	return versions, nil
// }

// // Parse versions from goose status output
// func parseVersionsFromStatus(output string) []int64 {
// 	var versions []int64
// 	lines := strings.Split(output, "\n")

// 	for _, line := range lines {
// 		line = strings.TrimSpace(line)
// 		// Look for lines with timestamp and migration version
// 		// Format: "2025-07-07 17:17:59 UTC -- 20250707100918_create_table.sql"
// 		re := regexp.MustCompile(`(\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}\s+\w+\s+--\s+)?(\d{14})_.*\.sql`)
// 		matches := re.FindStringSubmatch(line)

// 		if len(matches) > 2 {
// 			version, err := strconv.ParseInt(matches[2], 10, 64)
// 			if err == nil {
// 				versions = append(versions, version)
// 			}
// 		}
// 	}

// 	return versions
// }

// // Extract version from migrate-specific error patterns
// func extractVersionFromMigrateError(output string) int64 {
// 	// Pattern: "found 1 missing migrations before current version 20250707100918"
// 	re := regexp.MustCompile(`before current version\s+(\d+)`)
// 	matches := re.FindStringSubmatch(output)

// 	if len(matches) > 1 {
// 		version, err := strconv.ParseInt(matches[1], 10, 64)
// 		if err == nil {
// 			// The missing migration is likely the one before this version
// 			// We need to find what's actually missing by checking the database
// 			return version
// 		}
// 	}

// 	// Alternative: try to extract from "missing migrations: [numbers]"
// 	re2 := regexp.MustCompile(`missing migrations?:?\s*\[?(\d+)`)
// 	matches2 := re2.FindStringSubmatch(output)
// 	if len(matches2) > 1 {
// 		version, err := strconv.ParseInt(matches2[1], 10, 64)
// 		if err == nil {
// 			return version
// 		}
// 	}

// 	return 0
// }

// // Auto cleanup temp files without user interaction (for automated workflows)
// func autoCleanupTempFiles() error {
// 	// Look for files with temp_ prefix in their name
// 	pattern := "db/migrations/*_temp_*.sql"
// 	matches, err := filepath.Glob(pattern)
// 	if err != nil {
// 		return fmt.Errorf("failed to find temp files: %w", err)
// 	}

// 	if len(matches) == 0 {
// 		fmt.Println("📭 Không có temp migration files nào cần cleanup")
// 		return nil
// 	}

// 	fmt.Printf("🔍 Tìm thấy %d temp migration files, đang tự động xóa...\n", len(matches))

// 	removedCount := 0
// 	for _, file := range matches {
// 		err := os.Remove(file)
// 		if err != nil {
// 			fmt.Printf("⚠️  Failed to remove %s: %v\n", file, err)
// 		} else {
// 			fmt.Printf("🗑️  Removed: %s\n", file)
// 			removedCount++
// 		}
// 	}

// 	if removedCount > 0 {
// 		fmt.Printf("✅ Tự động xóa %d/%d temp files\n", removedCount, len(matches))
// 	}

// 	return nil
// }

// // Create empty migration file with goose template
// // @param name: string - migration name
// func CreateEmptyMigration(name string) error {
// 	ctx := context.Background()

// 	// Use goose create command to generate migration file
// 	cmd := exec.CommandContext(ctx, "goose",
// 		"-dir", "db/migrations",
// 		"create", name, "sql")

// 	output, err := cmd.CombinedOutput()
// 	if err != nil {
// 		return fmt.Errorf("❌ failed to create migration: %w\nOutput: %s", err, string(output))
// 	}

// 	fmt.Printf("📝 Goose output: %s", string(output))

// 	// Extract the created file name from goose output
// 	createdFile := extractCreatedFileName(string(output))
// 	if createdFile == "" {
// 		fmt.Println("✅ Migration file created successfully!")
// 		return nil
// 	}

// 	fmt.Printf("📁 Created file: %s\n", createdFile)

// 	// Enhance the template with more detailed comments
// 	err = enhanceMigrationTemplate(createdFile, name)
// 	if err != nil {
// 		fmt.Printf("⚠️  Warning: Could not enhance template: %v\n", err)
// 		// Don't fail the whole operation for template enhancement
// 	}

// 	return nil
// }

// // Extract created file name from goose create output
// func extractCreatedFileName(output string) string {
// 	// Pattern: "Created new file: db/migrations/20250707181234_migration_name.sql"
// 	re := regexp.MustCompile(`Created new file:\s+(db/migrations/\d+_[^\.]+\.sql)`)
// 	matches := re.FindStringSubmatch(output)

// 	if len(matches) > 1 {
// 		return matches[1]
// 	}

// 	return ""
// }

// // Enhance migration template with better comments and examples
// func enhanceMigrationTemplate(filePath, migrationName string) error {
// 	// Create enhanced template
// 	enhancedContent := fmt.Sprintf(`-- +goose Up
// -- +goose StatementBegin

// -- Migration: %s
// -- Created: %s
// -- Description: Add your migration description here

// -- Example SQL commands (uncomment and modify as needed):
// -- CREATE TABLE example (
// --     id SERIAL PRIMARY KEY,
// --     name VARCHAR(255) NOT NULL,
// --     created_at TIMESTAMP DEFAULT NOW()
// -- );

// -- ALTER TABLE existing_table ADD COLUMN new_column VARCHAR(255);
// -- CREATE INDEX idx_example_name ON example(name);

// -- +goose StatementEnd

// -- +goose Down
// -- +goose StatementBegin

// -- Rollback migration: %s
// -- Description: Add rollback description here

// -- Example rollback SQL commands (uncomment and modify as needed):
// -- DROP TABLE IF EXISTS example;
// -- ALTER TABLE existing_table DROP COLUMN IF EXISTS new_column;
// -- DROP INDEX IF EXISTS idx_example_name;

// -- +goose StatementEnd
// `, migrationName, time.Now().Format("2006-01-02 15:04:05"), migrationName)

// 	// Write enhanced content
// 	err := os.WriteFile(filePath, []byte(enhancedContent), 0644)
// 	if err != nil {
// 		return fmt.Errorf("failed to write enhanced template: %w", err)
// 	}

// 	fmt.Println("✨ Enhanced migration template with examples and comments")
// 	return nil
// }

// // Create only migration file
