# openai-sdk-mcp

一个基于Go语言的OpenAI SDK与MCP（Multi-Client Protocol）集成项目，支持多服务器连接管理和OpenAI API调用功能。

## 功能特性

- OpenAI API集成：支持自定义API密钥和BaseURL配置
- MCP服务器管理：
  - 支持多服务器并发连接
  - 服务器配置动态管理
  - 连接状态监控
- 内置功能模块：
  - 计算模块：支持基础数学运算
  - 随机数模块：生成随机数并进行计算

## 安装

```bash
go get github.com/weibaohui/openai-sdk-mcp
```

## 使用示例

```go
// 初始化OpenAI客户端
openaiConfig := OpenAIConfig{
    APIKey:  os.Getenv("OPENAI_API_KEY"),
    BaseURL: os.Getenv("OPENAI_BASE_URL"),
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
}

// 处理请求
content, result, err := host.ProcessWithOpenAI(ctx, "请计算22+33的结果")
```

## 项目结构

```
.
├── client/         # 客户端实现
│   └── client.go   # OpenAI客户端和MCP服务器管理
├── server/         # 服务器端实现
│   ├── calc/       # 计算模块
│   └── random/     # 随机数模块
├── go.mod          # Go模块定义
└── README.md       # 项目文档
```

## 依赖

- github.com/mark3labs/mcp-go v0.14.1
- github.com/sashabaranov/go-openai v1.38.0

## 许可证

MIT License