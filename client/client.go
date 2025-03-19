package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sashabaranov/go-openai"
)

// OpenAI客户端配置
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

	// 创建一个新的 SSE MCP 客户端
	cli, err := client.NewSSEMCPClient("http://localhost:9292/sse")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 启动客户端
	if err := cli.Start(ctx); err != nil {
		log.Fatalf("Failed to start client: %v", err)
	}

	// 初始化客户端
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}

	result, err := cli.Initialize(ctx, initRequest)
	if err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}

	log.Printf("Server name: %s", result.ServerInfo.Name)

	// 测试 Ping
	if err := cli.Ping(ctx); err != nil {
		log.Printf("Ping failed: %v", err)
	}

	toolResult, err := processWithOpenAI(context.Background(), cli, "测试一下test tool，这是一个哈哈哈")
	if err != nil {
		log.Printf("processWithOpenAI failed: %v", err)
		return
	}
	log.Printf("result = \n %s\n", toolResult)
}

// 使用OpenAI处理工具调用
func processWithOpenAI(ctx context.Context, cli *client.SSEMCPClient, prompt string) (string, error) {
	// 获取可用的工具列表
	toolsRequest := mcp.ListToolsRequest{}
	toolsResult, err := cli.ListTools(ctx, toolsRequest)
	if err != nil {
		return "", fmt.Errorf("failed to list tools: %v", err)
	}

	// 构建工具描述
	tools := make([]openai.Tool, 0, len(toolsResult.Tools))
	for _, tool := range toolsResult.Tools {
		tools = append(tools, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.InputSchema,
			},
		})
	}

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
			Tools: tools,
		},
	)
	if err != nil {
		return "", fmt.Errorf("OpenAI API call failed: %v", err)
	}

	// 处理工具调用
	if len(resp.Choices) > 0 && resp.Choices[0].Message.ToolCalls != nil {
		for _, toolCall := range resp.Choices[0].Message.ToolCalls {
			// 执行工具调用
			callRequest := mcp.CallToolRequest{}
			callRequest.Params.Name = toolCall.Function.Name

			// 解析参数
			var args map[string]interface{}
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
				return "", fmt.Errorf("failed to parse tool arguments: %v", err)
			}
			callRequest.Params.Arguments = args

			// 执行工具
			callResult, err := cli.CallTool(ctx, callRequest)
			if err != nil {
				return "", fmt.Errorf("tool execution failed: %v", err)
			}

			// 处理工具执行结果
			if len(callResult.Content) > 0 {
				if textContent, ok := callResult.Content[0].(mcp.TextContent); ok {
					return textContent.Text, nil
				}
			}
		}
	}

	return resp.Choices[0].Message.Content, nil
}

func ExecMcpTool(ctx context.Context, cli *client.SSEMCPClient) error {
	// 测试 CallTool
	callRequest := mcp.CallToolRequest{}
	callRequest.Params.Name = "test-tool"
	callRequest.Params.Arguments = map[string]interface{}{
		"parameter-1": "value1",
	}
	callResult, err := cli.CallTool(ctx, callRequest)
	if err != nil {
		return fmt.Errorf("CallTool failed: %v", err)
	}

	ret, err := json.MarshalIndent(callResult, "", "  ")
	if err != nil {
		return err
	}
	log.Printf("CallTool result: %s", string(ret))
	if len(callResult.Content) > 0 {
		if textContent, ok := callResult.Content[0].(mcp.TextContent); ok {
			log.Printf("CallTool result: %s", textContent.Text)
		}
	}
	return nil
}

func ListTools(ctx context.Context, cli *client.SSEMCPClient) {
	// 测试 ListTools
	toolsRequest := mcp.ListToolsRequest{}

	toolsResult, err := cli.ListTools(ctx, toolsRequest)
	if err != nil {
		log.Printf("ListTools failed: %v", err)
	} else {
		log.Printf("Available tools: %d", len(toolsResult.Tools))
		bytes, err := json.MarshalIndent(toolsResult.Tools, "", "  ")
		if err != nil {
			return
		}
		log.Printf("tools \n%s\n", string(bytes))
	}
}

