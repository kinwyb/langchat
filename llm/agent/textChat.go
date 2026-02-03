package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/kinwyb/langchat/llm/skills"
	mcpclient "github.com/smallnest/goskills/mcp"
	"github.com/smallnest/langgraphgo/adapter/mcp"
	"github.com/smallnest/langgraphgo/prebuilt"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
)

// TextChatAgent manages conversation history for a session
type TextChatAgent struct {
	llm           llms.Model
	messages      []llms.MessageContent
	mu            sync.RWMutex
	mcpClient     *mcpclient.Client
	mcpTools      []tools.Tool
	skills        []*skills.Skill
	cfg           *config
	selectedSkill string // Currently selected skill name
	toolsEnabled  bool
	toolsLoading  bool // true when tools are being loaded asynchronously
	toolsLoaded   bool // true when tools have finished loading
}

// NewTextChatAgent creates a text chat agent
func NewTextChatAgent(llm llms.Model, opts ...Option) *TextChatAgent {
	// Add system message
	systemMsg := llms.MessageContent{
		Role:  llms.ChatMessageTypeSystem,
		Parts: []llms.ContentPart{llms.TextPart("You are a helpful AI assistant. Be concise and friendly.")},
	}

	agent := &TextChatAgent{
		llm:      llm,
		messages: []llms.MessageContent{systemMsg},
		cfg:      &config{},
	}
	for _, opt := range opts {
		opt(agent.cfg)
	}
	agent.InitializeToolsAsync()
	return agent
}

// InitializeToolsAsync asynchronously loads Skills and MCP tools in the background
// This prevents blocking server startup while tools are being loaded
func (a *TextChatAgent) InitializeToolsAsync() {
	// Mark as loading
	a.mu.Lock()
	a.toolsLoading = true
	a.toolsLoaded = false
	a.mu.Unlock()

	defer func() {
		// Mark as loaded regardless of success/failure to prevent blocking
		a.mu.Lock()
		a.toolsLoading = false
		a.toolsLoaded = true
		skillsCount := len(a.skills)
		mcpToolsCount := len(a.mcpTools)
		a.mu.Unlock()
		log.Printf("✓ Tools pre-warming complete: %d Skills, %d MCP tools loaded", skillsCount, mcpToolsCount)
	}()

	log.Println("Starting background tools initialization...")

	// Add recovery for any panics during tool loading
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic during tools initialization: %v", r)
		}
	}()

	// Load Skills
	var serr error
	skillsDir := a.cfg.skillDir
	if skillsDir != "" {
		a.skills, serr = skills.LoadSkills(skillsDir)
		if serr != nil {
			log.Print(serr.Error())
		}
	}

	// Load MCP
	mcpConfigPath := a.cfg.mcpDir
	if mcpConfigPath != "" {
		// Safely initialize MCP with error recovery
		if err := a.initializeMCP(mcpConfigPath); err != nil {
			log.Printf("MCP initialization failed (continuing without MCP): %v", err)
		}
	}
}

// Chat implements the Agent interface for synchronous chat
func (a *TextChatAgent) Chat(ctx context.Context, message string, enableSkills bool, enableMCP bool) (string, error) {
	return a.ChatStream(ctx, message, enableSkills, enableMCP, nil)
}

