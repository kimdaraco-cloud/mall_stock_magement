// @ai-modified 2026-07-02 add seed script creating the default admin user
package main

import (
	"context"
	"fmt"
	"os"

	"mallstock/internal/config"
	"mallstock/internal/database"
	"mallstock/internal/service"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "seed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx := context.Background()
	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	hash, err := service.HashPassword("admin123")
	if err != nil {
		return err
	}
	tag, err := pool.Exec(ctx,
		`INSERT INTO users (email, password_hash, full_name, role, is_active)
		 VALUES ($1, $2, 'Administrator', 'admin', TRUE)
		 ON CONFLICT (email) DO NOTHING`,
		"admin@mall.local", hash)
	if err != nil {
		return fmt.Errorf("seed admin: %w", err)
	}
	if tag.RowsAffected() == 1 {
		fmt.Println("seed: created admin@mall.local / admin123 — change it immediately")
	} else {
		fmt.Println("seed: admin@mall.local already exists, skipped")
	}
	return nil
}
