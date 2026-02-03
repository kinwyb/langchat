package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/kinwyb/langchat/llm/skills"
	"github.com/kinwyb/langchat/llm/tools"
	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/prebuilt"
	"github.com/tmc/langchaingo/llms"
	tls "github.com/tmc/langchaingo/tools"
)

type ReactEvent int

const (
	ReactNodeStart ReactEvent = iota + 1
	ReactNodeEnd
	ReactLLMContent
)

type ReactOption func(*ReactAgent)

func ReactSupportTool(supportTool bool) ReactOption {
	return func(a *ReactAgent) {
		a.supportTool = supportTool
	}
}

func ReactWithStream(onChunk func(context.Context, []byte) error) ReactOption {
	return func(a *ReactAgent) {
		a.streaming = func(ctx context.Context, event ReactEvent, data []byte) {
			if event == ReactLLMContent && onChunk != nil {
				_ = onChunk(ctx, data)
			}
		}
	}
}

func ReactWithStreamEvent(event func(context.Context, ReactEvent, []byte)) ReactOption {
	return func(a *ReactAgent) {
		a.streaming = func(ctx context.Context, ev ReactEvent, data []byte) {
			if event != nil {
				event(ctx, ev, data)
			}
		}
	}
}

func ReactWithTools(tool []tools.ITool) ReactOption {
	return func(a *ReactAgent) {
		a.inputTools = tool
	}
}

func ReactWithMaxIterations(maxIterations int) ReactOption {
	return func(a *ReactAgent) {
		if maxIterations <= 0 {
			maxIterations = 5
		}
		a.maxIterations = maxIterations
	}
}

type ReactAgent struct {
	model         llms.Model
	inputTools    []tools.ITool
	toolExecutor  *prebuilt.ToolExecutor
	maxIterations int
	supportTool   bool
	initLock      sync.Mutex
	isInit        bool
	streaming     func(context.Context, ReactEvent, []byte)
	runnable      *graph.StateRunnable[map[string]any]
	message       []llms.MessageContent
}

func NewReactAgent(model llms.Model, systemPrompt []llms.MessageContent, option ...ReactOption) *ReactAgent {
	ret := &ReactAgent{
		model:         model,
		maxIterations: 1,
		message:       systemPrompt,
	}
	for _, opt := range option {
		opt(ret)
	}
	return ret
}

func (r *ReactAgent) InitAgent() error {
	r.initLock.Lock()
	defer r.initLock.Unlock()
	if r.isInit {
		return nil
	}

	if r.maxIterations == 0 {
		r.maxIterations = 20
	}

	var inputTools []tls.Tool
	for _, tool := range r.inputTools {
		inputTools = append(inputTools, tool)
	}
	// Define the tool executor
	r.toolExecutor = prebuilt.NewToolExecutor(inputTools)

	// Define the graph
	workflow := graph.NewStateGraph[map[string]any]()
	// Define the state schema
	agentSchema := graph.NewMapSchema()
	agentSchema.RegisterReducer("messages", graph.AppendReducer)
	workflow.SetSchema(agentSchema)

	// Define the agent node
	workflow.AddNode("agent", "ReAct agent decision maker", r.agentNode)
	// Define the tools node
	workflow.AddNode("tools", "Tool execution node", r.toolNode)
	workflow.SetEntryPoint("agent")
	workflow.AddConditionalEdge("agent", r.nodeConditionalEdge)
	workflow.AddEdge("tools", "agent")

	var err error
	r.runnable, err = workflow.Compile()
	return err
}

func (r *ReactAgent) streamEvent(ctx context.Context, event ReactEvent, data []byte) {
	if r.streaming == nil {
		return
	}
	r.streaming(ctx, event, data)
}

