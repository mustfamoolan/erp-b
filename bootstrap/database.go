package bootstrap

import (
	"fmt"
	"log"

	"github.com/fatih/color"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"m3aml-erp/config"
)

// DB is the shared database instance.
var DB *gorm.DB

// InitializeDatabase connects to PostgreSQL and returns the connection.
// Only PostgreSQL is supported — Roadmap §0 specifies PostgreSQL exclusively.
func InitializeDatabase() *gorm.DB {
	dbConfig := config.Global.Database

	if dbConfig.Driver != "postgres" {
		log.Fatalf("%s Only PostgreSQL is supported. Set DB_DRIVER=postgres in .env", color.RedString("FATAL:"))
	}

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Riyadh",
		dbConfig.Host,
		dbConfig.Username,
		dbConfig.Password,
		dbConfig.Database,
		dbConfig.Port,
	)

	gormLogger := logger.Default.LogMode(logger.Silent)
	if config.Global.App.Debug {
		gormLogger = logger.Default.LogMode(logger.Info)
	}

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		log.Fatalf("%s Failed to connect to PostgreSQL: %v", color.RedString("FATAL:"), err)
	}

	// Connection pool settings
	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatalf("%s Failed to get underlying DB: %v", color.RedString("FATAL:"), err)
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)

	fmt.Printf("%s PostgreSQL connected (%s:%s/%s)\n",
		color.CyanString("⚙️"),
		dbConfig.Host,
		dbConfig.Port,
		dbConfig.Database,
	)
	return DB
}
