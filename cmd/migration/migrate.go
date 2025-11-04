package main

import (
	"errors"
	"log"
	"os"
	"project/config"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	config := config.NewConfig()
	if config.Debug {
		log.Println("Debug mode enabled")
	}

	m, err := migrate.New(
		config.MigrationsPath,
		config.PostgresURL,
	)
	if err != nil {
		log.Panicf("Error initializing migrations: %v", err)
	}

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "down":
			if err = m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
				log.Fatalf("Error rolling back migrations: %v", err)
			}
			log.Println("Migrations rolled back successfully.")

		default:
			if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
				log.Fatalf("Error applying migrations: %v", err)
			}
			log.Println("Migrations applied successfully.")
		}

	} else {
		if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			log.Fatalf("Error applying migrations: %v", err)
		}
		log.Println("Migrations applied successfully.")
	}
}
