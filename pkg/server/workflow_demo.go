package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/agent/workflow"
	"github.com/astercloud/aster/pkg/evals"
	"github.com/astercloud/aster/pkg/session"
	"github.com/astercloud/aster/pkg/types"
)

// WorkflowRunRequest 表示一次工作流运行请求。
// 当前仅提供基于 MockAgent 的演示工作流, 用于快速体验工作流编排与事件结构。
type WorkflowRunRequest struct {
	// WorkflowID 工作流标识:
	// - "sequential_demo"
	// - "parallel_demo"
	// - "loop_demo"
	// - "nested_demo"
	WorkflowID string `json:"workflow_id"`

	// Input 传递给工作流的输入消息。
	Input string `json:"input"`
}

// WorkflowEvent 表示通过 HTTP 返回的精简事件结构。
type WorkflowEvent struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	AgentID   string                 `json:"agent_id"`
	Branch    string                 `json:"branch,omitempty"`
	Author    string                 `json:"author"`
	Text      string                 `json:"text"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// WorkflowRunResponse 表示工作流运行的结果。
type WorkflowRunResponse struct {
	RunID       string          `json:"run_id,omitempty"`
	Events       []WorkflowEvent `json:"events"`
	ErrorMessage string          `json:"error_message,omitempty"`
}

// WorkflowRunEvalRequest 在 WorkflowRunRequest 基础上增加可选的 eval 参数。
type WorkflowRunEvalRequest struct {
	WorkflowRunRequest

	// Reference 可选参考答案, 用于词汇相似度评估。
	Reference string `json:"reference,omitempty"`
	// Keywords 可选关键词列表, 用于关键词覆盖率评估。
	Keywords []string `json:"keywords,omitempty"`
	// Scorers 指定要启用的 scorer 名称, 为空时默认启用:
	// ["keyword_coverage", "lexical_similarity"]。
	Scorers []string `json:"scorers,omitempty"`
}

// WorkflowRunEvalResponse 在 WorkflowRunResponse 基础上增加 eval_scores 字段。
type WorkflowRunEvalResponse struct {
	WorkflowRunResponse
	EvalScores []evals.ScoreResult `json:"eval_scores,omitempty"`
}

// WorkflowRunRecord 表示持久化存储的一次工作流运行记录。
type WorkflowRunRecord struct {
	ID           string              `json:"id"`
	WorkflowID   string              `json:"workflow_id"`
	Input        string              `json:"input"`
	Events       []WorkflowEvent     `json:"events"`
	EvalScores   []evals.ScoreResult `json:"eval_scores,omitempty"`
	ErrorMessage string              `json:"error_message,omitempty"`
	CreatedAt    time.Time           `json:"created_at"`
}

// workflowRunStore 简单的内存工作流运行记录存储, 仅用于 demo。
type workflowRunStore struct {
	mu   sync.RWMutex
	runs map[string]*WorkflowRunRecord
}

func newWorkflowRunStore() *workflowRunStore {
	return &workflowRunStore{
		runs: make(map[string]*WorkflowRunRecord),
	}
}

func (s *workflowRunStore) Save(rec *WorkflowRunRecord) {
	if rec == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runs[rec.ID] = rec
}

func (s *workflowRunStore) Get(id string) (*WorkflowRunRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.runs[id]
	return rec, ok
}

// WorkflowDemoGetRunHandler 返回指定 run_id 的工作流运行记录。
//
// 路径示例:
//   GET /v1/workflows/demo/runs?id=<run_id>
//
// 响应体:
//   {
//     "id": "run-...",
//     "workflow_id": "sequential_demo",
//     "input": "处理用户数据",
//     "events": [...],
//     "eval_scores": [...],
//     "created_at": "..."
//   }
func (s *Server) WorkflowDemoGetRunHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		runID := r.URL.Query().Get("id")
		if runID == "" {
			http.Error(w, "id is required", http.StatusBadRequest)
			return
		}

		rec, ok := s.workflowRuns.Get(runID)
		if !ok {
			http.Error(w, "run not found", http.StatusNotFound)
			return
		}

		writeJSON(w, http.StatusOK, rec)
	})
}

// WorkflowDemoRunHandler 返回一个 HTTP handler, 用于运行内置的演示工作流。
//
// 路径示例:
//   POST /v1/workflows/demo/run
//
// 请求体:
//   {
//     "workflow_id": "sequential_demo",
//     "input": "处理用户数据"
//   }
//
// 响应体:
//   {
//     "events": [
//       {"id":"...","agent_id":"DataCollector","text":"...","metadata":{...}},
//       ...
//     ]
//   }
func (s *Server) WorkflowDemoRunHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req WorkflowRunRequest
		if err := jsonNewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}

		if req.WorkflowID == "" {
			http.Error(w, "workflow_id is required", http.StatusBadRequest)
			return
		}
		if req.Input == "" {
			http.Error(w, "input is required", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()

		wf, err := s.buildDemoWorkflow(req.WorkflowID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var events []WorkflowEvent

		for ev, err := range wf.Execute(ctx, req.Input) {
			if err != nil {
				resp := &WorkflowRunResponse{
					ErrorMessage: err.Error(),
				}
				writeJSON(w, http.StatusInternalServerError, resp)
				return
			}
			if ev == nil {
				continue
			}
			events = append(events, WorkflowEvent{
				ID:        ev.ID,
				Timestamp: ev.Timestamp,
				AgentID:   ev.AgentID,
				Branch:    ev.Branch,
				Author:    ev.Author,
				Text:      ev.Content.Content,
				Metadata:  ev.Metadata,
			})
		}

		runID := uuid.New().String()
		record := &WorkflowRunRecord{
			ID:         runID,
			WorkflowID: req.WorkflowID,
			Input:      req.Input,
			Events:     events,
			CreatedAt:  time.Now(),
		}
		s.workflowRuns.Save(record)

		writeJSON(w, http.StatusOK, &WorkflowRunResponse{
			RunID: runID,
			Events: events,
		})
	})
}

// WorkflowDemoRunEvalHandler 返回一个 HTTP handler, 用于运行内置 demo 工作流并对最终回答进行评估。
//
// 路径示例:
//   POST /v1/workflows/demo/run-eval
//
// 请求体:
//   {
//     "workflow_id": "sequential_demo",
//     "input": "处理用户数据",
//     "reference": "期望的总结结果...",
//     "keywords": ["收集", "分析", "报告"],
//     "scorers": ["keyword_coverage", "lexical_similarity"]
//   }
//
// 响应体:
//   {
//     "events": [...],
//     "eval_scores": [
//       {"name": "keyword_coverage", "value": 0.75, ...},
//       {"name": "lexical_similarity", "value": 0.82, ...}
//     ]
//   }
func (s *Server) WorkflowDemoRunEvalHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req WorkflowRunEvalRequest
		if err := jsonNewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}

		if req.WorkflowID == "" {
			http.Error(w, "workflow_id is required", http.StatusBadRequest)
			return
		}
		if req.Input == "" {
			http.Error(w, "input is required", http.StatusBadRequest)
			return
		}

		// 验证 scorers 列表
		scorerNames := req.Scorers
		if len(scorerNames) == 0 {
			scorerNames = []string{"keyword_coverage", "lexical_similarity"}
		}
		for _, name := range scorerNames {
			if name != "keyword_coverage" && name != "lexical_similarity" {
				http.Error(w, "unsupported scorer: "+name, http.StatusBadRequest)
				return
			}
		}

		ctx := r.Context()
		ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()

		wf, err := s.buildDemoWorkflow(req.WorkflowID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var (
			events    []WorkflowEvent
			rawEvents []session.Event
		)

		for ev, err := range wf.Execute(ctx, req.Input) {
			if err != nil {
				resp := &WorkflowRunEvalResponse{
					WorkflowRunResponse: WorkflowRunResponse{
						ErrorMessage: err.Error(),
					},
				}
				writeJSON(w, http.StatusInternalServerError, resp)
				return
			}
			if ev == nil {
				continue
			}

			events = append(events, WorkflowEvent{
				ID:        ev.ID,
				Timestamp: ev.Timestamp,
				AgentID:   ev.AgentID,
				Branch:    ev.Branch,
				Author:    ev.Author,
				Text:      ev.Content.Content,
				Metadata:  ev.Metadata,
			})

			rawEvents = append(rawEvents, *ev)
		}

		// 使用 session 事件构建 TextEvalInput
		textInput := evals.BuildTextEvalInputFromEvents(rawEvents)
		textInput.Reference = req.Reference

		evalScores := make([]evals.ScoreResult, 0, len(scorerNames))
		for _, name := range scorerNames {
			switch name {
			case "keyword_coverage":
				scorer := evals.NewKeywordCoverageScorer(evals.KeywordCoverageConfig{
					Keywords:        req.Keywords,
					CaseInsensitive: true,
				})
				res, err := scorer.Score(ctx, textInput)
				if err != nil {
					resp := &WorkflowRunEvalResponse{
						WorkflowRunResponse: WorkflowRunResponse{
							ErrorMessage: err.Error(),
						},
					}
					writeJSON(w, http.StatusInternalServerError, resp)
					return
				}
				evalScores = append(evalScores, *res)
			case "lexical_similarity":
				scorer := evals.NewLexicalSimilarityScorer(evals.LexicalSimilarityConfig{
					MinTokenLength: 2,
				})
				res, err := scorer.Score(ctx, textInput)
				if err != nil {
					resp := &WorkflowRunEvalResponse{
						WorkflowRunResponse: WorkflowRunResponse{
							ErrorMessage: err.Error(),
						},
					}
					writeJSON(w, http.StatusInternalServerError, resp)
					return
				}
				evalScores = append(evalScores, *res)
			}
		}

		runID := uuid.New().String()
		record := &WorkflowRunRecord{
			ID:         runID,
			WorkflowID: req.WorkflowID,
			Input:      req.Input,
			Events:     events,
			EvalScores: evalScores,
			CreatedAt:  time.Now(),
		}
		s.workflowRuns.Save(record)

		writeJSON(w, http.StatusOK, &WorkflowRunEvalResponse{
			WorkflowRunResponse: WorkflowRunResponse{
				RunID: runID,
				Events: events,
			},
			EvalScores: evalScores,
		})
	})
}

// buildDemoWorkflow 根据 workflow_id 构建一个内置演示工作流。
func (s *Server) buildDemoWorkflow(id string) (workflow.Agent, error) {
	switch id {
	case "sequential_demo":
		return s.buildSequentialDemo()
	case "parallel_demo":
		return s.buildParallelDemo()
	case "loop_demo":
		return s.buildLoopDemo()
	case "nested_demo":
		return s.buildNestedDemo()
	default:
		return nil, fmt.Errorf("unsupported workflow_id: %s", id)
	}
}

// ======== Demo workflow builders ========

func (s *Server) buildSequentialDemo() (workflow.Agent, error) {
	agents := []workflow.Agent{
		NewLLMWorkflowAgent("DataCollector", "收集数据", s.deps),
		NewLLMWorkflowAgent("Analyzer", "分析数据", s.deps),
		NewLLMWorkflowAgent("Reporter", "生成报告", s.deps),
	}

	return workflow.NewSequentialAgent(workflow.SequentialConfig{
		Name:      "DataPipeline",
		SubAgents: agents,
	})
}

func (s *Server) buildParallelDemo() (workflow.Agent, error) {
	agents := []workflow.Agent{
		NewLLMWorkflowAgent("AlgorithmA", "方案A：快速但粗糙", s.deps),
		NewLLMWorkflowAgent("AlgorithmB", "方案B：慢但精确", s.deps),
		NewLLMWorkflowAgent("AlgorithmC", "方案C：平衡", s.deps),
	}

	return workflow.NewParallelAgent(workflow.ParallelConfig{
		Name:      "MultiAlgorithm",
		SubAgents: agents,
	})
}

func (s *Server) buildLoopDemo() (workflow.Agent, error) {
	critic := NewLLMWorkflowAgent("Critic", "评估当前方案", s.deps)
	improver := NewLLMWorkflowAgent("Improver", "提出改进建议", s.deps)

	return workflow.NewLoopAgent(workflow.LoopConfig{
		Name:          "OptimizationLoop",
		SubAgents:     []workflow.Agent{critic, improver},
		MaxIterations: 3,
		StopCondition: func(ev *session.Event) bool {
			if ev == nil || ev.Metadata == nil {
				return false
			}
			if score, ok := ev.Metadata["quality_score"].(int); ok {
				return score >= 90
			}
			return false
		},
	})
}

func (s *Server) buildNestedDemo() (workflow.Agent, error) {
	dataCollectors := []workflow.Agent{
		NewLLMWorkflowAgent("Source1", "数据源1", s.deps),
		NewLLMWorkflowAgent("Source2", "数据源2", s.deps),
		NewLLMWorkflowAgent("Source3", "数据源3", s.deps),
	}
	parallelCollector, _ := workflow.NewParallelAgent(workflow.ParallelConfig{
		Name:      "ParallelCollector",
		SubAgents: dataCollectors,
	})

	analyzer := NewLLMWorkflowAgent("Analyzer", "数据分析", s.deps)
	reporter := NewLLMWorkflowAgent("Reporter", "生成报告", s.deps)

	return workflow.NewSequentialAgent(workflow.SequentialConfig{
		Name: "NestedWorkflow",
		SubAgents: []workflow.Agent{
			parallelCollector,
			analyzer,
			reporter,
		},
	})
}

// ======== DemoAgent: Mock workflow.Agent 实现 ========

// LLMWorkflowAgent 是一个基于 agent.Agent 封装的 workflow.Agent 实现,
// 用于在 demo 工作流中调用真实 LLM。
type LLMWorkflowAgent struct {
	name        string
	description string
	templateID  string
	deps        *agent.Dependencies
}

// NewLLMWorkflowAgent 创建一个 LLMWorkflowAgent。
// 当前默认使用 templateID = "assistant"。
func NewLLMWorkflowAgent(name, description string, deps *agent.Dependencies) *LLMWorkflowAgent {
	return &LLMWorkflowAgent{
		name:        name,
		description: description,
		templateID:  "assistant",
		deps:        deps,
	}
}

// Name 实现 workflow.Agent 接口。
func (a *LLMWorkflowAgent) Name() string {
	return a.name
}

// Execute 实现 workflow.Agent 接口。
func (a *LLMWorkflowAgent) Execute(ctx context.Context, message string) iter.Seq2[*session.Event, error] {
	return func(yield func(*session.Event, error) bool) {
		// 为每次执行创建一个独立的 Agent 实例
		agentConfig := &types.AgentConfig{
			TemplateID: a.templateID,
			Metadata: map[string]interface{}{
				"workflow_step": a.name,
			},
		}

		ag, err := agent.Create(ctx, agentConfig, a.deps)
		if err != nil {
			_ = yield(nil, err)
			return
		}
		defer func() { _ = ag.Close() }()

		// 为当前工作流步骤构造提示词
		prompt := fmt.Sprintf("[%s] %s - 处理: %s", a.name, a.description, message)

		// 调用底层 Agent
		res, err := ag.Chat(ctx, prompt)
		if !yield(&session.Event{
			ID:           fmt.Sprintf("evt-%s-%d", a.name, time.Now().UnixNano()),
			Timestamp:    time.Now(),
			InvocationID: "workflow-llm-demo",
			AgentID:      a.name,
			Author:       "agent",
			Content: types.Message{
				Role:    types.RoleAssistant,
				Content: res.Text,
			},
			Metadata: map[string]interface{}{
				"agent_description": a.description,
			},
		}, err) {
			return
		}
	}
}

// jsonNewDecoder 抽象出 json.NewDecoder, 方便测试/替换。
var jsonNewDecoder = func(r io.Reader) *json.Decoder {
	return json.NewDecoder(r)
}
