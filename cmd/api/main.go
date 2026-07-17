// starts the server, wires everything together
package main

import (
	"log"
	"net/http"

	"github.com/Inengs/ficmart-payment-gateway/configs"
	"github.com/Inengs/ficmart-payment-gateway/internal/client"
	db "github.com/Inengs/ficmart-payment-gateway/internal/db"
	"github.com/Inengs/ficmart-payment-gateway/internal/handler"
	"github.com/Inengs/ficmart-payment-gateway/internal/repository"
	"github.com/Inengs/ficmart-payment-gateway/internal/service"
	"github.com/Inengs/ficmart-payment-gateway/pkg/logger"
	"github.com/gin-gonic/gin"
)

func main() {
	logger.Init()

	cfg := configs.Load()
	conn, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		logger.Logger.Error("failed to connect to database", "error", err)
		log.Fatalf("failed to connect to database, error: %s", err)
	}
	logger.Logger.Info("database connected successfully")

	bankClient := client.NewBankClient(cfg.BankBaseURL)
	repo := repository.NewPaymentRepository(conn)
	svc := service.NewPaymentService(repo, bankClient)
	h := handler.NewPaymentHandler(svc)

	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Hello world"})
	})
	r.POST("/payments/authorize", h.Authorize)

	logger.Logger.Info("server starting", "port", cfg.Port)
	r.Run(":" + cfg.Port)
}
