package main

import (
	"fmt"
	"log"
	"os"

	migroCMD "github.com/ChungNQ511/migro/cmd"
	"github.com/urfave/cli/v2"
)

var GlobalConfig *migroCMD.CONFIG

func setGlobalConfig(config *migroCMD.CONFIG) {
	GlobalConfig = config
}

func getGlobalConfig() *migroCMD.CONFIG {
	return GlobalConfig
}

func main() {
	app := &cli.App{
		Name:  "migro",
		Usage: "A tool for managing database migrations",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Path to config file (default: migro.yaml)",
				EnvVars: []string{"MIGRO_CONFIG"},
			},
		},
		Before: func(c *cli.Context) error {
			configPath := c.String("config")

			// If no config specified, try to find migro.yaml in current directory
			if configPath == "" {
				// Try multiple possible config file names
				possibleConfigs := []string{
					"migro.yaml",
					"migro.yml",
					"config.yaml",
					"config.yml",
				}

				for _, fileName := range possibleConfigs {
					if _, err := os.Stat(fileName); err == nil {
						configPath = fileName
						fmt.Printf("📄 Using config file: %s\n", fileName)
						break
					}
				}

				if configPath == "" {
					return fmt.Errorf("❌ No config file found. Please create migro.yaml or specify --config flag")
				}
			}

			cfg, err := migroCMD.LoadConfig(configPath)
			if err != nil {
				return fmt.Errorf("❌ Failed to load config from %s: %w", configPath, err)
			}
			setGlobalConfig(cfg)
			return nil
		},
		Commands: []*cli.Command{
			{
				Name:  "create-migration",
				Usage: "Create a new migration file",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "name",
						Aliases:  []string{"n"},
						Usage:    "Name of the migration",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					return migroCMD.CreateEmptyMigration(getGlobalConfig().MIGRATION_DIR, c.String("name"))
				},
			},
			{
				Name:  "create-table",
				Usage: "Create a new table with columns",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "table",
						Aliases:  []string{"t"},
						Usage:    "Table name to create",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "columns",
						Aliases:  []string{"c"},
						Usage:    "Column definitions in format: name:type[:options...],name2:type2[:options...]",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					pool := migroCMD.DBConnection(getGlobalConfig())
					defer pool.Close()
					return migroCMD.CreateTable(getGlobalConfig(), pool, c.String("table"), c.String("columns"))
				},
			},
			{
				Name:  "add-column",
				Usage: "Add columns to an existing table",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "table",
						Aliases:  []string{"t"},
						Usage:    "Table name to add columns to",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "columns",
						Aliases:  []string{"c"},
						Usage:    "Column definitions in format: name:type[:options...],name2:type2[:options...]",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					pool := migroCMD.DBConnection(getGlobalConfig())
					defer pool.Close()
					return migroCMD.AddColumn(getGlobalConfig(), pool, c.String("table"), c.String("columns"))
				},
			},
			{
				Name:  "delete-column",
				Usage: "Delete columns from an existing table",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "table",
						Aliases:  []string{"t"},
						Usage:    "Table name to delete columns from",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "columns",
						Aliases:  []string{"c"},
						Usage:    "Column names to delete (comma-separated): name1,name2,name3",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					pool := migroCMD.DBConnection(getGlobalConfig())
					defer pool.Close()
					return migroCMD.DeleteColumn(getGlobalConfig(), pool, c.String("table"), c.String("columns"))
				},
			},
			{
				Name:  "read-table",
				Usage: "Read column information of a table",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "table",
						Aliases:  []string{"t"},
						Usage:    "Table name to read columns from",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					pool := migroCMD.DBConnection(getGlobalConfig())
					defer pool.Close()
					columns, err := migroCMD.ReadColumnOfTable(pool, c.String("table"))
					if err != nil {
						return err
					}
					fmt.Println("✅ Read column of table", c.String("table"), "success")
					fmt.Println(columns)
					return nil
				},
			},
			{
				Name:  "reset",
				Usage: "Reset the sequence of a table",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "table",
						Aliases:  []string{"t"},
						Usage:    "Table name to reset sequence for",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					pool := migroCMD.DBConnection(getGlobalConfig())
					defer pool.Close()
					return migroCMD.ResetSequenceOfTable(pool, c.String("table"))
				},
			},
			{
				Name:  "rollback",
				Usage: "Rollback specified number of migrations",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:     "count",
						Aliases:  []string{"c"},
						Usage:    "Number of migrations to rollback",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					pool := migroCMD.DBConnection(getGlobalConfig())
					defer pool.Close()
					return migroCMD.Rollback(getGlobalConfig(), pool, c.Int("count"))
				},
			},
			{
				Name:  "rollback-all",
				Usage: "Rollback ALL migrations (WARNING: This will reset your database)",
				Action: func(c *cli.Context) error {
					pool := migroCMD.DBConnection(getGlobalConfig())
					defer pool.Close()
					return migroCMD.RollbackAll(getGlobalConfig(), pool)
				},
			},
			{
				Name:  "migrate",
				Usage: "Run database migration up to latest version",
				Action: func(c *cli.Context) error {
					pool := migroCMD.DBConnection(getGlobalConfig())
					defer pool.Close()
					return migroCMD.MigrateUp(getGlobalConfig(), pool)
				},
			},
			{
				Name:  "status",
				Usage: "Show current migration status",
				Action: func(c *cli.Context) error {
					return migroCMD.ShowMigrationStatus(getGlobalConfig())
				},
			},
			{
				Name:  "insert",
				Usage: "Insert data into a table",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "table",
						Aliases:  []string{"t"},
						Usage:    "Table name to insert data into",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "data",
						Aliases:  []string{"d"},
						Usage:    "Data to insert in format: column1=value1,column2=value2",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					pool := migroCMD.DBConnection(getGlobalConfig())
					defer pool.Close()
					return migroCMD.InsertData(getGlobalConfig(), pool, c.String("table"), c.String("data"))
				},
			},
			{
				Name:  "update",
				Usage: "Update data in a table",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "table",
						Aliases:  []string{"t"},
						Usage:    "Table name to update data in",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "data",
						Aliases:  []string{"d"},
						Usage:    "Data to update in format: column1=value1,column2=value2",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "where",
						Aliases:  []string{"w"},
						Usage:    "WHERE clause in format: column=value",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					pool := migroCMD.DBConnection(getGlobalConfig())
					defer pool.Close()
					return migroCMD.UpdateData(getGlobalConfig(), pool, c.String("table"), c.String("data"), c.String("where"))
				},
			},
			{
				Name:  "select-one",
				Usage: "Select one record from a table",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "table",
						Aliases:  []string{"t"},
						Usage:    "Table name to select from",
						Required: true,
					},
					&cli.StringFlag{
						Name:    "columns",
						Aliases: []string{"c"},
						Usage:   "Columns to select (default: *)",
						Value:   "*",
					},
					&cli.StringFlag{
						Name:     "where",
						Aliases:  []string{"w"},
						Usage:    "WHERE clause in format: column=value",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					pool := migroCMD.DBConnection(getGlobalConfig())
					defer pool.Close()
					return migroCMD.SelectOne(getGlobalConfig(), pool, c.String("table"), c.String("columns"), c.String("where"))
				},
			},
			{
				Name:  "select-many",
				Usage: "Select multiple records from a table",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "table",
						Aliases:  []string{"t"},
						Usage:    "Table name to select from",
						Required: true,
					},
					&cli.StringFlag{
						Name:    "columns",
						Aliases: []string{"c"},
						Usage:   "Columns to select (default: *)",
						Value:   "*",
					},
					&cli.StringFlag{
						Name:    "where",
						Aliases: []string{"w"},
						Usage:   "WHERE clause in format: column=value (optional)",
					},
					&cli.IntFlag{
						Name:    "limit",
						Aliases: []string{"l"},
						Usage:   "Maximum number of records to return (default: 100)",
						Value:   100,
					},
				},
				Action: func(c *cli.Context) error {
					pool := migroCMD.DBConnection(getGlobalConfig())
					defer pool.Close()
					return migroCMD.SelectMany(getGlobalConfig(), pool, c.String("table"), c.String("columns"), c.String("where"), c.Int("limit"))
				},
			},
			{
				Name:  "delete",
				Usage: "Soft delete record from a table (sets deleted_at)",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "table",
						Aliases:  []string{"t"},
						Usage:    "Table name to delete from",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "where",
						Aliases:  []string{"w"},
						Usage:    "WHERE clause in format: column=value",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					pool := migroCMD.DBConnection(getGlobalConfig())
					defer pool.Close()
					return migroCMD.SoftDelete(getGlobalConfig(), pool, c.String("table"), c.String("where"))
				},
			},
			{
				Name:  "sqlc-init",
				Usage: "Initialize SQLC configuration (creates sqlc.yaml and example queries)",
				Action: func(c *cli.Context) error {
					return migroCMD.InitSQLC(getGlobalConfig())
				},
			},
			{
				Name:  "sqlc",
				Usage: "Generate SQLC code from database",
				Action: func(c *cli.Context) error {
					return migroCMD.GenerateSQLC(getGlobalConfig())
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
