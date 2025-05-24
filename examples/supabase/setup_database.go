// setup_database.go - Initialize Supabase database schema
package main

import (
	"database/sql"
	"io/ioutil"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
		log.Println("Make sure to set environment variables or copy .env.example to .env")
	}

	// Get database URL
	dbURL := os.Getenv("SUPABASE_DB_URL")
	if dbURL == "" {
		log.Fatal("SUPABASE_DB_URL environment variable is required")
	}

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("✅ Connected to Supabase database successfully")

	// Read initialization script
	sqlContent, err := ioutil.ReadFile("init.sql")
	if err != nil {
		log.Fatalf("Failed to read init.sql: %v", err)
	}

	// Execute initialization script
	log.Println("🔄 Executing database initialization script...")
	if _, err := db.Exec(string(sqlContent)); err != nil {
		log.Fatalf("Failed to execute initialization script: %v", err)
	}

	log.Println("✅ Database schema initialized successfully!")
	log.Println("🚀 You can now run the userprefs examples with 'make run'")
}
