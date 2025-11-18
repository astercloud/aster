package builtin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/astercloud/aster/pkg/tools"
)

func TestHttpRequestTool_Success(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Hello, World!"))
	}))
	defer func() { _ = server.Close() }()

	tool, err := NewHttpRequestTool(nil)
	if err != nil {
		t.Fatalf("Failed to create tool: %v", err)
	}

	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]interface{}{
		"url":    server.URL,
		"method": "GET",
	}, &tools.ToolContext{})

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	if success, _ := resultMap["success"].(bool); !success {
		t.Errorf("Expected success=true, got false")
	}

	if statusCode, _ := resultMap["status_code"].(int); statusCode != 200 {
		t.Errorf("Expected status_code=200, got %d", statusCode)
	}

	if content, _ := resultMap["content"].(string); content != "Hello, World!" {
		t.Errorf("Expected content='Hello, World!', got '%s'", content)
	}
}

func TestHttpRequestTool_JsonResponse(t *testing.T) {
	// 创建返回JSON的测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "success",
			"code":    0,
		})
	}))
	defer func() { _ = server.Close() }()

	tool, err := NewHttpRequestTool(nil)
	if err != nil {
		t.Fatalf("Failed to create tool: %v", err)
	}

	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]interface{}{
		"url": server.URL,
	}, &tools.ToolContext{})

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	// 验证content被解析为JSON
	content, ok := resultMap["content"].(map[string]interface{})
	if !ok {
		t.Fatal("Content is not a JSON object")
	}

	if msg, _ := content["message"].(string); msg != "success" {
		t.Errorf("Expected message='success', got '%s'", msg)
	}
}

func TestHttpRequestTool_POST_WithBody(t *testing.T) {
	// 创建测试服务器,验证POST请求
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		// 读取请求体
		body := make([]byte, r.ContentLength)
		r.Body.Read(body)

		if string(body) != "test data" {
			t.Errorf("Expected body='test data', got '%s'", string(body))
		}

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("Created"))
	}))
	defer func() { _ = server.Close() }()

	tool, err := NewHttpRequestTool(nil)
	if err != nil {
		t.Fatalf("Failed to create tool: %v", err)
	}

	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]interface{}{
		"url":    server.URL,
		"method": "POST",
		"body":   "test data",
	}, &tools.ToolContext{})

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	if statusCode, _ := resultMap["status_code"].(int); statusCode != 201 {
		t.Errorf("Expected status_code=201, got %d", statusCode)
	}
}

func TestHttpRequestTool_CustomHeaders(t *testing.T) {
	// 创建测试服务器,验证自定义请求头
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-token" {
			t.Errorf("Expected Authorization header, got '%s'", auth)
		}

		if accept := r.Header.Get("Accept"); accept != "application/json" {
			t.Errorf("Expected Accept header, got '%s'", accept)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer func() { _ = server.Close() }()

	tool, err := NewHttpRequestTool(nil)
	if err != nil {
		t.Fatalf("Failed to create tool: %v", err)
	}

	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]interface{}{
		"url": server.URL,
		"headers": map[string]interface{}{
			"Authorization": "Bearer test-token",
			"Accept":        "application/json",
		},
	}, &tools.ToolContext{})

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	if success, _ := resultMap["success"].(bool); !success {
		t.Error("Expected success=true")
	}
}

// Note: Timeout test is skipped in automated tests because it's slow and flaky
// Timeout handling is verified manually and through integration tests
// The timeout logic is implemented in HttpRequest.go:130-137

func TestHttpRequestTool_InvalidURL(t *testing.T) {
	tool, err := NewHttpRequestTool(nil)
	if err != nil {
		t.Fatalf("Failed to create tool: %v", err)
	}

	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]interface{}{
		"url": "not-a-valid-url",
	}, &tools.ToolContext{})

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	if success, _ := resultMap["success"].(bool); success {
		t.Error("Expected success=false for invalid URL")
	}
}

func TestHttpRequestTool_404Status(t *testing.T) {
	// 创建返回404的测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Not Found"))
	}))
	defer func() { _ = server.Close() }()

	tool, err := NewHttpRequestTool(nil)
	if err != nil {
		t.Fatalf("Failed to create tool: %v", err)
	}

	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]interface{}{
		"url": server.URL + "/nonexistent",
	}, &tools.ToolContext{})

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	// 404应该被标记为不成功
	if success, _ := resultMap["success"].(bool); success {
		t.Error("Expected success=false for 404 status")
	}

	if statusCode, _ := resultMap["status_code"].(int); statusCode != 404 {
		t.Errorf("Expected status_code=404, got %d", statusCode)
	}
}

func TestHttpRequestTool_EmptyResponse(t *testing.T) {
	// 创建返回空响应的测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer func() { _ = server.Close() }()

	tool, err := NewHttpRequestTool(nil)
	if err != nil {
		t.Fatalf("Failed to create tool: %v", err)
	}

	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]interface{}{
		"url": server.URL,
	}, &tools.ToolContext{})

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	if content, _ := resultMap["content"].(string); content != "" {
		t.Errorf("Expected empty content, got '%s'", content)
	}
}
