package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kinwyb/langchat/api"
	"github.com/kinwyb/langchat/llm/agent"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

func main() {
	// 配置 LLM
	llm, err := openai.New(
		openai.WithToken("your-api-key"),                // 替换为你的 API key
		openai.WithBaseURL("http://localhost:11434/v1"), // Ollama 示例
		openai.WithModel("gemma3:12b"),
	)
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}

	// 创建 TextChatAgent
	// 可以替换为其他 agent 实现，如 ReactAgent
	textAgent := agent.NewTextChatAgent(llm,
		agent.WithSkill("./skills"), // 配置技能目录
		// agent.WithMCP("./mcp"),    // 配置 MCP 目录
	)

	// 创建 API 服务器
	serverCfg := api.DefaultServerConfig()
	serverCfg.Port = 8080
	server := api.NewServer(textAgent, serverCfg)

	// 启动服务器
	go func() {
		log.Println("Server starting...")
		if err := server.Start(); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// 测试 LLM 连接
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Println("Testing LLM connection...")
	testResp, err := llm.GenerateContent(ctx, []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, "Hello, this is a connection test."),
	})
	if err != nil {
		log.Printf("Warning: LLM connection test failed: %v", err)
	} else if len(testResp.Choices) > 0 {
		log.Printf("LLM connected successfully. Test response: %s", testResp.Choices[0].Content)
	}

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// 优雅关闭
	log.Println("Shutting down server...")
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
	log.Println("Server stopped")
}
