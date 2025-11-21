package config

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"

	common "github.com/GunarsK-portfolio/portfolio-common/config"
)

type Config struct {
	common.DatabaseConfig
	common.ServiceConfig
	common.S3Config
	JWTSecret        string   `validate:"required,min=32"`
	MaxFileSize      int64    `validate:"gt=0"`
	AllowedFileTypes []string `validate:"required,min=1,dive,required"`
}

func Load() *Config {
	maxFileSizeStr := common.GetEnv("MAX_FILE_SIZE", "10485760") // 10MB default
	maxFileSize, err := strconv.ParseInt(maxFileSizeStr, 10, 64)
	if err != nil {
		log.Fatalf("Invalid MAX_FILE_SIZE value: %s", maxFileSizeStr)
	}

	allowedTypesStr := common.GetEnv("ALLOWED_FILE_TYPES", "image/jpeg,image/jpg,image/png,image/gif,image/webp,application/pdf")
	allowedTypes := strings.Split(allowedTypesStr, ",")
	for i := range allowedTypes {
		allowedTypes[i] = strings.TrimSpace(allowedTypes[i])
	}

	cfg := &Config{
		DatabaseConfig:   common.NewDatabaseConfig(),
		ServiceConfig:    common.NewServiceConfig("8085"),
		S3Config:         common.NewS3Config(),
		JWTSecret:        common.GetEnvRequired("JWT_SECRET"),
		MaxFileSize:      maxFileSize,
		AllowedFileTypes: allowedTypes,
	}

	// Validate service-specific fields
	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		panic(fmt.Sprintf("Invalid configuration: %v", err))
	}

	return cfg
}
