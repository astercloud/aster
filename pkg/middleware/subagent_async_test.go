package middleware

import (
	"context"
	"testing"
	"time"

	"github.com/astercloud/aster/pkg/tools/builtin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		execFn := func(ctx context.Context, description string, parentContext map[string]interface{}) (string, error) {
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

	// 获取 task 工具
	tools := mw.Tools()
	var taskTool *TaskTool
	for _, tool := range tools {
		if tool.Name() == "task" {
			taskTool = tool.(*TaskTool)
			break
		}
	}
	require.NotNil(t, taskTool)

	// 测试异步执行
	ctx := context.Background()
	result, err := taskTool.Execute(ctx, map[string]interface{}{
		"description":   "Test async task",
		"subagent_type": "test-agent",
		"async":         true,
	}, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]interface{})
	assert.True(t, resultMap["ok"].(bool))
	assert.NotEmpty(t, resultMap["task_id"])
	assert.Equal(t, "test-agent", resultMap["subagent_type"])

	taskID := resultMap["task_id"].(string)

	// 等待任务完成
	time.Sleep(200 * time.Millisecond)

	// 查询任务状态
	var queryTool *QuerySubagentTool
	for _, tool := range tools {
		if tool.Name() == "query_subagent" {
			queryTool = tool.(*QuerySubagentTool)
			break
		}
	}
	require.NotNil(t, queryTool)

	queryResult, err := queryTool.Execute(ctx, map[string]interface{}{
		"task_id": taskID,
	}, nil)
	require.NoError(t, err)

	queryMap := queryResult.(map[string]interface{})
	assert.True(t, queryMap["ok"].(bool))
	assert.Equal(t, "completed", queryMap["status"])
	assert.Contains(t, queryMap["output"], "Task completed")
}

// TestSubAgentMiddleware_QuerySubagent 测试查询子代理
func TestSubAgentMiddleware_QuerySubagent(t *testing.T) {
	// 创建模拟管理器
	manager := builtin.NewFileSubagentManager()

	// 创建中间件
	mw, err := NewSubAgentMiddleware(&SubAgentMiddlewareConfig{
		Manager:     manager,
		EnableAsync: true,
	})
	require.NoError(t, err)

	// 启动一个子代理
	ctx := context.Background()
	config := &builtin.SubagentConfig{
		Type:    "test",
		Prompt:  "echo 'Hello World'",
		Timeout: 5 * time.Second,
	}

	instance, err := manager.StartSubagent(ctx, config)
	require.NoError(t, err)

	// 等待一会儿
	time.Sleep(100 * time.Millisecond)

	// 查询状态
	tools := mw.Tools()
	var queryTool *QuerySubagentTool
	for _, tool := range tools {
		if tool.Name() == "query_subagent" {
			queryTool = tool.(*QuerySubagentTool)
			break
		}
	}
	require.NotNil(t, queryTool)

	result, err := queryTool.Execute(ctx, map[string]interface{}{
		"task_id": instance.ID,
	}, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]interface{})
	assert.True(t, resultMap["ok"].(bool))
	assert.Equal(t, instance.ID, resultMap["task_id"])
	assert.NotEmpty(t, resultMap["status"])
}

// TestSubAgentMiddleware_StopSubagent 测试停止子代理
func TestSubAgentMiddleware_StopSubagent(t *testing.T) {
	// 创建模拟管理器
	manager := builtin.NewFileSubagentManager()

	// 创建中间件
	mw, err := NewSubAgentMiddleware(&SubAgentMiddlewareConfig{
		Manager:     manager,
		EnableAsync: true,
	})
	require.NoError(t, err)

	// 启动一个长时间运行的子代理
	ctx := context.Background()
	config := &builtin.SubagentConfig{
		Type:    "test",
		Prompt:  "sleep 10",
		Timeout: 30 * time.Second,
	}

	instance, err := manager.StartSubagent(ctx, config)
	require.NoError(t, err)

	// 等待启动
	time.Sleep(100 * time.Millisecond)

	// 停止子代理
	tools := mw.Tools()
	var stopTool *StopSubagentTool
	for _, tool := range tools {
		if tool.Name() == "stop_subagent" {
			stopTool = tool.(*StopSubagentTool)
			break
		}
	}
	require.NotNil(t, stopTool)

	result, err := stopTool.Execute(ctx, map[string]interface{}{
		"task_id": instance.ID,
	}, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]interface{})
	assert.True(t, resultMap["ok"].(bool))
	assert.Equal(t, instance.ID, resultMap["task_id"])

	// 验证状态
	status, err := manager.GetSubagent(instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "stopped", status.Status)
}

