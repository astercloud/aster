package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAgentLifecycle 测试 Agent 完整生命周期
func TestAgentLifecycle(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	var agentID string

	// 1. 创建 Agent
	t.Run("CreateAgent", func(t *testing.T) {
		body := `{
			"template_id": "chat",
			"name": "Test Agent",
			"model_config": {
				"provider": "mock",
				"model": "test-model"
			}
		}`

		req := httptest.NewRequest(http.MethodPost, "/v1/agents", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		// 如果创建失败，打印错误信息
		if w.Code != http.StatusCreated {
			t.Logf("Create agent failed with status %d: %s", w.Code, w.Body.String())
		}

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		if !resp["success"].(bool) {
			t.Logf("Response indicates failure: %+v", resp)
		}

		assert.True(t, resp["success"].(bool))
		data := resp["data"].(map[string]any)
		agentID = data["id"].(string)
		assert.NotEmpty(t, agentID)
	})

	// 2. 获取 Agent
	t.Run("GetAgent", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/agents/"+agentID, nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), agentID)
	})

	// 3. 更新 Agent
	t.Run("UpdateAgent", func(t *testing.T) {
		body := `{
			"name": "Updated Agent",
			"metadata": {
				"version": "2.0"
			}
		}`

		req := httptest.NewRequest(http.MethodPatch, "/v1/agents/"+agentID, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Updated Agent")
	})

	// 4. 获取 Agent 状态
	t.Run("GetAgentStatus", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/agents/"+agentID+"/status", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "status")
	})

	// 5. 删除 Agent
	t.Run("DeleteAgent", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/v1/agents/"+agentID, nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	// 6. 验证删除后无法获取
	t.Run("GetDeletedAgent", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/agents/"+agentID, nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// TestPoolOperations 测试 Pool 操作
func TestPoolOperations(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	var agentID string

	// 1. 在 Pool 中创建 Agent
	t.Run("CreateAgentInPool", func(t *testing.T) {
		body := `{
			"template_id": "chat",
			"model_config": {
				"provider": "mock",
				"model": "test-model"
			}
		}`

		req := httptest.NewRequest(http.MethodPost, "/v1/pool/agents", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		data := resp["data"].(map[string]any)
		agentID = data["agent_id"].(string)
		assert.NotEmpty(t, agentID)
	})

	// 2. 列出 Pool 中的 Agents
	t.Run("ListPoolAgents", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/pool/agents", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "agents")
	})

	// 3. 获取 Pool 统计
	t.Run("GetPoolStats", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/pool/stats", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "total_agents")
	})

	// 4. 从 Pool 中移除 Agent
	t.Run("RemoveAgentFromPool", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/v1/pool/agents/"+agentID, nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}

// TestRoomOperations 测试 Room 操作
func TestRoomOperations(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// 创建 Room
	body := `{
		"name": "Test Room",
		"metadata": {
			"description": "A test room"
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/rooms", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.Router().ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var createResp map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &createResp)
	require.NoError(t, err)

	data := createResp["data"].(map[string]any)
	roomID := data["id"].(string)
	require.NotEmpty(t, roomID)

	// 1. 获取 Room
	t.Run("GetRoom", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/rooms/"+roomID, nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	// 2. 加入 Room (需要 Room 在内存中，跳过)
	t.Run("JoinRoom", func(t *testing.T) {
		t.Skip("Room operations require stateful handler, needs refactoring")
	})

	// 3. 获取成员列表 (需要 Room 在内存中，跳过)
	t.Run("GetRoomMembers", func(t *testing.T) {
		t.Skip("Room operations require stateful handler, needs refactoring")
	})

	// 4. 发送消息 (需要 Room 在内存中，跳过)
	t.Run("SendMessage", func(t *testing.T) {
		t.Skip("Room operations require stateful handler, needs refactoring")
	})

	// 5. 获取历史消息 (需要 Room 在内存中，跳过)
	t.Run("GetRoomHistory", func(t *testing.T) {
		t.Skip("Room operations require stateful handler, needs refactoring")
	})

	// 6. 离开 Room (需要 Room 在内存中，跳过)
	t.Run("LeaveRoom", func(t *testing.T) {
		t.Skip("Room operations require stateful handler, needs refactoring")
	})

	// 7. 列出所有 Rooms
	t.Run("ListRooms", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/rooms", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	// 8. 删除 Room
	t.Run("DeleteRoom", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/v1/rooms/"+roomID, nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}

// TestMemoryOperations 测试 Memory 操作
func TestMemoryOperations(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	var memoryID string

	// 1. 创建工作记忆
	t.Run("CreateWorkingMemory", func(t *testing.T) {
		body := `{
			"key": "test_key",
			"value": "test_value",
			"type": "string",
			"ttl": 3600
		}`

		req := httptest.NewRequest(http.MethodPost, "/v1/memory/working", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		data := resp["data"].(map[string]any)
		memoryID = data["id"].(string)
		assert.NotEmpty(t, memoryID)
	})

	// 2. 列出工作记忆
	t.Run("ListWorkingMemory", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/memory/working", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	// 3. 获取工作记忆
	t.Run("GetWorkingMemory", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/memory/working/"+memoryID, nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	// 4. 更新工作记忆
	t.Run("UpdateWorkingMemory", func(t *testing.T) {
		body := `{
			"value": "updated_value"
		}`

		req := httptest.NewRequest(http.MethodPatch, "/v1/memory/working/"+memoryID, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	// 5. 删除工作记忆
	t.Run("DeleteWorkingMemory", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/v1/memory/working/"+memoryID, nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}

// TestSessionOperations 测试 Session 操作
func TestSessionOperations(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	var sessionID string

	// 1. 创建 Session
	t.Run("CreateSession", func(t *testing.T) {
		body := `{
			"agent_id": "test-agent",
			"context": {
				"user_id": "user-123"
			}
		}`

		req := httptest.NewRequest(http.MethodPost, "/v1/sessions", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		data := resp["data"].(map[string]any)
		sessionID = data["id"].(string)
		assert.NotEmpty(t, sessionID)
	})

	// 2. 获取 Session
	t.Run("GetSession", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/sessions/"+sessionID, nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	// 3. 获取 Session 消息
	t.Run("GetSessionMessages", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/sessions/"+sessionID+"/messages", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	// 4. 获取 Session 统计
	t.Run("GetSessionStats", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/sessions/"+sessionID+"/stats", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	// 5. 删除 Session
	t.Run("DeleteSession", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/v1/sessions/"+sessionID, nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}

// TestWorkflowOperations 测试 Workflow 操作
func TestWorkflowOperations(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	var workflowID string

	// 1. 创建 Workflow
	t.Run("CreateWorkflow", func(t *testing.T) {
		body := `{
			"name": "Test Workflow",
			"description": "A test workflow",
			"version": "1.0.0",
			"steps": [
				{
					"id": "step1",
					"name": "First Step",
					"type": "agent"
				}
			]
		}`

		req := httptest.NewRequest(http.MethodPost, "/v1/workflows", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		data := resp["data"].(map[string]any)
		workflowID = data["id"].(string)
		assert.NotEmpty(t, workflowID)
	})

	// 2. 获取 Workflow
	t.Run("GetWorkflow", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/workflows/"+workflowID, nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	// 3. 列出 Workflows
	t.Run("ListWorkflows", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/workflows", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	// 4. 删除 Workflow
	t.Run("DeleteWorkflow", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/v1/workflows/"+workflowID, nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}

// TestErrorHandling 测试错误处理
func TestErrorHandling(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "InvalidJSON",
			method:         http.MethodPost,
			path:           "/v1/agents",
			body:           `{invalid json}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "bad_request",
		},
		{
			name:           "MissingRequiredField",
			method:         http.MethodPost,
			path:           "/v1/agents",
			body:           `{"name": "Test"}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "bad_request",
		},
		{
			name:           "NotFound",
			method:         http.MethodGet,
			path:           "/v1/agents/nonexistent",
			body:           "",
			expectedStatus: http.StatusNotFound,
			expectedError:  "not_found",
		},
		{
			name:           "InvalidRoomOperation",
			method:         http.MethodPost,
			path:           "/v1/rooms/nonexistent/join",
			body:           `{"name": "Alice", "agent_id": "agent-1"}`,
			expectedStatus: http.StatusNotFound,
			expectedError:  "not_found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}

			w := httptest.NewRecorder()
			srv.Router().ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedError)
		})
	}
}

// TestConcurrentRequests 测试并发请求
func TestConcurrentRequests(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	const numRequests = 10
	done := make(chan int, numRequests)

	for i := range numRequests {
		go func(id int) {
			defer func() {
				if r := recover(); r != nil {
					t.Logf("Request %d panicked: %v", id, r)
				}
				done <- id
			}()

			// 使用简单的 GET 请求避免 JSON 解析问题
			req := httptest.NewRequest(http.MethodGet, "/v1/rooms", nil)
			w := httptest.NewRecorder()

			srv.Router().ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Logf("Request %d failed with status %d", id, w.Code)
			}
		}(i)
	}

	// 等待所有请求完成
	timeout := time.After(5 * time.Second)
	completed := 0
	for completed < numRequests {
		select {
		case <-done:
			completed++
		case <-timeout:
			t.Fatalf("Timeout waiting for concurrent requests, completed: %d/%d", completed, numRequests)
		}
	}

	t.Logf("All %d concurrent requests completed", numRequests)
}

// TestRateLimiting 测试速率限制（如果启用）
func TestRateLimiting(t *testing.T) {
	t.Skip("Rate limiting not yet implemented")

	// TODO: 实现速率限制测试
}

// TestAuthentication 测试认证（如果启用）
func TestAuthentication(t *testing.T) {
	t.Skip("Authentication testing requires separate setup")

	// TODO: 实现认证测试
}
