package seeders

import (
	"fmt"
	"github.com/fatih/color"
	"gorm.io/gorm"
)

func Run(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		// Admin user — username: admin | password: 12345678
		if err := runSeeder(tx, &AdminSeeder{}); err != nil {
			return err
		}
		
		// Removed FactorySeeder to keep database clean for production/testing

		fmt.Printf("%s All seeders completed successfully.\n", color.GreenString("🏁"))
		return nil
	})
}

func runSeeder(db *gorm.DB, seeder interface{ Run(db *gorm.DB) error }) error {
	name := fmt.Sprintf("%T", seeder)
	fmt.Printf("%s Running Seeder: %s...\n", color.CyanString("⚡"), color.YellowString(name))
	
	if err := seeder.Run(db); err != nil {
		fmt.Printf("%s Seeder %s failed: %v\n", color.RedString("❌"), name, err)
		return err
	}
	
	fmt.Printf("%s Seeder %s completed.\n", color.GreenString("✅"), name)
	return nil
}
