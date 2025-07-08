package migroCMD

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Create empty migration file with goose template
// @param migrationDir string
// @param name string
// @return error
func CreateEmptyMigration(migrationDir, name string) error {
	ctx := context.Background()

	// Use goose create command to generate migration file
	cmd := exec.CommandContext(ctx, "goose",
		"-dir", migrationDir,
		"create", name, "sql")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("❌ failed to create migration: %w\nOutput: %s", err, string(output))
	}

	fmt.Printf("📝 Goose output: %s", string(output))

	// Extract the created file name from goose output
	createdFile := extractCreatedFileName(migrationDir, string(output))
	if createdFile == "" {
		fmt.Println("✅ Migration file created successfully!")
		return nil
	}

	fmt.Printf("📁 Created file: %s\n", createdFile)

	// Enhance the template with more detailed comments
	err = enhanceMigrationTemplate(createdFile, name)
	if err != nil {
		fmt.Printf("⚠️  Warning: Could not enhance template: %v\n", err)
		// Don't fail the whole operation for template enhancement
	}

	return nil
}

// Add Column to Table
// @param db: *pgxpool.Pool
// @param table: string
// @param column: string
func CreateTable(config *CONFIG, db *pgxpool.Pool, table string, columns string) error {
	// rename migration filename
	migrationFilename := fmt.Sprintf("create_%s", table)
	// check if any file with pattern 14-digit-number_table.sql exists
	matches, err := filepath.Glob(fmt.Sprintf("%s/[0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9]_%s.sql", config.MIGRATION_DIR, migrationFilename))
	if err != nil {
		return fmt.Errorf("❌ error checking migration files: %w", err)
	}
	if len(matches) > 0 {
		return fmt.Errorf("❌ migration file for %s already exists", table)
	}

	// validate column type
	valid, err := validateColumnType(columns)
	if !valid {
		return fmt.Errorf("❌ invalid column type: %w", err)
	}
	// rename column type format
	columns = renameColumnTypeEnhanceFormat(columns)

	// Create migration file using goose and then modify it
	err = createMigrationTableFile(config, migrationFilename, table, columns)
	if err != nil {
		return fmt.Errorf("❌ create migration failed: %w", err)
	}
	return nil
}

// createMigrationTableFile creates a migration file with table creation SQL
func createMigrationTableFile(config *CONFIG, migrationName, tableName, columns string) error {
	ctx := context.Background()
	migrationsDir := config.MIGRATION_DIR

	// Create empty migration file first
	cmd := exec.CommandContext(ctx, "goose", "-dir", migrationsDir, "create", migrationName, "sql")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create migration: %w\nOutput: %s", err, string(output))
	}

	// Find the latest migration file that was just created
	pattern := filepath.Join(migrationsDir, "[0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9]_"+migrationName+".sql")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("error finding migration file: %w", err)
	}

	if len(matches) == 0 {
		return fmt.Errorf("no migration file found after creation")
	}

	// Sort by modification time to get the latest file
	sort.Slice(matches, func(i, j int) bool {
		infoI, _ := os.Stat(matches[i])
		infoJ, _ := os.Stat(matches[j])
		return infoI.ModTime().After(infoJ.ModTime())
	})

	fileName := matches[0]

	// Generate the SQL content
	sqlContent, err := generateCreateTableSQL(tableName, columns)
	if err != nil {
		return fmt.Errorf("error generating SQL: %w", err)
	}

	// Write the SQL content to the file
	content := fmt.Sprintf(`-- +goose Up
-- +goose StatementBegin
%s
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS %s;
-- +goose StatementEnd
`, sqlContent, tableName)

	err = os.WriteFile(fileName, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("error writing migration file: %w", err)
	}

	fmt.Printf("✅ Created migration file: %s\n", fileName)
	return nil
}

