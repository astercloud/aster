package middleware

import (
	"context"
	"strings"
	"testing"

	"github.com/astercloud/aster/pkg/backends"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/types"
)

// TestFilesystemMiddleware_PathValidation 测试路径安全验证
func TestFilesystemMiddleware_PathValidation(t *testing.T) {
	backend := backends.NewStateBackend()

	tests := []struct {
		name          string
		config        *FilesystemMiddlewareConfig
		inputPath     string
		expectError   bool
		errorContains string
	}{
		{
			name: "禁用验证 - 允许所有路径",
			config: &FilesystemMiddlewareConfig{
				Backend:              backend,
				EnablePathValidation: false,
			},
			inputPath:   "../etc/passwd",
			expectError: false,
		},
		{
			name: "启用验证 - 阻止路径遍历 (..)",
			config: &FilesystemMiddlewareConfig{
				Backend:              backend,
				EnablePathValidation: true,
			},
			inputPath:     "../etc/passwd",
			expectError:   true,
			errorContains: "路径遍历不允许",
		},
		{
			name: "启用验证 - 阻止 home 目录访问 (~)",
			config: &FilesystemMiddlewareConfig{
				Backend:              backend,
				EnablePathValidation: true,
			},
			inputPath:     "~/secrets.txt",
			expectError:   true,
			errorContains: "路径遍历不允许",
		},
		{
			name: "启用验证 + 前缀白名单 - 允许合法路径",
			config: &FilesystemMiddlewareConfig{
				Backend:              backend,
				EnablePathValidation: true,
				AllowedPathPrefixes:  []string{"/workspace/", "/tmp/"},
			},
			inputPath:   "/workspace/file.txt",
			expectError: false,
		},
		{
			name: "启用验证 + 前缀白名单 - 阻止非白名单路径",
			config: &FilesystemMiddlewareConfig{
				Backend:              backend,
				EnablePathValidation: true,
				AllowedPathPrefixes:  []string{"/workspace/", "/tmp/"},
			},
			inputPath:     "/etc/passwd",
			expectError:   true,
			errorContains: "路径必须以以下前缀之一开头",
		},
		{
			name: "启用验证 - 路径规范化",
			config: &FilesystemMiddlewareConfig{
				Backend:              backend,
				EnablePathValidation: true,
				AllowedPathPrefixes:  []string{"/workspace/"},
			},
			inputPath:   "workspace/file.txt", // 缺少前导 /
			expectError: false,                // 规范化后变为 /workspace/file.txt
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := NewFilesystemMiddleware(tt.config)

			// 测试 validatePath 函数
			validatedPath, err := middleware.validatePath(tt.inputPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if validatedPath == "" {
					t.Errorf("Expected non-empty validated path")
				}
			}
		})
	}
}

