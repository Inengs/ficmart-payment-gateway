package main

import (
	"log"

	db "github.com/Inengs/ficmart-payment-gateway/internal/db"

	"github.com/Inengs/ficmart-payment-gateway/configs"
)

func main() {
	cfg := configs.Load()
	_, err := db.Connect(cfg.DatabaseURL) // change _ to database when you want to pass it down

	if err != nil {
		log.Fatalf("failed to connect to database, error: %s", err)
	}


}