// generateCreateTableSQL generates the CREATE TABLE SQL statement
func generateCreateTableSQL(tableName, columns string) (string, error) {
	// Get singular name for primary key
	singularName := getSingularName(tableName)
	primaryKey := singularName + "_id"

	var columnDefs []string

	// Add primary key
	columnDefs = append(columnDefs, fmt.Sprintf("    %s serial primary key", primaryKey))

	// Process column definitions
	if strings.TrimSpace(columns) != "" {
		columnLines := strings.Split(columns, ",")
		for _, column := range columnLines {
			column = strings.TrimSpace(column)
			if column == "" {
				continue
			}

			columnDef, err := parseColumnDefinition(column)
			if err != nil {
				return "", fmt.Errorf("error parsing column '%s': %w", column, err)
			}
			columnDefs = append(columnDefs, fmt.Sprintf("    %s", columnDef))
		}
	}

	// Add standard timestamp columns
	columnDefs = append(columnDefs, "    created_at timestamp DEFAULT CURRENT_TIMESTAMP")
	columnDefs = append(columnDefs, "    updated_at timestamp DEFAULT CURRENT_TIMESTAMP")
	columnDefs = append(columnDefs, "    deleted_at timestamp")

	// Build CREATE TABLE query
	sql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s(\n%s\n);", tableName, strings.Join(columnDefs, ",\n"))
	return sql, nil
}

// getSingularName converts plural table names to singular for primary key
func getSingularName(tableName string) string {
	if strings.HasSuffix(tableName, "ses") {
		return strings.TrimSuffix(tableName, "es")
	}
	if strings.HasSuffix(tableName, "s") {
		return strings.TrimSuffix(tableName, "s")
	}
	return tableName
}

// parseColumnDefinition parses a column definition string like "name:VARCHAR:not_null:default=test"
func parseColumnDefinition(column string) (string, error) {
	parts := strings.Split(column, ":")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid column format, expected name:type[:options...]")
	}

	columnName := strings.TrimSpace(parts[0])
	columnType := strings.TrimSpace(parts[1])

	var columnExtra strings.Builder
	isArray := false

	// Process options
	if len(parts) > 2 {
		for i := 2; i < len(parts); i++ {
			option := strings.TrimSpace(parts[i])
			if option == "" {
				continue
			}

			if option == "array" {
				columnType = columnType + "[]"
				isArray = true
			} else if strings.HasPrefix(option, "default=") {
				defaultVal := strings.TrimPrefix(option, "default=")
				defaultClause, err := formatDefaultValue(defaultVal, columnType, isArray)
				if err != nil {
					return "", err
				}
				columnExtra.WriteString(" " + defaultClause)
			} else if option == "not_null" {
				columnExtra.WriteString(" NOT NULL")
			} else if option == "unique" {
				columnExtra.WriteString(" UNIQUE")
			} else if strings.HasPrefix(option, "check=") {
				checkVal := strings.TrimPrefix(option, "check=")
				columnExtra.WriteString(fmt.Sprintf(" CHECK(%s)", checkVal))
			} else {
				// Any other option is added as-is
				columnExtra.WriteString(" " + option)
			}
		}
	}

	// Add default empty array for array types if no default was specified
	if isArray && !strings.Contains(columnExtra.String(), "DEFAULT") {
		columnExtra.WriteString(fmt.Sprintf(" DEFAULT ARRAY[]::%s", columnType))
	}

	return fmt.Sprintf("%s %s%s", columnName, columnType, columnExtra.String()), nil
}

// formatDefaultValue formats default values based on column type
func formatDefaultValue(defaultVal, columnType string, isArray bool) (string, error) {
	columnTypeLower := strings.ToLower(columnType)

	if isArray {
		// Handle array defaults
		if defaultVal == "{}" || defaultVal == "'{}'" {
			// Remove single quotes if present
			cleanVal := strings.Trim(defaultVal, "'")
			return fmt.Sprintf("DEFAULT '%s'::%s", cleanVal, columnType), nil
		}
		return fmt.Sprintf("DEFAULT %s", defaultVal), nil
	}

	// Handle string types
	if strings.Contains(columnTypeLower, "varchar") || strings.Contains(columnTypeLower, "text") {
		// Check if already quoted
		if strings.HasPrefix(defaultVal, "'") && strings.HasSuffix(defaultVal, "'") {
			return fmt.Sprintf("DEFAULT %s", defaultVal), nil
		}
		return fmt.Sprintf("DEFAULT '%s'", defaultVal), nil
	}

	// For other types, use as-is
	return fmt.Sprintf("DEFAULT %s", defaultVal), nil
}

