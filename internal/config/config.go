package config

import (
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost             string
	DBPort             string
	DBUser             string
	DBPassword         string
	DBName             string
	GCSBucketName      string
	GCSCredentialsFile string
}

func NewConfig() *Config {
	// Only load .env if explicitly running in development mode
	if os.Getenv("ENV") == "development" {
		projectRoot, _ := os.Getwd()
		log.Println("Running in development mode, loading .env file")
		if err := godotenv.Load(filepath.Join(projectRoot, ".env")); err != nil {
			log.Println("Warning: .env file not found, using default values or env vars")
		}
	} else {
		log.Println("Running in production mode, using environment variables directly")
	}

	return &Config{
		DBHost:             getEnv("DB_HOST", "localhost"),
		DBPort:             getEnv("DB_PORT", "5432"),
		DBUser:             getEnv("DB_USER", "admin"),
		DBPassword:         getEnv("DB_PASSWORD", "password"),
		DBName:             getEnv("DB_NAME", "api"),
		GCSBucketName:      getEnv("GCS_BUCKET_NAME", "bucket_name"),
		GCSCredentialsFile: getEnv("GCS_CREDENTIALS_FILE", "credentials.json"),
	}
}

// getEnv retrieves an environment variable with a fallback value
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
