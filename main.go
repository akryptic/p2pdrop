package main

import (
	"fmt"
	"log"

	"github.com/akryptic/p2pdrop/internal/config"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Config initialization failed: %v", err)
	}

	fmt.Printf("Config Initialized! Ready: %v | Device Name: %s\n", cfg.IsReady(), cfg.Get().DeviceName)
}