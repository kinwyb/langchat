# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

LangChat is a Go-based LLM chat agent framework with a plugin-style skills system. It provides two types of agents:
- **TextChatAgent**: Simple conversation manager with message history and skill integration
- **ReactAgent**: Advanced ReAct (Reasoning and Acting) agent using LangGraph for iterative tool-based problem solving

The project uses LangChain Go for LLM integration and supports both OpenAI-compatible APIs and local models (like Ollama).

## Commands

```bash
# Install/update dependencies
go mod tidy

# Run all tests
go test ./...

# Run tests in a specific package
go test ./llm/agent
go test ./llm/skills

# Run a specific test
go test -run TestNewTextChatAgent

# Build the project
go build .

# Run API server example
go run examples/server/main.go

# Install frontend dependencies
cd ui && npm install

# Run frontend development server
cd ui && npm run dev

# Build frontend for production
cd ui && npm run build
```

## Architecture

### Core Components

**llm/agent/**
- `agent.go`: Agent interface and configuration options (WithSkill, WithMCP)
- `textChat.go`: TextChatAgent - maintains conversation history, selects and executes skills via LLM
- `reactAgent.go`: ReactAgent - implements ReAct pattern with LangGraph state management, agent/tool nodes, conditional edges

**llm/skills/**
- `skills.go`: Skill loading and management (LoadSkills loads from directory)
- `parser.go`: Parses skill packages (supports Claude SKILL.md with YAML frontmatter and OpenAI skill.md)
- `tool.go`: Converts skill packages into LangChain Tool definitions

**llm/tools/**
- Base tools: file operations (read/write), shell execution, Python execution, web search (Tavily, Wikipedia), web fetching
- Tools implement LangChain's `tools.Tool` interface

**api/**
- `server.go`: HTTP server with CORS middleware, graceful shutdown
- `handler.go`: HTTP handlers for chat endpoints (Chat, ChatStream, HealthCheck)
- `model.go`: Request/response data models
- `handler_test.go`: Unit tests with mock agent
- See `examples/server/main.go` for usage example

**ui/** - Vue 3 frontend with DeepSeek-style design
- `src/components/Sidebar.vue` - Sidebar with chat history
- `src/components/Message.vue` - Message display with Markdown rendering
- `src/components/ChatInput.vue` - Input component with streaming support
- `src/api.js` - API client with SSE support
- `src/App.vue` - Main application component
- See `ui/README.md` for frontend setup

### Key Architectural Patterns

1. **Skill Selection**: LLM-based skill selection using JSON prompts. The LLM decides which skill (if any) to use based on user input and skill descriptions.

2. **ReAct Pattern**: ReactAgent uses LangGraph for stateful execution:
   - Agent node: LLM decides whether to call tools
   - Tools node: Executes tool calls
   - Conditional edge: Routes between agent/tools based on tool calls in the message
   - Iteration limit prevents infinite loops

3. **Async Tool Loading**: Both agents initialize tools asynchronously in the background to prevent blocking startup (see `InitializeToolsAsync` in TextChatAgent)

4. **Streaming Support**: All agents support streaming responses via `ChatStream` interface

5. **Skill Execution**: When a skill is selected, it creates a ReactAgent with the skill's tools and prompt, then executes the task (see `skillDoTask` in reactAgent.go:410)

### Tool Modes

ReactAgent supports two tool invocation modes:
- **Native tool calling** (`supportTool=true`): Passes tools as OpenAI function definitions to the LLM
- **JSON-based tool calling** (`supportTool=false`): LLM outputs JSON with tool name and arguments, which are then parsed and executed

### Configuration

Agents are created with functional options:
```go
agent.NewTextChatAgent(llm, agent.WithSkill("./skills"))
agent.NewReactAgent(model, systemPrompt,
    ReactWithTools(tools),
    ReactWithMaxIterations(5),
    ReactWithSupportTool(true),
)
```

### HTTP API

The `api` package provides a REST API for agent interaction:

**Endpoints:**
- `GET /health` - Health check
- `POST /api/chat` - Non-streaming chat
- `POST /api/chat/stream` - Streaming chat (SSE)

**Usage:**
```go
server := api.NewServer(textAgent, api.DefaultServerConfig())
server.Start()
```

**Request format:**
```json
{
  "message": "Your message here",
  "enableSkills": true,
  "enableMCP": false
}
```

## Testing

Tests use a local Ollama instance at `http://localhost:11434/v1` with model `gemma3:12b`. The test in `llm/agent/textChat_test.go` demonstrates agent initialization and usage.

## Dependencies

- `github.com/tmc/langchaingo` - LangChain Go for LLM integration
- `github.com/sashabaranov/go-openai` - OpenAI client
- `github.com/smallnest/langgraphgo` - LangGraph for agent workflows
- `github.com/PuerkitoBio/goquery` - HTML parsing
- `gopkg.in/yaml.v3` - YAML parsing for skill metadata
- `github.com/modelcontextprotocol/go-sdk` - MCP support (currently disabled)

### Frontend Stack
- **Vue 3** - Progressive JavaScript framework
- **Vite** - Fast build tool and dev server
- **Marked** - Markdown parser
- **Highlight.js** - Code syntax highlighting
- **Axios** - HTTP client
