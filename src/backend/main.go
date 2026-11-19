package main

import (
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"zenith-store/backend/config"
	"zenith-store/backend/handlers"
	"zenith-store/backend/services"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = cfg.CORSOrigins
	corsConfig.AllowMethods = []string{"GET", "POST", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}
	router.Use(cors.New(corsConfig))

	vipayment := services.NewVIPaymentService(cfg)
	gameHandler := handlers.NewGameHandler(vipayment)
	orderHandler := handlers.NewOrderHandler(vipayment)

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	api := router.Group("/api")
	{
		api.GET("/services", gameHandler.GetServices)
		api.GET("/nickname", orderHandler.GetNickname)
		api.POST("/checkout", orderHandler.Checkout)
		api.GET("/orders/:trxid/status", orderHandler.GetStatus)
	}
	router.POST("/webhook/vipayment", orderHandler.Webhook)

	log.Printf("Backend listening on http://localhost:%s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("start server: %v", err)
	}
}
