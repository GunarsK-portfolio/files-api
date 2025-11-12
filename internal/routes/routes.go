package routes

import (
	"github.com/GunarsK-portfolio/files-api/internal/config"
	"github.com/GunarsK-portfolio/files-api/internal/handlers"
	"github.com/GunarsK-portfolio/portfolio-common/metrics"
	common "github.com/GunarsK-portfolio/portfolio-common/middleware"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func Setup(router *gin.Engine, handler *handlers.Handler, cfg *config.Config, metricsCollector *metrics.Metrics) {
	// Security middleware with CORS validation
	securityMiddleware := common.NewSecurityMiddleware(
		cfg.AllowedOrigins,
		"GET,POST,DELETE,OPTIONS",
		"Content-Type,Authorization",
		true,
	)
	router.Use(securityMiddleware.Apply())

	// Health check
	router.GET("/health", handler.HealthCheck)
	// Metrics
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Public routes (no auth)
		v1.GET("/files/:fileType/*key", handler.DownloadFile)

		// Protected routes (JWT required)
		authMiddleware := common.NewAuthMiddleware(cfg.AuthServiceURL)
		protected := v1.Group("/")
		protected.Use(authMiddleware.ValidateToken())
		{
			protected.POST("/files", handler.UploadFile)
			protected.DELETE("/files/:id", handler.DeleteFile)
		}
	}

	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
