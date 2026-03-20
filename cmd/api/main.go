package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/FrontWorksDev/Loki/internal/api"
)

func main() {
	cfg := api.DefaultConfig()
	srv := api.NewServer(cfg)

	// シグナルによるGraceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := srv.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	fmt.Println("\nShutting down server...")

	if err := srv.Shutdown(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "Shutdown error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Server stopped gracefully.")
}