// Validate column type
// @param column: string, format: "column_name:column_type,column_name:column_type,..."
// @return bool
func validateColumnType(column string) (bool, error) {
	parts := strings.Split(column, ",")
	for _, part := range parts {
		parts := strings.Split(part, ":")
		if len(parts) < 2 {
			return false, fmt.Errorf("invalid column type of column %s", parts[0])
		}
		columnType := strings.TrimSpace(strings.ToLower(parts[1]))
		if _, ok := columnTypeMap[columnType]; !ok {
			return false, fmt.Errorf("invalid column type of column %s", parts[0])
		}
	}
	return true, nil
}

var columnTypeMap = map[string]string{
	"varchar":     "VARCHAR",
	"string":      "VARCHAR",
	"int":         "INTEGER",
	"integer":     "INTEGER",
	"bigint":      "BIGINT",
	"bool":        "BOOLEAN",
	"boolean":     "BOOLEAN",
	"float":       "FLOAT",
	"double":      "DOUBLE PRECISION",
	"decimal":     "NUMERIC",
	"numeric":     "NUMERIC",
	"text":        "TEXT",
	"json":        "JSON",
	"jsonb":       "JSONB",
	"uuid":        "UUID",
	"date":        "DATE",
	"timestamp":   "TIMESTAMP",
	"datetime":    "TIMESTAMP",
	"timestamptz": "TIMESTAMP WITH TIME ZONE",
}

// Add Column to Table
// @param config: *CONFIG
// @param db: *pgxpool.Pool
// @param table: string
// @param columns: string
func AddColumn(config *CONFIG, db *pgxpool.Pool, table string, columns string) error {
	// get column names
	columnNames := strings.Split(columns, ",")

	var migrationFilename string
	if len(columnNames) == 1 {
		// Single column: add_column_{columnName}_to_{table}
		columnName := strings.TrimSpace(strings.Split(columnNames[0], ":")[0])
		migrationFilename = fmt.Sprintf("add_column_%s_to_%s", columnName, table)
	} else {
		// Multiple columns: add_columns_{col1}_{col2}_to_{table}
		// Extract column names and limit filename length
		var columnNamesForFile []string
		for _, col := range columnNames {
			colName := strings.TrimSpace(strings.Split(col, ":")[0])
			if colName != "" {
				columnNamesForFile = append(columnNamesForFile, colName)
			}
		}

		// Join column names with underscore
		columnsStr := strings.Join(columnNamesForFile, "_")

		// If filename would be too long, truncate and add hash
		maxLength := 80 // reasonable filename length
		baseFilename := fmt.Sprintf("add_columns_%s_to_%s", columnsStr, table)

		if len(baseFilename) > maxLength {
			// Create a shorter version with first few columns + hash
			shortColumnsStr := ""
			if len(columnNamesForFile) > 0 {
				shortColumnsStr = columnNamesForFile[0]
				if len(columnNamesForFile) > 1 {
					shortColumnsStr += "_and_more"
				}
			}

			// Simple hash of all column names
			hash := fmt.Sprintf("%x", len(columns)+len(strings.Join(columnNamesForFile, "")))[:6]
			migrationFilename = fmt.Sprintf("add_columns_%s_%s_to_%s", shortColumnsStr, hash, table)
		} else {
			migrationFilename = baseFilename
		}
	}

	// check if any file with pattern 14-digit-number_migration.sql exists
	matches, err := filepath.Glob(fmt.Sprintf("%s/[0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9]_%s.sql", config.MIGRATION_DIR, migrationFilename))
	if err != nil {
		return fmt.Errorf("❌ error checking migration files: %w", err)
	}
	if len(matches) > 0 {
		return fmt.Errorf("❌ migration file for %s already exists", migrationFilename)
	}

	// validate column type
	valid, err := validateColumnType(columns)
	if !valid {
		return fmt.Errorf("❌ invalid column type: %w", err)
	}

	// rename column type format
	columns = renameColumnTypeEnhanceFormat(columns)

	// Create migration file using goose and then modify it
	err = createMigrationAddColumnsFile(config, migrationFilename, table, columns)
	if err != nil {
		return fmt.Errorf("❌ create migration failed: %w", err)
	}
	return nil
}

