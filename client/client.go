package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
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

// ServerConfig 服务器配置
type ServerConfig struct {
	URL     string
	Name    string
	Enabled bool
}

// MCPHost MCP服务器管理器
type MCPHost struct {
	clients map[string]*client.SSEMCPClient
	configs map[string]ServerConfig
	mutex   sync.RWMutex
}

// NewMCPHost 创建新的MCP管理器
func NewMCPHost() *MCPHost {
	return &MCPHost{
		clients: make(map[string]*client.SSEMCPClient),
		configs: make(map[string]ServerConfig),
	}
}

// AddServer 添加服务器配置
func (m *MCPHost) AddServer(config ServerConfig) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.configs[config.Name] = config
	return nil
}

// ConnectServer 连接到指定服务器
func (m *MCPHost) ConnectServer(ctx context.Context, serverName string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	config, exists := m.configs[serverName]
	if !exists {
		return fmt.Errorf("server config not found: %s", serverName)
	}

	if !config.Enabled {
		return fmt.Errorf("server is disabled: %s", serverName)
	}

	cli, err := client.NewSSEMCPClient(config.URL)
	if err != nil {
		return fmt.Errorf("failed to create client for %s: %v", serverName, err)
	}

	if err := cli.Start(ctx); err != nil {
		cli.Close()
		return fmt.Errorf("failed to start client for %s: %v", serverName, err)
	}

	// 初始化客户端
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "multi-server-client",
		Version: "1.0.0",
	}

	_, err = cli.Initialize(ctx, initRequest)
	if err != nil {
		cli.Close()
		return fmt.Errorf("failed to initialize client for %s: %v", serverName, err)
	}

	m.clients[serverName] = cli
	return nil
}

// DisconnectServer 断开与指定服务器的连接
func (m *MCPHost) DisconnectServer(serverName string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if cli, exists := m.clients[serverName]; exists {
		cli.Close()
		delete(m.clients, serverName)
	}
	return nil
}

// GetClient 获取指定服务器的客户端
func (m *MCPHost) GetClient(serverName string) (*client.SSEMCPClient, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	cli, exists := m.clients[serverName]
	if !exists {
		return nil, fmt.Errorf("client not found: %s", serverName)
	}
	return cli, nil
}

// Close 关闭所有连接
func (m *MCPHost) Close() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for _, cli := range m.clients {
		cli.Close()
	}
	m.clients = make(map[string]*client.SSEMCPClient)
}

func (m *MCPHost) GetAllTools(ctx context.Context) []openai.Tool {
	// 从所有可用的MCP服务器收集工具列表
	var allTools []openai.Tool

	// 遍历所有服务器获取工具
	m.mutex.RLock()
	for serverName, cli := range m.clients {
		toolsRequest := mcp.ListToolsRequest{}
		toolsResult, err := cli.ListTools(ctx, toolsRequest)
		if err != nil {
			log.Printf("从服务器 %s 获取工具列表失败: %v", serverName, err)
			continue
		}

		// 为每个工具添加服务器标识
		for _, tool := range toolsResult.Tools {
			allTools = append(allTools, openai.Tool{
				Type: openai.ToolTypeFunction,
				Function: &openai.FunctionDefinition{
					// 在工具名称中添加服务器标识
					Name:        buildToolName(tool.Name, serverName),
					Description: tool.Description,
					Parameters:  tool.InputSchema,
				},
			})
		}
	}
	m.mutex.RUnlock()
	return allTools
}

// ToolCallResult 存储工具调用的结果
type ToolCallResult struct {
	ToolName   string                 `json:"tool_name"`
	Parameters map[string]interface{} `json:"parameters"`
	Result     string                 `json:"result"`
	Error      string                 `json:"error,omitempty"`
}

func (m *MCPHost) ProcessWithOpenAI(ctx context.Context, prompt string) (string, []ToolCallResult, error) {
	// 创建带有工具的聊天完成请求
	resp, err := openaiClient.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: os.Getenv("OPENAI_MODEL_NAME"),
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			Tools: m.GetAllTools(ctx),
		},
	)
	if err != nil {
		return "", nil, fmt.Errorf("OpenAI API call failed: %v", err)
	}

	if len(resp.Choices) == 0 {
		return "", nil, fmt.Errorf("OpenAI API call failed: %v", err)
	}

	choice := resp.Choices[0]
	content := choice.Message.Content

	// 存储所有工具调用的结果
	var results []ToolCallResult

	// 处理工具调用
	if len(resp.Choices) > 0 && choice.Message.ToolCalls != nil {
		for _, toolCall := range choice.Message.ToolCalls {
			result := ToolCallResult{
				ToolName: toolCall.Function.Name,
			}

			// 解析参数
			var args map[string]interface{}
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
				result.Error = fmt.Sprintf("failed to parse tool arguments: %v", err)
				results = append(results, result)
				continue
			}
			result.Parameters = args

			toolName, serverName, err := parseToolName(toolCall.Function.Name)

			// 执行工具调用
			callRequest := mcp.CallToolRequest{}
			callRequest.Params.Name = toolName
			callRequest.Params.Arguments = args

			if err != nil {
				result.Error = fmt.Sprintf("解析MCP Server 名称失败: %v", err)
				results = append(results, result)
				continue
			}
			cli, err := m.GetClient(serverName)
			if err != nil {
				result.Error = fmt.Sprintf("获取MCP Client 失败: %v", err)
				results = append(results, result)
				continue
			}

			// 执行工具
			callResult, err := cli.CallTool(ctx, callRequest)
			if err != nil {
				result.Error = fmt.Sprintf("工具执行失败: %v", err)
				results = append(results, result)
				continue
			}

			// 处理工具执行结果
			if len(callResult.Content) > 0 {
				if textContent, ok := callResult.Content[0].(mcp.TextContent); ok {
					result.Result = textContent.Text
				}
			}
			results = append(results, result)
		}
	}

	return content, results, nil

}

// buildToolName 构建完整的工具名称
func buildToolName(toolName, serverName string) string {
	return fmt.Sprintf("%s@%s", toolName, serverName)
}

// parseToolName 从完整的工具名称中解析出服务器名称
func parseToolName(fullToolName string) (toolName, serverName string, err error) {
	lastIndex := strings.LastIndex(fullToolName, "@")
	if lastIndex == -1 {
		return "", "", fmt.Errorf("invalid tool name format: %s", fullToolName)
	}
	return fullToolName[:lastIndex], fullToolName[lastIndex+1:], nil
}
