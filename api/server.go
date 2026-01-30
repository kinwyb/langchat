package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/kinwyb/langchat/llm/agent"
)

// Server represents an HTTP server for the chat API
type Server struct {
	httpServer *http.Server
	handler    *Handler
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Host string
	Port int
}

// DefaultServerConfig returns default server configuration
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		Host: "0.0.0.0",
		Port: 8080,
	}
}

// NewServer creates a new HTTP server for the chat API
func NewServer(a agent.Agent, cfg ServerConfig) *Server {
	handler := NewHandler(a)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", handler.HealthCheck)
	mux.HandleFunc("/api/chat", handler.Chat)
	mux.HandleFunc("/api/chat/stream", handler.ChatStream)

	// Add CORS middleware
	corsMux := corsMiddleware(mux)

	return &Server{
		httpServer: &http.Server{
			Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
			Handler:      corsMux,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 2 * time.Minute,
			IdleTimeout:  120 * time.Second,
		},
		handler: handler,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("Starting server on %s", s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server failed: %w", err)
	}
	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down server...")
	return s.httpServer.Shutdown(ctx)
}

// corsMiddleware adds CORS headers to all responses
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