// createMigrationAddColumnsFile creates a migration file with ALTER TABLE ADD COLUMN SQL
func createMigrationAddColumnsFile(config *CONFIG, migrationName, tableName, columns string) error {
	ctx := context.Background()
	migrationsDir := config.MIGRATION_DIR

	// Create empty migration file first
	cmd := exec.CommandContext(ctx, "goose", "-dir", migrationsDir, "create", migrationName, "sql")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create migration: %w\nOutput: %s", err, string(output))
	}

	// Find the latest migration file that was just created
	pattern := filepath.Join(migrationsDir, "[0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9]_"+migrationName+".sql")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("error finding migration file: %w", err)
	}

	if len(matches) == 0 {
		return fmt.Errorf("no migration file found after creation")
	}

	// Sort by modification time to get the latest file
	sort.Slice(matches, func(i, j int) bool {
		infoI, _ := os.Stat(matches[i])
		infoJ, _ := os.Stat(matches[j])
		return infoI.ModTime().After(infoJ.ModTime())
	})

	fileName := matches[0]

	// Generate the SQL content
	upSQL, downSQL, err := generateAddColumnsSQL(tableName, columns)
	if err != nil {
		return fmt.Errorf("error generating SQL: %w", err)
	}

	// Write the SQL content to the file
	content := fmt.Sprintf(`-- +goose Up
-- +goose StatementBegin
%s
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
%s
-- +goose StatementEnd
`, upSQL, downSQL)

	err = os.WriteFile(fileName, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("error writing migration file: %w", err)
	}

	fmt.Printf("✅ Created migration file: %s\n", fileName)
	return nil
}

// generateAddColumnsSQL generates ALTER TABLE ADD COLUMN and DROP COLUMN SQL statements
func generateAddColumnsSQL(tableName, columns string) (string, string, error) {
	var upStatements []string
	var downStatements []string

	// Process column definitions
	columnLines := strings.Split(columns, ",")
	for _, column := range columnLines {
		column = strings.TrimSpace(column)
		if column == "" {
			continue
		}

		columnDef, err := parseColumnDefinitionForAlter(column)
		if err != nil {
			return "", "", fmt.Errorf("error parsing column '%s': %w", column, err)
		}

		// Extract column name for DROP statement
		parts := strings.Split(column, ":")
		if len(parts) < 1 {
			return "", "", fmt.Errorf("invalid column format: %s", column)
		}
		columnName := strings.TrimSpace(parts[0])

		upStatements = append(upStatements, fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS %s;", tableName, columnDef))
		downStatements = append(downStatements, fmt.Sprintf("ALTER TABLE %s DROP COLUMN IF EXISTS %s;", tableName, columnName))
	}

	return strings.Join(upStatements, "\n"), strings.Join(downStatements, "\n"), nil
}

// parseColumnDefinitionForAlter parses a column definition for ALTER TABLE ADD COLUMN
func parseColumnDefinitionForAlter(column string) (string, error) {
	parts := strings.Split(column, ":")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid column format, expected name:type[:options...]")
	}

	columnName := strings.TrimSpace(parts[0])
	columnType := strings.TrimSpace(parts[1])

	var columnExtra strings.Builder
	isArray := false

	// Process options
	if len(parts) > 2 {
		for i := 2; i < len(parts); i++ {
			option := strings.TrimSpace(parts[i])
			if option == "" {
				continue
			}

			if option == "array" {
				columnType = columnType + "[]"
				isArray = true
			} else if strings.HasPrefix(option, "default=") {
				defaultVal := strings.TrimPrefix(option, "default=")
				defaultClause, err := formatDefaultValueForAlter(defaultVal, columnType, isArray)
				if err != nil {
					return "", err
				}
				columnExtra.WriteString(" " + defaultClause)
			} else if option == "not_null" {
				columnExtra.WriteString(" NOT NULL")
			} else if option == "unique" {
				columnExtra.WriteString(" UNIQUE")
			} else if strings.HasPrefix(option, "check=") {
				checkVal := strings.TrimPrefix(option, "check=")
				columnExtra.WriteString(fmt.Sprintf(" CHECK(%s)", checkVal))
			} else if option != "array" {
				// Any other option is added as-is
				columnExtra.WriteString(" " + option)
			}
		}
	}

	// Add default empty array for array types if no default was specified
	if isArray && !strings.Contains(columnExtra.String(), "DEFAULT") {
		columnExtra.WriteString(fmt.Sprintf(" DEFAULT ARRAY[]::%s", columnType))
	}

	return fmt.Sprintf("%s %s%s", columnName, columnType, columnExtra.String()), nil
}