// ChatStream implements the Agent interface for streaming chat
func (a *TextChatAgent) ChatStream(ctx context.Context, message string, enableSkills bool, enableMCP bool, onChunk func(context.Context, []byte) error) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Add user message to history
	a.messages = append(a.messages, llms.TextParts(llms.ChatMessageTypeHuman, message))

	// Accumulator for the full response content (including tool logs)
	var fullResponseBuilder strings.Builder

	if enableSkills && len(a.skills) > 0 {
		selectedSkill, err := a.selectSkillForTask(ctx, message)
		if err != nil {
			log.Printf("Skill selection error: %v", err)
		} else if selectedSkill != "" { // 选中了一个技能，使用技能
			for _, skill := range a.skills {
				if skill.Name == selectedSkill {
					skillResp, se := skillDoTask(ctx, a.llm, skill, a.cfg.toolSupport, message, onChunk)
					if se != nil {
						log.Printf("Error during task creation: %v", se)
					} else if skillResp != "" {
						// Add assistant response to history
						assistantMsg := llms.MessageContent{
							Role:  llms.ChatMessageTypeAI,
							Parts: []llms.ContentPart{llms.TextPart(skillResp)},
						}
						a.messages = append(a.messages, assistantMsg)
						return skillResp, nil
					}
				}
			}
		}
	}
	if enableMCP && len(a.mcpTools) > 0 {
		if a.cfg.toolSupport {
			var tools []llms.Tool
			for _, t := range a.mcpTools {
				if param, ok := mcp.GetToolSchema(t); ok {
					tools = append(tools, llms.Tool{
						Type: "function",
						Function: &llms.FunctionDefinition{
							Name:        t.Name(),
							Description: t.Description(),
							Parameters:  param,
						},
					})
				}
			}
			var opt []llms.CallOption
			opt = append(opt, llms.WithTools(tools))
			if onChunk != nil {
				opt = append(opt, llms.WithStreamingFunc(onChunk))
			}
			response, err := a.llm.GenerateContent(ctx, a.messages)
			if err != nil {
				return "", fmt.Errorf("LLM call failed: %w", err)
			}
			toolCalls := response.Choices[0].ToolCalls
			if len(toolCalls) > 0 {
				toolExecutor := prebuilt.NewToolExecutor(a.mcpTools)
				for _, tc := range toolCalls {
					var args map[string]any
					_ = json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args)

					inputVal := ""
					if val, ok := args["input"].(string); ok {
						inputVal = val
					} else {
						inputVal = tc.FunctionCall.Arguments
					}
					res, err := toolExecutor.Execute(ctx, prebuilt.ToolInvocation{
						Tool:      tc.FunctionCall.Name,
						ToolInput: inputVal,
					})
					if err != nil {
						res = fmt.Sprintf("Error: %v", err)
					}
					toolMsg := llms.MessageContent{
						Role: llms.ChatMessageTypeTool,
						Parts: []llms.ContentPart{
							llms.ToolCallResponse{
								ToolCallID: tc.ID,
								Name:       tc.FunctionCall.Name,
								Content:    res,
							},
						},
					}
					a.messages = append(a.messages, toolMsg)
				}
			}
		} else {
			toolResp, useTool, err := a.selectToolForTask(ctx, message)
			if err != nil {
				log.Printf(err.Error())
			} else if useTool {
				// Add assistant response to history
				assistantMsg := llms.MessageContent{
					Role:  llms.ChatMessageTypeAI,
					Parts: []llms.ContentPart{llms.TextPart(toolResp)},
				}
				a.messages = append(a.messages, assistantMsg)
				return toolResp, nil
			}
		}
	}
	var opt []llms.CallOption
	if onChunk != nil {
		opt = append(opt, llms.WithStreamingFunc(onChunk))
	}
	// Call LLM with full history and streaming
	response, err := a.llm.GenerateContent(ctx, a.messages, opt...)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	// Extract response text
	var responseText string
	if response != nil && len(response.Choices) > 0 {
		responseText = response.Choices[0].Content
	}

	// Append LLM response to full response
	fullResponseBuilder.WriteString(responseText)
	fullResponse := fullResponseBuilder.String()

	// Add assistant response to history
	assistantMsg := llms.MessageContent{
		Role:  llms.ChatMessageTypeAI,
		Parts: []llms.ContentPart{llms.TextPart(fullResponse)},
	}
	a.messages = append(a.messages, assistantMsg)

	return fullResponse, nil
}

