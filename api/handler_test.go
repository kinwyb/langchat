package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kinwyb/langchat/llm/agent"
)

// mockAgent is a mock implementation of the Agent interface for testing
type mockAgent struct {
	chatResponse    string
	chatError       error
	streamChunks    []string
	streamError     error
	chunksSentCount int
}

func (m *mockAgent) Chat(ctx context.Context, message string, enableSkills bool, enableMCP bool) (string, error) {
	if m.chatError != nil {
		return "", m.chatError
	}
	return m.chatResponse, nil
}

func (m *mockAgent) ChatStream(ctx context.Context, message string, enableSkills bool, enableMCP bool, onChunk func(context.Context, []byte) error) (string, error) {
	if m.streamError != nil {
		return "", m.streamError
	}

	for _, chunk := range m.streamChunks {
		if err := onChunk(ctx, []byte(chunk)); err != nil {
			return "", err
		}
		m.chunksSentCount++
	}

	return strings.Join(m.streamChunks, ""), nil
}

func TestHealthCheck(t *testing.T) {
	handler := NewHandler(&mockAgent{})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.HealthCheck(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", resp.Status)
	}
}

func TestChat(t *testing.T) {
	tests := []struct {
		name         string
		agent        agent.Agent
		requestBody  string
		expectedCode int
		checkResp    func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:  "successful chat",
			agent: &mockAgent{chatResponse: "Hello! How can I help you?"},
			requestBody: `{
				"message": "Hello",
				"enableSkills": true,
				"enableMCP": false
			}`,
			expectedCode: http.StatusOK,
			checkResp: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp ChatResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if resp.Response != "Hello! How can I help you?" {
					t.Errorf("Expected response 'Hello! How can I help you?', got '%s'", resp.Response)
				}
			},
		},
		{
			name:         "empty message",
			agent:        &mockAgent{},
			requestBody:  `{"message": ""}`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "invalid JSON",
			agent:        &mockAgent{},
			requestBody:  `{invalid json}`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "agent error",
			agent:        &mockAgent{chatError: context.DeadlineExceeded},
			requestBody:  `{"message": "test"}`,
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(tt.agent)

			req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(tt.requestBody))
			w := httptest.NewRecorder()

			handler.Chat(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("Expected status %d, got %d", tt.expectedCode, w.Code)
			}

			if tt.checkResp != nil {
				tt.checkResp(t, w)
			}
		})
	}
}

func TestChatStream(t *testing.T) {
	tests := []struct {
		name        string
		agent       agent.Agent
		requestBody string
		checkResp   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "successful stream",
			agent: &mockAgent{
				streamChunks: []string{"Hello", " ", "World", "!"},
			},
			requestBody: `{"message": "Hello"}`,
			checkResp: func(t *testing.T, w *httptest.ResponseRecorder) {
				body := w.Body.String()
				if !strings.Contains(body, "data: Hello") {
					t.Errorf("Expected 'data: Hello' in response, got: %s", body)
				}
				if !strings.Contains(body, "event: done") {
					t.Errorf("Expected 'event: done' in response, got: %s", body)
				}
				if !strings.Contains(body, "event: end") {
					t.Errorf("Expected 'event: end' in response, got: %s", body)
				}
			},
		},
		{
			name:        "empty message",
			agent:       &mockAgent{},
			requestBody: `{"message": ""}`,
			checkResp: func(t *testing.T, w *httptest.ResponseRecorder) {
				body := w.Body.String()
				if !strings.Contains(body, "event: error") {
					t.Errorf("Expected error event in response, got: %s", body)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(tt.agent)

			req := httptest.NewRequest(http.MethodPost, "/api/chat/stream", strings.NewReader(tt.requestBody))
			w := httptest.NewRecorder()

			handler.ChatStream(w, req)

			if tt.checkResp != nil {
				tt.checkResp(t, w)
			}
		})
	}
}

func TestMethodNotAllowed(t *testing.T) {
	handler := NewHandler(&mockAgent{})

	tests := []struct {
		name      string
		method    string
		path      string
		handlerFn func(http.ResponseWriter, *http.Request)
	}{
		{"health with POST", http.MethodPost, "/health", handler.HealthCheck},
		{"chat with GET", http.MethodGet, "/api/chat", handler.Chat},
		{"stream with GET", http.MethodGet, "/api/chat/stream", handler.ChatStream},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			tt.handlerFn(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405, got %d", w.Code)
			}
		})
	}
}
