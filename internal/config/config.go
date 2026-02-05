package config

import (
	"os"
)

type Config struct {
	DatabaseURL string
	ServerPort  string
	JWTSecret   string
}

func Load() *Config {
	return &Config{
		DatabaseURL: getEnv("DATABASE_URL", "host=localhost user=max password=123456 dbname=kurs port=5432 sslmode=disable"),
		ServerPort:  getEnv("SERVER_PORT", ":8080"),
		JWTSecret:   getEnv("JWT_SECRET", "BpR0cOjcNNiskIZu9ZtS3Q3o3M2RzNEEAQIZVJFX5uC"),
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
