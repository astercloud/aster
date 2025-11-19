package asteros

import (
	"context"
	"fmt"

	"github.com/astercloud/aster/pkg/stars"
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

// handleListStars 列出所有 Stars
func (os *AsterOS) handleListStars(c *gin.Context) {
	starsList := os.registry.ListStars()
	c.JSON(200, gin.H{
		"stars": starsList,
		"count": len(starsList),
	})
}

// StarsRunRequest Stars 运行请求
type StarsRunRequest struct {
	Task    string                 `json:"task" binding:"required"`
	Context map[string]interface{} `json:"context,omitempty"`
}

// handleStarsRun 运行 Stars
func (os *AsterOS) handleStarsRun(c *gin.Context) {
	starsID := c.Param("id")

	// 获取 Stars
	s, exists := os.registry.GetStars(starsID)
	if !exists {
		c.JSON(404, gin.H{"error": "stars not found"})
		return
	}

	// 解析请求
	var req StarsRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// 运行 Stars
	ctx := context.Background()
	events := make([]string, 0)

	for event, err := range s.Run(ctx, req.Task) {
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		events = append(events, fmt.Sprintf("[%s] %s", event.Type, event.Content))
	}

	c.JSON(200, gin.H{
		"status": "success",
		"events": events,
	})
}

// StarsJoinRequest Stars 加入请求
type StarsJoinRequest struct {
	AgentID string `json:"agent_id" binding:"required"`
	Role    string `json:"role" binding:"required"`
}

// handleStarsJoin 添加成员到 Stars
func (os *AsterOS) handleStarsJoin(c *gin.Context) {
	starsID := c.Param("id")

	// 获取 Stars
	s, exists := os.registry.GetStars(starsID)
	if !exists {
		c.JSON(404, gin.H{"error": "stars not found"})
		return
	}

	// 解析请求
	var req StarsJoinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// 解析角色
	var role stars.Role
	switch req.Role {
	case "leader":
		role = stars.RoleLeader
	case "worker":
		role = stars.RoleWorker
	default:
		c.JSON(400, gin.H{"error": "invalid role"})
		return
	}

	// 添加成员
	if err := s.Join(req.AgentID, role); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"status":  "success",
		"message": fmt.Sprintf("agent %s joined as %s", req.AgentID, req.Role),
	})
}

// StarsLeaveRequest Stars 离开请求
type StarsLeaveRequest struct {
	AgentID string `json:"agent_id" binding:"required"`
}

// handleStarsLeave 从 Stars 移除成员
func (os *AsterOS) handleStarsLeave(c *gin.Context) {
	starsID := c.Param("id")

	// 获取 Stars
	s, exists := os.registry.GetStars(starsID)
	if !exists {
		c.JSON(404, gin.H{"error": "stars not found"})
		return
	}

	// 解析请求
	var req StarsLeaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// 移除成员
	if err := s.Leave(req.AgentID); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"status":  "success",
		"message": fmt.Sprintf("agent %s left", req.AgentID),
	})
}

// handleStarsMembers 获取 Stars 成员列表
func (os *AsterOS) handleStarsMembers(c *gin.Context) {
	starsID := c.Param("id")

	// 获取 Stars
	s, exists := os.registry.GetStars(starsID)
	if !exists {
		c.JSON(404, gin.H{"error": "stars not found"})
		return
	}

	// 获取成员
	members := s.Members()

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
