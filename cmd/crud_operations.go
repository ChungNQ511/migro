package migroCMD

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Insert data into table
// @param config *CONFIG
// @param db *pgxpool.Pool
// @param table string
// @param data string (format: "column1=value1,column2=value2")
// @return error
func InsertData(config *CONFIG, db *pgxpool.Pool, table string, data string) error {
	ctx := context.Background()

	// check table exists in migration files
	exists, err := checkTableExistsInMigrations(config.MIGRATION_DIR, table)
	if err != nil {
		return fmt.Errorf("❌ error checking table exists: %w", err)
	}
	if !exists {
		return fmt.Errorf("❌ table '%s' does not exist in migration files", table)
	}

	// Parse data
	columns, values, err := parseInsertData(data)
	if err != nil {
		return fmt.Errorf("❌ error parsing data: %w", err)
	}

	// Build INSERT query
	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) RETURNING *",
		table,
		strings.Join(columns, ", "),
		buildPlaceholders(len(values)),
	)

	fmt.Printf("🔄 Executing: %s\n", query)
	fmt.Printf("📝 Values: %v\n", values)

	// Execute query
	rows, err := db.Query(ctx, query, values...)
	if err != nil {
		return fmt.Errorf("❌ insert failed: %w", err)
	}
	defer rows.Close()

	// Print result
	fmt.Println("✅ Insert successful!")
	err = printQueryResults(rows)
	if err != nil {
		return fmt.Errorf("❌ error printing results: %w", err)
	}

	return nil
}

// Update data in table
// @param config *CONFIG
// @param db *pgxpool.Pool
// @param table string
// @param data string (format: "column1=value1,column2=value2")
// @param where string (format: "id=1" or "name='test'")
// @return error
func UpdateData(config *CONFIG, db *pgxpool.Pool, table string, data string, where string) error {
	ctx := context.Background()

	// check table exists in migration files
	exists, err := checkTableExistsInMigrations(config.MIGRATION_DIR, table)
	if err != nil {
		return fmt.Errorf("❌ error checking table exists: %w", err)
	}
	if !exists {
		return fmt.Errorf("❌ table '%s' does not exist in migration files", table)
	}

	// Parse data
	setClauses, values, err := parseUpdateData(data)
	if err != nil {
		return fmt.Errorf("❌ error parsing data: %w", err)
	}

	// Parse WHERE clause
	whereClause, whereValues, err := parseWhereClause(where, len(values))
	if err != nil {
		return fmt.Errorf("❌ error parsing where clause: %w", err)
	}

	// Combine values
	allValues := append(values, whereValues...)

	// Add updated_at if column exists
	setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", len(allValues)+1))
	allValues = append(allValues, time.Now())

	// Build UPDATE query
	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s RETURNING *",
		table,
		strings.Join(setClauses, ", "),
		whereClause,
	)

	fmt.Printf("🔄 Executing: %s\n", query)
	fmt.Printf("📝 Values: %v\n", allValues)

	// Execute query
	rows, err := db.Query(ctx, query, allValues...)
	if err != nil {
		return fmt.Errorf("❌ update failed: %w", err)
	}
	defer rows.Close()

	// Print result
	fmt.Println("✅ Update successful!")
	err = printQueryResults(rows)
	if err != nil {
		return fmt.Errorf("❌ error printing results: %w", err)
	}

	return nil
}

// Select one record from table
// @param config *CONFIG
// @param db *pgxpool.Pool
// @param table string
// @param columns string (optional, default: "*")
// @param where string (format: "id=1")
// @return error
func SelectOne(config *CONFIG, db *pgxpool.Pool, table string, columns string, where string) error {
	ctx := context.Background()

	// check table exists in migration files
	exists, err := checkTableExistsInMigrations(config.MIGRATION_DIR, table)
	if err != nil {
		return fmt.Errorf("❌ error checking table exists: %w", err)
	}
	if !exists {
		return fmt.Errorf("❌ table '%s' does not exist in migration files", table)
	}

	// Default columns
	if columns == "" {
		columns = "*"
	}

	// Parse WHERE clause
	whereClause, values, err := parseWhereClause(where, 0)
	if err != nil {
		return fmt.Errorf("❌ error parsing where clause: %w", err)
	}

	// Build SELECT query
	query := fmt.Sprintf(
		"SELECT %s FROM %s WHERE %s AND deleted_at IS NULL LIMIT 1",
		columns,
		table,
		whereClause,
	)

	fmt.Printf("🔄 Executing: %s\n", query)
	fmt.Printf("📝 Values: %v\n", values)

	// Execute query
	rows, err := db.Query(ctx, query, values...)
	if err != nil {
		return fmt.Errorf("❌ select failed: %w", err)
	}
	defer rows.Close()

	// Check if any results
	if !rows.Next() {
		fmt.Println("📭 No records found")
		return nil
	}

	// Print result
	fmt.Println("✅ Record found!")
	err = printQueryResults(rows)
	if err != nil {
		return fmt.Errorf("❌ error printing results: %w", err)
	}

	return nil
}

