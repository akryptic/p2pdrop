package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/akryptic/p2pdrop/internal/config"
	"github.com/akryptic/p2pdrop/internal/server"
	"github.com/pkg/browser"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Failed to initialize config: %v", err)
	}

	srv := server.NewServer(cfg)

	// Channel to listen for OS interrupt signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	serverURL := "http://localhost:6969"

	// Run HTTP server in a background goroutine
	go func() {
		fmt.Printf("🚀 P2P Drop running at %s\n", serverURL)

		// Open default browser right before blocking on ListenAndServe
		if err := browser.OpenURL(serverURL); err != nil {
			log.Printf("Failed to open browser automatically: %v", err)
		}

		if err := srv.Start(":6969"); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server unexpected crash: %v", err)
		}
	}()

	// Block until SIGINT or SIGTERM signal is received
	<-stop
	fmt.Println("\n⌛ Shutting down P2P Drop gracefully...")

	// Create a 5-second context timeout for active requests to complete
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown with error: %v", err)
	}

	fmt.Println("👋 Server stopped cleanly.")
}
