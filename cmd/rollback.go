package migroCMD

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// import (
// 	"bufio"
// 	"context"
// 	"fmt"
// 	"os"
// 	"os/exec"
// 	"strings"

// 	"github.com/jackc/pgx/v5/pgxpool"
// )

var createdTempFiles []string // Track temp files for cleanup

// Rollback migrations by count
// @param config: util.Config
// @param db: *pgxpool.Pool
// @param count: int - number of migrations to rollback
func Rollback(config *CONFIG, db *pgxpool.Pool, count int) error {
	ctx := context.Background()

	if count <= 0 {
		return fmt.Errorf("rollback count must be greater than 0")
	}

	fmt.Printf("🔄 Rolling back %d migration(s)...\n", count)

	// Clear temp files tracker
	createdTempFiles = []string{}

	for i := 1; i <= count; i++ {
		fmt.Printf("📉 Rollback round %d/%d...\n", i, count)

		err := performSingleRollback(config, ctx)
		if err != nil {
			return fmt.Errorf("rollback failed at round %d: %w", i, err)
		}

		fmt.Printf("✅ Rollback round %d completed\n", i)
	}

	// Ask if user wants to cleanup temp files
	if len(createdTempFiles) > 0 {
		fmt.Printf("\n🧹 Đã tạo %d file temp migration. Bạn có muốn xóa chúng không? (y/N): ", len(createdTempFiles))
		reader := bufio.NewReader(os.Stdin)
		cleanup, _ := reader.ReadString('\n')
		cleanup = strings.TrimSpace(strings.ToLower(cleanup))

		if cleanup == "y" || cleanup == "yes" {
			cleanupTempFiles()
		} else {
			fmt.Println("📝 Temp files được giữ lại:")
			for _, file := range createdTempFiles {
				fmt.Printf("   - %s\n", file)
			}
		}
	}

	// Show final migration status
	return showMigrationStatus(config)
}

// Rollback all migrations
// @param config *CONFIG
// @param db *pgxpool.Pool
// @return error
func RollbackAll(config *CONFIG, db *pgxpool.Pool) error {
	ctx := context.Background()

	fmt.Println("🔥 Rolling back ALL migrations...")
	fmt.Print("⚠️  This will rollback ALL migrations. Are you sure? (y/N): ")

	reader := bufio.NewReader(os.Stdin)
	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(strings.ToLower(confirm))

	if confirm != "y" && confirm != "yes" {
		return fmt.Errorf("rollback all cancelled by user")
	}

	// Clear temp files tracker
	createdTempFiles = []string{}

	// Use goose reset to rollback all migrations
	cmd := exec.CommandContext(ctx, "goose",
		"-dir", config.MIGRATION_DIR,
		config.DATABASE_CONNECTION_STRING,
		"reset")

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Try to handle missing migration error
		if strings.Contains(string(output), "missing") && strings.Contains(string(output), "migration") {
			fmt.Println("🔧 Found missing migration, creating temp files...")
			err = handleMissingMigrationForReset(config, ctx)
			if err != nil {
				return fmt.Errorf("failed to handle missing migration: %w", err)
			}
			// Retry reset after creating temp files
			output, err = cmd.CombinedOutput()
		}

		if err != nil {
			return fmt.Errorf("rollback all failed: %w\nOutput: %s", err, string(output))
		}
	}

	fmt.Println("✅ All migrations rolled back successfully")

	// Ask if user wants to cleanup temp files
	if len(createdTempFiles) > 0 {
		fmt.Printf("\n🧹 Đã tạo %d file temp migration. Bạn có muốn xóa chúng không? (y/N): ", len(createdTempFiles))
		cleanup, _ := reader.ReadString('\n')
		cleanup = strings.TrimSpace(strings.ToLower(cleanup))

		if cleanup == "y" || cleanup == "yes" {
			cleanupTempFiles()
		}
	}

	// Show final migration status
	return showMigrationStatus(config)
}

// Handle missing migration for reset operation
// @param config *CONFIG
// @param ctx context.Context
// @return error
func handleMissingMigrationForReset(config *CONFIG, ctx context.Context) error {
	// Get migration status to find missing versions
	cmd := exec.CommandContext(ctx, "goose",
		"-dir", config.MIGRATION_DIR,
		config.DATABASE_CONNECTION_STRING,
		"status")

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Even if status fails, try to extract missing migration info
		version, name := extractMissingMigrationInfo(config.MIGRATION_DIR, string(output))
		if version > 0 {
			tempFile := createTempMigration(config.MIGRATION_DIR, version, name)
			if tempFile != "" {
				createdTempFiles = append(createdTempFiles, tempFile)
				return nil
			}
		}
		return fmt.Errorf("failed to get migration status and create temp file: %w", err)
	}

	return nil
}

