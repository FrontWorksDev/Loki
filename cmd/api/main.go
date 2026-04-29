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
	cfg, err := api.LoadConfig(api.LoadConfigOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "設定の読み込みに失敗しました: %v\n", err)
		os.Exit(1)
	}

	srv := api.NewServer(cfg)

	// シグナルによるGraceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)

	go func() {
		if err := srv.Start(); err != nil {
			errCh <- err
		}
	}()

	exitCode := 0

	select {
	case <-ctx.Done():
		fmt.Println("\nShutting down server...")
	case err := <-errCh:
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		fmt.Println("\nShutting down server...")
		exitCode = 1
	}

	if err := srv.Shutdown(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "Shutdown error: %v\n", err)
		exitCode = 1
	} else {
		fmt.Println("Server stopped gracefully.")
	}

	if exitCode != 0 {
		os.Exit(exitCode)
	}
}
