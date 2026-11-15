package config

import (
	"os"
)

type Config struct {
	DatabaseURL string
	ServerPort  string
	JWTSecret   string
	S3Endpoint  string
	S3AccessKey string
	S3SecretKey string
	S3Bucket    string
	S3UseSSL    bool
}

func Load() *Config {
	return &Config{
		DatabaseURL: getEnv("DATABASE_URL", "host=localhost user=stpkurs_app password=Qwezxc098 dbname=kurs port=5433 sslmode=disable"),
		ServerPort:  getEnv("SERVER_PORT", ":8080"),
		JWTSecret:   getEnv("JWT_SECRET", "BpR0cOjcNNiskIZu9ZtS3Q3o3M2RzNEEAQIZVJFX5uC"),
		S3Endpoint:  getEnv("S3_ENDPOINT", "localhost:9000"),
		S3AccessKey: getEnv("S3_ACCESS_KEY", "stpkurs-app"),
		S3SecretKey: getEnv("S3_SECRET_KEY", "Qwezxc098"),
		S3Bucket:    getEnv("S3_BUCKET", "event-media"),
		S3UseSSL:    getEnv("S3_USE_SSL", "false") == "true",
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
