package config

import (
	"os"
)

type Config struct {
	Port        string
	DatabaseURL string
	RedisURL    string
	DBHost      string
	DBName      string
	DBUser      string
	DBPassword  string
	DBPort      string
	DBSchema    string
	JWTSecret   string
}

func Load() *Config {
	return &Config{
		Port:        getEnv("PORT", "3000"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://aquaflow:aquaflow_dev@localhost:5432/aquaflowdb?sslmode=disable&search_path=aquaflow"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379"),
		DBHost:      getEnv("DB_HOST", "localhost"),
		DBName:      getEnv("DB_NAME", "aquaflowdb"),
		DBUser:      getEnv("DB_USER", "aquaflow"),
		DBPassword:  getEnv("DB_PASSWORD", "aquaflow_dev"),
		DBPort:      getEnv("DB_PORT", "5432"),
		DBSchema:    getEnv("DB_SCHEMA", "aquaflow"),
		JWTSecret:   getEnv("JWT_SECRET", "your-secret-key-here"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}