// selectSkillForTask uses LLM to determine which skill (if any) should be used for the task
func (a *TextChatAgent) selectSkillForTask(ctx context.Context, message string) (string, error) {
	if len(a.skills) == 0 {
		return "", nil // No skills available
	}

	var info strings.Builder
	info.WriteString("Available Skills:\n\n")

	for _, skill := range a.skills {
		info.WriteString(fmt.Sprintf("- %s: %s\n", skill.Name, skill.Description))
	}
	skillsOverview := info.String()

	skillPrompt := fmt.Sprintf(`Based on the user's message, determine if any of the available skills should be used to help with this task.

%s

User message: %s

Respond with a JSON object:
- If no skill is needed: {"use_skill": false, "reason": "reason why no skill is needed"}
- If a skill is needed: {"use_skill": true, "skill_name": "exact skill name", "reason": "why this skill is appropriate"}

IMPORTANT:
- Return ONLY valid JSON
- Do NOT use markdown code fences
- Do NOT use `+"```json"+` wrapper
- Choose the skill that best matches the user's needs`, skillsOverview, message)

	// Create LLM call for skill selection
	skillMsg := []llms.MessageContent{
		{Role: llms.ChatMessageTypeSystem, Parts: []llms.ContentPart{llms.TextPart("You are a helpful assistant that selects appropriate skills for tasks. Respond only with valid JSON.")}},
		{Role: llms.ChatMessageTypeHuman, Parts: []llms.ContentPart{llms.TextPart(skillPrompt)}},
	}

	response, err := a.llm.GenerateContent(ctx, skillMsg)
	if err != nil {
		return "", fmt.Errorf("LLM call failed for skill selection: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response from LLM")
	}

	decision := response.Choices[0].Content
	log.Printf("Skill selection decision: %s", decision)

	// Clean up the decision
	cleanDecision := strings.TrimSpace(decision)
	if after, ok := strings.CutPrefix(cleanDecision, "```json"); ok {
		cleanDecision = after
		cleanDecision = strings.TrimSuffix(cleanDecision, "```")
		cleanDecision = strings.TrimSpace(cleanDecision)
	} else if after, ok := strings.CutPrefix(cleanDecision, "```"); ok {
		cleanDecision = after
		cleanDecision = strings.TrimSuffix(cleanDecision, "```")
		cleanDecision = strings.TrimSpace(cleanDecision)
	}

	// Parse the decision
	var skillDecision struct {
		UseSkill  bool   `json:"use_skill"`
		SkillName string `json:"skill_name"`
		Reason    string `json:"reason"`
	}

	if err := json.Unmarshal([]byte(cleanDecision), &skillDecision); err != nil {
		return "", fmt.Errorf("failed to parse skill decision: %w", err)
	}

	if skillDecision.UseSkill {
		log.Printf("Selected skill '%s' because: %s", skillDecision.SkillName, skillDecision.Reason)
		return skillDecision.SkillName, nil
	}

	log.Printf("No skill selected: %s", skillDecision.Reason)
	return "", nil
}

func (a *TextChatAgent) selectToolForTask(ctx context.Context, message string) (string, bool, error) {
	if len(a.mcpTools) == 0 {
		return "", false, nil // No mcp tool available
	}

	// Build tools info
	var toolsInfo strings.Builder
	for _, tool := range a.mcpTools {
		toolsInfo.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name(), tool.Description()))
	}

	toolPrompt := fmt.Sprintf(`Based on the user's message, determine which tool should be used.

Available tools:
%s

User message: %s

Respond with a JSON object:
- If no tool is needed: {"use_tool": false, "reason": "reason why no tool is needed"}
- If a tool is needed: {"use_tool": true, "tool_name": "exact tool name", "args": {parameter: "value"}, "reason": "why this tool is appropriate"}

IMPORTANT:
- Return ONLY valid JSON
- Do NOT use markdown code fences
- Do NOT use `+"```json"+` wrapper
- Select the tool that can best accomplish the user's request`, toolsInfo.String(), message)

	// Create LLM call for tool selection
	toolMsg := []llms.MessageContent{
		{Role: llms.ChatMessageTypeSystem, Parts: []llms.ContentPart{llms.TextPart("You are a helpful assistant that selects appropriate tools for tasks. Respond only with valid JSON.")}},
		{Role: llms.ChatMessageTypeHuman, Parts: []llms.ContentPart{llms.TextPart(toolPrompt)}},
	}

	response, err := a.llm.GenerateContent(ctx, toolMsg)
	if err != nil {
		return "", false, fmt.Errorf("LLM call failed for tool selection: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", false, fmt.Errorf("no response from LLM")
	}

	decision := response.Choices[0].Content
	log.Printf("Tool selection decision: %s", decision)

	// Clean up the decision
	cleanDecision := strings.TrimSpace(decision)
	if after, ok := strings.CutPrefix(cleanDecision, "```json"); ok {
		cleanDecision = after
		cleanDecision = strings.TrimSuffix(cleanDecision, "```")
		cleanDecision = strings.TrimSpace(cleanDecision)
	} else if after, ok := strings.CutPrefix(cleanDecision, "```"); ok {
		cleanDecision = after
		cleanDecision = strings.TrimSuffix(cleanDecision, "```")
		cleanDecision = strings.TrimSpace(cleanDecision)
	}

	// Parse the decision
	var toolDecision struct {
		UseTool  bool           `json:"use_tool"`
		ToolName string         `json:"tool_name"`
		Args     map[string]any `json:"args"`
		Reason   string         `json:"reason"`
	}

	if err := json.Unmarshal([]byte(cleanDecision), &toolDecision); err != nil {
		return "", false, fmt.Errorf("failed to parse tool decision: %w", err)
	}

	if toolDecision.UseTool {
		// Find the selected tool
		for _, tool := range a.mcpTools {
			if strings.EqualFold(tool.Name(), toolDecision.ToolName) {
				log.Printf("Selected tool '%s' because: %s", toolDecision.ToolName, toolDecision.Reason)
				// Convert args to JSON string
				argsJSON, _ := json.Marshal(toolDecision.Args)
				argsStr := string(argsJSON)
				if argsStr == "null" {
					argsStr = "{}"
				}
				// Call the tool
				result, err := tool.Call(ctx, argsStr)
				if err != nil {
					log.Printf("MCP tool %s call failed: %v", tool.Name(), err)
					return "", false, fmt.Errorf("tool %s call failed: %w", tool.Name(), err)
				}
				log.Printf("Successfully used MCP tool '%s'", tool.Name())
				return fmt.Sprintf("I used the '%s' tool to help with your request. Here's the result:\n\n%s", tool.Name(), result), true, nil
			}
		}
		return "", false, fmt.Errorf("tool '%s' not found in available tools", toolDecision.ToolName)
	}

	log.Printf("No tool selected: %s", toolDecision.Reason)
	return "", false, nil
}

