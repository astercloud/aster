package a2a

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/astercloud/aster/pkg/logging"
	"github.com/gin-gonic/gin"
)

// Handler HTTP 处理器,实现 A2A 协议的 HTTP 端点
type Handler struct {
	server *Server
}

// NewHandler 创建新的 HTTP 处理器
func NewHandler(server *Server) *Handler {
	return &Handler{
		server: server,
	}
}

// GetAgentCard 处理 Agent Card 请求
// GET /.well-known/{agentId}/agent-card.json
func (h *Handler) GetAgentCard(c *gin.Context) {
	ctx := c.Request.Context()
	agentID := c.Param("agentId")

	if agentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "bad_request",
				"message": "Missing agentId parameter",
			},
		})
		return
	}

	logging.Info(ctx, "a2a.get_agent_card", map[string]interface{}{
		"agent_id": agentID,
	})

	card, err := h.server.GetAgentCard(agentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "not_found",
				"message": fmt.Sprintf("Agent not found: %s", err.Error()),
			},
		})
		return
	}

	// A2A 规范要求直接返回 Agent Card JSON,不包装在 success/data 中
	c.JSON(http.StatusOK, card)
}

// HandleJSONRPC 处理 JSON-RPC 2.0 请求
// POST /a2a/{agentId}
func (h *Handler) HandleJSONRPC(c *gin.Context) {
	ctx := c.Request.Context()
	agentID := c.Param("agentId")

	if agentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "bad_request",
				"message": "Missing agentId parameter",
			},
		})
		return
	}

	// 解析 JSON-RPC 请求
	var req JSONRPCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 返回 JSON-RPC 错误响应
		c.JSON(http.StatusBadRequest, &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      nil, // 解析错误时 ID 可能为 nil
			Error: &RPCError{
				Code:    -32700, // Parse error
				Message: "Invalid JSON was received",
				Data:    err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "a2a.jsonrpc_request", map[string]interface{}{
		"agent_id": agentID,
		"method":   req.Method,
		"id":       req.ID,
	})

	// 验证 JSON-RPC 版本
	if req.JSONRPC != "2.0" {
		c.JSON(http.StatusBadRequest, &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    -32600, // Invalid Request
				Message: "Invalid JSON-RPC version, must be '2.0'",
			},
		})
		return
	}

	// 处理请求
	resp := h.server.HandleRequest(ctx, agentID, &req)

	// 记录响应
	if resp.Error != nil {
		logging.Warn(ctx, "a2a.jsonrpc_error", map[string]interface{}{
			"agent_id":   agentID,
			"method":     req.Method,
			"error_code": resp.Error.Code,
			"error_msg":  resp.Error.Message,
		})
	} else {
		logging.Info(ctx, "a2a.jsonrpc_success", map[string]interface{}{
			"agent_id": agentID,
			"method":   req.Method,
		})
	}

	// 返回 JSON-RPC 响应
	c.JSON(http.StatusOK, resp)
}

// HandleBatchJSONRPC 处理批量 JSON-RPC 2.0 请求
// POST /a2a/{agentId}/batch
func (h *Handler) HandleBatchJSONRPC(c *gin.Context) {
	ctx := c.Request.Context()
	agentID := c.Param("agentId")

	if agentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "bad_request",
				"message": "Missing agentId parameter",
			},
		})
		return
	}

	// 解析批量请求
	var requests []JSONRPCRequest
	if err := c.ShouldBindJSON(&requests); err != nil {
		c.JSON(http.StatusBadRequest, &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      nil,
			Error: &RPCError{
				Code:    -32700, // Parse error
				Message: "Invalid JSON was received",
				Data:    err.Error(),
			},
		})
		return
	}

	if len(requests) == 0 {
		c.JSON(http.StatusBadRequest, &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      nil,
			Error: &RPCError{
				Code:    -32600, // Invalid Request
				Message: "Empty batch request",
			},
		})
		return
	}

	logging.Info(ctx, "a2a.batch_jsonrpc_request", map[string]interface{}{
		"agent_id": agentID,
		"count":    len(requests),
	})

	// 处理每个请求
	responses := make([]*JSONRPCResponse, 0, len(requests))
	for _, req := range requests {
		if req.JSONRPC != "2.0" {
			responses = append(responses, &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &RPCError{
					Code:    -32600,
					Message: "Invalid JSON-RPC version, must be '2.0'",
				},
			})
			continue
		}

		resp := h.server.HandleRequest(ctx, agentID, &req)
		responses = append(responses, resp)
	}

	logging.Info(ctx, "a2a.batch_jsonrpc_success", map[string]interface{}{
		"agent_id": agentID,
		"count":    len(responses),
	})

	// 返回批量响应
	c.JSON(http.StatusOK, responses)
}

// GetTaskStatus 获取任务状态的便捷端点
// GET /a2a/{agentId}/tasks/{taskId}
func (h *Handler) GetTaskStatus(c *gin.Context) {
	ctx := c.Request.Context()
	agentID := c.Param("agentId")
	taskID := c.Param("taskId")

	if agentID == "" || taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "bad_request",
				"message": "Missing agentId or taskId parameter",
			},
		})
		return
	}

	logging.Info(ctx, "a2a.get_task_status", map[string]interface{}{
		"agent_id": agentID,
		"task_id":  taskID,
	})

	// 构造 tasks/get JSON-RPC 请求
	params := TasksGetParams{
		TaskID: taskID,
	}
	paramsJSON, _ := json.Marshal(params)

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      taskID, // 使用 taskID 作为请求 ID
		Method:  "tasks/get",
		Params:  paramsJSON,
	}

	resp := h.server.HandleRequest(ctx, agentID, req)

	if resp.Error != nil {
		var statusCode int
		switch resp.Error.Code {
		case -32602: // Invalid params
			statusCode = http.StatusBadRequest
		case -32001: // Task not found
			statusCode = http.StatusNotFound
		default:
			statusCode = http.StatusInternalServerError
		}

		c.JSON(statusCode, gin.H{
			"success": false,
			"error": gin.H{
				"code":    resp.Error.Message,
				"message": fmt.Sprintf("%v", resp.Error.Data),
			},
		})
		return
	}

	// 解析任务响应
	var result TasksGetResult
	if resultBytes, ok := resp.Result.([]byte); ok {
		if err := json.Unmarshal(resultBytes, &result); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to parse task result",
				},
			})
			return
		}
	} else if err := json.Unmarshal(resp.Result.(json.RawMessage), &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to parse task result",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result.Task,
	})
}

// RegisterRoutes 注册 A2A 路由到 Gin RouterGroup
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	// Agent Card 端点 (符合 A2A 规范的路径)
	rg.GET("/.well-known/:agentId/agent-card.json", h.GetAgentCard)

	// A2A JSON-RPC 端点
	a2a := rg.Group("/a2a")
	{
		// 单个 JSON-RPC 请求
		a2a.POST("/:agentId", h.HandleJSONRPC)

		// 批量 JSON-RPC 请求
		a2a.POST("/:agentId/batch", h.HandleBatchJSONRPC)

		// 便捷端点:直接获取任务状态
		a2a.GET("/:agentId/tasks/:taskId", h.GetTaskStatus)
	}
}
