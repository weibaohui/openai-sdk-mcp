package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/sashabaranov/go-openai"
)

type OpenAIConfig struct {
	APIKey  string
	BaseURL string
}

// 全局OpenAI客户端
var openaiClient *openai.Client

// 初始化OpenAI客户端
func initOpenAIClient(config OpenAIConfig) error {
	cfg := openai.DefaultConfig(config.APIKey)
	if config.BaseURL != "" {
		cfg.BaseURL = config.BaseURL
	}
	openaiClient = openai.NewClientWithConfig(cfg)
	return nil
}

func main() {
	// 初始化OpenAI客户端
	openaiConfig := OpenAIConfig{
		APIKey:  os.Getenv("OPENAI_API_KEY"),  // 请替换为实际的API密钥
		BaseURL: os.Getenv("OPENAI_BASE_URL"), // 可选，如果需要自定义BaseURL
	}
	if err := initOpenAIClient(openaiConfig); err != nil {
		log.Fatalf("Failed to initialize OpenAI client: %v", err)
	}

	// 创建MCP管理器
	host := NewMCPHost()
	defer host.Close()

	// 添加服务器配置
	servers := []ServerConfig{
		{
			Name:    "server1",
			URL:     "http://localhost:9292/sse",
			Enabled: true,
		},
		{
			Name:    "server2",
			URL:     "http://localhost:9293/sse",
			Enabled: true,
		},
	}

	// 添加并连接服务器
	ctx := context.Background()
	for _, server := range servers {
		if err := host.AddServer(server); err != nil {
			log.Printf("Failed to add server %s: %v", server.Name, err)
			continue
		}

		if err := host.ConnectServer(ctx, server.Name); err != nil {
			log.Printf("Failed to connect to server %s: %v", server.Name, err)
			continue
		}

		// 测试服务器连接状态
		if err := host.Ping(ctx, server.Name); err != nil {
			log.Printf("Ping failed for server %s: %v", server.Name, err)
			continue
		}

		log.Printf("Successfully connected to server: %s", server.Name)
	}

	// content, toolResult, err := host.ProcessWithOpenAI(ctx, "请生成两个不大于10的随机数，并将这两个随机数进行求和，告诉我最终的结果")
	content, toolResult, err := host.ProcessWithOpenAI(ctx, "请分析22+33=?这个算式，并调用对应方法，求结果")

	if err != nil {
		log.Printf("processWithOpenAI failed: %v", err)
		return
	}
	log.Printf("content = \n %s\n", content)
	bytes, _ := json.MarshalIndent(toolResult, "", "  ")
	log.Printf("result = \n %s\n", bytes)
}