// initializeMCP safely initializes MCP client with error recovery
func (a *TextChatAgent) initializeMCP(mcpConfigPath string) (err error) {
	// Add panic recovery to prevent crashes from MCP initialization
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic during MCP initialization: %v", r)
			log.Printf("Recovered from MCP initialization panic: %v", r)
		}
	}()

	// Use a longer timeout for initialization as npx downloads may be slow
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Load MCP config
	config, err := mcpclient.LoadConfig(mcpConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load MCP config: %w", err)
	}

	// Create MCP client with error handling
	client, err := mcpclient.NewClient(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create MCP client: %w", err)
	}

	// Get tools from MCP with timeout
	toolsCtx, toolsCancel := context.WithTimeout(ctx, 30*time.Second)
	defer toolsCancel()

	tools, err := mcp.MCPToTools(toolsCtx, client)
	if err != nil {
		// Close client if tool loading fails
		if closeErr := a.closeMCPClient(client); closeErr != nil {
			log.Printf("Failed to close MCP client after error: %v", closeErr)
		}
		return fmt.Errorf("failed to get MCP tools: %w", err)
	}

	if len(tools) == 0 {
		log.Printf("No MCP tools found, closing client")
		if closeErr := a.closeMCPClient(client); closeErr != nil {
			log.Printf("Failed to close MCP client: %v", closeErr)
		}
		return nil
	}

	// Successfully initialized
	a.mu.Lock()
	a.mcpClient = client
	a.mcpTools = tools
	a.toolsEnabled = true
	a.mu.Unlock()
	log.Printf("Successfully loaded %d MCP tools", len(tools))

	return nil
}

// closeMCPClient safely closes an MCP client with panic recovery and timeout
func (a *TextChatAgent) closeMCPClient(client *mcpclient.Client) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic during MCP client close: %v", r)
			log.Printf("Recovered from MCP client close panic: %v", r)
		}
	}()

	if client == nil {
		return nil
	}

	// Use a goroutine with timeout to prevent hanging on close
	done := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Errorf("panic in close goroutine: %v", r)
			}
		}()
		done <- client.Close()
	}()

	// Wait for close with timeout
	select {
	case closeErr := <-done:
		if closeErr != nil {
			return fmt.Errorf("failed to close MCP client: %w", closeErr)
		}
		return nil
	case <-time.After(5 * time.Second):
		log.Printf("Warning: MCP client close timed out after 5 seconds")
		return fmt.Errorf("MCP client close timed out")
	}
}

// Close releases resources held by the agent
func (a *TextChatAgent) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	log.Printf("Closing agent and cleaning up resources...")

	if a.mcpClient != nil {
		log.Printf("Closing MCP client...")
		if err := a.closeMCPClient(a.mcpClient); err != nil {
			// Log error but don't return - we want to continue cleanup
			log.Printf("Error closing MCP client (continuing cleanup): %v", err)
		}
		a.mcpClient = nil
		a.mcpTools = nil
		log.Printf("MCP client closed and cleared")
	}

	return nil
}
