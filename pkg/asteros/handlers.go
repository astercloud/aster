package asteros

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
)

// handleHealth 健康检查
func (os *AsterOS) handleHealth(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": "healthy",
		"name":   os.opts.Name,
	})
}

// handleMetrics Prometheus 指标
func (os *AsterOS) handleMetrics(c *gin.Context) {
	// TODO: 实现 Prometheus 指标
	c.String(200, "# Prometheus metrics\n")
}

// handleListAgents 列出所有 Agents
func (os *AsterOS) handleListAgents(c *gin.Context) {
	agents := os.registry.ListAgents()
	c.JSON(200, gin.H{
		"agents": agents,
		"count":  len(agents),
	})
}

// AgentRunRequest Agent 运行请求
type AgentRunRequest struct {
	Message string                 `json:"message" binding:"required"`
	Stream  bool                   `json:"stream,omitempty"`
	Context map[string]interface{} `json:"context,omitempty"`
}

// handleAgentRun 运行 Agent
func (os *AsterOS) handleAgentRun(c *gin.Context) {
	agentID := c.Param("id")

	// 获取 Agent
	ag, exists := os.registry.GetAgent(agentID)
	if !exists {
		c.JSON(404, gin.H{"error": "agent not found"})
		return
	}

	// 解析请求
	var req AgentRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// 运行 Agent
	ctx := context.Background()
	if err := ag.Send(ctx, req.Message); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"status":  "success",
		"message": "agent task started",
	})
}

// handleAgentStatus 获取 Agent 状态
func (os *AsterOS) handleAgentStatus(c *gin.Context) {
	agentID := c.Param("id")

	// 获取 Agent
	ag, exists := os.registry.GetAgent(agentID)
	if !exists {
		c.JSON(404, gin.H{"error": "agent not found"})
		return
	}

	// 获取状态
	status := ag.Status()

	c.JSON(200, gin.H{
		"agent_id": status.AgentID,
		"state":    status.State,
	})
}

// handleListRooms 列出所有 Rooms
func (os *AsterOS) handleListRooms(c *gin.Context) {
	roomsList := os.registry.ListRooms()
	c.JSON(200, gin.H{
		"rooms": roomsList,
		"count": len(roomsList),
	})
}

// RoomSayRequest Room 发送消息请求
type RoomSayRequest struct {
	From string `json:"from" binding:"required"`
	Text string `json:"text" binding:"required"`
}

// handleRoomSay 在 Room 中发送消息
func (os *AsterOS) handleRoomSay(c *gin.Context) {
	roomID := c.Param("id")

	// 获取 Room
	room, exists := os.registry.GetRoom(roomID)
	if !exists {
		c.JSON(404, gin.H{"error": "room not found"})
		return
	}

	// 解析请求
	var req RoomSayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// 发送消息
	ctx := context.Background()
	if err := room.Say(ctx, req.From, req.Text); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"status":  "success",
		"message": "message sent",
	})
}

// RoomJoinRequest Room 加入请求
type RoomJoinRequest struct {
	Name    string `json:"name" binding:"required"`
	AgentID string `json:"agent_id" binding:"required"`
}

// handleRoomJoin 添加成员到 Room
func (os *AsterOS) handleRoomJoin(c *gin.Context) {
	roomID := c.Param("id")

	// 获取 Room
	room, exists := os.registry.GetRoom(roomID)
	if !exists {
		c.JSON(404, gin.H{"error": "room not found"})
		return
	}

	// 解析请求
	var req RoomJoinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// 添加成员
	if err := room.Join(req.Name, req.AgentID); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"status":  "success",
		"message": fmt.Sprintf("member %s joined", req.Name),
	})
}

// RoomLeaveRequest Room 离开请求
type RoomLeaveRequest struct {
	Name string `json:"name" binding:"required"`
}

// handleRoomLeave 从 Room 移除成员
func (os *AsterOS) handleRoomLeave(c *gin.Context) {
	roomID := c.Param("id")

	// 获取 Room
	room, exists := os.registry.GetRoom(roomID)
	if !exists {
		c.JSON(404, gin.H{"error": "room not found"})
		return
	}

	// 解析请求
	var req RoomLeaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// 移除成员
	if err := room.Leave(req.Name); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"status":  "success",
		"message": fmt.Sprintf("member %s left", req.Name),
	})
}

// handleRoomMembers 获取 Room 成员列表
func (os *AsterOS) handleRoomMembers(c *gin.Context) {
	roomID := c.Param("id")

	// 获取 Room
	room, exists := os.registry.GetRoom(roomID)
	if !exists {
		c.JSON(404, gin.H{"error": "room not found"})
		return
	}

	// 获取成员
	members := room.GetMembers()

	c.JSON(200, gin.H{
		"members": members,
		"count":   len(members),
	})
}

// handleListWorkflows 列出所有 Workflows
func (os *AsterOS) handleListWorkflows(c *gin.Context) {
	workflows := os.registry.ListWorkflows()
	c.JSON(200, gin.H{
		"workflows": workflows,
		"count":     len(workflows),
	})
}

// WorkflowExecuteRequest Workflow 执行请求
type WorkflowExecuteRequest struct {
	Message string                 `json:"message" binding:"required"`
	Context map[string]interface{} `json:"context,omitempty"`
}

// handleWorkflowExecute 执行 Workflow
func (os *AsterOS) handleWorkflowExecute(c *gin.Context) {
	workflowID := c.Param("id")

	// 获取 Workflow
	wf, exists := os.registry.GetWorkflow(workflowID)
	if !exists {
		c.JSON(404, gin.H{"error": "workflow not found"})
		return
	}

	// 解析请求
	var req WorkflowExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// 执行 Workflow
	ctx := context.Background()
	events := make([]string, 0)

	for event, err := range wf.Execute(ctx, req.Message) {
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		events = append(events, fmt.Sprintf("Event: %+v", event))
	}

	c.JSON(200, gin.H{
		"status": "success",
		"events": events,
	})
}
