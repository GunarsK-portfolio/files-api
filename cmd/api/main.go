package main

import (
	"fmt"
	"log"

	_ "github.com/GunarsK-portfolio/files-api/docs"
	"github.com/GunarsK-portfolio/files-api/internal/config"
	"github.com/GunarsK-portfolio/files-api/internal/database"
	"github.com/GunarsK-portfolio/files-api/internal/handlers"
	"github.com/GunarsK-portfolio/files-api/internal/repository"
	"github.com/GunarsK-portfolio/files-api/internal/routes"
	"github.com/GunarsK-portfolio/files-api/internal/storage"
	"github.com/gin-gonic/gin"
)

// @title Portfolio Files API
// @version 1.0
// @description File upload/download service for portfolio - handles S3/MinIO storage operations
// @host localhost:8085
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	// Load configuration
	cfg := config.Load()

	// Connect to database
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Initialize storage
	stor, err := storage.New(cfg)
	if err != nil {
		log.Fatal("Failed to initialize storage:", err)
	}

	// Initialize repository
	repo := repository.New(db)

	// Initialize handlers
	handler := handlers.New(repo, stor, cfg)

	// Setup router
	router := gin.Default()

	// Setup routes
	routes.Setup(router, handler, cfg)

	// Start server
	log.Printf("Starting files API on port %s", cfg.Port)
	if err := router.Run(fmt.Sprintf(":%s", cfg.Port)); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