// AgentNode 思考节点
func (r *ReactAgent) agentNode(ctx context.Context, state map[string]any) (map[string]any, error) {
	messages, ok := state["messages"].([]llms.MessageContent)
	if !ok {
		return nil, fmt.Errorf("messages key not found or invalid type")
	}
	r.streamEvent(ctx, ReactNodeStart, []byte("agent"))
	// Check iteration count
	iterationCount := 0
	if count, ok := state["iteration_count"].(int); ok {
		iterationCount = count
	}
	if iterationCount >= r.maxIterations {
		// Max iterations reached, return final message
		finalMsg := llms.MessageContent{
			Role: llms.ChatMessageTypeAI,
			Parts: []llms.ContentPart{
				llms.TextPart("Maximum iterations reached. Please try a simpler query."),
			},
		}
		return map[string]any{
			"messages": []llms.MessageContent{finalMsg},
		}, nil
	}

	// Convert tools to ToolInfo for the model
	var toolDefs = r.initTool()

	opts := []llms.CallOption{llms.WithTools(toolDefs)}
	if r.streaming != nil {
		opts = append(opts, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
			r.streamEvent(ctx, ReactLLMContent, chunk)
			return nil
		}))
	}

	// Call model with tools
	resp, err := r.model.GenerateContent(ctx, messages, opts...)
	if err != nil {
		return nil, err
	}

	choice := resp.Choices[0]
	aiMsg := llms.MessageContent{
		Role: llms.ChatMessageTypeAI,
	}
	if choice.Content != "" {
		aiMsg.Parts = append(aiMsg.Parts, llms.TextPart(choice.Content))
	}
	for _, tc := range choice.ToolCalls {
		aiMsg.Parts = append(aiMsg.Parts, tc)
	}
	r.streamEvent(ctx, ReactNodeEnd, []byte("agent"))
	return map[string]any{
		"messages":        []llms.MessageContent{aiMsg},
		"iteration_count": iterationCount + 1,
	}, nil
}

// ToolNode 工具执行节点
func (r *ReactAgent) toolNode(ctx context.Context, state map[string]any) (map[string]any, error) {
	messages := state["messages"].([]llms.MessageContent)
	lastMsg := messages[len(messages)-1]

	if lastMsg.Role != llms.ChatMessageTypeAI {
		return nil, fmt.Errorf("last message is not an AI message")
	}

	r.streamEvent(ctx, ReactNodeStart, []byte("tool"))

	var toolMessages []llms.MessageContent
	for _, part := range lastMsg.Parts {
		if tc, ok := part.(llms.ToolCall); ok {
			var args map[string]any
			_ = json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args)

			inputVal := ""
			if val, ok := args["input"].(string); ok {
				inputVal = val
			} else {
				inputVal = tc.FunctionCall.Arguments
			}

			res, err := r.toolExecutor.Execute(ctx, prebuilt.ToolInvocation{
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
			r.streamEvent(ctx, ReactLLMContent, []byte("tool result: "+res))
			toolMessages = append(toolMessages, toolMsg)
		}
	}

	r.streamEvent(ctx, ReactNodeEnd, []byte("tool"))

	return map[string]any{
		"messages": toolMessages,
	}, nil
}

// NodeConditionalEdge 节点判断
func (r *ReactAgent) nodeConditionalEdge(ctx context.Context, state map[string]any) string {
	messages := state["messages"].([]llms.MessageContent)
	lastMsg := messages[len(messages)-1]
	for _, part := range lastMsg.Parts {
		if _, ok := part.(llms.ToolCall); ok {
			return "tools"
		}
		if !r.supportTool {
			if textPart, ok := part.(llms.TextContent); ok {
				decision := textPart.Text
				if strings.Contains(decision, "{\"use_tool\":") {
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
						continue
					}
					if toolDecision.UseTool {
						// Find the selected tool
						for _, tool := range r.inputTools {
							if strings.EqualFold(tool.Name(), toolDecision.ToolName) {
								log.Printf("Selected tool '%s' because: %s", toolDecision.ToolName, toolDecision.Reason)
								argsJSON, _ := json.MarshalIndent(toolDecision.Args, "", "  ")
								argsStr := string(argsJSON)
								if argsStr == "null" {
									argsStr = "{}"
								}
								res, err := r.toolExecutor.Execute(ctx, prebuilt.ToolInvocation{
									Tool:      toolDecision.ToolName,
									ToolInput: argsStr,
								})
								if err != nil {
									res = fmt.Sprintf("Error: %v", err)
								}
								aiMsg := llms.MessageContent{
									Role: llms.ChatMessageTypeAI,
									Parts: []llms.ContentPart{
										llms.TextPart(" tool " + tool.Name() + " do complete result content : " + res),
									},
								}
								state["messages"] = append(state["messages"].([]llms.MessageContent), aiMsg)
								return "agent"
							}
						}
					}
				}
			}
		}
	}
	return graph.END
}

