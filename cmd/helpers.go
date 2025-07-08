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
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Extract missing migration version and name from goose error
// @param output string
// @return int64, string
func extractMissingMigrationInfo(migrationDir, output string) (int64, string) {
	// New pattern for goose output: "version 20250702043810: %s/20250702043810_migration_name.sql"
	re := regexp.MustCompile(fmt.Sprintf(`version\s+(\d+):\s+%s/\d+_([^\.]+)\.sql`, migrationDir))
	matches := re.FindStringSubmatch(output)

	if len(matches) > 2 {
		version, err := strconv.ParseInt(matches[1], 10, 64)
		if err == nil {
			migrationName := matches[2]
			return version, migrationName
		}
	}

	// Fallback pattern: "missing migration: 20240101123456"
	re2 := regexp.MustCompile(`missing migration[s]?:?\s*(\d+)(?:_([^,\s\n]+))?`)
	matches2 := re2.FindStringSubmatch(output)

	if len(matches2) > 1 {
		version, err := strconv.ParseInt(matches2[1], 10, 64)
		if err == nil {
			name := "temp_migration"
			if len(matches2) > 2 && matches2[2] != "" {
				name = matches2[2]
			}
			return version, name
		}
	}

	// Alternative pattern: "found 1 missing migrations before current version 20240101123456"
	re3 := regexp.MustCompile(`version\s+(\d+)`)
	matches3 := re3.FindStringSubmatch(output)
	if len(matches3) > 1 {
		version, err := strconv.ParseInt(matches3[1], 10, 64)
		if err == nil {
			return version, "temp_migration"
		}
	}

	return 0, ""
}

// Create temporary migration file
// @param version int64
// @param name string
// @return string
func createTempMigration(migrationDir string, version int64, name string) string {
	// Use consistent naming pattern for temp files
	if name == "temp_migration" || name == "" {
		name = "temp_migration"
	} else {
		name = fmt.Sprintf("temp_%s", name)
	}

	filename := fmt.Sprintf("%s/%d_%s.sql", migrationDir, version, name)

	// Check if file already exists
	if _, err := os.Stat(filename); err == nil {
		return filename // File already exists
	}

	content := fmt.Sprintf(`-- +goose Up
-- +goose StatementBegin
-- Temp migration created automatically for rollback
-- Version: %d
-- Name: %s
-- Created: %s
-- NOTE: This is a temporary file created to handle missing migrations
-- You can safely delete this file after rollback is complete
SELECT 1; -- Placeholder
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Temp migration rollback
SELECT 1; -- Placeholder
-- +goose StatementEnd
`, version, name, time.Now().Format("2006-01-02 15:04:05"))

	err := os.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		fmt.Printf("❌ Failed to create temp file %s: %v\n", filename, err)
		return ""
	}

	return filename
}

func showMigrationStatus(config *CONFIG) error {
	ctx := context.Background()

	fmt.Println("\n📊 Current migration status:")

	cmd := exec.CommandContext(ctx, "goose",
		"-dir", config.MIGRATION_DIR,
		config.DATABASE_CONNECTION_STRING,
		"status")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get migration status: %w\nOutput: %s", err, string(output))
	}

	fmt.Print(string(output))
	return nil
}

// ShowMigrationStatus - public wrapper for showing migration status
func ShowMigrationStatus(config *CONFIG) error {
	return showMigrationStatus(config)
}

// Extract created file name from goose create output
// @param migrationDir string
// @param output string
// @return string
func extractCreatedFileName(migrationDir, output string) string {
	// Pattern: "Created new file: db/migrations/20250707181234_migration_name.sql"
	re := regexp.MustCompile(fmt.Sprintf(`Created new file:\s+(%s/\d+_[^\.]+\.sql)`, migrationDir))
	matches := re.FindStringSubmatch(output)

	if len(matches) > 1 {
		return matches[1]
	}

	return ""
}

