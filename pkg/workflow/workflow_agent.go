package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// WorkflowAgent 专门用于 Workflow 编排的受限 Agent
// A restricted Agent class specifically designed for workflow orchestration.
type WorkflowAgent struct {
	ID                 string
	Name               string
	Instructions       string
	Model              string
	AddWorkflowHistory bool
	NumHistoryRuns     int
	workflow           *Workflow
	mu                 sync.RWMutex
}

// NewWorkflowAgent 创建新的 WorkflowAgent
func NewWorkflowAgent(model, instructions string, addHistory bool, numRuns int) *WorkflowAgent {
	defaultInstructions := `You are a workflow orchestration agent. Your job is to help users by either:
1. Answering directly from workflow history if the question can be answered
2. Running the workflow when needed for new queries`

	if instructions == "" {
		instructions = defaultInstructions
	}

	return &WorkflowAgent{
		ID:                 uuid.New().String(),
		Name:               "WorkflowAgent",
		Instructions:       instructions,
		Model:              model,
		AddWorkflowHistory: addHistory,
		NumHistoryRuns:     numRuns,
	}
}

// AttachWorkflow 将 workflow 附加到 agent
func (wa *WorkflowAgent) AttachWorkflow(wf *Workflow) *WorkflowAgent {
	wa.mu.Lock()
	defer wa.mu.Unlock()
	wa.workflow = wf
	return wa
}

// CreateWorkflowTool 创建 workflow 执行工具
func (wa *WorkflowAgent) CreateWorkflowTool(
	session *WorkflowSession,
	executionInput *WorkflowInput,
	stream bool,
) WorkflowToolFunc {
	return func(ctx context.Context, query string) (interface{}, error) {
		if wa.workflow == nil {
			return nil, fmt.Errorf("no workflow attached to agent")
		}

		workflowInput := &WorkflowInput{
			Input:          query,
			AdditionalData: executionInput.AdditionalData,
			Images:         executionInput.Images,
			Videos:         executionInput.Videos,
			Audio:          executionInput.Audio,
			Files:          executionInput.Files,
			SessionID:      session.ID,
			SessionState:   session.State,
		}

		if stream {
			return wa.executeStreamingWorkflow(ctx, workflowInput)
		}

		return wa.executeWorkflow(ctx, workflowInput)
	}
}

// executeWorkflow 执行 workflow（非流式）
func (wa *WorkflowAgent) executeWorkflow(ctx context.Context, input *WorkflowInput) (interface{}, error) {
	var finalOutput interface{}
	var finalError error

	for event, err := range wa.workflow.Execute(ctx, input) {
		if err != nil {
			finalError = err
			continue
		}

		if event.Type == EventWorkflowCompleted {
			if data, ok := event.Data.(map[string]interface{}); ok {
				finalOutput = data["output"]
			}
		}
	}

	if finalError != nil {
		return nil, finalError
	}

	return wa.formatOutput(finalOutput), nil
}

// executeStreamingWorkflow 执行 workflow（流式）
func (wa *WorkflowAgent) executeStreamingWorkflow(ctx context.Context, input *WorkflowInput) (interface{}, error) {
	resultChan := make(chan interface{}, 100)

	go func() {
		defer close(resultChan)

		for event, err := range wa.workflow.Execute(ctx, input) {
			if err != nil {
				resultChan <- map[string]interface{}{"error": err.Error()}
				continue
			}
			resultChan <- event
		}
	}()

	return resultChan, nil
}

// formatOutput 格式化输出为字符串
func (wa *WorkflowAgent) formatOutput(output interface{}) string {
	if output == nil {
		return ""
	}

	switch v := output.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(jsonBytes)
	}
}

// GetWorkflowHistory 获取 workflow 历史
func (wa *WorkflowAgent) GetWorkflowHistory() []WorkflowHistoryItem {
	wa.mu.RLock()
	defer wa.mu.RUnlock()

	if wa.workflow == nil || wa.workflow.workflowSession == nil {
		return []WorkflowHistoryItem{}
	}

	runs := wa.workflow.workflowSession.History
	numRuns := len(runs)
	if numRuns > wa.NumHistoryRuns {
		numRuns = wa.NumHistoryRuns
	}

	history := make([]WorkflowHistoryItem, numRuns)
	for i := 0; i < numRuns; i++ {
		run := runs[len(runs)-numRuns+i]
		history[i] = WorkflowHistoryItem{
			RunID:     run.RunID,
			Input:     run.Input,
			Output:    run.Output,
			Status:    string(run.Status),
			StartTime: run.StartTime,
			EndTime:   run.EndTime,
			Duration:  run.Duration,
			Metrics:   run.Metrics,
		}
	}

	return history
}

// Run 运行 WorkflowAgent
func (wa *WorkflowAgent) Run(ctx context.Context, input string) (string, error) {
	wa.mu.RLock()
	workflow := wa.workflow
	wa.mu.RUnlock()

	if workflow == nil {
		return "", fmt.Errorf("no workflow attached to agent")
	}

	workflowInput := &WorkflowInput{
		Input: input,
	}

	result, err := wa.executeWorkflow(ctx, workflowInput)
	if err != nil {
		return "", err
	}

	return wa.formatOutput(result), nil
}

