package middleware

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTool[T any](t *testing.T, mw *SubAgentMiddleware, name string) T {
	t.Helper()
	var zero T
	for _, tool := range mw.Tools() {
		if tool.Name() == name {
			typed, ok := tool.(T)
			require.True(t, ok, "tool %s has unexpected type", name)
			return typed
		}
	}
	t.Fatalf("tool %s not found", name)
	return zero
}

// TestSubAgentMiddleware_AsyncExecution 测试异步执行
func TestSubAgentMiddleware_AsyncExecution(t *testing.T) {
	// 创建子代理规格
	specs := []SubAgentSpec{
		{
			Name:        "test-agent",
			Description: "测试子代理",
			Prompt:      "你是一个测试子代理",
		},
	}

	// 创建工厂
	factory := func(ctx context.Context, spec SubAgentSpec) (SubAgent, error) {
		execFn := func(ctx context.Context, description string, parentContext map[string]any) (string, error) {
			time.Sleep(100 * time.Millisecond) // 模拟处理时间
			return "Task completed: " + description, nil
		}
		return NewSimpleSubAgent(spec.Name, spec.Prompt, execFn), nil
	}

	// 创建中间件（启用异步）
	mw, err := NewSubAgentMiddleware(&SubAgentMiddlewareConfig{
		Specs:       specs,
		Factory:     factory,
		EnableAsync: true,
	})
	require.NoError(t, err)
	require.NotNil(t, mw.manager)

	taskTool := getTool[*TaskTool](t, mw, "task")

	// 测试异步执行
	ctx := context.Background()
	result, err := taskTool.Execute(ctx, map[string]any{
		"description":   "Test async task",
		"subagent_type": "test-agent",
		"async":         true,
	}, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	assert.True(t, resultMap["ok"].(bool))
	assert.NotEmpty(t, resultMap["task_id"])
	assert.Equal(t, "test-agent", resultMap["subagent_type"])

	taskID := resultMap["task_id"].(string)

	// 等待任务完成
	time.Sleep(200 * time.Millisecond)

	// 查询任务状态
	queryTool := getTool[*QuerySubagentTool](t, mw, "query_subagent")

	queryResult, err := queryTool.Execute(ctx, map[string]any{
		"task_id": taskID,
	}, nil)
	require.NoError(t, err)

	queryMap := queryResult.(map[string]any)
	assert.True(t, queryMap["ok"].(bool))
	assert.Equal(t, "completed", queryMap["status"])
	assert.Contains(t, queryMap["output"], "Task completed")
}

// TestSubAgentMiddleware_QuerySubagent 测试查询子代理
func TestSubAgentMiddleware_QuerySubagent(t *testing.T) {
	specs := []SubAgentSpec{
		{
			Name:        "test-agent",
			Description: "测试查询",
		},
	}

	factory := func(ctx context.Context, spec SubAgentSpec) (SubAgent, error) {
		execFn := func(ctx context.Context, description string, parentContext map[string]any) (string, error) {
			time.Sleep(50 * time.Millisecond)
			return "result: " + description, nil
		}
		return NewSimpleSubAgent(spec.Name, spec.Prompt, execFn), nil
	}

	mw, err := NewSubAgentMiddleware(&SubAgentMiddlewareConfig{
		Specs:       specs,
		Factory:     factory,
		EnableAsync: true,
	})
	require.NoError(t, err)

	taskTool := getTool[*TaskTool](t, mw, "task")
	ctx := context.Background()
	taskResult, err := taskTool.Execute(ctx, map[string]any{
		"description":   "Query status",
		"subagent_type": "test-agent",
		"async":         true,
	}, nil)
	require.NoError(t, err)

	taskMap := taskResult.(map[string]any)
	taskID := taskMap["task_id"].(string)

	time.Sleep(150 * time.Millisecond)

	queryTool := getTool[*QuerySubagentTool](t, mw, "query_subagent")
	result, err := queryTool.Execute(ctx, map[string]any{
		"task_id": taskID,
	}, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	assert.True(t, resultMap["ok"].(bool))
	assert.Equal(t, taskID, resultMap["task_id"])
	assert.Equal(t, "completed", resultMap["status"])
	assert.Contains(t, resultMap["output"].(string), "result")
}

// TestSubAgentMiddleware_StopSubagent 测试停止子代理
func TestSubAgentMiddleware_StopSubagent(t *testing.T) {
	specs := []SubAgentSpec{
		{
			Name:        "slow-agent",
			Description: "可停止的子代理",
		},
	}

	factory := func(ctx context.Context, spec SubAgentSpec) (SubAgent, error) {
		execFn := func(ctx context.Context, description string, parentContext map[string]any) (string, error) {
			ticker := time.NewTicker(50 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return "", ctx.Err()
				case <-ticker.C:
				}
			}
		}
		return NewSimpleSubAgent(spec.Name, spec.Prompt, execFn), nil
	}

	mw, err := NewSubAgentMiddleware(&SubAgentMiddlewareConfig{
		Specs:       specs,
		Factory:     factory,
		EnableAsync: true,
	})
	require.NoError(t, err)

	taskTool := getTool[*TaskTool](t, mw, "task")
	ctx := context.Background()
	taskResult, err := taskTool.Execute(ctx, map[string]any{
		"description":   "long running",
		"subagent_type": "slow-agent",
		"async":         true,
	}, nil)
	require.NoError(t, err)

	taskID := taskResult.(map[string]any)["task_id"].(string)
	stopTool := getTool[*StopSubagentTool](t, mw, "stop_subagent")

	result, err := stopTool.Execute(ctx, map[string]any{
		"task_id": taskID,
	}, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	assert.True(t, resultMap["ok"].(bool))
	assert.Equal(t, taskID, resultMap["task_id"])

	queryTool := getTool[*QuerySubagentTool](t, mw, "query_subagent")
	statusResult, err := queryTool.Execute(ctx, map[string]any{
		"task_id": taskID,
	}, nil)
	require.NoError(t, err)

	status := statusResult.(map[string]any)
	assert.Equal(t, "stopped", status["status"])
}

// TestSubAgentMiddleware_ResumeSubagent 测试恢复子代理
func TestSubAgentMiddleware_ResumeSubagent(t *testing.T) {
	specs := []SubAgentSpec{
		{
			Name:        "resumable-agent",
			Description: "支持恢复的子代理",
		},
	}

	factory := func(ctx context.Context, spec SubAgentSpec) (SubAgent, error) {
		execFn := func(ctx context.Context, description string, parentContext map[string]any) (string, error) {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(500 * time.Millisecond):
				return "resumed task completed", nil
			}
		}
		return NewSimpleSubAgent(spec.Name, spec.Prompt, execFn), nil
	}

	mw, err := NewSubAgentMiddleware(&SubAgentMiddlewareConfig{
		Specs:       specs,
		Factory:     factory,
		EnableAsync: true,
	})
	require.NoError(t, err)

	taskTool := getTool[*TaskTool](t, mw, "task")
	ctx := context.Background()
	taskResult, err := taskTool.Execute(ctx, map[string]any{
		"description":   "needs resume",
		"subagent_type": "resumable-agent",
		"async":         true,
	}, nil)
	require.NoError(t, err)

	taskID := taskResult.(map[string]any)["task_id"].(string)
	stopTool := getTool[*StopSubagentTool](t, mw, "stop_subagent")
	_, err = stopTool.Execute(ctx, map[string]any{"task_id": taskID}, nil)
	require.NoError(t, err)

	resumeTool := getTool[*ResumeSubagentTool](t, mw, "resume_subagent")
	result, err := resumeTool.Execute(ctx, map[string]any{
		"task_id": taskID,
	}, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	assert.True(t, resultMap["ok"].(bool))
	assert.Equal(t, taskID, resultMap["old_task_id"])
	assert.NotEmpty(t, resultMap["new_task_id"])
}

// TestSubAgentMiddleware_ListSubagents 测试列出子代理
func TestSubAgentMiddleware_ListSubagents(t *testing.T) {
	specs := []SubAgentSpec{
		{
			Name:        "worker",
			Description: "测试列出子代理",
		},
	}

	factory := func(ctx context.Context, spec SubAgentSpec) (SubAgent, error) {
		execFn := func(ctx context.Context, description string, parentContext map[string]any) (string, error) {
			time.Sleep(50 * time.Millisecond)
			return "done", nil
		}
		return NewSimpleSubAgent(spec.Name, spec.Prompt, execFn), nil
	}

	mw, err := NewSubAgentMiddleware(&SubAgentMiddlewareConfig{
		Specs:       specs,
		Factory:     factory,
		EnableAsync: true,
	})
	require.NoError(t, err)

	taskTool := getTool[*TaskTool](t, mw, "task")
	ctx := context.Background()

	for range 3 {
		_, err := taskTool.Execute(ctx, map[string]any{
			"description":   "list job",
			"subagent_type": "worker",
			"async":         true,
		}, nil)
		require.NoError(t, err)
	}

	time.Sleep(100 * time.Millisecond)

	listTool := getTool[*ListSubagentsTool](t, mw, "list_subagents")
	result, err := listTool.Execute(ctx, map[string]any{}, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	assert.True(t, resultMap["ok"].(bool))
	count := resultMap["count"].(int)
	assert.GreaterOrEqual(t, count, 3)
}

// TestSubAgentMiddleware_SyncVsAsync 测试同步和异步执行的区别
func TestSubAgentMiddleware_SyncVsAsync(t *testing.T) {
	// 创建子代理规格
	specs := []SubAgentSpec{
		{
			Name:        "slow-agent",
			Description: "慢速子代理",
			Prompt:      "慢速处理",
		},
	}

	// 创建工厂
	factory := func(ctx context.Context, spec SubAgentSpec) (SubAgent, error) {
		execFn := func(ctx context.Context, description string, parentContext map[string]any) (string, error) {
			time.Sleep(500 * time.Millisecond) // 模拟慢速处理
			return "Slow task completed", nil
		}
		return NewSimpleSubAgent(spec.Name, spec.Prompt, execFn), nil
	}

	// 创建中间件
	mw, err := NewSubAgentMiddleware(&SubAgentMiddlewareConfig{
		Specs:       specs,
		Factory:     factory,
		EnableAsync: true,
	})
	require.NoError(t, err)

	taskTool := getTool[*TaskTool](t, mw, "task")

	ctx := context.Background()

	// 测试同步执行（应该阻塞）
	t.Run("Sync", func(t *testing.T) {
		start := time.Now()
		result, err := taskTool.Execute(ctx, map[string]any{
			"description":   "Sync task",
			"subagent_type": "slow-agent",
			"async":         false,
		}, nil)
		duration := time.Since(start)

		require.NoError(t, err)
		resultMap := result.(map[string]any)
		assert.True(t, resultMap["ok"].(bool))
		assert.Contains(t, resultMap["result"], "Slow task completed")

		// 同步执行应该至少花费 500ms
		assert.GreaterOrEqual(t, duration.Milliseconds(), int64(500))
	})

	// 测试异步执行（应该立即返回）
	t.Run("Async", func(t *testing.T) {
		start := time.Now()
		result, err := taskTool.Execute(ctx, map[string]any{
			"description":   "Async task",
			"subagent_type": "slow-agent",
			"async":         true,
		}, nil)
		duration := time.Since(start)

		require.NoError(t, err)
		resultMap := result.(map[string]any)
		assert.True(t, resultMap["ok"].(bool))
		assert.NotEmpty(t, resultMap["task_id"])

		// 异步执行应该立即返回（< 100ms）
		assert.Less(t, duration.Milliseconds(), int64(100))
	})
}
