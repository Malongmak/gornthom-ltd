package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/gornhom/backend/config"
	"github.com/gornhom/backend/db"
	"github.com/gornhom/backend/routes"
	"github.com/gornhom/backend/services"
)

func main() {
	godotenv.Load()

	cfg := config.Load()
	db.Init()

	if cfg.NodeEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()
	rs := services.NewRouterService(cfg)

	routes.Register(r, cfg, rs)

	log.Printf("🚀 GORNHOM Backend (Go/Gin) starting on port %s", cfg.Port)
	log.Printf("🌐 Router Type: %s", cfg.RouterType)
	log.Printf("📋 Environment: %s", cfg.NodeEnv)

	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