func ListResources(ctx context.Context, cli *client.SSEMCPClient) {
	// 测试 ListResources
	resourcesRequest := mcp.ListResourcesRequest{}

	resourcesResult, err := cli.ListResources(ctx, resourcesRequest)
	if err != nil {
		log.Printf("ListResources failed: %v", err)
	} else {
		log.Printf("Available resources: %d", len(resourcesResult.Resources))
		bytes, err := json.MarshalIndent(resourcesResult.Resources, "", "  ")
		if err != nil {
			return
		}
		log.Printf("resources \n%s\n", string(bytes))
	}
}

func ReadResource(ctx context.Context, cli *client.SSEMCPClient) error {
	// 测试 ReadResource
	readRequest := mcp.ReadResourceRequest{}
	readRequest.Params.URI = "resource://testresource"

	resource, err := cli.ReadResource(ctx, readRequest)
	if err != nil {
		return fmt.Errorf("ReadResource failed: %v", err)
	}

	if len(resource.Contents) > 0 {
		if textContent, ok := resource.Contents[0].(mcp.TextResourceContents); ok {
			log.Printf("Resource content: %s", textContent.Text)
		}
	}
	return nil
}

func ListPrompts(ctx context.Context, cli *client.SSEMCPClient) {
	// 测试 ListPrompts
	promptsRequest := mcp.ListPromptsRequest{}

	promptsResult, err := cli.ListPrompts(ctx, promptsRequest)
	if err != nil {
		log.Printf("ListPrompts failed: %v", err)
	} else {
		log.Printf("Available prompts: %d", len(promptsResult.Prompts))
		bytes, err := json.MarshalIndent(promptsResult.Prompts, "", "  ")
		if err != nil {
			return
		}
		log.Printf("prompts \n%s\n", string(bytes))
	}
}

func ExecPrompt(ctx context.Context, cli *client.SSEMCPClient) error {
	// 测试 GetPrompt
	promptRequest := mcp.GetPromptRequest{}
	promptRequest.Params.Name = "test-prompt"
	promptRequest.Params.Arguments = map[string]string{
		"parameter-1": "测试参数值",
	}

	promptResult, err := cli.GetPrompt(ctx, promptRequest)
	if err != nil {
		return fmt.Errorf("GetPrompt failed: %v", err)
	}

	ret, err := json.MarshalIndent(promptResult, "", "  ")
	if err != nil {
		return err
	}
	log.Printf("GetPrompt result: %s", string(ret))

	if len(promptResult.Messages) > 0 {
		if textContent, ok := promptResult.Messages[0].Content.(mcp.TextContent); ok {
			log.Printf("Prompt message: %s", textContent.Text)
		}
	}
	return nil
}
func ExecPromptCodeReview(ctx context.Context, cli *client.SSEMCPClient) error {
	// 测试 GetPrompt
	promptRequest := mcp.GetPromptRequest{}
	promptRequest.Params.Name = "code_review"
	promptRequest.Params.Arguments = map[string]string{
		"pr_number": "3333333",
	}

	// 调用GetPrompt并处理可能的资源类型错误
	promptResult, err := cli.GetPrompt(ctx, promptRequest)
	if err != nil {
		if err.Error() == "unsupported resource type" {
			log.Printf("警告: 收到不支持的资源类型，继续执行")
			return nil
		}
		return fmt.Errorf("GetPrompt failed: %v", err)
	}

	log.Printf("Code Review Feedback:")
	for _, msg := range promptResult.Messages {
		switch content := msg.Content.(type) {
		case mcp.TextContent:
			log.Printf("Message: %s", content.Text)
		case mcp.EmbeddedResource:
			log.Printf("Resource URI: %s", content.Resource)
			if textResource, ok := content.Resource.(mcp.TextResourceContents); ok {
				log.Printf("Diff Content:\n%s", textResource.Text)
			} else {
				log.Printf("Unsupported resource type: %T", content.Resource)
			}
		}
	}
	return nil
}

func ReadUserProfile(ctx context.Context, cli *client.SSEMCPClient, userID string) error {
	// 测试动态资源模板
	readRequest := mcp.ReadResourceRequest{}
	readRequest.Params.URI = fmt.Sprintf("users://%s/profile", userID)

	// Method参数不需要设置
	resource, err := cli.ReadResource(ctx, readRequest)
	if err != nil {
		return fmt.Errorf("ReadUserProfile failed: %v", err)
	}

	if len(resource.Contents) > 0 {
		if textContent, ok := resource.Contents[0].(mcp.TextResourceContents); ok {
			log.Printf("User profile content: %s", textContent.Text)
		}
	}
	return nil
}
