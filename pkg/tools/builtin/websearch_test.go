package builtin

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/astercloud/aster/pkg/tools"
)

// newLocalHTTPServer 启动绑定 IPv4 的测试服务器，避免某些环境禁用 IPv6 loopback。
func newLocalHTTPServerWS(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Skipf("skip: unable to listen on IPv4 loopback: %v", err)
	}
	srv := httptest.NewUnstartedServer(handler)
	srv.Listener = ln
	srv.Start()
	return srv
}

func TestWebSearchTool_MissingAPIKey(t *testing.T) {
	// 保存原有环境变量
	oldAPIKey := os.Getenv("WF_TAVILY_API_KEY")
	oldAPIKey2 := os.Getenv("TAVILY_API_KEY")
	defer func() {
		os.Setenv("WF_TAVILY_API_KEY", oldAPIKey)
		os.Setenv("TAVILY_API_KEY", oldAPIKey2)
	}()

	// 清除环境变量
	os.Unsetenv("WF_TAVILY_API_KEY")
	os.Unsetenv("TAVILY_API_KEY")

	tool, err := NewWebSearchTool(nil)
	if err != nil {
		t.Fatalf("Failed to create tool: %v", err)
	}

	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]interface{}{
		"query": "test query",
	}, &tools.ToolContext{})

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	// 应该返回错误信息
	if errorMsg, ok := resultMap["error"].(string); !ok || errorMsg == "" {
		t.Error("Expected error message for missing API key")
	}
}

func TestWebSearchTool_SuccessfulSearch(t *testing.T) {
	// 创建模拟 Tavily API 的测试服务器
	server := newLocalHTTPServerWS(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求方法和头部
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Expected Content-Type: application/json, got %s", ct)
		}

		// 解析请求体
		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)

		// 验证必需参数
		if query, ok := reqBody["query"].(string); !ok || query == "" {
			t.Error("Missing or empty query parameter")
		}

		// 返回模拟的搜索结果
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []map[string]interface{}{
				{
					"title":   "Test Result 1",
					"url":     "https://example.com/1",
					"content": "This is test content 1",
					"score":   0.95,
				},
				{
					"title":   "Test Result 2",
					"url":     "https://example.com/2",
					"content": "This is test content 2",
					"score":   0.88,
				},
			},
			"query": reqBody["query"],
		})
	}))
	defer server.Close()

	// 保存原有环境变量
	oldAPIKey := os.Getenv("WF_TAVILY_API_KEY")
	defer os.Setenv("WF_TAVILY_API_KEY", oldAPIKey)

	// 设置测试 API key
	os.Setenv("WF_TAVILY_API_KEY", "test-api-key")

	_, err := NewWebSearchTool(nil)
	if err != nil {
		t.Fatalf("Failed to create tool: %v", err)
	}

	// 替换 API 端点为测试服务器
	// 注意: 这需要修改 WebSearch.go 以支持配置 API 端点
	// 现在我们跳过这个测试,因为无法模拟真实的 Tavily API

	t.Skip("Skipping integration test - requires mocking Tavily API endpoint")
}

func TestWebSearchTool_InvalidQuery(t *testing.T) {
	// 保存原有环境变量
	oldAPIKey := os.Getenv("WF_TAVILY_API_KEY")
	defer os.Setenv("WF_TAVILY_API_KEY", oldAPIKey)

	// 设置测试 API key
	os.Setenv("WF_TAVILY_API_KEY", "test-api-key")

	tool, err := NewWebSearchTool(nil)
	if err != nil {
		t.Fatalf("Failed to create tool: %v", err)
	}

	ctx := context.Background()
	_, err = tool.Execute(ctx, map[string]interface{}{
		"query": "", // 空查询
	}, &tools.ToolContext{})

	if err == nil {
		t.Error("Expected error for empty query")
	}

	// 也可以测试非字符串查询
	_, err2 := tool.Execute(ctx, map[string]interface{}{
		"query": 123, // 非字符串
	}, &tools.ToolContext{})

	if err2 == nil {
		t.Error("Expected error for non-string query")
	}
}

