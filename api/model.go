package api

// ChatRequest represents a chat request
type ChatRequest struct {
	Message      string `json:"message"`
	EnableSkills bool   `json:"enableSkills"`
	EnableMCP    bool   `json:"enableMCP"`
}

// ChatResponse represents a chat response
type ChatResponse struct {
	Response string `json:"response"`
	Error    string `json:"error,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// HealthResponse represents a health check response
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}
