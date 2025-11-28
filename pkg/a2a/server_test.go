package a2a

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/astercloud/aster/pkg/actor"
	pkgagent "github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockAgentActor 用于测试的简单 Agent Actor
type MockAgentActor struct {
	responses []string
	index     int
}

func (a *MockAgentActor) Receive(ctx *actor.Context, msg actor.Message) {
	switch m := msg.(type) {
	case *pkgagent.ChatMsg:
		// 模拟延迟
		time.Sleep(10 * time.Millisecond)

		// 返回预设的响应
		response := "Mock response"
		if a.index < len(a.responses) {
			response = a.responses[a.index]
			a.index++
		}

		result := &pkgagent.ChatResultMsg{
			Result: &types.CompleteResult{
				Text: response,
			},
		}

		select {
		case m.ReplyTo <- result:
		case <-time.After(time.Second):
		}
	}
}

func TestServer_GetAgentCard(t *testing.T) {
	// 创建 Actor 系统
	system := actor.NewSystem("test-a2a")
	defer system.Shutdown()

	// 创建 A2A 服务器
	taskStore := NewInMemoryTaskStore()
	server := NewServer(system, taskStore)

	// 创建测试 Agent
	agentID := "test-agent"
	mockAgent := &MockAgentActor{responses: []string{"Hello"}}
	system.Spawn(mockAgent, agentID)

	// 获取 Agent Card
	card, err := server.GetAgentCard(agentID)
	require.NoError(t, err)
	assert.Equal(t, agentID, card.Name)
	assert.Contains(t, card.Description, agentID)
	assert.True(t, card.Capabilities.Streaming)
	assert.Len(t, card.Skills, 1)
}

func TestServer_GetAgentCard_NotFound(t *testing.T) {
	system := actor.NewSystem("test-a2a")
	defer system.Shutdown()

	taskStore := NewInMemoryTaskStore()
	server := NewServer(system, taskStore)

	// 尝试获取不存在的 Agent
	_, err := server.GetAgentCard("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent not found")
}

func TestServer_HandleRequest_MessageSend(t *testing.T) {
	// 创建 Actor 系统
	system := actor.NewSystem("test-a2a")
	defer system.Shutdown()

	// 创建 A2A 服务器
	taskStore := NewInMemoryTaskStore()
	server := NewServer(system, taskStore)

	// 创建测试 Agent
	agentID := "test-agent"
	mockAgent := &MockAgentActor{responses: []string{"Hello from agent"}}
	system.Spawn(mockAgent, agentID)

	// 构造 message/send 请求
	params := MessageSendParams{
		Message: Message{
			MessageID: "msg-1",
			Role:      "user",
			Parts:     []Part{{Kind: "text", Text: "Hello"}},
		},
		ContextID: "context-1",
	}
	paramsJSON, _ := json.Marshal(params)

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      "req-1",
		Method:  "message/send",
		Params:  json.RawMessage(paramsJSON),
	}

	// 处理请求
	ctx := context.Background()
	resp := server.HandleRequest(ctx, agentID, req)

	// 验证响应
	require.NotNil(t, resp)
	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.Equal(t, "req-1", resp.ID)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Result)

	// 验证任务已创建
	var result MessageSendResult
	resultBytes, _ := json.Marshal(resp.Result)
	err := json.Unmarshal(resultBytes, &result)
	require.NoError(t, err)
	assert.NotEmpty(t, result.TaskID)

	// 验证任务存储中的任务
	task, err := taskStore.Load(agentID, result.TaskID)
	require.NoError(t, err)
	assert.Equal(t, TaskStateCompleted, task.Status.State)
	assert.Len(t, task.History, 2) // 用户消息 + Agent 响应
}

