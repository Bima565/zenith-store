package main

import (
	"context"
	"log"
	"time"

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
	steamgrid := services.NewSteamGridService(cfg)
	gameHandler := handlers.NewGameHandler(vipayment, steamgrid)
	orderHandler := handlers.NewOrderHandler(vipayment)

	// Pre-warm SteamGridDB icons (all games) + portrait grids (top 16 popular games).
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if list, err := vipayment.GetServices(ctx); err == nil {
			steamgrid.EnrichServices(ctx, list, 16)
			log.Printf("SteamGridDB cache warmed (all icons + top 16 popular grids)")
		}
	}()

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
