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
	Port             string `validate:"required,number,min=1,max=65535"`
	DBHost           string `validate:"required"`
	DBPort           string `validate:"required,number,min=1,max=65535"`
	DBUser           string `validate:"required"`
	DBPassword       string `validate:"required"`
	DBName           string `validate:"required"`
	S3Endpoint       string `validate:"required,url"`
	S3AccessKey      string `validate:"required"`
	S3SecretKey      string `validate:"required"`
	S3UseSSL         bool
	AuthServiceURL   string   `validate:"required,url"`
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
		Port:             common.GetEnv("PORT", "8085"),
		DBHost:           common.GetEnvRequired("DB_HOST"),
		DBPort:           common.GetEnvRequired("DB_PORT"),
		DBUser:           common.GetEnvRequired("DB_USER"),
		DBPassword:       common.GetEnvRequired("DB_PASSWORD"),
		DBName:           common.GetEnvRequired("DB_NAME"),
		S3Endpoint:       common.GetEnvRequired("S3_ENDPOINT"),
		S3AccessKey:      common.GetEnvRequired("S3_ACCESS_KEY"),
		S3SecretKey:      common.GetEnvRequired("S3_SECRET_KEY"),
		S3UseSSL:         common.GetEnvBool("S3_USE_SSL", false),
		AuthServiceURL:   common.GetEnvRequired("AUTH_SERVICE_URL"),
		MaxFileSize:      maxFileSize,
		AllowedFileTypes: allowedTypes,
	}

	// Validate configuration
	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		panic(fmt.Sprintf("Invalid configuration: %v", err))
	}

	return cfg
}
