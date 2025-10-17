package main

import (
	"fmt"
	"log"

	_ "github.com/GunarsK-portfolio/files-api/docs"
	"github.com/GunarsK-portfolio/files-api/internal/config"
	"github.com/GunarsK-portfolio/files-api/internal/handlers"
	"github.com/GunarsK-portfolio/files-api/internal/repository"
	"github.com/GunarsK-portfolio/files-api/internal/routes"
	"github.com/GunarsK-portfolio/files-api/internal/storage"
	commondb "github.com/GunarsK-portfolio/portfolio-common/database"
	"github.com/gin-gonic/gin"
)

// @title Portfolio Files API
// @version 1.0
// @description File upload/download service for portfolio - handles S3/MinIO storage operations
// @host localhost:8085
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	// Load configuration
	cfg := config.Load()

	// Connect to database
	db, err := commondb.Connect(commondb.PostgresConfig{
		Host:     cfg.DBHost,
		Port:     cfg.DBPort,
		User:     cfg.DBUser,
		Password: cfg.DBPassword,
		DBName:   cfg.DBName,
		SSLMode:  "disable",
		TimeZone: "UTC",
	})
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