// Enhance migration template with better comments and examples
// @param filePath string
// @param migrationName string
// @return error
func enhanceMigrationTemplate(filePath, migrationName string) error {
	// Create enhanced template
	enhancedContent := fmt.Sprintf(`-- +goose Up
-- +goose StatementBegin

-- Migration: %s
-- Created: %s
-- Description: Add your migration description here

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Rollback migration: %s
-- Description: Add rollback description here
-- +goose StatementEnd
`, migrationName, time.Now().Format("2006-01-02 15:04:05"), migrationName)

	// Write enhanced content
	err := os.WriteFile(filePath, []byte(enhancedContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write enhanced template: %w", err)
	}

	fmt.Println("✨ Enhanced migration template with examples and comments")
	return nil
}

// rename column type format
// @param column: string, format: "column_name:column_type,column_name:column_type,..."
// @return string
func renameColumnTypeEnhanceFormat(column string) string {
	parts := strings.Split(column, ",")
	columnTypes := make([]string, len(parts))
	for i, part := range parts {
		subParts := strings.SplitN(part, ":", 3)
		columnName := subParts[0]
		columnType := ""
		options := ""
		if len(subParts) > 1 {
			columnType = strings.TrimSpace(strings.ToLower(subParts[1]))
			if mappedType, ok := columnTypeMap[columnType]; ok {
				columnType = mappedType
			}
		}
		if len(subParts) > 2 {
			options = subParts[2]
			columnTypes[i] = fmt.Sprintf("%s:%s:%s", columnName, columnType, options)
		} else {
			columnTypes[i] = fmt.Sprintf("%s:%s", columnName, columnType)
		}
	}
	return strings.Join(columnTypes, ",")
}

// Check Table Exists
// @param db: *pgxpool.Pool
// @param table: string
// @return bool, error
func checkTableExists(db *pgxpool.Pool, table string) (bool, error) {
	ctx := context.Background()
	var exists bool
	err := db.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = $1)", table).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("❌ error checking table exists: %w", err)
	}
	return exists, nil
}

// Check Column Exists
// @param db: *pgxpool.Pool
// @param table: string
// @param column: string
// @return bool, error
func checkColumnExists(db *pgxpool.Pool, table string, column string) (bool, error) {
	ctx := context.Background()
	var exists bool
	err := db.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = $1 AND column_name = $2)", table, column).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("❌ error checking column exists: %w", err)
	}
	return exists, nil
}

type MigrationScript string

const (
	MigrationScriptUp     MigrationScript = "up"
	MigrationScriptDown   MigrationScript = "down"
	MigrationScriptStatus MigrationScript = "status"
)

// run migration
// @param config *CONFIG
// @param script MigrationScript
// @return []byte, error
func executeGoose(config *CONFIG, script MigrationScript) ([]byte, error) {
	ctx := context.Background()

	cmd := exec.CommandContext(ctx, "goose",
		"-dir", config.MIGRATION_DIR,
		config.DATABASE_CONNECTION_STRING,
		string(script))

	return cmd.CombinedOutput()
}

// Calculate how many migrations need to be rolled back to reach the missing version
// @param config *CONFIG
// @param missingVersions []int64
// @return int, error
func calculateRollbackCount(config *CONFIG, missingVersions []int64) (int, error) {
	// Get current applied migrations
	output, err := executeGoose(config, MigrationScriptStatus)
	if err != nil {
		return 0, fmt.Errorf("failed to get migration status: %w", err)
	}

	// Parse applied migration versions
	appliedVersions := parseVersionsFromStatus(string(output))
	if len(appliedVersions) == 0 {
		return 0, fmt.Errorf("no applied migrations found")
	}

	// Find the earliest missing version
	earliestMissing := missingVersions[0]
	for _, version := range missingVersions {
		if version < earliestMissing {
			earliestMissing = version
		}
	}

	// Count how many applied migrations are after the earliest missing version
	rollbackCount := 0
	for _, appliedVersion := range appliedVersions {
		if appliedVersion > earliestMissing {
			rollbackCount++
		}
	}

	return rollbackCount, nil
}

// Get migration versions from local files
// @param migrationDir string
// @return []int64, error
func getLocalMigrationVersions(migrationDir string) ([]int64, error) {
	pattern := fmt.Sprintf("%s/[0-9]*.sql", migrationDir)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob migration files: %w", err)
	}

	var versions []int64
	re := regexp.MustCompile(`(\d{14})_.*\.sql`)

	for _, file := range matches {
		filename := filepath.Base(file)
		matches := re.FindStringSubmatch(filename)
		if len(matches) > 1 {
			version, err := strconv.ParseInt(matches[1], 10, 64)
			if err == nil {
				versions = append(versions, version)
			}
		}
	}

	return versions, nil
}
