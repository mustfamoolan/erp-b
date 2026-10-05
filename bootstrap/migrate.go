package bootstrap

import (
	"fmt"
	"log"

	"github.com/fatih/color"
	"github.com/golang-migrate/migrate/v4"
	migratepostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"m3aml-erp/config"
)

// RunMigrations applies all pending SQL migrations from database/migrations/.
// Uses golang-migrate — migrations are explicit SQL files, never GORM AutoMigrate.
// This ensures immutability and auditability of schema changes (Roadmap Rule 14).
func RunMigrations() {
	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatalf("%s Could not get sql.DB from GORM: %v", color.RedString("FATAL:"), err)
	}

	driver, err := migratepostgres.WithInstance(sqlDB, &migratepostgres.Config{
		DatabaseName: config.Global.Database.Database,
	})
	if err != nil {
		log.Fatalf("%s Could not create migrate driver: %v", color.RedString("FATAL:"), err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://database/migrations",
		"postgres",
		driver,
	)
	if err != nil {
		log.Fatalf("%s Could not create migrate instance: %v", color.RedString("FATAL:"), err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("%s Migration failed: %v", color.RedString("FATAL:"), err)
	}

	version, dirty, _ := m.Version()
	if dirty {
		log.Fatalf("%s Database is in dirty state at version %d. Fix migrations.", color.RedString("FATAL:"), version)
	}

	fmt.Printf("%s Migrations applied (version: %d)\n", color.CyanString("⚙️"), version)
}
