package routes

import (
	"log"

	"github.com/GunarsK-portfolio/files-api/docs"
	"github.com/GunarsK-portfolio/files-api/internal/config"
	"github.com/GunarsK-portfolio/files-api/internal/handlers"
	"github.com/GunarsK-portfolio/portfolio-common/health"
	"github.com/GunarsK-portfolio/portfolio-common/jwt"
	"github.com/GunarsK-portfolio/portfolio-common/metrics"
	common "github.com/GunarsK-portfolio/portfolio-common/middleware"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func Setup(router *gin.Engine, handler *handlers.Handler, cfg *config.Config, metricsCollector *metrics.Metrics, healthAgg *health.Aggregator) {
	// Security middleware with CORS validation
	securityMiddleware := common.NewSecurityMiddleware(
		cfg.AllowedOrigins,
		"GET,POST,DELETE,OPTIONS",
		"Content-Type,Authorization",
		true,
	)
	router.Use(securityMiddleware.Apply())

	// Health check
	router.GET("/health", healthAgg.Handler())
	// Metrics
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Public routes (no auth)
		v1.GET("/files/:fileType/*key", handler.DownloadFile)

		// Protected routes (JWT required)
		jwtService, err := jwt.NewValidatorOnly(cfg.JWTSecret)
		if err != nil {
			log.Fatalf("Failed to create JWT service: %v", err)
		}
		authMiddleware := common.NewAuthMiddleware(jwtService)
		protected := v1.Group("/")
		protected.Use(authMiddleware.ValidateToken())
		{
			protected.POST("/files", common.RequirePermission(common.ResourceFiles, common.LevelEdit), handler.UploadFile)
			protected.DELETE("/files/:id", common.RequirePermission(common.ResourceFiles, common.LevelDelete), handler.DeleteFile)
		}
	}

	// Swagger documentation (only if SWAGGER_HOST is configured)
	if cfg.SwaggerHost != "" {
		docs.SwaggerInfo.Host = cfg.SwaggerHost
		router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}
}
