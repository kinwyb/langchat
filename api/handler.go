package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/kinwyb/langchat/llm/agent"
)

// Handler handles HTTP requests for the chat agent
type Handler struct {
	agent agent.Agent
}

// NewHandler creates a new HTTP handler
func NewHandler(a agent.Agent) *Handler {
	return &Handler{
		agent: a,
	}
}

// HealthCheck handles health check requests
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendErrorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	response := HealthResponse{
		Status:  "ok",
		Version: "1.0.0",
	}
	sendJSONResponse(w, http.StatusOK, response)
}

// Chat handles regular (non-streaming) chat requests
func (h *Handler) Chat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	if req.Message == "" {
		sendErrorResponse(w, http.StatusBadRequest, "message is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	response, err := h.agent.Chat(ctx, req.Message, req.EnableSkills, req.EnableMCP)
	if err != nil {
		log.Printf("Chat error: %v", err)
		sendErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("chat failed: %v", err))
		return
	}

	sendJSONResponse(w, http.StatusOK, ChatResponse{
		Response: response,
	})
}

// ChatStream handles streaming chat requests using Server-Sent Events (SSE)
func (h *Handler) ChatStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendSSEError(w, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	if req.Message == "" {
		sendSSEError(w, "message is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	// Flusher ensures SSE data is sent immediately
	flusher, ok := w.(http.Flusher)
	if !ok {
		log.Printf("Streaming not supported")
		sendSSEError(w, "streaming not supported")
		return
	}

	// Stream chunks
	chunkCount := 0
	response, err := h.agent.ChatStream(ctx, req.Message, req.EnableSkills, req.EnableMCP,
		func(ctx context.Context, chunk []byte) error {
			// Send SSE event
			chunkStr := string(chunk)
			fmt.Fprintf(w, "data: %s\n\n", chunkStr)
			flusher.Flush()
			chunkCount++
			if chunkCount <= 5 {
				log.Printf("Sent chunk #%d: %q", chunkCount, chunkStr)
			}
			return nil
		})

	if err != nil {
		log.Printf("Chat stream error: %v", err)
		sendSSEError(w, fmt.Sprintf("chat failed: %v", err))
		return
	}

	// Send final response
	sendSSEEvent(w, "done", response)
	sendSSEEvent(w, "end", "")
}

// sendJSONResponse sends a JSON response
func sendJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

// sendErrorResponse sends an error response
func sendErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error: message,
	})
}

// sendSSEEvent sends an SSE event
func sendSSEEvent(w http.ResponseWriter, event string, data string) {
	fmt.Fprintf(w, "event: %s\n", event)
	fmt.Fprintf(w, "data: %s\n\n", data)
}

// sendSSEError sends an error via SSE
func sendSSEError(w http.ResponseWriter, message string) {
	sendSSEEvent(w, "error", message)
}

// StreamReader wraps an io.Reader for SSE streaming
type StreamReader struct {
	io.Reader
}