// formatDefaultValueForAlter formats default values for ALTER TABLE context
func formatDefaultValueForAlter(defaultVal, columnType string, isArray bool) (string, error) {
	columnTypeLower := strings.ToLower(columnType)

	if isArray {
		// Handle array defaults
		if defaultVal == "{}" || defaultVal == "'{}'" {
			// Remove single quotes if present
			cleanVal := strings.Trim(defaultVal, "'")
			return fmt.Sprintf("DEFAULT '%s'::%s", cleanVal, columnType), nil
		}
		return fmt.Sprintf("DEFAULT %s", defaultVal), nil
	}

	// Handle string types
	if strings.Contains(columnTypeLower, "varchar") || strings.Contains(columnTypeLower, "text") {
		// Check if already quoted
		if strings.HasPrefix(defaultVal, "'") && strings.HasSuffix(defaultVal, "'") {
			return fmt.Sprintf("DEFAULT %s", defaultVal), nil
		}
		return fmt.Sprintf("DEFAULT '%s'", defaultVal), nil
	}

	// For other types, use as-is
	return fmt.Sprintf("DEFAULT %s", defaultVal), nil
}

// getColumnDefinition gets the full definition of a column for rollback purposes
func getColumnDefinition(db *pgxpool.Pool, tableName, columnName string) (string, error) {
	ctx := context.Background()
	query := `
		SELECT 
			column_name,
			data_type,
			character_maximum_length,
			is_nullable,
			column_default
		FROM information_schema.columns 
		WHERE table_name = $1 AND column_name = $2
	`

	var colName, dataType string
	var maxLength *int
	var isNullable, columnDefault *string

	err := db.QueryRow(ctx, query, tableName, columnName).Scan(&colName, &dataType, &maxLength, &isNullable, &columnDefault)
	if err != nil {
		return "", fmt.Errorf("error getting column definition: %w", err)
	}

	// Build column definition string
	var definition string = fmt.Sprintf("%s %s", colName, strings.ToUpper(dataType))

	// Add length for varchar types
	if maxLength != nil && (dataType == "character varying" || dataType == "varchar") {
		definition += fmt.Sprintf("(%d)", *maxLength)
	}

	// Add NOT NULL constraint
	if isNullable != nil && *isNullable == "NO" {
		definition += " NOT NULL"
	}

	// Add default value
	if columnDefault != nil && *columnDefault != "" {
		definition += fmt.Sprintf(" DEFAULT %s", *columnDefault)
	}

	return definition, nil
}

// Delete Column from Table
// @param config: *CONFIG
// @param db: *pgxpool.Pool
// @param table: string
// @param columns: string
func DeleteColumn(config *CONFIG, db *pgxpool.Pool, table string, columns string) error {
	// check table exists
	exists, err := checkTableExists(db, table)
	if err != nil {
		return fmt.Errorf("❌ error checking table exists: %w", err)
	}
	if !exists {
		return fmt.Errorf("❌ table '%s' does not exist", table)
	}

	// get column names
	columnNames := strings.Split(columns, ",")

	var migrationFilename string
	if len(columnNames) == 1 {
		columnName := strings.TrimSpace(strings.Split(columnNames[0], ":")[0])
		// check column exists
		exists, err := checkColumnExists(db, table, columnName)
		if err != nil {
			return fmt.Errorf("❌ error checking column exists: %w", err)
		}
		if !exists {
			return fmt.Errorf("❌ column '%s' does not exist in table '%s'", columnName, table)
		}
		migrationFilename = fmt.Sprintf("delete_column_%s_from_%s", columnName, table)
	} else {
		// Multiple columns: delete_columns_{col1}_{col2}_from_{table}
		// Extract column names and limit filename length
		var columnNamesForFile []string
		for _, col := range columnNames {
			colName := strings.TrimSpace(strings.Split(col, ":")[0])
			if colName != "" {
				// check column exists
				exists, err := checkColumnExists(db, table, colName)
				if err != nil {
					return fmt.Errorf("❌ error checking column exists: %w", err)
				}
				if !exists {
					return fmt.Errorf("❌ column '%s' does not exist in table '%s'", colName, table)
				}
				columnNamesForFile = append(columnNamesForFile, colName)
			}
		}

		// Join column names with underscore
		columnsStr := strings.Join(columnNamesForFile, "_")

		// If filename would be too long, truncate and add hash
		maxLength := 80 // reasonable filename length
		baseFilename := fmt.Sprintf("delete_columns_%s_from_%s", columnsStr, table)

		if len(baseFilename) > maxLength {
			// Create a shorter version with first few columns + hash
			shortColumnsStr := ""
			if len(columnNamesForFile) > 0 {
				shortColumnsStr = columnNamesForFile[0]
				if len(columnNamesForFile) > 1 {
					shortColumnsStr += "_and_more"
				}
			}

			// Simple hash of all column names
			hash := fmt.Sprintf("%x", len(columns)+len(strings.Join(columnNamesForFile, "")))[:6]
			migrationFilename = fmt.Sprintf("delete_columns_%s_%s_from_%s", shortColumnsStr, hash, table)
		} else {
			migrationFilename = baseFilename
		}
	}

	// check if migration file already exists
	matches, err := filepath.Glob(fmt.Sprintf("%s/[0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9]_%s.sql", config.MIGRATION_DIR, migrationFilename))
	if err != nil {
		return fmt.Errorf("❌ error checking migration files: %w", err)
	}
	if len(matches) > 0 {
		return fmt.Errorf("❌ migration file for %s already exists", migrationFilename)
	}

	// Create migration file using goose and then modify it
	err = createMigrationDeleteColumnsFile(config, db, migrationFilename, table, columns)
	if err != nil {
		return fmt.Errorf("❌ create migration failed: %w", err)
	}
	return nil
}

