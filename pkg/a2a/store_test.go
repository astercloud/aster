package a2a

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryTaskStore_SaveAndLoad(t *testing.T) {
	store := NewInMemoryTaskStore()
	agentID := "agent-1"
	task := NewTask("task-1", "context-1")
	task.AddMessage(Message{
		MessageID: "msg-1",
		Role:      "user",
		Parts:     []Part{{Kind: "text", Text: "Hello"}},
	})

	// 保存任务
	err := store.Save(agentID, task)
	require.NoError(t, err)

	// 加载任务
	loaded, err := store.Load(agentID, task.ID)
	require.NoError(t, err)
	assert.Equal(t, task.ID, loaded.ID)
	assert.Equal(t, task.ContextID, loaded.ContextID)
	assert.Len(t, loaded.History, 1)
	assert.Equal(t, "msg-1", loaded.History[0].MessageID)
}

func TestInMemoryTaskStore_LoadNotFound(t *testing.T) {
	store := NewInMemoryTaskStore()

	// 尝试加载不存在的任务
	_, err := store.Load("agent-1", "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task not found")
}

func TestInMemoryTaskStore_Delete(t *testing.T) {
	store := NewInMemoryTaskStore()
	agentID := "agent-1"
	task := NewTask("task-1", "context-1")

	// 保存任务
	err := store.Save(agentID, task)
	require.NoError(t, err)

	// 删除任务
	err = store.Delete(agentID, task.ID)
	require.NoError(t, err)

	// 确认任务已删除
	_, err = store.Load(agentID, task.ID)
	assert.Error(t, err)
}

func TestInMemoryTaskStore_List(t *testing.T) {
	store := NewInMemoryTaskStore()
	agentID := "agent-1"

	// 创建多个任务
	task1 := NewTask("task-1", "context-1")
	task2 := NewTask("task-2", "context-1")
	task3 := NewTask("task-3", "context-2")

	require.NoError(t, store.Save(agentID, task1))
	require.NoError(t, store.Save(agentID, task2))
	require.NoError(t, store.Save(agentID, task3))

	// 列出所有任务
	tasks, err := store.List(agentID)
	require.NoError(t, err)
	assert.Len(t, tasks, 3)

	// 验证任务 ID
	taskIDs := make(map[string]bool)
	for _, task := range tasks {
		taskIDs[task.ID] = true
	}
	assert.True(t, taskIDs["task-1"])
	assert.True(t, taskIDs["task-2"])
	assert.True(t, taskIDs["task-3"])
}

func TestInMemoryTaskStore_Cancellation(t *testing.T) {
	store := NewInMemoryTaskStore()
	task := NewTask("task-1", "context-1")

	// 添加取消信号
	store.AddCancellation(task.ID)

	// 检查取消状态
	canceled := store.IsCanceled(task.ID)
	assert.True(t, canceled)

	// 移除取消信号
	store.RemoveCancellation(task.ID)

	// 再次检查
	canceled = store.IsCanceled(task.ID)
	assert.False(t, canceled)
}

func TestInMemoryTaskStore_Concurrency(t *testing.T) {
	store := NewInMemoryTaskStore()
	agentID := "agent-1"

	// 并发保存任务
	const numTasks = 100
	done := make(chan bool, numTasks)

	for i := range numTasks {
		go func(id int) {
			task := NewTask(string(rune('a'+id)), "context-1")
			_ = store.Save(agentID, task)
			done <- true
		}(i)
	}

	// 等待所有任务完成
	for range numTasks {
		<-done
	}

	// 验证任务已保存(数量可能少于numTasks,因为有ID冲突)
	tasks, err := store.List(agentID)
	require.NoError(t, err)
	assert.NotEmpty(t, tasks)
}

func TestTask_StateTransitions(t *testing.T) {
	task := NewTask("task-1", "context-1")

	// 初始状态
	assert.Equal(t, TaskStateSubmitted, task.Status.State)

	// 转换到 working
	msg := Message{MessageID: "msg-1", Role: "agent"}
	task.UpdateStatus(TaskStateWorking, &msg)
	assert.Equal(t, TaskStateWorking, task.Status.State)

	// 转换到 completed
	task.UpdateStatus(TaskStateCompleted, nil)
	assert.Equal(t, TaskStateCompleted, task.Status.State)
}

func TestTask_AddMessage(t *testing.T) {
	task := NewTask("task-1", "context-1")

	// 添加用户消息
	userMsg := Message{
		MessageID: "msg-1",
		Role:      "user",
		Parts:     []Part{{Kind: "text", Text: "Hello"}},
	}
	task.AddMessage(userMsg)

	// 添加代理消息
	agentMsg := Message{
		MessageID: "msg-2",
		Role:      "agent",
		Parts:     []Part{{Kind: "text", Text: "Hi there"}},
	}
	task.AddMessage(agentMsg)

	// 验证历史记录
	assert.Len(t, task.History, 2)
	assert.Equal(t, "msg-1", task.History[0].MessageID)
	assert.Equal(t, "user", task.History[0].Role)
	assert.Equal(t, "msg-2", task.History[1].MessageID)
	assert.Equal(t, "agent", task.History[1].Role)
}
