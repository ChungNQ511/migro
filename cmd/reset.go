package migroCMD

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ResetSequenceOfTable reset sequence of table
// @param db: *pgxpool.Pool
// @param table: string
func ResetSequenceOfTable(db *pgxpool.Pool, table string) error {
	ctx := context.Background()
	var primaryKey string
	err := db.QueryRow(ctx,
		"SELECT a.attname FROM pg_index i JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey) WHERE i.indrelid = $1::regclass AND i.indisprimary;",
		table,
	).Scan(&primaryKey)
	if err != nil {
		return fmt.Errorf("get primary key failed: %w", err)
	}

	// Get sequence name
	var seqName string
	err = db.QueryRow(ctx,
		"SELECT pg_get_serial_sequence($1, $2)",
		table, primaryKey,
	).Scan(&seqName)
	if err != nil {
		return fmt.Errorf("get serial sequence failed: %w", err)
	}

	// Get MAX id
	query := fmt.Sprintf("SELECT MAX(%s) FROM %s", primaryKey, table)
	var maxID sql.NullInt64
	err = db.QueryRow(ctx, query).Scan(&maxID)
	if err != nil {
		return fmt.Errorf("get max id failed: %w", err)
	}
	if !maxID.Valid {
		return nil // table is empty
	}

	// Reset sequence value
	_, err = db.Exec(ctx,
		"SELECT setval($1, $2, true)",
		seqName, maxID.Int64,
	)
	if err != nil {
		return fmt.Errorf("setval failed: %w", err)
	}

	return nil
}