// createMigrationDeleteColumnsFile creates a migration file with ALTER TABLE DROP COLUMN SQL
func createMigrationDeleteColumnsFile(config *CONFIG, db *pgxpool.Pool, migrationName, tableName, columns string) error {
	ctx := context.Background()
	migrationsDir := config.MIGRATION_DIR

	// Create empty migration file first
	cmd := exec.CommandContext(ctx, "goose", "-dir", migrationsDir, "create", migrationName, "sql")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create migration: %w\nOutput: %s", err, string(output))
	}

	// Find the latest migration file that was just created
	pattern := filepath.Join(migrationsDir, "[0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9]_"+migrationName+".sql")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("error finding migration file: %w", err)
	}

	if len(matches) == 0 {
		return fmt.Errorf("no migration file found after creation")
	}

	// Sort by modification time to get the latest file
	sort.Slice(matches, func(i, j int) bool {
		infoI, _ := os.Stat(matches[i])
		infoJ, _ := os.Stat(matches[j])
		return infoI.ModTime().After(infoJ.ModTime())
	})

	fileName := matches[0]

	// Generate the SQL content
	upSQL, downSQL, err := generateDeleteColumnsSQL(db, tableName, columns)
	if err != nil {
		return fmt.Errorf("error generating SQL: %w", err)
	}

	// Write the SQL content to the file
	content := fmt.Sprintf(`-- +goose Up
-- +goose StatementBegin
%s
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
%s
-- +goose StatementEnd
`, upSQL, downSQL)

	err = os.WriteFile(fileName, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("error writing migration file: %w", err)
	}

	fmt.Printf("✅ Created migration file: %s\n", fileName)
	return nil
}

// generateDeleteColumnsSQL generates ALTER TABLE DROP COLUMN and ADD COLUMN SQL statements
func generateDeleteColumnsSQL(db *pgxpool.Pool, tableName, columns string) (string, string, error) {
	var upStatements []string
	var downStatements []string

	// Process column names
	columnNames := strings.Split(columns, ",")
	for _, columnName := range columnNames {
		columnName = strings.TrimSpace(strings.Split(columnName, ":")[0])
		if columnName == "" {
			continue
		}

		// Get the full column definition for rollback
		columnDef, err := getColumnDefinition(db, tableName, columnName)
		if err != nil {
			return "", "", fmt.Errorf("error getting column definition for '%s': %w", columnName, err)
		}

		upStatements = append(upStatements, fmt.Sprintf("ALTER TABLE %s DROP COLUMN IF EXISTS %s;", tableName, columnName))
		downStatements = append(downStatements, fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS %s;", tableName, columnDef))
	}

	return strings.Join(upStatements, "\n"), strings.Join(downStatements, "\n"), nil
}