// Select many records from table
// @param config *CONFIG
// @param db *pgxpool.Pool
// @param table string
// @param columns string (optional, default: "*")
// @param where string (optional, format: "status='active'")
// @param limit int (optional, default: 100)
// @return error
func SelectMany(config *CONFIG, db *pgxpool.Pool, table string, columns string, where string, limit int) error {
	ctx := context.Background()

	// check table exists in migration files
	exists, err := checkTableExistsInMigrations(config.MIGRATION_DIR, table)
	if err != nil {
		return fmt.Errorf("❌ error checking table exists: %w", err)
	}
	if !exists {
		return fmt.Errorf("❌ table '%s' does not exist in migration files", table)
	}

	// Default columns
	if columns == "" {
		columns = "*"
	}

	// Default limit
	if limit <= 0 {
		limit = 100
	}

	var query string
	var values []interface{}

	// Build query with or without WHERE
	if where != "" {
		whereClause, whereValues, err := parseWhereClause(where, 0)
		if err != nil {
			return fmt.Errorf("❌ error parsing where clause: %w", err)
		}
		values = whereValues

		query = fmt.Sprintf(
			"SELECT %s FROM %s WHERE %s AND deleted_at IS NULL ORDER BY created_at DESC LIMIT %d",
			columns,
			table,
			whereClause,
			limit,
		)
	} else {
		query = fmt.Sprintf(
			"SELECT %s FROM %s WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT %d",
			columns,
			table,
			limit,
		)
	}

	fmt.Printf("🔄 Executing: %s\n", query)
	if len(values) > 0 {
		fmt.Printf("📝 Values: %v\n", values)
	}

	// Execute query
	rows, err := db.Query(ctx, query, values...)
	if err != nil {
		return fmt.Errorf("❌ select failed: %w", err)
	}
	defer rows.Close()

	// Count results
	count := 0
	fmt.Println("✅ Records found:")
	err = printQueryResults(rows)
	if err != nil {
		return fmt.Errorf("❌ error printing results: %w", err)
	}

	// Count rows (approximate)
	rows.Close()
	countQuery := strings.Replace(query, fmt.Sprintf("SELECT %s FROM", columns), "SELECT COUNT(*) FROM", 1)
	countQuery = strings.Replace(countQuery, fmt.Sprintf("ORDER BY created_at DESC LIMIT %d", limit), "", 1)

	err = db.QueryRow(ctx, countQuery, values...).Scan(&count)
	if err == nil {
		fmt.Printf("\n📊 Total records: %d (showing max %d)\n", count, limit)
	}

	return nil
}

// Soft delete record (update deleted_at)
// @param config *CONFIG
// @param db *pgxpool.Pool
// @param table string
// @param where string (format: "id=1")
// @return error
func SoftDelete(config *CONFIG, db *pgxpool.Pool, table string, where string) error {
	ctx := context.Background()

	// check table exists in migration files
	exists, err := checkTableExistsInMigrations(config.MIGRATION_DIR, table)
	if err != nil {
		return fmt.Errorf("❌ error checking table exists: %w", err)
	}
	if !exists {
		return fmt.Errorf("❌ table '%s' does not exist in migration files", table)
	}

	// Parse WHERE clause
	whereClause, values, err := parseWhereClause(where, 0)
	if err != nil {
		return fmt.Errorf("❌ error parsing where clause: %w", err)
	}

	// Add deleted_at timestamp
	deletedAt := time.Now()
	values = append(values, deletedAt)

	// Build UPDATE query for soft delete
	query := fmt.Sprintf(
		"UPDATE %s SET deleted_at = $%d, updated_at = $%d WHERE %s AND deleted_at IS NULL RETURNING *",
		table,
		len(values),
		len(values)+1,
		whereClause,
	)
	values = append(values, time.Now())

	fmt.Printf("🔄 Executing soft delete: %s\n", query)
	fmt.Printf("📝 Values: %v\n", values)

	// Execute query
	rows, err := db.Query(ctx, query, values...)
	if err != nil {
		return fmt.Errorf("❌ soft delete failed: %w", err)
	}
	defer rows.Close()

	// Check if any rows affected
	if !rows.Next() {
		fmt.Println("📭 No records found to delete (may already be deleted)")
		return nil
	}

	// Print result
	fmt.Println("✅ Soft delete successful!")
	err = printQueryResults(rows)
	if err != nil {
		return fmt.Errorf("❌ error printing results: %w", err)
	}

	return nil
}