// TestFilesystemTools_PathValidationIntegration 测试工具集成路径验证
func TestFilesystemTools_PathValidationIntegration(t *testing.T) {
	ctx := context.Background()
	backend := backends.NewStateBackend()

	// 准备测试数据
	if _, err := backend.Write(ctx, "/workspace/safe.txt", "safe content"); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if _, err := backend.Write(ctx, "/etc/dangerous.txt", "dangerous content"); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	middleware := NewFilesystemMiddleware(&FilesystemMiddlewareConfig{
		Backend:              backend,
		EnablePathValidation: true,
		AllowedPathPrefixes:  []string{"/workspace/"},
	})

	// 注意: 只测试 backend-based 工具 (Ls, Edit, Glob, Grep)
	// Read 和 Write 来自 builtin,需要 ToolContext/Sandbox,不在此测试
	toolTests := []struct {
		name          string
		toolName      string
		input         map[string]any
		expectOk      bool
		errorContains string
	}{
		{
			name:     "Ls - 允许的路径",
			toolName: "Ls",
			input:    map[string]any{"path": "/workspace/"},
			expectOk: true,
		},
		{
			name:          "Ls - 阻止路径遍历",
			toolName:      "Ls",
			input:         map[string]any{"path": "../etc/"},
			expectOk:      false,
			errorContains: "path validation failed",
		},
		{
			name:     "Edit - 允许的路径",
			toolName: "Edit",
			input: map[string]any{
				"path":        "/workspace/safe.txt",
				"old_content": "safe",
				"new_content": "safer",
			},
			expectOk: true,
		},
		{
			name:     "Edit - 阻止非白名单路径",
			toolName: "Edit",
			input: map[string]any{
				"path":        "/etc/dangerous.txt",
				"old_content": "dangerous",
				"new_content": "safer",
			},
			expectOk:      false,
			errorContains: "path validation failed",
		},
		{
			name:     "Glob - 允许的路径",
			toolName: "Glob",
			input:    map[string]any{"pattern": "*.txt", "path": "/workspace/"},
			expectOk: true,
		},
		{
			name:          "Glob - 阻止非白名单路径",
			toolName:      "Glob",
			input:         map[string]any{"pattern": "*.txt", "path": "/etc/"},
			expectOk:      false,
			errorContains: "path validation failed",
		},
		{
			name:     "Grep - 允许的路径",
			toolName: "Grep",
			input:    map[string]any{"pattern": "content", "path": "/workspace/"},
			expectOk: true,
		},
		{
			name:          "Grep - 阻止非白名单路径",
			toolName:      "Grep",
			input:         map[string]any{"pattern": "content", "path": "/etc/"},
			expectOk:      false,
			errorContains: "path validation failed",
		},
	}

	middlewareTools := middleware.Tools()
	toolMap := make(map[string]tools.Tool)
	for _, tool := range middlewareTools {
		toolMap[tool.Name()] = tool
	}

	for _, tt := range toolTests {
		t.Run(tt.name, func(t *testing.T) {
			tool, exists := toolMap[tt.toolName]
			if !exists {
				t.Fatalf("Tool %s not found", tt.toolName)
			}

			result, err := tool.Execute(ctx, tt.input, nil)
			if err != nil {
				t.Fatalf("Execute failed: %v", err)
			}

			resultMap, ok := result.(map[string]any)
			if !ok {
				t.Fatal("Result is not a map")
			}

			actualOk := resultMap["ok"].(bool)
			if actualOk != tt.expectOk {
				errorMsg, _ := resultMap["error"].(string)
				t.Errorf("Expected ok=%v, got ok=%v (error: %s)", tt.expectOk, actualOk, errorMsg)
			}

			if !tt.expectOk {
				errorMsg, hasError := resultMap["error"].(string)
				if !hasError {
					t.Errorf("Expected error message in result")
				} else if !strings.Contains(errorMsg, tt.errorContains) {
					t.Errorf("Expected error containing '%s', got: %s", tt.errorContains, errorMsg)
				}
			}
		})
	}
}

// TestFilesystemMiddleware_CustomToolDescriptions 测试自定义工具描述
func TestFilesystemMiddleware_CustomToolDescriptions(t *testing.T) {
	backend := backends.NewStateBackend()

	middleware := NewFilesystemMiddleware(&FilesystemMiddlewareConfig{
		Backend: backend,
		CustomToolDescriptions: map[string]string{
			"Ls":   "自定义列表目录描述",
			"Edit": "自定义编辑文件描述",
		},
	})

	middlewareTools := middleware.Tools()
	toolMap := make(map[string]tools.Tool)
	for _, tool := range middlewareTools {
		toolMap[tool.Name()] = tool
	}

	// 注意: 只测试 backend-based 工具的自定义描述
	// Read 和 Write 来自 builtin,不支持自定义描述

	// 测试自定义描述
	if lsTool, ok := toolMap["Ls"]; ok {
		desc := lsTool.Description()
		if desc != "自定义列表目录描述" {
			t.Errorf("Expected custom description for Ls, got: %s", desc)
		}
	} else {
		t.Error("Ls tool not found")
	}

	if editTool, ok := toolMap["Edit"]; ok {
		desc := editTool.Description()
		if desc != "自定义编辑文件描述" {
			t.Errorf("Expected custom description for Edit, got: %s", desc)
		}
	} else {
		t.Error("Edit tool not found")
	}

	// 验证未自定义的工具保留默认描述
	if globTool, ok := toolMap["Glob"]; ok {
		desc := globTool.Description()
		if strings.Contains(desc, "自定义") {
			t.Errorf("Glob should not have custom description, got: %s", desc)
		}
		// 应该包含默认描述的部分内容
		if !strings.Contains(desc, "Find files") && !strings.Contains(desc, "glob") {
			t.Errorf("Glob should have default description, got: %s", desc)
		}
	}
}

