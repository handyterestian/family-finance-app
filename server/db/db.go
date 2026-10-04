package db

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

var Pool *pgxpool.Pool

// Init membuka connection pool ke PostgreSQL dari environment variables.
func Init(ctx context.Context) error {
	host := getenv("DB_HOST", "localhost")
	port := getenv("DB_PORT", "5432")
	user := getenv("DB_USER", "finance_user")
	pass := getenv("DB_PASSWORD", "finance_pass")
	name := getenv("DB_NAME", "finance_db")

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user, pass, host, port, name,
	)

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return fmt.Errorf("db: gagal buat pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("db: gagal ping: %w", err)
	}
	Pool = pool
	return nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
