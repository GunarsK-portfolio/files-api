package routes

import (
	"github.com/GunarsK-portfolio/files-api/internal/config"
	"github.com/GunarsK-portfolio/files-api/internal/handlers"
	"github.com/GunarsK-portfolio/files-api/internal/middleware"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func Setup(router *gin.Engine, handler *handlers.Handler, cfg *config.Config) {
	// Enable CORS
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Health check
	router.GET("/health", handler.HealthCheck)

	// Public routes (no auth)
	router.GET("/files/:fileType/*key", handler.DownloadFile)

	// Protected routes (JWT required)
	protected := router.Group("/")
	protected.Use(middleware.JWTAuth(cfg.JWTSecret))
	{
		protected.POST("/files", handler.UploadFile)
		protected.DELETE("/files/:id", handler.DeleteFile)
	}

	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
