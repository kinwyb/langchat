package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kinwyb/langchat/llm/tools"
	openai "github.com/sashabaranov/go-openai"
)

// Tool implements tools.Tool for skills.
type Tool struct {
	scriptMap map[string]string
	skillPath string
	tool      openai.Tool
}

func (t *Tool) Paramters() any {
	return t.tool.Function.Parameters
}

func (t *Tool) DescriptionWithParamters() string {
	sb := strings.Builder{}
	sb.WriteString(t.Description() + " ")
	params := tools.OpenaiToolConvertToolParamter(t.tool)
	for k, v := range params {
		sb.WriteString(k + "(" + v.Type + ") " + v.Description + " ")
	}
	return sb.String()
}

func (t *Tool) Name() string {
	return t.tool.Function.Name
}

func (t *Tool) Description() string {
	return t.tool.Function.Description
}

func (t *Tool) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"skillPath": t.skillPath,
		"scriptMap": t.scriptMap,
		"tool":      t.tool,
	})
}

func (t *Tool) Call(ctx context.Context, input string) (string, error) {
	// input is the JSON string of arguments
	// We need to parse it based on the tool name, similar to goskills runner.go
	switch t.Name() {
	case "run_shell_code":
		var params struct {
			Code string         `json:"code"`
			Args map[string]any `json:"args"`
		}
		if err := json.Unmarshal([]byte(input), &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal run_shell_code arguments: %w", err)
		}
		shellTool := tools.ShellTool{}
		return shellTool.Run(params.Args, params.Code)

	case "run_shell_script":
		var params struct {
			ScriptPath string   `json:"scriptPath"`
			Args       []string `json:"args"`
		}
		if err := json.Unmarshal([]byte(input), &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal run_shell_script arguments: %w", err)
		}
		return tools.RunShellScript(params.ScriptPath, params.Args)

	case "run_python_code":
		var params struct {
			Code string         `json:"code"`
			Args map[string]any `json:"args"`
		}
		if err := json.Unmarshal([]byte(input), &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal run_python_code arguments: %w", err)
		}
		pythonTool := tools.PythonTool{}
		return pythonTool.Run(params.Args, params.Code)

	case "run_python_script":
		var params struct {
			ScriptPath string   `json:"scriptPath"`
			Args       []string `json:"args"`
		}
		if err := json.Unmarshal([]byte(input), &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal run_python_script arguments: %w", err)
		}
		return tools.RunPythonScript(params.ScriptPath, params.Args)

	case "read_file":
		var params struct {
			FilePath string `json:"filePath"`
		}
		if err := json.Unmarshal([]byte(input), &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal read_file arguments: %w", err)
		}
		path := params.FilePath
		if !filepath.IsAbs(path) && t.skillPath != "" {
			resolvedPath := filepath.Join(t.skillPath, path)
			if _, err := os.Stat(resolvedPath); err == nil {
				path = resolvedPath
			}
		}
		return tools.ReadFile(path)

	case "write_file":
		var params struct {
			FilePath string `json:"filePath"`
			Content  string `json:"content"`
		}
		if err := json.Unmarshal([]byte(input), &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal write_file arguments: %w", err)
		}
		err := tools.WriteFile(params.FilePath, params.Content)
		if err == nil {
			return fmt.Sprintf("Successfully wrote to file: %s", params.FilePath), nil
		}
		return "", err

	case "wikipedia_search":
		var params struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal([]byte(input), &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal wikipedia_search arguments: %w", err)
		}
		return tools.WikipediaSearch(params.Query)

	case "tavily_search":
		var params struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal([]byte(input), &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal tavily_search arguments: %w", err)
		}
		return tools.TavilySearch(params.Query)

	case "web_fetch":
		var params struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal([]byte(input), &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal web_fetch arguments: %w", err)
		}
		return tools.WebFetch(params.URL)

	default:
		if scriptPath, ok := t.scriptMap[t.Name()]; ok {
			var params struct {
				Args []string `json:"args"`
			}
			if input != "" {
				if err := json.Unmarshal([]byte(input), &params); err != nil {
					return "", fmt.Errorf("failed to unmarshal script arguments: %w", err)
				}
			}
			if strings.HasSuffix(scriptPath, ".py") {
				return tools.RunPythonScript(scriptPath, params.Args)
			}

			return tools.RunShellScript(scriptPath, params.Args)
		}
		return "", fmt.Errorf("unknown tool: %s", t.Name())
	}
}

// Tools converts a SkillPackage to a slice of tools.Tool.
func Tools(skill *Package) ([]tools.ITool, error) {
	availableTools, scriptMap := generateToolDefinitions(skill)
	var result []tools.ITool

	for _, t := range availableTools {
		if t.Function.Name == "" {
			continue
		}

		result = append(result, &Tool{
			scriptMap: scriptMap,
			skillPath: skill.Path,
			tool:      t,
		})
	}
	return result, nil
}

// generateToolDefinitions generates the list of OpenAI tools for a given skill.
// It returns the tool definitions and a map of tool names to script paths for execution.
func generateToolDefinitions(skill *Package) ([]openai.Tool, map[string]string) {
	var tool []openai.Tool
	scriptMap := make(map[string]string)

	// 1. Base Tools
	baseTools := tools.GetBaseTools()

	if len(skill.Meta.AllowedTools) > 0 {
		allowedMap := make(map[string]bool)
		for _, t := range skill.Meta.AllowedTools {
			allowedMap[t] = true
		}

		for _, t := range baseTools {
			if allowedMap[t.Function.Name] {
				tool = append(tool, t)
			}
		}
	} else {
		tool = append(tool, baseTools...)
	}

	// 2. Script Tools
	for _, scriptRelPath := range skill.Resources.Scripts {
		toolDef, toolName := generateScriptTool(skill.Path, scriptRelPath)
		tool = append(tool, toolDef)
		scriptMap[toolName] = filepath.Join(skill.Path, scriptRelPath)
	}

	return tool, scriptMap
}

func generateScriptTool(skillPath, scriptRelPath string) (openai.Tool, string) {
	// Normalize name: replace non-alphanumeric with underscore
	safeName := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, scriptRelPath)
	toolName := "run_" + safeName

	// Determine type based on extension
	ext := filepath.Ext(scriptRelPath)
	var description string
	if ext == ".py" {
		description = fmt.Sprintf("Executes the python script '%s'.", scriptRelPath)
	} else {
		description = fmt.Sprintf("Executes the shell script '%s'.", scriptRelPath)
	}

	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        toolName,
			Description: description,
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"args": map[string]any{
						"type":        "array",
						"description": "Arguments to pass to the script.",
						"items": map[string]any{
							"type": "string",
						},
					},
				},
			},
		},
	}, toolName
}
