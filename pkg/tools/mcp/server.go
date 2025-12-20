package mcp

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/astercloud/aster/pkg/sandbox/cloud"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/tools/search"
)

// MCPServer MCP Server 连接管理器
type MCPServer struct {
	mu       sync.RWMutex
	client   *cloud.MCPClient
	serverID string
	tools    []cloud.MCPTool
	registry *tools.Registry
}

// MCPServerConfig MCP Server 配置
type MCPServerConfig struct {
	ServerID        string
	Endpoint        string
	AccessKeyID     string
	AccessKeySecret string
	SecurityToken   string
}

// NewMCPServer 创建 MCP Server 连接
func NewMCPServer(config *MCPServerConfig, registry *tools.Registry) (*MCPServer, error) {
	if config.ServerID == "" {
		return nil, errors.New("server_id is required")
	}

	if config.Endpoint == "" {
		return nil, errors.New("endpoint is required")
	}

	// 创建 MCP 客户端
	client := cloud.NewMCPClient(&cloud.MCPClientConfig{
		Endpoint:        config.Endpoint,
		AccessKeyID:     config.AccessKeyID,
		AccessKeySecret: config.AccessKeySecret,
		SecurityToken:   config.SecurityToken,
	})

	return &MCPServer{
		client:   client,
		serverID: config.ServerID,
		tools:    make([]cloud.MCPTool, 0),
		registry: registry,
	}, nil
}

// Connect 连接到 MCP Server 并发现工具
func (s *MCPServer) Connect(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 列出服务端提供的工具
	mcpTools, err := s.client.ListTools(ctx)
	if err != nil {
		return fmt.Errorf("list mcp tools: %w", err)
	}

	s.tools = mcpTools
	return nil
}

// RegisterTools 将 MCP 工具注册到 aster Registry
func (s *MCPServer) RegisterTools() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.tools) == 0 {
		return errors.New("no tools available, call Connect() first")
	}

	// 为每个 MCP 工具创建工厂并注册
	for _, mcpTool := range s.tools {
		// 使用 server_id 作为前缀避免工具名冲突
		toolName := fmt.Sprintf("%s:%s", s.serverID, mcpTool.Name)

		// 创建工具工厂
		factory := ToolFactory(s.client, mcpTool)

		// 注册到 Registry
		s.registry.Register(toolName, factory)
	}

	return nil
}

// ListTools 返回已发现的工具列表
func (s *MCPServer) ListTools() []cloud.MCPTool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 返回副本
	tools := make([]cloud.MCPTool, len(s.tools))
	copy(tools, s.tools)
	return tools
}

// GetToolCount 获取工具数量
func (s *MCPServer) GetToolCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.tools)
}

// GetServerID 获取 Server ID
func (s *MCPServer) GetServerID() string {
	return s.serverID
}

// GetClient 获取底层 MCP 客户端
func (s *MCPServer) GetClient() *cloud.MCPClient {
	return s.client
}

// GetToolIndexEntries 获取工具索引条目（用于延迟加载）
// 返回工具的元数据，但不实际注册到 Registry
func (s *MCPServer) GetToolIndexEntries() []search.ToolIndexEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := make([]search.ToolIndexEntry, 0, len(s.tools))
	for _, mcpTool := range s.tools {
		// 使用 server_id 作为前缀
		toolName := fmt.Sprintf("%s:%s", s.serverID, mcpTool.Name)

		entry := search.ToolIndexEntry{
			Name:        toolName,
			Description: mcpTool.Description,
			InputSchema: mcpTool.InputSchema,
			Category:    "mcp",
			Keywords:    []string{"mcp", s.serverID, mcpTool.Name},
			Deferred:    true,
			Source:      "mcp",
			Metadata: map[string]any{
				"server_id":     s.serverID,
				"original_name": mcpTool.Name,
				"mcp_tool":      true,
			},
		}
		entries = append(entries, entry)
	}

	return entries
}

// RegisterToolDeferred 延迟注册单个工具（按需激活时调用）
func (s *MCPServer) RegisterToolDeferred(toolName string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 查找对应的 MCP 工具
	for _, mcpTool := range s.tools {
		fullName := fmt.Sprintf("%s:%s", s.serverID, mcpTool.Name)
		if fullName == toolName {
			// 创建工具工厂并注册
			factory := ToolFactory(s.client, mcpTool)
			s.registry.Register(toolName, factory)
			return nil
		}
	}

	return fmt.Errorf("tool not found: %s", toolName)
}

// IndexToolsToIndex 将工具添加到工具索引（延迟加载模式）
func (s *MCPServer) IndexToolsToIndex(index *search.ToolIndex) error {
	entries := s.GetToolIndexEntries()
	for _, entry := range entries {
		if err := index.IndexToolEntry(entry); err != nil {
			return fmt.Errorf("index tool %s: %w", entry.Name, err)
		}
	}
	return nil
}
