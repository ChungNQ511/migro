package migroCMD

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/viper"
)

type CONFIG struct {
	ENV                        string `mapstructure:"ENV"`
	DATABASE_DRIVER            string `mapstructure:"DATABASE_DRIVER"`
	DATABASE_HOST              string `mapstructure:"DATABASE_HOST"`
	DATABASE_PORT              string `mapstructure:"DATABASE_PORT"`
	DATABASE_USERNAME          string `mapstructure:"DATABASE_USERNAME"`
	DATABASE_PASSWORD          string `mapstructure:"DATABASE_PASSWORD"`
	DATABASE_NAME              string `mapstructure:"DATABASE_NAME"`
	DATABASE_CONNECTION_STRING string `mapstructure:"DATABASE_CONNECTION_STRING"`
	TIMEOUT_SECONDS            int    `mapstructure:"TIMEOUT_SECONDS"`
	MIGRATION_DIR              string `mapstructure:"MIGRATION_DIR"`
	QUERY_DIR                  string `mapstructure:"QUERY_DIR"`
	SQLC_DIR                   string `mapstructure:"SQLC_DIR"`
}

func DBConnection(config *CONFIG) *pgxpool.Pool {
	dbURL := config.DATABASE_CONNECTION_STRING

	poolConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		log.Fatalf("Unable to parse connection string: %v\n", err)
	}

	poolConfig.MaxConns = 20
	poolConfig.MinConns = 1
	poolConfig.HealthCheckPeriod = 5 * time.Second

	// Create a new connection pool
	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		log.Fatalf("Unable to create connection pool: %v\n", err)
	}

	return pool
}

func LoadConfig(configPath string) (*CONFIG, error) {
	var config CONFIG
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}
	config.DATABASE_CONNECTION_STRING = buildConnectionString(&config)

	return &config, nil
}

func buildConnectionString(config *CONFIG) string {
	return fmt.Sprintf("%s://%s:%s@%s:%s/%s?sslmode=disable",
		config.DATABASE_DRIVER,
		config.DATABASE_USERNAME,
		config.DATABASE_PASSWORD,
		config.DATABASE_HOST,
		config.DATABASE_PORT,
		config.DATABASE_NAME)
}
