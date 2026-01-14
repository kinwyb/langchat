package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/kinwyb/langchat/llm/skills"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
)

// TextChatAgent manages conversation history for a session
type TextChatAgent struct {
	llm      llms.Model
	messages []llms.MessageContent
	mu       sync.RWMutex
	//mcpClient     *mcpclient.Client
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

	go func() {
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
		//mcpConfigPath := a.cfg.mcpDir

		// Safely initialize MCP with error recovery
		//if err := a.initializeMCP(mcpConfigPath); err != nil {
		//	log.Printf("MCP initialization failed (continuing without MCP): %v", err)
		//}
	}()
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
					skillResp, se := skillDoTask(ctx, a.llm, skill, message, onChunk)
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
