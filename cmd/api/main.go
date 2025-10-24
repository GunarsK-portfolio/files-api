package main

import (
	"fmt"
	"log"
	"os"

	_ "github.com/GunarsK-portfolio/files-api/docs"
	"github.com/GunarsK-portfolio/files-api/internal/config"
	"github.com/GunarsK-portfolio/files-api/internal/handlers"
	"github.com/GunarsK-portfolio/files-api/internal/repository"
	"github.com/GunarsK-portfolio/files-api/internal/routes"
	"github.com/GunarsK-portfolio/files-api/internal/storage"
	commondb "github.com/GunarsK-portfolio/portfolio-common/database"
	"github.com/GunarsK-portfolio/portfolio-common/logger"
	"github.com/GunarsK-portfolio/portfolio-common/metrics"
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
	cfg := config.Load()

	appLogger := logger.New(logger.Config{
		Level:       os.Getenv("LOG_LEVEL"),
		Format:      os.Getenv("LOG_FORMAT"),
		ServiceName: "files-api",
		AddSource:   os.Getenv("LOG_SOURCE") == "true",
	})

	appLogger.Info("Starting files API", "version", "1.0")

	metricsCollector := metrics.New(metrics.Config{
		ServiceName: "files",
		Namespace:   "portfolio",
	})

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
		appLogger.Error("Failed to connect to database", "error", err)
		log.Fatal("Failed to connect to database:", err)
	}
	appLogger.Info("Database connection established")

	stor, err := storage.New(cfg)
	if err != nil {
		appLogger.Error("Failed to initialize storage", "error", err)
		log.Fatal("Failed to initialize storage:", err)
	}
	appLogger.Info("Storage initialized")

	repo := repository.New(db)
	handler := handlers.New(repo, stor, cfg)

	router := gin.New()
	router.Use(logger.Recovery(appLogger))
	router.Use(logger.RequestLogger(appLogger))
	router.Use(metricsCollector.Middleware())

	routes.Setup(router, handler, cfg, metricsCollector)

	appLogger.Info("Files API ready", "port", cfg.Port, "environment", os.Getenv("ENVIRONMENT"))
	if err := router.Run(fmt.Sprintf(":%s", cfg.Port)); err != nil {
		appLogger.Error("Failed to start server", "error", err)
		log.Fatal("Failed to start server:", err)
	}
}
