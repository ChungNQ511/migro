package migroCMD

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Read Column of Table
// @param db *pgxpool.Pool
// @param table string
// @return []string, error
func ReadColumnOfTable(db *pgxpool.Pool, table string) ([]string, error) {
	ctx := context.Background()
	rows, err := db.Query(ctx, "SELECT column_name FROM information_schema.columns WHERE table_name = $1", table)
	if err != nil {
		return nil, fmt.Errorf("query columns failed: %w", err)
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var column string
		err := rows.Scan(&column)
		if err != nil {
			return nil, fmt.Errorf("scan column failed: %w", err)
		}
		columns = append(columns, column)
	}

	return columns, nil
}