// TestSubAgentMiddleware_ResumeSubagent 测试恢复子代理
func TestSubAgentMiddleware_ResumeSubagent(t *testing.T) {
	// 创建模拟管理器
	manager := builtin.NewFileSubagentManager()

	// 创建中间件
	mw, err := NewSubAgentMiddleware(&SubAgentMiddlewareConfig{
		Manager:     manager,
		EnableAsync: true,
	})
	require.NoError(t, err)

	// 启动并停止一个子代理
	ctx := context.Background()
	config := &builtin.SubagentConfig{
		Type:    "test",
		Prompt:  "echo 'test'",
		Timeout: 5 * time.Second,
	}

	instance, err := manager.StartSubagent(ctx, config)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	err = manager.StopSubagent(instance.ID)
	require.NoError(t, err)

	// 恢复子代理
	tools := mw.Tools()
	var resumeTool *ResumeSubagentTool
	for _, tool := range tools {
		if tool.Name() == "resume_subagent" {
			resumeTool = tool.(*ResumeSubagentTool)
			break
		}
	}
	require.NotNil(t, resumeTool)

	result, err := resumeTool.Execute(ctx, map[string]interface{}{
		"task_id": instance.ID,
	}, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]interface{})
	assert.True(t, resultMap["ok"].(bool))
	assert.Equal(t, instance.ID, resultMap["old_task_id"])
	assert.NotEmpty(t, resultMap["new_task_id"])
}

// TestSubAgentMiddleware_ListSubagents 测试列出子代理
func TestSubAgentMiddleware_ListSubagents(t *testing.T) {
	// 创建模拟管理器
	manager := builtin.NewFileSubagentManager()

	// 创建中间件
	mw, err := NewSubAgentMiddleware(&SubAgentMiddlewareConfig{
		Manager:     manager,
		EnableAsync: true,
	})
	require.NoError(t, err)

	// 启动多个子代理
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		config := &builtin.SubagentConfig{
			Type:    "test",
			Prompt:  "echo 'test'",
			Timeout: 5 * time.Second,
		}
		_, err := manager.StartSubagent(ctx, config)
		require.NoError(t, err)
	}

	time.Sleep(100 * time.Millisecond)

	// 列出所有子代理
	tools := mw.Tools()
	var listTool *ListSubagentsTool
	for _, tool := range tools {
		if tool.Name() == "list_subagents" {
			listTool = tool.(*ListSubagentsTool)
			break
		}
	}
	require.NotNil(t, listTool)

	result, err := listTool.Execute(ctx, map[string]interface{}{}, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]interface{})
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
		execFn := func(ctx context.Context, description string, parentContext map[string]interface{}) (string, error) {
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

	tools := mw.Tools()
	var taskTool *TaskTool
	for _, tool := range tools {
		if tool.Name() == "task" {
			taskTool = tool.(*TaskTool)
			break
		}
	}
	require.NotNil(t, taskTool)

	ctx := context.Background()

	// 测试同步执行（应该阻塞）
	t.Run("Sync", func(t *testing.T) {
		start := time.Now()
		result, err := taskTool.Execute(ctx, map[string]interface{}{
			"description":   "Sync task",
			"subagent_type": "slow-agent",
			"async":         false,
		}, nil)
		duration := time.Since(start)

		require.NoError(t, err)
		resultMap := result.(map[string]interface{})
		assert.True(t, resultMap["ok"].(bool))
		assert.Contains(t, resultMap["result"], "Slow task completed")

		// 同步执行应该至少花费 500ms
		assert.GreaterOrEqual(t, duration.Milliseconds(), int64(500))
	})

	// 测试异步执行（应该立即返回）
	t.Run("Async", func(t *testing.T) {
		start := time.Now()
		result, err := taskTool.Execute(ctx, map[string]interface{}{
			"description":   "Async task",
			"subagent_type": "slow-agent",
			"async":         true,
		}, nil)
		duration := time.Since(start)

		require.NoError(t, err)
		resultMap := result.(map[string]interface{})
		assert.True(t, resultMap["ok"].(bool))
		assert.NotEmpty(t, resultMap["task_id"])

		// 异步执行应该立即返回（< 100ms）
		assert.Less(t, duration.Milliseconds(), int64(100))
	})
}
