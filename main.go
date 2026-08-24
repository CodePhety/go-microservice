package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"user-service/internal/config"
	"user-service/internal/metrics"
	"user-service/internal/server"
	"user-service/internal/user"
)

// main is the application entrypoint.
// It wires together configuration, the domain service, the HTTP router, and the
// process lifecycle so the application can start cleanly and shut down gracefully.
func main() {
	appConfig := config.Load()
	logger := log.New(os.Stdout, "user-service ", log.LstdFlags|log.LUTC)
	telemetry := metrics.New()

	// The repository is intentionally in-memory because the goal of this project is
	// to teach architecture and interview reasoning without introducing a database.
	// In a production system, a persistent store like PostgreSQL would replace this.
	repository := user.NewInMemoryRepository(logger)
	service := user.NewService(repository, logger, telemetry)
	api := server.New(service, appConfig, logger, telemetry)

	serverAddress := fmt.Sprintf(":%d", appConfig.Port)
	httpServer := &http.Server{
		Addr:              serverAddress,
		Handler:           api.Router(),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	// The application listens for OS signals before it attempts to exit, which is
	// a senior design habit because it allows in-flight requests to finish cleanly.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// This goroutine starts the HTTP server and keeps the process alive while the
	// app is running. The main goroutine then waits for shutdown signals.
	go func() {
		logger.Printf("starting user service on %s in %s mode", serverAddress, appConfig.Environment)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("http server failed: %v", err)
		}
	}()

	<-ctx.Done()
	logger.Println("shutdown signal received; stopping server gracefully")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Printf("graceful shutdown failed: %v", err)
		os.Exit(1)
	}

	logger.Println("service stopped cleanly")
}
