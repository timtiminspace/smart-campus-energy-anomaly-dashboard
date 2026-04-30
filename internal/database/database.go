package database

import (
	"database/sql"
	_ "embed"
	"fmt"
	"log"

	_ "github.com/glebarez/go-sqlite" // Pure-Go driver
)

//go:embed schema.sql
var schemaSQL string

func InitDB(dataSourceName string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error pinging the database: %w", err)
	}

	if _, err := db.Exec(schemaSQL); err != nil {
		return nil, fmt.Errorf("error executing schema migrations: %w", err)
	}

	log.Println("SQLite database initialized via embedded schema.")
	return db, nil
}
