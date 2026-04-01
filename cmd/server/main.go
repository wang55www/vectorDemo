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

	"vectorDemo/internal/config"
	"vectorDemo/internal/handler"
	"vectorDemo/internal/mcp"
	"vectorDemo/internal/repository"
	"vectorDemo/internal/service"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize repository
	repo, err := repository.NewImageRepository(&cfg.TiDB)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer repo.Close()

	// Initialize database schema
	if err := repo.InitSchema(); err != nil {
		log.Fatalf("Failed to initialize schema: %v", err)
	}
	log.Println("Database schema initialized")

	// Initialize services
	embeddingSvc := service.NewEmbeddingService(&cfg.Jina)

	// Initialize handlers
	h := handler.NewHandler(repo, embeddingSvc)

	// Setup HTTP server
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: h,
	}

	// Initialize MCP server
	mcpServer := mcp.NewMCPServer(repo, embeddingSvc)

	// Start servers in goroutines
	go func() {
		log.Printf("HTTP server starting on port %d", cfg.Server.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	go func() {
		log.Printf("MCP server starting on port %d", cfg.MCPServer.Port)
		if err := mcpServer.Start(fmt.Sprintf(":%d", cfg.MCPServer.Port)); err != nil {
			log.Printf("MCP server failed: %v", err)
		}
	}()

	log.Println("Server started successfully")

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down servers...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	if err := mcpServer.Stop(); err != nil {
		log.Printf("MCP server shutdown error: %v", err)
	}

	log.Println("Servers stopped")
}