// Convert tools to ToolInfo for the model
func (r *ReactAgent) initTool() []llms.Tool {
	if len(r.inputTools) < 1 {
		return nil
	}
	var toolDefs []llms.Tool
	if r.supportTool {
		for _, t := range r.inputTools {
			toolDefs = append(toolDefs, llms.Tool{
				Type: "function",
				Function: &llms.FunctionDefinition{
					Name:        t.Name(),
					Description: t.Description(),
					Parameters:  t.Paramters(),
				},
			})
		}
	} else {
		var toolsInfo strings.Builder
		for _, tool := range r.inputTools {
			if strings.HasPrefix(tool.Name(), "run_") {
				continue
			}
			desc := tool.DescriptionWithParamters()
			toolsInfo.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name(), desc))
		}

		toolPrompt := fmt.Sprintf(`Available tools:
%s

If need use tool respond with a JSON object:
- If a tool is needed: {"use_tool": true, "tool_name": "exact tool name", "args": {parameter: "value"}, "reason": "why this tool is appropriate"}
- Return ONLY valid JSON
- Do NOT use markdown code fences
- Do NOT use `+"```json"+` wrapper
- Select the tool that can best accomplish the user's request
		
If not use tool normal return content
`, toolsInfo.String())

		r.message = append(r.message,
			// llms.TextParts(llms.ChatMessageTypeSystem, "You are a helpful assistant that can selects appropriate tools for tasks. IF need use tool respond only with valid JSON."),
			llms.TextParts(llms.ChatMessageTypeSystem, toolPrompt),
		)
	}
	return toolDefs
}

func (r *ReactAgent) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	if !r.isInit {
		err := r.InitAgent()
		if err != nil {
			return nil, err
		}
	}
	r.initTool()
	ms := append(r.message, messages...)
	initialState := map[string]any{
		"messages": ms,
	}
	ret, err := r.runnable.Invoke(ctx, initialState)
	if err != nil {
		return nil, err
	}
	// Print Result
	messages = ret["messages"].([]llms.MessageContent)
	lastMsg := messages[len(messages)-1]
	if len(lastMsg.Parts) > 0 {
		if textPart, ok := lastMsg.Parts[0].(llms.TextContent); ok {
			return &llms.ContentResponse{Choices: []*llms.ContentChoice{
				{Content: textPart.Text},
			}}, nil
		}
	}
	return nil, errors.New("no messages found")
}

// skillDoTask skill 执行
func skillDoTask(ctx context.Context, model llms.Model, skill *skills.Skill, toolSupport bool, message string, onChunk func(ctx context.Context, data []byte) error) (string, error) {
	if skill == nil {
		return "", errors.New("skill is nil")
	}
	skillPropemt := fmt.Sprintf("Skill: %s\n%s\n\n", skill.Package.Meta.Name, skill.Package.Body)
	opts := []ReactOption{
		ReactWithTools(skill.Tools),
		ReactWithMaxIterations(5),
		ReactSupportTool(toolSupport),
	}
	if onChunk != nil {
		opts = append(opts, ReactWithStream(onChunk))
	}
	rac := NewReactAgent(model, []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeSystem, skillPropemt)}, opts...)
	response, err := rac.GenerateContent(ctx, []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeHuman, message)})
	if err != nil {
		return "", err
	}
	// Extract response text
	var responseText string
	if response != nil && len(response.Choices) > 0 {
		responseText = response.Choices[0].Content
	}

	return responseText, nil
}
