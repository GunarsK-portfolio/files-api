package config

import (
	"log"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port             string
	DBHost           string
	DBPort           string
	DBUser           string
	DBPassword       string
	DBName           string
	S3Endpoint       string
	S3AccessKey      string
	S3SecretKey      string
	S3UseSSL         bool
	AuthServiceURL   string
	MaxFileSize      int64
	AllowedFileTypes []string
}

func Load() *Config {
	maxFileSizeStr := getEnv("MAX_FILE_SIZE", "10485760") // 10MB default
	maxFileSize, err := strconv.ParseInt(maxFileSizeStr, 10, 64)
	if err != nil {
		log.Fatalf("Invalid MAX_FILE_SIZE value: %s", maxFileSizeStr)
	}

	allowedTypesStr := getEnv("ALLOWED_FILE_TYPES", "image/jpeg,image/jpg,image/png,image/gif,image/webp,application/pdf")
	allowedTypes := strings.Split(allowedTypesStr, ",")
	for i := range allowedTypes {
		allowedTypes[i] = strings.TrimSpace(allowedTypes[i])
	}

	cfg := &Config{
		Port:             getEnv("PORT", "8085"),
		DBHost:           getEnvRequired("DB_HOST"),
		DBPort:           getEnvRequired("DB_PORT"),
		DBUser:           getEnvRequired("DB_USER"),
		DBPassword:       getEnvRequired("DB_PASSWORD"),
		DBName:           getEnvRequired("DB_NAME"),
		S3Endpoint:       getEnvRequired("S3_ENDPOINT"),
		S3AccessKey:      getEnvRequired("S3_ACCESS_KEY"),
		S3SecretKey:      getEnvRequired("S3_SECRET_KEY"),
		S3UseSSL:         getEnvBool("S3_USE_SSL", false),
		AuthServiceURL:   getEnvRequired("AUTH_SERVICE_URL"),
		MaxFileSize:      maxFileSize,
		AllowedFileTypes: allowedTypes,
	}
	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvRequired(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("Required environment variable %s is not set", key)
	}
	return value
}

func getEnvBool(key string, defaultValue bool) bool {
	val := getEnv(key, "")
	if val == "" {
		return defaultValue
	}
	return strings.EqualFold(val, "true")
}
