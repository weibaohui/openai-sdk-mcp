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
		"My Test Server",
		"1.0.0",
		server.WithResourceCapabilities(true, true),
		server.WithPromptCapabilities(true),
		// server.WithToolCapabilities(true),
		server.WithLogging(),
	)

	// 添加一个测试资源
	s.AddResource(
		mcp.Resource{
			URI:  "resource://testresource",
			Name: "My Resource",
		},
		func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      "resource://testresource",
					MIMEType: "text/plain",
					Text:     "我是嵌入的资源",
				},
			}, nil
		},
	)

	// 添加一个测试工具
	s.AddTool(
		mcp.NewTool(
			"test-tool",
			mcp.WithDescription("Test tool"),
			mcp.WithString("parameter-1", mcp.Description("A string tool parameter")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			paramValue := request.Params.Arguments["parameter-1"].(string)
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: "hello Input parameter: " + paramValue,
					},
				},
			}, nil
		},
	)

	// 添加一个测试提示处理器
	s.AddPrompt(
		mcp.Prompt{
			Name:        "test-prompt",
			Description: "一个简单的提示处理器示例",
			Arguments: []mcp.PromptArgument{
				{
					Name:        "parameter-1",
					Description: "参数1",
					Required:    true,
				},
			},
		},
		func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			p1 := request.Params.Arguments["parameter-1"]
			return &mcp.GetPromptResult{
				Messages: []mcp.PromptMessage{
					{
						Role: mcp.RoleUser,
						Content: mcp.TextContent{
							Type: "text",
							Text: fmt.Sprintf("你输入的内容是: %s!", p1),
						},
					},
				},
			}, nil
		},
	)

	// Code review prompt with embedded resource
	s.AddPrompt(mcp.NewPrompt("code_review",
		mcp.WithPromptDescription("Code review assistance"),
		mcp.WithArgument("pr_number",
			mcp.ArgumentDescription("Pull request number to review"),
			mcp.RequiredArgument(),
		),
	), func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		prNumber := request.Params.Arguments["pr_number"]
		if prNumber == "" {
			return nil, fmt.Errorf("pr_number is required")
		}

		return mcp.NewGetPromptResult(
			"Code review assistance",
			[]mcp.PromptMessage{
				mcp.NewPromptMessage(
					mcp.RoleAssistant,
					mcp.NewTextContent("You are a helpful code reviewer. Review the changes and provide constructive feedback."),
				),
				// mcp.NewPromptMessage(
				// 	mcp.RoleAssistant,
				// 	mcp.NewEmbeddedResource(mcp.TextResourceContents{
				// 		URI:      fmt.Sprintf("git://pulls/%s/diff", prNumber),
				// 		MIMEType: "text/x-diff",
				// 	}),
				// ),
			},
		), nil
	})

	// Dynamic resource example - user profiles by ID
	template := mcp.NewResourceTemplate(
		"users://{id}/profile",
		"User Profile",
		mcp.WithTemplateDescription("Returns user profile information"),
		mcp.WithTemplateMIMEType("application/json"),
	)

	// Add template with its handler
	s.AddResourceTemplate(template, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		// Extract ID from the URI using regex matching
		// The server automatically matches URIs to templates

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "application/json",
				Text:     "{a:true}",
			},
		}, nil
	})

	// 创建 SSE 服务器
	sseServer := server.NewSSEServer(s, server.WithBaseURL("http://localhost:9292"))

	// 启动服务器，监听 9292 端口
	err := sseServer.Start(":9292")
	if err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
	// // 启动服务器使用标准输入输出
	// if err := server.ServeStdio(s); err != nil {
	// 	log.Fatalf("Server error: %v", err)
	// }
}
