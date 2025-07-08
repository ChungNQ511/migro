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
		config.DATABASE_DRIVER,
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

// Parse migration files to extract table and column information
// @param migrationDir string
// @return map[string][]string, error (table -> columns)
func parseMigrationFiles(migrationDir string) (map[string][]string, error) {
	tableColumns := make(map[string][]string)

	pattern := fmt.Sprintf("%s/[0-9]*.sql", migrationDir)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob migration files: %w", err)
	}

	for _, file := range matches {
		err := parseMigrationFile(file, tableColumns)
		if err != nil {
			// Continue parsing other files even if one fails
			fmt.Printf("⚠️  Warning: Failed to parse %s: %v\n", file, err)
		}
	}

	return tableColumns, nil
}

// Parse a single migration file
// @param filePath string
// @param tableColumns map[string][]string
// @return error
func parseMigrationFile(filePath string, tableColumns map[string][]string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	fileContent := string(content)

	// Extract only the -- +goose Up section
	upContent := extractGooseUpContent(fileContent)

	// Parse CREATE TABLE statements
	parseCreateTableStatements(upContent, tableColumns)

	// Parse ADD COLUMN statements
	parseAddColumnStatements(upContent, tableColumns)

	// Parse DROP COLUMN statements (remove columns)
	parseDropColumnStatements(upContent, tableColumns)

	return nil
}

// Extract content from -- +goose Up section
// @param content string
// @return string
func extractGooseUpContent(content string) string {
	// Find -- +goose Up section
	upStart := strings.Index(content, "-- +goose Up")
	if upStart == -1 {
		return ""
	}

	// Find -- +goose Down section
	downStart := strings.Index(content, "-- +goose Down")
	if downStart == -1 {
		return content[upStart:]
	}

	return content[upStart:downStart]
}

// Parse CREATE TABLE statements
// @param content string
// @param tableColumns map[string][]string
func parseCreateTableStatements(content string, tableColumns map[string][]string) {
	// Regex to match CREATE TABLE statements
	re := regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(\w+)\s*\(([^;]+)\)`)
	matches := re.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			tableName := strings.TrimSpace(match[1])
			columnsStr := match[2]

			columns := parseColumnDefinitions(columnsStr)
			if len(columns) > 0 {
				tableColumns[tableName] = columns
			}
		}
	}
}

// Parse ADD COLUMN statements
// @param content string
// @param tableColumns map[string][]string
func parseAddColumnStatements(content string, tableColumns map[string][]string) {
	// Regex to match ALTER TABLE ADD COLUMN statements
	re := regexp.MustCompile(`(?i)ALTER\s+TABLE\s+(\w+)\s+ADD\s+COLUMN\s+(?:IF\s+NOT\s+EXISTS\s+)?(\w+)`)
	matches := re.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			tableName := strings.TrimSpace(match[1])
			columnName := strings.TrimSpace(match[2])

			if _, exists := tableColumns[tableName]; !exists {
				tableColumns[tableName] = []string{}
			}

			// Add column if not already exists
			if !contains(tableColumns[tableName], columnName) {
				tableColumns[tableName] = append(tableColumns[tableName], columnName)
			}
		}
	}
}

// Parse DROP COLUMN statements
// @param content string
// @param tableColumns map[string][]string
func parseDropColumnStatements(content string, tableColumns map[string][]string) {
	// Regex to match ALTER TABLE DROP COLUMN statements
	re := regexp.MustCompile(`(?i)ALTER\s+TABLE\s+(\w+)\s+DROP\s+COLUMN\s+(?:IF\s+EXISTS\s+)?(\w+)`)
	matches := re.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			tableName := strings.TrimSpace(match[1])
			columnName := strings.TrimSpace(match[2])

			if columns, exists := tableColumns[tableName]; exists {
				// Remove column from list
				for i, col := range columns {
					if col == columnName {
						tableColumns[tableName] = append(columns[:i], columns[i+1:]...)
						break
					}
				}
			}
		}
	}
}

// Parse column definitions from CREATE TABLE
// @param columnsStr string
// @return []string
func parseColumnDefinitions(columnsStr string) []string {
	var columns []string

	// Split by comma, but be careful with nested parentheses
	lines := strings.Split(columnsStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}

		// Remove trailing comma
		line = strings.TrimSuffix(line, ",")

		// Extract column name (first word)
		parts := strings.Fields(line)
		if len(parts) > 0 {
			columnName := strings.TrimSpace(parts[0])
			if columnName != "" && !strings.HasPrefix(columnName, "CONSTRAINT") && !strings.HasPrefix(columnName, "PRIMARY") && !strings.HasPrefix(columnName, "FOREIGN") {
				columns = append(columns, columnName)
			}
		}
	}

	return columns
}

// Check if table exists in migration files
// @param migrationDir string
// @param tableName string
// @return bool, error
func checkTableExistsInMigrations(migrationDir, tableName string) (bool, error) {
	tableColumns, err := parseMigrationFiles(migrationDir)
	if err != nil {
		return false, err
	}

	_, exists := tableColumns[tableName]
	return exists, nil
}

// Check if column exists in migration files
// @param migrationDir string
// @param tableName string
// @param columnName string
// @return bool, error
func checkColumnExistsInMigrations(migrationDir, tableName, columnName string) (bool, error) {
	tableColumns, err := parseMigrationFiles(migrationDir)
	if err != nil {
		return false, err
	}

	columns, tableExists := tableColumns[tableName]
	if !tableExists {
		return false, nil
	}

	return contains(columns, columnName), nil
}

// Helper function to check if slice contains string
// @param slice []string
// @param item string
// @return bool
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
