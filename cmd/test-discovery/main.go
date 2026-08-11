package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid" // Or generate a simple string ID if uuid package isn't pulled
	"github.com/akryptic/p2pdrop/internal/discovery"
)

func main() {
	// 1. Generate a unique Instance ID for this running process
	instanceID := uuid.New().String()

	// 2. Read Device Name from terminal args, or default to OS Hostname
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "Unknown-Device"
	}

	if len(os.Args) > 1 {
		hostname = os.Args[1] // Custom device name if passed: go run main.go "Laptop-2"
	}

	// Ports configuration
	httpPort := uint16(6969)
	udpDiscoveryPort := uint16(9696)

	log.Printf("🚀 Starting P2P Drop Discovery Test Node...")
	log.Printf("Name: %s | UUID: %s", hostname, instanceID)

	// 3. Initialize & Start Discovery Engine
	engine := discovery.NewEngine(instanceID, hostname, httpPort, udpDiscoveryPort)
	if err := engine.Start(); err != nil {
		log.Fatalf("Fatal error starting engine: %v", err)
	}
	defer engine.Stop()

	// 4. Background goroutine to periodically print currently discovered peers list
	go func() {
		ticker := time.NewTicker(6 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			peers := engine.GetPeers()
			fmt.Println("\n================ 👥 DISCOVERED PEERS LIST ================")
			if len(peers) == 0 {
				fmt.Println("  (No other peers found on local network yet)")
			} else {
				for i, p := range peers {
					fmt.Printf("  [%d] Device: %-15s | IP: %-15s | HTTP Port: %d | ID: %s\n",
						i+1, p.DeviceName, p.IP, p.HTTPPort, p.ID[:8])
				}
			}
			fmt.Println("==========================================================")
		}
	}()

	// 5. Block main thread until SIGINT / SIGTERM (Ctrl+C)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("\nCleaning up and exiting...")
}