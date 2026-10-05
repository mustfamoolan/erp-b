package main

import (
	"fmt"
	"m3aml-erp/bootstrap"
)

func main() {
	bootstrap.InitializeConfig()
	bootstrap.InitializeDatabase()
	bootstrap.RunMigrations()
	fmt.Println("Done migrating!")
}
