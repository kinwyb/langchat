package agent

import "context"

// Agent interface defines the contract for chat agents
type Agent interface {
	Chat(ctx context.Context, message string, enableSkills bool, enableMCP bool) (string, error)
	ChatStream(ctx context.Context, message string, enableSkills bool, enableMCP bool, onChunk func(context.Context, []byte) error) (string, error)
}

// config agent config
type config struct {
	skillDir    string
	mcpDir      string
	toolSupport bool
}

type Option func(*config)

// WithSkill 配置技能目录
func WithSkill(skillDir string) Option {
	return func(c *config) {
		c.skillDir = skillDir
	}
}

// WithMCP 配置MCP目录
func WithMCP(mcpDir string) Option {
	return func(c *config) {
		c.mcpDir = mcpDir
	}
}

// ModelToolSupport 模型支持工具调用
func ModelToolSupport(support bool) Option {
	return func(c *config) {
		c.toolSupport = support
	}
}
