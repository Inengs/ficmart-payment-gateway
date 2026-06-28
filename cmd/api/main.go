package main

import (
	"log"
	"net/http"

	"github.com/Inengs/ficmart-payment-gateway/configs"
	db "github.com/Inengs/ficmart-payment-gateway/internal/db"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := configs.Load()
	_, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database, error: %s", err)
	}

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Hello world",
		})
	})

	r.Run()
}