func executeGooseDown(config *CONFIG, ctx context.Context) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "goose",
		"-dir", config.MIGRATION_DIR,
		config.DATABASE_CONNECTION_STRING,
		"down")

	return cmd.CombinedOutput()
}

func performSingleRollback(config *CONFIG, ctx context.Context) error {
	maxRetries := 5 // Prevent infinite loop

	for retry := 0; retry < maxRetries; retry++ {
		output, err := executeGooseDown(config, ctx)

		if err != nil {
			// Check if it's a missing migration error for rollback
			if strings.Contains(string(output), "no current version found") ||
				strings.Contains(string(output), "migration") && strings.Contains(string(output), "no current version") {
				fmt.Println("🔧 Found missing migration in rollback, creating temp file...")
				fmt.Printf("🔍 Rollback error output: %s\n", string(output))

				// Extract version from error
				version := extractVersionFromRollbackError(string(output))
				if version > 0 {
					tempFile := createTempMigration(config.MIGRATION_DIR, version, "temp_migration")
					if tempFile != "" {
						fmt.Printf("📝 Created temp file for rollback: %s\n", tempFile)
						createdTempFiles = append(createdTempFiles, tempFile)
						continue // Retry rollback
					}
				}

				// Fallback: try to get current version from database
				currentVersion, err := getCurrentVersionFromDB(config)
				if err == nil && currentVersion > 0 {
					tempFile := createTempMigration(config.MIGRATION_DIR, currentVersion, "temp_migration")
					if tempFile != "" {
						fmt.Printf("📝 Created temp file for rollback (from DB): %s\n", tempFile)
						createdTempFiles = append(createdTempFiles, tempFile)
						continue // Retry rollback
					}
				}
			}

			// Check if it's a missing migration error (original logic)
			if strings.Contains(string(output), "missing") && strings.Contains(string(output), "migration") {
				fmt.Println("🔧 Found missing migration, creating temp file...")

				// Extract missing version from error
				version, migrationName := extractMissingMigrationInfo(config.MIGRATION_DIR, string(output))
				if version > 0 {
					tempFile := createTempMigration(config.MIGRATION_DIR, version, migrationName)
					if tempFile != "" {
						fmt.Printf("📝 Created temp file: %s\n", tempFile)
						createdTempFiles = append(createdTempFiles, tempFile)
						continue // Retry rollback
					}
				}
			}

			return fmt.Errorf("goose down failed: %w\nOutput: %s", err, string(output))
		}

		// Success
		return nil
	}

	return fmt.Errorf("failed after %d retries", maxRetries)
}

// Extract version from rollback error "migration 20250707100918: no current version found"
// @param output string
// @return int64
func extractVersionFromRollbackError(output string) int64 {
	// Pattern: "migration 20250707100918: no current version found"
	re := regexp.MustCompile(`migration\s+(\d+):\s+no current version`)
	matches := re.FindStringSubmatch(output)

	if len(matches) > 1 {
		version, err := strconv.ParseInt(matches[1], 10, 64)
		if err == nil {
			return version
		}
	}
	// Alternative pattern: "goose run: migration 20250707100918: no current version found"
	re2 := regexp.MustCompile(`goose run: migration\s+(\d+):`)
	matches2 := re2.FindStringSubmatch(output)
	if len(matches2) > 1 {
		version, err := strconv.ParseInt(matches2[1], 10, 64)
		if err == nil {
			return version
		}
	}
	return 0
}

// Get current migration version from database
// @param config *CONFIG
// @return int64, error
func getCurrentVersionFromDB(config *CONFIG) (int64, error) {
	ctx := context.Background()

	cmd := exec.CommandContext(ctx, "goose",
		"-dir", config.MIGRATION_DIR,
		config.DATABASE_CONNECTION_STRING,
		"version")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("failed to get current version: %w", err)
	}

	// Parse version from output
	version := parseCurrentVersionFromOutput(string(output))
	return version, nil
}

// Parse current version from goose version command output
// @param output string
// @return int64
func parseCurrentVersionFromOutput(output string) int64 {
	// Pattern: "goose: version 20250707100918"
	re := regexp.MustCompile(`version\s+(\d+)`)
	matches := re.FindStringSubmatch(output)

	if len(matches) > 1 {
		version, err := strconv.ParseInt(matches[1], 10, 64)
		if err == nil {
			return version
		}
	}

	return 0
}

// Cleanup temporary migration files
// @param createdTempFiles []string
func cleanupTempFiles() {
	for _, file := range createdTempFiles {
		err := os.Remove(file)
		if err != nil {
			fmt.Printf("⚠️  Failed to remove temp file %s: %v\n", file, err)
		} else {
			fmt.Printf("🗑️  Removed temp file: %s\n", file)
		}
	}
	createdTempFiles = []string{}
}
