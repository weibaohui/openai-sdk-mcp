package client

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

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

func Run() {
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

		log.Printf("Successfully connected to server: %s", server.Name)
	}

	// 启动定期ping检查
	go func() {
		ticker := time.NewTicker(30 * time.Second) // 每30秒执行一次
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				status := host.PingAll(ctx)
				for serverName, serverStatus := range status {
					if serverStatus.LastPingSuccess {
						log.Printf("Server %s is healthy, last ping time: %v", serverName, serverStatus.LastPingTime)
					} else {
						log.Printf("Server %s is unhealthy, last ping time: %v, error: %s", serverName, serverStatus.LastPingTime, serverStatus.LastError)
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// 示例：处理用户输入
	content, toolResult, err := host.ProcessWithOpenAI(ctx, "请分析22+33=?这个算式，并调用对应方法，求结果")
	if err != nil {
		log.Printf("processWithOpenAI failed: %v", err)
		return
	}
	log.Printf("content = \n %s\n", content)
	bytes, _ := json.MarshalIndent(toolResult, "", "  ")
	log.Printf("result = \n %s\n", bytes)

	// 保持程序运行
	select {}
}
