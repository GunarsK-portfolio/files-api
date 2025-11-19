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
	"github.com/GunarsK-portfolio/portfolio-common/audit"
	commondb "github.com/GunarsK-portfolio/portfolio-common/database"
	"github.com/GunarsK-portfolio/portfolio-common/logger"
	"github.com/GunarsK-portfolio/portfolio-common/metrics"
	commonrepo "github.com/GunarsK-portfolio/portfolio-common/repository"
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

	//nolint:staticcheck // Embedded field name required due to ambiguous fields
	db, err := commondb.Connect(commondb.PostgresConfig{
		Host:     cfg.DatabaseConfig.Host,
		Port:     cfg.DatabaseConfig.Port,
		User:     cfg.DatabaseConfig.User,
		Password: cfg.DatabaseConfig.Password,
		DBName:   cfg.DatabaseConfig.Name,
		SSLMode:  cfg.DatabaseConfig.SSLMode,
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
	actionLogRepo := commonrepo.NewActionLogRepository(db)
	handler := handlers.New(repo, stor, cfg, actionLogRepo)

	router := gin.New()
	router.Use(logger.Recovery(appLogger))
	router.Use(logger.RequestLogger(appLogger))
	router.Use(audit.ContextMiddleware())
	router.Use(metricsCollector.Middleware())

	routes.Setup(router, handler, cfg, metricsCollector)

	appLogger.Info("Files API ready", "port", cfg.ServiceConfig.Port, "environment", os.Getenv("ENVIRONMENT"))
	if err := router.Run(fmt.Sprintf(":%s", cfg.ServiceConfig.Port)); err != nil {
		appLogger.Error("Failed to start server", "error", err)
		log.Fatal("Failed to start server:", err)
	}
}
