package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sashabaranov/go-openai"
)

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
