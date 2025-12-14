package main

import (
	"log"
	"os"

	"github.com/container-make/cm/cloud/api"
)

func main() {
	config := api.Config{
		Port:      8080,
		JWTSecret: getEnv("JWT_SECRET", "dev-secret-key-change-in-production"),

		// OAuth (optional - will work without)
		GitHubClientID:     getEnv("GITHUB_CLIENT_ID", ""),
		GitHubClientSecret: getEnv("GITHUB_CLIENT_SECRET", ""),
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),

		// Database
		DatabaseDriver: getEnv("DB_DRIVER", "sqlite"),
		DatabaseURL:    getEnv("DATABASE_URL", ""),

		// Stripe
		StripeSecretKey: getEnv("STRIPE_SECRET_KEY", ""),
	}

	server, err := api.NewServer(config)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	log.Printf("🚀 Cloud Control Plane API running on port %d", config.Port)
	log.Printf("📦 Database: %s", config.DatabaseDriver)
	log.Printf("🔗 Dashboard: http://localhost:%d", config.Port)

	if err := server.Start(); err != nil {
		log.Fatal(err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
