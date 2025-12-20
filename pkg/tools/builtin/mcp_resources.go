package builtin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/astercloud/aster/pkg/tools"
)

// MCPResource MCP 资源定义
type MCPResource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
	Server      string `json:"server"`
}

// MCPResourceContent MCP 资源内容
type MCPResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"`
}

// ListMcpResourcesTool 列出 MCP 资源工具
type ListMcpResourcesTool struct{}

// NewListMcpResourcesTool 创建 ListMcpResources 工具
func NewListMcpResourcesTool(config map[string]any) (tools.Tool, error) {
	return &ListMcpResourcesTool{}, nil
}

func (t *ListMcpResourcesTool) Name() string {
	return "ListMcpResources"
}

func (t *ListMcpResourcesTool) Description() string {
	return "Lists available MCP resources from connected servers"
}

func (t *ListMcpResourcesTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"server": map[string]any{
				"type":        "string",
				"description": "Optional server name to filter resources by",
			},
		},
	}
}

func (t *ListMcpResourcesTool) Execute(ctx context.Context, input map[string]any, tc *tools.ToolContext) (any, error) {
	serverFilter := GetStringParam(input, "server", "")
	start := time.Now()

	// 获取 MCP 资源列表
	resources, err := t.listResources(ctx, serverFilter, tc)
	if err != nil {
		return map[string]any{
			"ok":    false,
			"error": fmt.Sprintf("failed to list MCP resources: %v", err),
			"recommendations": []string{
				"Check if MCP servers are connected",
				"Verify server name if filter is specified",
				"Ensure MCP servers support resources",
			},
			"duration_ms": time.Since(start).Milliseconds(),
		}, nil
	}

	return map[string]any{
		"ok":          true,
		"resources":   resources,
		"total":       len(resources),
		"server":      serverFilter,
		"duration_ms": time.Since(start).Milliseconds(),
	}, nil
}

func (t *ListMcpResourcesTool) listResources(ctx context.Context, serverFilter string, tc *tools.ToolContext) ([]MCPResource, error) {
	resources := make([]MCPResource, 0)

	// 从 ToolContext 获取 MCP Manager
	if tc == nil || tc.MCPManager == nil {
		return resources, nil
	}

	// 获取所有服务器
	servers := tc.MCPManager.ListServers()

	for _, serverID := range servers {
		// 如果指定了服务器过滤，跳过不匹配的
		if serverFilter != "" && serverID != serverFilter {
			continue
		}

		server, exists := tc.MCPManager.GetServer(serverID)
		if !exists {
			continue
		}

		// 获取服务器的资源列表
		serverResources, err := t.getServerResources(ctx, server, serverID)
		if err != nil {
			// 记录错误但继续处理其他服务器
			continue
		}

		resources = append(resources, serverResources...)
	}

	return resources, nil
}

func (t *ListMcpResourcesTool) getServerResources(ctx context.Context, server any, serverID string) ([]MCPResource, error) {
	// 尝试调用服务器的 ListResources 方法
	// 这里需要根据实际的 MCP 服务器接口来实现
	// 当前返回空列表，等待 MCP 服务器支持资源列表功能
	return []MCPResource{}, nil
}

func (t *ListMcpResourcesTool) Prompt() string {
	return `Lists available MCP resources from connected servers.

This tool queries connected MCP servers for available resources that can be read.

Parameters:
- server: (optional) Filter resources by server name

Returns:
- resources: Array of available resources with uri, name, description, mimeType, and server
- total: Total number of resources found

Use this tool to discover what MCP resources are available before reading them with ReadMcpResource.`
}

// Examples 返回 ListMcpResources 工具的使用示例
func (t *ListMcpResourcesTool) Examples() []tools.ToolExample {
	return []tools.ToolExample{
		{
			Description: "List all MCP resources",
			Input:       map[string]any{},
		},
		{
			Description: "List resources from a specific server",
			Input: map[string]any{
				"server": "my-mcp-server",
			},
		},
	}
}

// ReadMcpResourceTool 读取 MCP 资源工具
type ReadMcpResourceTool struct{}

// NewReadMcpResourceTool 创建 ReadMcpResource 工具
func NewReadMcpResourceTool(config map[string]any) (tools.Tool, error) {
	return &ReadMcpResourceTool{}, nil
}

func (t *ReadMcpResourceTool) Name() string {
	return "ReadMcpResource"
}

func (t *ReadMcpResourceTool) Description() string {
	return "Reads a specific MCP resource from a server"
}

func (t *ReadMcpResourceTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"server": map[string]any{
				"type":        "string",
				"description": "The MCP server name",
			},
			"uri": map[string]any{
				"type":        "string",
				"description": "The resource URI to read",
			},
		},
		"required": []string{"server", "uri"},
	}
}

func (t *ReadMcpResourceTool) Execute(ctx context.Context, input map[string]any, tc *tools.ToolContext) (any, error) {
	// 验证必需参数
	if err := ValidateRequired(input, []string{"server", "uri"}); err != nil {
		return NewClaudeErrorResponse(err), nil
	}

	server := GetStringParam(input, "server", "")
	uri := GetStringParam(input, "uri", "")
	start := time.Now()

	if server == "" {
		return NewClaudeErrorResponse(errors.New("server is required")), nil
	}

	if uri == "" {
		return NewClaudeErrorResponse(errors.New("uri is required")), nil
	}

	// 读取 MCP 资源
	contents, err := t.readResource(ctx, server, uri, tc)
	if err != nil {
		return map[string]any{
			"ok":    false,
			"error": fmt.Sprintf("failed to read MCP resource: %v", err),
			"recommendations": []string{
				"Verify the server name is correct",
				"Check if the resource URI exists",
				"Use ListMcpResources to discover available resources",
			},
			"server":      server,
			"uri":         uri,
			"duration_ms": time.Since(start).Milliseconds(),
		}, nil
	}

	return map[string]any{
		"ok":          true,
		"contents":    contents,
		"server":      server,
		"uri":         uri,
		"duration_ms": time.Since(start).Milliseconds(),
	}, nil
}

func (t *ReadMcpResourceTool) readResource(ctx context.Context, serverName, uri string, tc *tools.ToolContext) ([]MCPResourceContent, error) {
	if tc == nil || tc.MCPManager == nil {
		return nil, errors.New("MCP manager not available")
	}

	server, exists := tc.MCPManager.GetServer(serverName)
	if !exists {
		return nil, fmt.Errorf("server not found: %s", serverName)
	}

	// 尝试调用服务器的 ReadResource 方法
	// 这里需要根据实际的 MCP 服务器接口来实现
	_ = server // 使用 server 变量

	return nil, errors.New("resource reading not yet implemented for this server")
}

func (t *ReadMcpResourceTool) Prompt() string {
	return `Reads a specific MCP resource from a server.

This tool retrieves the contents of an MCP resource identified by its URI.

Parameters:
- server: (required) The MCP server name
- uri: (required) The resource URI to read

Returns:
- contents: Array of resource contents with uri, mimeType, text or blob
- server: The server that provided the resource

Use ListMcpResources first to discover available resources and their URIs.`
}

// Examples 返回 ReadMcpResource 工具的使用示例
func (t *ReadMcpResourceTool) Examples() []tools.ToolExample {
	return []tools.ToolExample{
		{
			Description: "Read a specific MCP resource",
			Input: map[string]any{
				"server": "my-mcp-server",
				"uri":    "file:///path/to/resource",
			},
		},
	}
}
