package main

import (
	"fmt"
	"log"
	"os"

	"github.com/derangedhermits/website/internal/config"
	"github.com/derangedhermits/website/internal/db"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: seed <username> <password>\n")
		os.Exit(1)
	}

	username := os.Args[1]
	password := os.Args[2]

	cfg := config.Load()
	database, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	if err := db.CreateAdminUser(database, username, password); err != nil {
		log.Fatalf("Failed to create admin user: %v", err)
	}

	fmt.Printf("Admin user '%s' created successfully.\n", username)
}
