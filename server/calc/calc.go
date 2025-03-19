package main

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	// 创建一个新的 MCP 服务器
	s := server.NewMCPServer(
		"Calculator Server",
		"1.0.0",
		server.WithResourceCapabilities(true, true),
		server.WithPromptCapabilities(true),
		server.WithLogging(),
	)

	// 添加一个加法计算工具
	s.AddTool(
		mcp.NewTool(
			"add-numbers",
			mcp.WithDescription("Add two numbers together"),
			mcp.WithNumber("number1", mcp.Description("First number to add")),
			mcp.WithNumber("number2", mcp.Description("Second number to add")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			num1 := request.Params.Arguments["number1"].(float64)
			num2 := request.Params.Arguments["number2"].(float64)
			result := num1 + num2

			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("The sum of %.2f and %.2f is %.2f", num1, num2, result),
					},
				},
			}, nil
		},
	)

	// 创建 SSE 服务器
	sseServer := server.NewSSEServer(s, server.WithBaseURL("http://localhost:9293"))

	// 启动服务器，监听 9293 端口
	err := sseServer.Start(":9293")
	if err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
