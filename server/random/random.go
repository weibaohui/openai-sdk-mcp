package main

import (
	"context"
	"fmt"
	"math/rand/v2"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	// 创建一个新的 MCP 服务器
	s := server.NewMCPServer(
		"Random Number Generator Server",
		"1.0.0",
		server.WithResourceCapabilities(true, true),
		server.WithPromptCapabilities(true),
		server.WithLogging(),
	)

	// 添加生成单个随机数的工具
	s.AddTool(
		mcp.NewTool(
			"generate-random",
			mcp.WithDescription("Generate a random number within the specified range"),
			mcp.WithNumber("min", mcp.Description("Minimum value of the range")),
			mcp.WithNumber("max", mcp.Description("Maximum value of the range")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			min := request.Params.Arguments["min"].(float64)
			max := request.Params.Arguments["max"].(float64)
			result := min + rand.Float64()*(max-min)
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Generated random number: %.2f", result),
					},
				},
			}, nil
		},
	)

	// 添加生成随机数序列的工具
	s.AddTool(
		mcp.NewTool(
			"generate-random-sequence",
			mcp.WithDescription("Generate a sequence of random numbers within the specified range"),
			mcp.WithNumber("min", mcp.Description("Minimum value of the range")),
			mcp.WithNumber("max", mcp.Description("Maximum value of the range")),
			mcp.WithNumber("count", mcp.Description("Number of random numbers to generate")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			min := request.Params.Arguments["min"].(float64)
			max := request.Params.Arguments["max"].(float64)
			count := int(request.Params.Arguments["count"].(float64))

			var numbers []float64
			for i := 0; i < count; i++ {
				numbers = append(numbers, min+rand.Float64()*(max-min))
			}

			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Generated random numbers: %.2f", numbers),
					},
				},
			}, nil
		},
	)

	// 创建 SSE 服务器
	sseServer := server.NewSSEServer(s, server.WithBaseURL("http://localhost:9292"))

	// 启动服务器，监听 9292 端口
	err := sseServer.Start(":9292")
	if err != nil {
		fmt.Printf("Server error: %v\n", err)
	}

}
