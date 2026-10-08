// database connection
package db

import (
	"fmt"
	"log"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

func Connect(dataSourceName string) (*sqlx.DB, error) {
	var lastErr error // variable to store the last error encountered during connection attempts

	for attempts := 0; attempts < 15; attempts++ { // attempt to connect to the database up to 15 times
		db, err := sqlx.Connect("pgx", dataSourceName) // attempt to connect to the database
		if err == nil {
			// successful connection
			log.Println("database connected successfully")
			return db, nil
		}

		lastErr = err // store the last error encountered
		log.Printf("failed to connect to database (attempt %d/15): %v", attempts+1, err) // log the error and the attempt number
		time.Sleep(2 * time.Second) // wait for 2 seconds before retrying
	}

	return nil, fmt.Errorf("failed to connect to database after retries: %w", lastErr) // return an error if all attempts fail, wrapping the last encountered error
}
