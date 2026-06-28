package internal

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
)

func Connect(dataSourceName string) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", dataSourceName)

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("database connected successfully")
	return db, nil
}