// RunStream 流式运行 WorkflowAgent
func (wa *WorkflowAgent) RunStream(ctx context.Context, input string) <-chan AgentStreamEvent {
	eventChan := make(chan AgentStreamEvent, 100)

	go func() {
		defer close(eventChan)

		eventChan <- AgentStreamEvent{
			Type:      AgentEventStart,
			Timestamp: time.Now(),
			Data:      map[string]interface{}{"input": input},
		}

		workflowInput := &WorkflowInput{Input: input}
		for event, err := range wa.workflow.Execute(ctx, workflowInput) {
			if err != nil {
				eventChan <- AgentStreamEvent{
					Type:      AgentEventError,
					Timestamp: time.Now(),
					Error:     err,
				}
				continue
			}

			eventChan <- AgentStreamEvent{
				Type:      AgentEventWorkflowEvent,
				Timestamp: time.Now(),
				Data:      map[string]interface{}{"workflow_event": event},
			}

			if event.Type == EventWorkflowCompleted {
				if data, ok := event.Data.(map[string]interface{}); ok {
					if output, ok := data["output"]; ok {
						eventChan <- AgentStreamEvent{
							Type:      AgentEventResponse,
							Timestamp: time.Now(),
							Data:      map[string]interface{}{"response": output},
						}
					}
				}
			}
		}

		eventChan <- AgentStreamEvent{
			Type:      AgentEventComplete,
			Timestamp: time.Now(),
		}
	}()

	return eventChan
}

// ===== Types =====

// WorkflowToolFunc Workflow 工具函数类型
type WorkflowToolFunc func(ctx context.Context, query string) (interface{}, error)

// WorkflowHistoryItem Workflow 历史项
type WorkflowHistoryItem struct {
	RunID     string
	Input     interface{}
	Output    interface{}
	Status    string
	StartTime time.Time
	EndTime   time.Time
	Duration  float64
	Metrics   *RunMetrics
}

// AgentStreamEvent Agent 流式事件
type AgentStreamEvent struct {
	Type      AgentEventType
	Timestamp time.Time
	Data      map[string]interface{}
	Error     error
}

// AgentEventType Agent 事件类型
type AgentEventType string

const (
	AgentEventStart         AgentEventType = "agent_start"
	AgentEventWorkflowStart AgentEventType = "workflow_start"
	AgentEventWorkflowEvent AgentEventType = "workflow_event"
	AgentEventResponse      AgentEventType = "agent_response"
	AgentEventComplete      AgentEventType = "agent_complete"
	AgentEventError         AgentEventType = "agent_error"
)

// ===== Builder Methods =====

// WithInstructions 设置指令
func (wa *WorkflowAgent) WithInstructions(instructions string) *WorkflowAgent {
	wa.mu.Lock()
	defer wa.mu.Unlock()
	wa.Instructions = instructions
	return wa
}

// WithModel 设置模型
func (wa *WorkflowAgent) WithModel(model string) *WorkflowAgent {
	wa.mu.Lock()
	defer wa.mu.Unlock()
	wa.Model = model
	return wa
}

// WithHistorySize 设置历史记录数量
func (wa *WorkflowAgent) WithHistorySize(num int) *WorkflowAgent {
	wa.mu.Lock()
	defer wa.mu.Unlock()
	wa.NumHistoryRuns = num
	return wa
}

// EnableHistory 启用/禁用历史记录
func (wa *WorkflowAgent) EnableHistory(enable bool) *WorkflowAgent {
	wa.mu.Lock()
	defer wa.mu.Unlock()
	wa.AddWorkflowHistory = enable
	return wa
}

// GetWorkflow 获取关联的 workflow
func (wa *WorkflowAgent) GetWorkflow() *Workflow {
	wa.mu.RLock()
	defer wa.mu.RUnlock()
	return wa.workflow
}

// ===== Workflow Methods with Agent =====

// WithAgent 为 Workflow 设置 Agent（Agentic Workflow）
func (wf *Workflow) WithAgent(agent *WorkflowAgent) *Workflow {
	agent.AttachWorkflow(wf)
	return wf
}

// AgenticExecute Agentic 方式执行 - Agent 决定何时运行 workflow
func (wf *Workflow) AgenticExecute(ctx context.Context, agent *WorkflowAgent, input string) (string, error) {
	if agent.GetWorkflow() == nil {
		agent.AttachWorkflow(wf)
	}
	return agent.Run(ctx, input)
}

// AgenticExecuteStream Agentic 方式流式执行
func (wf *Workflow) AgenticExecuteStream(ctx context.Context, agent *WorkflowAgent, input string) <-chan AgentStreamEvent {
	if agent.GetWorkflow() == nil {
		agent.AttachWorkflow(wf)
	}
	return agent.RunStream(ctx, input)
}