// Helper functions

// parseInsertData parses insert data string into columns and values
// @param data string (format: "name=John,age=25,email=john@example.com")
// @return []string, []interface{}, error
func parseInsertData(data string) ([]string, []interface{}, error) {
	if data == "" {
		return nil, nil, fmt.Errorf("data cannot be empty")
	}

	pairs := strings.Split(data, ",")
	var columns []string
	var values []interface{}

	for _, pair := range pairs {
		parts := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(parts) != 2 {
			return nil, nil, fmt.Errorf("invalid data format: %s (expected key=value)", pair)
		}

		column := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
			value = strings.Trim(value, "'")
		}

		columns = append(columns, column)
		values = append(values, value)
	}

	return columns, values, nil
}

// parseUpdateData parses update data string into SET clauses and values
// @param data string (format: "name=John,age=25")
// @return []string, []interface{}, error
func parseUpdateData(data string) ([]string, []interface{}, error) {
	columns, values, err := parseInsertData(data)
	if err != nil {
		return nil, nil, err
	}

	var setClauses []string
	for i, column := range columns {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", column, i+1))
	}

	return setClauses, values, nil
}

// parseWhereClause parses WHERE clause string
// @param where string (format: "id=1" or "name='test'")
// @param startIndex int (for parameter numbering)
// @return string, []interface{}, error
func parseWhereClause(where string, startIndex int) (string, []interface{}, error) {
	if where == "" {
		return "", nil, fmt.Errorf("where clause cannot be empty")
	}

	// Simple parsing for basic conditions
	parts := strings.SplitN(strings.TrimSpace(where), "=", 2)
	if len(parts) != 2 {
		return "", nil, fmt.Errorf("invalid where format: %s (expected column=value)", where)
	}

	column := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	// Remove quotes if present
	if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
		value = strings.Trim(value, "'")
	}

	whereClause := fmt.Sprintf("%s = $%d", column, startIndex+1)
	values := []interface{}{value}

	return whereClause, values, nil
}

// buildPlaceholders builds PostgreSQL placeholders ($1, $2, ...)
// @param count int
// @return string
func buildPlaceholders(count int) string {
	var placeholders []string
	for i := 1; i <= count; i++ {
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
	}
	return strings.Join(placeholders, ", ")
}

// printQueryResults prints query results in a formatted table
// @param rows pgx.Rows
// @return error
func printQueryResults(rows interface{}) error {
	switch r := rows.(type) {
	case pgx.Rows:
		return printPgxRows(r)
	default:
		fmt.Println("📋 Query results (unsupported type)")
		return nil
	}
}

// printPgxRows prints pgx rows in formatted table
func printPgxRows(rows pgx.Rows) error {
	// Get field descriptions
	fieldDescriptions := rows.FieldDescriptions()
	if len(fieldDescriptions) == 0 {
		fmt.Println("📭 No columns in result")
		return nil
	}

	// Print header
	fmt.Printf("\n")
	for i, field := range fieldDescriptions {
		if i > 0 {
			fmt.Printf(" | ")
		}
		fmt.Printf("%-15s", string(field.Name))
	}
	fmt.Printf("\n")

	// Print separator
	for i := range fieldDescriptions {
		if i > 0 {
			fmt.Printf("-+-")
		}
		fmt.Printf("%-15s", strings.Repeat("-", 15))
	}
	fmt.Printf("\n")

	// Print rows
	rowCount := 0
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return fmt.Errorf("error scanning row: %w", err)
		}

		for i, value := range values {
			if i > 0 {
				fmt.Printf(" | ")
			}
			// Format value based on type
			formatted := formatValue(value)
			if len(formatted) > 15 {
				formatted = formatted[:12] + "..."
			}
			fmt.Printf("%-15s", formatted)
		}
		fmt.Printf("\n")
		rowCount++
	}

	if rowCount == 0 {
		fmt.Println("📭 No rows returned")
	} else {
		fmt.Printf("\n📊 %d row(s) returned\n", rowCount)
	}

	return rows.Err()
}

// formatValue formats a value for display
func formatValue(value interface{}) string {
	if value == nil {
		return "<NULL>"
	}

	switch v := value.(type) {
	case string:
		return v
	case int, int32, int64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%.2f", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case time.Time:
		return v.Format("2006-01-02 15:04:05")
	default:
		return fmt.Sprintf("%v", v)
	}
}
