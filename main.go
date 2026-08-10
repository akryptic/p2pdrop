package main

import (
	"fmt"
	"log"

	"github.com/akryptic/p2pdrop/internal/config"
	"github.com/akryptic/p2pdrop/internal/server"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Failed to initialize config: %v", err)
	}

	srv := server.NewServer(cfg)

	fmt.Println("🚀 P2P Drop running at http://localhost:6969")
	if err := srv.Start(":6969"); err != nil {
		log.Fatalf("Server stopped: %v", err)
	}
}