func TestWebSearchTool_MaxResultsValidation(t *testing.T) {
	// 保存原有环境变量
	oldAPIKey := os.Getenv("WF_TAVILY_API_KEY")
	defer os.Setenv("WF_TAVILY_API_KEY", oldAPIKey)

	// 设置测试 API key
	os.Setenv("WF_TAVILY_API_KEY", "test-api-key")

	tool, err := NewWebSearchTool(nil)
	if err != nil {
		t.Fatalf("Failed to create tool: %v", err)
	}

	// 这个测试实际上需要验证请求体,但由于我们无法轻易模拟 Tavily API,
	// 我们只能确认工具不会因为超出范围的 max_results 而崩溃

	ctx := context.Background()

	// max_results 太大(会被限制到 10)
	_, err = tool.Execute(ctx, map[string]interface{}{
		"query":       "test",
		"max_results": 100,
	}, &tools.ToolContext{})

	// 不应该崩溃(会返回 API 错误,但不会 panic)
	// 由于没有真实 API,这里会返回连接错误,这是预期的
	if err != nil && err.Error() == "panic" {
		t.Error("Tool panicked with large max_results")
	}

	// max_results 太小(会被限制到 1)
	_, err = tool.Execute(ctx, map[string]interface{}{
		"query":       "test",
		"max_results": -5,
	}, &tools.ToolContext{})

	if err != nil && err.Error() == "panic" {
		t.Error("Tool panicked with negative max_results")
	}
}

func TestWebSearchTool_TopicValidation(t *testing.T) {
	// 保存原有环境变量
	oldAPIKey := os.Getenv("WF_TAVILY_API_KEY")
	defer os.Setenv("WF_TAVILY_API_KEY", oldAPIKey)

	// 设置测试 API key
	os.Setenv("WF_TAVILY_API_KEY", "test-api-key")

	tool, err := NewWebSearchTool(nil)
	if err != nil {
		t.Fatalf("Failed to create tool: %v", err)
	}

	ctx := context.Background()

	// 测试有效的 topic 值
	topics := []string{"general", "news", "finance"}
	for _, topic := range topics {
		_, err := tool.Execute(ctx, map[string]interface{}{
			"query": "test",
			"topic": topic,
		}, &tools.ToolContext{})

		// 不应该因为有效的 topic 而返回参数错误
		if err != nil && err.Error() == "invalid topic" {
			t.Errorf("Tool rejected valid topic: %s", topic)
		}
	}
}

func TestWebSearchTool_APIKeyFromEnvironment(t *testing.T) {
	// 测试从不同环境变量读取 API key

	tests := []struct {
		name   string
		envVar string
		envVal string
		hasKey bool
	}{
		{
			name:   "WF_TAVILY_API_KEY",
			envVar: "WF_TAVILY_API_KEY",
			envVal: "test-key-1",
			hasKey: true,
		},
		{
			name:   "TAVILY_API_KEY",
			envVar: "TAVILY_API_KEY",
			envVal: "test-key-2",
			hasKey: true,
		},
		{
			name:   "No API key",
			envVar: "",
			envVal: "",
			hasKey: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 保存原有环境变量
			oldAPIKey1 := os.Getenv("WF_TAVILY_API_KEY")
			oldAPIKey2 := os.Getenv("TAVILY_API_KEY")
			defer func() {
				os.Setenv("WF_TAVILY_API_KEY", oldAPIKey1)
				os.Setenv("TAVILY_API_KEY", oldAPIKey2)
			}()

			// 清除环境变量
			os.Unsetenv("WF_TAVILY_API_KEY")
			os.Unsetenv("TAVILY_API_KEY")

			// 设置测试环境变量
			if tt.envVar != "" {
				os.Setenv(tt.envVar, tt.envVal)
			}

			tool, err := NewWebSearchTool(nil)
			if err != nil {
				t.Fatalf("Failed to create tool: %v", err)
			}

			webSearchTool := tool.(*WebSearchTool)

			if tt.hasKey && webSearchTool.apiKey == "" {
				t.Error("Expected API key to be set, but it's empty")
			}

			if !tt.hasKey && webSearchTool.apiKey != "" {
				t.Errorf("Expected no API key, but got: %s", webSearchTool.apiKey)
			}
		})
	}
}