// TestFilesystemMiddleware_SystemPromptOverride 测试 SystemPrompt 覆盖
func TestFilesystemMiddleware_SystemPromptOverride(t *testing.T) {
	ctx := context.Background()
	backend := backends.NewStateBackend()

	tests := []struct {
		name                 string
		systemPromptOverride string
		expectContains       string
	}{
		{
			name:                 "无覆盖 - 使用默认提示词",
			systemPromptOverride: "",
			expectContains:       "### Filesystem Tools",
		},
		{
			name:                 "自定义覆盖",
			systemPromptOverride: "这是自定义的文件系统提示词\n请遵循项目规范",
			expectContains:       "这是自定义的文件系统提示词",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := NewFilesystemMiddleware(&FilesystemMiddlewareConfig{
				Backend:                 backend,
				SystemPromptOverride:    tt.systemPromptOverride,
				HasSystemPromptOverride: tt.systemPromptOverride != "",
			})

			req := &ModelRequest{
				SystemPrompt: "原始提示词",
				Messages:     []types.Message{},
			}

			handler := func(ctx context.Context, req *ModelRequest) (*ModelResponse, error) {
				// 验证 SystemPrompt 是否包含预期内容
				if !strings.Contains(req.SystemPrompt, tt.expectContains) {
					t.Errorf("SystemPrompt should contain '%s', got: %s", tt.expectContains, req.SystemPrompt)
				}
				return &ModelResponse{}, nil
			}

			if _, err := middleware.WrapModelCall(ctx, req, handler); err != nil {
				t.Errorf("WrapModelCall failed: %v", err)
			}
		})
	}
}

// TestFilesystemMiddleware_PathNormalization 测试路径规范化
func TestFilesystemMiddleware_PathNormalization(t *testing.T) {
	backend := backends.NewStateBackend()
	middleware := NewFilesystemMiddleware(&FilesystemMiddlewareConfig{
		Backend:              backend,
		EnablePathValidation: true,
	})

	tests := []struct {
		input    string
		expected string
	}{
		{"workspace/file.txt", "/workspace/file.txt"},
		{"/workspace/./file.txt", "/workspace/file.txt"},
		{"/workspace//file.txt", "/workspace/file.txt"},
		{"///workspace/file.txt", "/workspace/file.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			normalized, err := middleware.validatePath(tt.input)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if normalized != tt.expected {
				t.Errorf("Expected normalized path '%s', got '%s'", tt.expected, normalized)
			}
		})
	}
}

// BenchmarkPathValidation 性能测试: 路径验证
func BenchmarkPathValidation(b *testing.B) {
	backend := backends.NewStateBackend()
	middleware := NewFilesystemMiddleware(&FilesystemMiddlewareConfig{
		Backend:              backend,
		EnablePathValidation: true,
		AllowedPathPrefixes:  []string{"/workspace/", "/tmp/"},
	})

	paths := []string{
		"/workspace/file.txt",
		"/tmp/cache.dat",
		"workspace/nested/deep/file.go",
		"/workspace/./redundant/../path/file.txt",
	}

	for i := 0; b.Loop(); i++ {
		path := paths[i%len(paths)]
		_, _ = middleware.validatePath(path)
	}
}

// BenchmarkPathValidation_Disabled 性能测试: 禁用路径验证
func BenchmarkPathValidation_Disabled(b *testing.B) {
	backend := backends.NewStateBackend()
	middleware := NewFilesystemMiddleware(&FilesystemMiddlewareConfig{
		Backend:              backend,
		EnablePathValidation: false,
	})

	paths := []string{
		"/workspace/file.txt",
		"../etc/passwd",
		"~/secrets.txt",
	}

	for i := 0; b.Loop(); i++ {
		path := paths[i%len(paths)]
		_, _ = middleware.validatePath(path)
	}
}