func TestServer_HandleRequest_TasksGet(t *testing.T) {
	// 创建 Actor 系统
	system := actor.NewSystem("test-a2a")
	defer system.Shutdown()

	// 创建 A2A 服务器
	taskStore := NewInMemoryTaskStore()
	server := NewServer(system, taskStore)

	agentID := "test-agent"

	// 创建一个任务
	task := NewTask("task-1", "context-1")
	task.AddMessage(Message{
		MessageID: "msg-1",
		Role:      "user",
		Parts:     []Part{{Kind: "text", Text: "Hello"}},
	})
	task.UpdateStatus(TaskStateCompleted, nil)
	require.NoError(t, taskStore.Save(agentID, task))

	// 构造 tasks/get 请求
	params := TasksGetParams{
		TaskID: task.ID,
	}
	paramsJSON, _ := json.Marshal(params)

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      "req-1",
		Method:  "tasks/get",
		Params:  json.RawMessage(paramsJSON),
	}

	// 处理请求
	ctx := context.Background()
	resp := server.HandleRequest(ctx, agentID, req)

	// 验证响应
	require.NotNil(t, resp)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Result)

	// 验证返回的任务
	var result TasksGetResult
	resultBytes, _ := json.Marshal(resp.Result)
	err := json.Unmarshal(resultBytes, &result)
	require.NoError(t, err)
	assert.Equal(t, task.ID, result.Task.ID)
	assert.Equal(t, TaskStateCompleted, result.Task.Status.State)
}

func TestServer_HandleRequest_TasksCancel(t *testing.T) {
	// 创建 Actor 系统
	system := actor.NewSystem("test-a2a")
	defer system.Shutdown()

	// 创建 A2A 服务器
	taskStore := NewInMemoryTaskStore()
	server := NewServer(system, taskStore)

	agentID := "test-agent"

	// 创建一个任务
	task := NewTask("task-1", "context-1")
	task.UpdateStatus(TaskStateWorking, nil)
	require.NoError(t, taskStore.Save(agentID, task))

	// 构造 tasks/cancel 请求
	params := TasksCancelParams{
		TaskID: task.ID,
	}
	paramsJSON, _ := json.Marshal(params)

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      "req-1",
		Method:  "tasks/cancel",
		Params:  json.RawMessage(paramsJSON),
	}

	// 处理请求
	ctx := context.Background()
	resp := server.HandleRequest(ctx, agentID, req)

	// 验证响应
	require.NotNil(t, resp)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Result)

	// 验证取消标记已设置
	cancelled := taskStore.IsCanceled(task.ID)
	assert.True(t, cancelled)

	// 验证任务状态已更新
	updatedTask, err := taskStore.Load(agentID, task.ID)
	require.NoError(t, err)
	assert.Equal(t, TaskStateCanceled, updatedTask.Status.State)
}

func TestServer_HandleRequest_InvalidMethod(t *testing.T) {
	system := actor.NewSystem("test-a2a")
	defer system.Shutdown()

	taskStore := NewInMemoryTaskStore()
	server := NewServer(system, taskStore)

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      "req-1",
		Method:  "invalid/method",
		Params:  []byte("{}"),
	}

	ctx := context.Background()
	resp := server.HandleRequest(ctx, "agent-1", req)

	require.NotNil(t, resp)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, ErrorCodeMethodNotFound, resp.Error.Code)
}

func TestServer_HandleRequest_InvalidParams(t *testing.T) {
	system := actor.NewSystem("test-a2a")
	defer system.Shutdown()

	taskStore := NewInMemoryTaskStore()
	server := NewServer(system, taskStore)

	// 无效的参数
	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      "req-1",
		Method:  "message/send",
		Params:  []byte("invalid json"),
	}

	ctx := context.Background()
	resp := server.HandleRequest(ctx, "agent-1", req)

	require.NotNil(t, resp)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, ErrorCodeInvalidParams, resp.Error.Code)
}

func TestJSONRPCError(t *testing.T) {
	// 测试错误响应的创建
	resp := NewErrorResponse("req-1", ErrorCodeInvalidParams, "Invalid parameters", map[string]string{"field": "message"})

	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.Equal(t, "req-1", resp.ID)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, ErrorCodeInvalidParams, resp.Error.Code)
	assert.Equal(t, "Invalid parameters", resp.Error.Message)
	assert.Nil(t, resp.Result)
}

func TestJSONRPCSuccess(t *testing.T) {
	// 测试成功响应的创建
	result := map[string]string{"status": "ok"}
	resp := NewSuccessResponse("req-1", result)

	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.Equal(t, "req-1", resp.ID)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Result)
}
