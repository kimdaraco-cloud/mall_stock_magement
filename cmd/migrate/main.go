// @ai-modified 2026-07-02 add migration CLI wrapping golang-migrate as a library
package main

import (
	"fmt"
	"os"

	"mallstock/internal/config"
	"mallstock/internal/database"
)

const usage = `usage: migrate <up|down>

  up    apply all pending migrations
  down  roll back the last migration
`

func main() {
	if len(os.Args) != 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrate: %v\n", err)
		os.Exit(1)
	}

	src := "file://migrations"
	switch os.Args[1] {
	case "up":
		err = database.MigrateUp(src, cfg.DatabaseURL)
	case "down":
		err = database.MigrateDown(src, cfg.DatabaseURL)
	default:
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrate: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("migrate:", os.Args[1], "ok")
}
