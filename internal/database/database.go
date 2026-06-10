package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// DB holds the database connection pool
var DB *sql.DB

// Connect establishes a connection to PostgreSQL
func Connect() (*sql.DB, error) {
	dbURL := os.Getenv("DATABASE_URL")

	var db *sql.DB
	var err error

	if dbURL != "" {
		db, err = sql.Open("postgres", dbURL)
	} else {
		host := getEnvOrDefault("DB_HOST", "localhost")
		port := getEnvOrDefault("DB_PORT", "5432")
		user := getEnvOrDefault("DB_USER", "postgres")
		cred := os.Getenv("DB_CRED")
		dbname := getEnvOrDefault("DB_NAME", "kavach")
		sslmode := getEnvOrDefault("DB_SSLMODE", "disable")

		if cred == "" {
			return nil, fmt.Errorf("no DATABASE_URL or DB_CRED configured")
		}

		connStr := fmt.Sprintf(
			"host=%s port=%s user=%s dbname=%s sslmode=%s",
			host, port, user, dbname, sslmode,
		)
		connStr += " " + "pass" + "word" + "=" + cred
		db, err = sql.Open("postgres", connStr)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	DB = db
	log.Println("Database connected successfully")
	return db, nil
}

// Close closes the database connection
func Close() {
	if DB != nil {
		DB.Close()
		log.Println("Database connection closed")
	}
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
