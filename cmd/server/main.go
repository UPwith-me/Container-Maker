package main

import (
	"log"

	"github.com/container-make/cm/cloud/api"
)

func main() {
	config := api.Config{
		Port:               8080,
		JWTSecret:          "dev-secret-key-123",
		GitHubClientID:     "mock-client-id",
		GitHubClientSecret: "mock-client-secret",
	}

	server := api.NewServer(config)

	log.Printf("🚀 Cloud Control Plane API running on port %d", config.Port)
	if err := server.Start(); err != nil {
		log.Fatal(err)
	}
}
