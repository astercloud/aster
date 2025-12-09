package builtin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/astercloud/aster/pkg/logging"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/types"
	"github.com/google/uuid"
)

var askUserLog = logging.ForComponent("AskUserTool")

// askUserRequest 表示一个待处理的 AskUser 请求
type askUserRequest struct {
	responseChan chan map[string]any
	answered     bool       // 是否已回答
	mu           sync.Mutex // 保护 answered 字段
}

// 全局 pending requests 注册表（用于外部响应）
var (
	globalAskUserRequests   = make(map[string]*askUserRequest)
	globalAskUserRequestsMu sync.RWMutex
)

// RespondToAskUser 响应 AskUser 请求（供外部调用）
func RespondToAskUser(requestID string, answers map[string]any) error {
	globalAskUserRequestsMu.RLock()
	req, ok := globalAskUserRequests[requestID]
	globalAskUserRequestsMu.RUnlock()

	if !ok {
		return fmt.Errorf("no pending AskUser request with ID: %s", requestID)
	}

	// 使用互斥锁确保只能回答一次
	req.mu.Lock()
	if req.answered {
		req.mu.Unlock()
		return fmt.Errorf("request %s has already been answered", requestID)
	}
	req.answered = true
	req.mu.Unlock()

	// 发送答案到 channel
	select {
	case req.responseChan <- answers:
		askUserLog.Info(context.Background(), "successfully sent answer for request", map[string]any{"request_id": requestID})
		// 注意：不在这里删除 globalAskUserRequests，让 Execute 函数的 goroutine 来清理
		return nil
	default:
		return fmt.Errorf("response channel full or closed for request: %s", requestID)
	}
}

// AskUserQuestionTool 结构化用户提问工具
// 用于在执行过程中向用户提出结构化问题并获取回答
type AskUserQuestionTool struct {
	// 待处理的请求映射: requestID -> response channel
	pendingRequests map[string]chan map[string]any
}

// NewAskUserQuestionTool 创建AskUserQuestion工具
func NewAskUserQuestionTool(config map[string]any) (tools.Tool, error) {
	return &AskUserQuestionTool{
		pendingRequests: make(map[string]chan map[string]any),
	}, nil
}

func (t *AskUserQuestionTool) Name() string {
	return "AskUserQuestion"
}

func (t *AskUserQuestionTool) Description() string {
	return "向用户提出结构化问题，用于澄清需求、确认方案或收集偏好"
}

func (t *AskUserQuestionTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"questions": map[string]any{
				"type":        "array",
				"description": "要向用户提出的问题列表（1-4个问题）",
				"minItems":    1,
				"maxItems":    4,
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"question": map[string]any{
							"type":        "string",
							"description": "完整的问题文本，应清晰、具体，以问号结尾",
						},
						"header": map[string]any{
							"type":        "string",
							"description": "简短标签，最多12字符，如'Auth method'、'Library'",
							"maxLength":   12,
						},
						"options": map[string]any{
							"type":        "array",
							"description": "可选答案列表（2-4个选项）",
							"minItems":    2,
							"maxItems":    4,
							"items": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"label": map[string]any{
										"type":        "string",
										"description": "选项标签，1-5个词",
									},
									"description": map[string]any{
										"type":        "string",
										"description": "选项说明，解释该选项的含义或影响",
									},
								},
								"required": []string{"label", "description"},
							},
						},
						"multi_select": map[string]any{
							"type":        "boolean",
							"description": "是否允许多选，默认为false",
							"default":     false,
						},
					},
					"required": []string{"question", "header", "options"},
				},
			},
		},
		"required": []string{"questions"},
	}
}

func (t *AskUserQuestionTool) Execute(ctx context.Context, input map[string]any, tc *tools.ToolContext) (any, error) {
	// 验证必需参数
	if err := ValidateRequired(input, []string{"questions"}); err != nil {
		return NewClaudeErrorResponse(err), nil
	}

	// 解析问题列表
	questions, err := t.parseQuestions(input["questions"])
	if err != nil {
		return NewClaudeErrorResponse(err), nil
	}

	// 验证问题数量
	if len(questions) < 1 || len(questions) > 4 {
		return NewClaudeErrorResponse(fmt.Errorf("questions count must be between 1 and 4, got %d", len(questions))), nil
	}

	// 验证每个问题
	for i, q := range questions {
		if err := t.validateQuestion(q, i); err != nil {
			return NewClaudeErrorResponse(err), nil
		}
	}

	// 生成请求ID
	requestID := uuid.New().String()

	// 创建响应通道（带缓冲，确保发送不会阻塞）
	responseChan := make(chan map[string]any, 1)
	t.pendingRequests[requestID] = responseChan

	// 创建请求对象并注册到全局表
	req := &askUserRequest{
		responseChan: responseChan,
		answered:     false,
	}
	globalAskUserRequestsMu.Lock()
	globalAskUserRequests[requestID] = req
	globalAskUserRequestsMu.Unlock()

	askUserLog.Info(context.Background(), "registered request, waiting for user response", map[string]any{"request_id": requestID})

	// 创建响应回调函数（用于 Emit 事件）
	respond := func(answers map[string]any) error {
		return RespondToAskUser(requestID, answers)
	}

	// 发送事件到 Control 通道
	if tc.Reporter != nil {
		// 通过 Reporter 发送中间结果，包含事件信息
		tc.Reporter.Intermediate("ask_user_event", map[string]any{
			"request_id": requestID,
			"questions":  questions,
			"event_type": "ask_user",
		})
	}

	// 如果有 Emit 函数，使用它发送事件
	if tc.Emit != nil {
		tc.Emit("ask_user", &types.ControlAskUserEvent{
			RequestID: requestID,
			Questions: questions,
			Respond:   respond,
		})
	}

	// 等待用户响应或超时
	// 使用独立的 30 分钟超时，不受外层 context 影响
	// 这是因为 AskUserQuestion 需要等待用户响应，可能需要很长时间
	timeout := time.After(30 * time.Minute)

	// 清理函数
	cleanup := func() {
		delete(t.pendingRequests, requestID)
		globalAskUserRequestsMu.Lock()
		delete(globalAskUserRequests, requestID)
		globalAskUserRequestsMu.Unlock()
		askUserLog.Debug(context.Background(), "cleaned up request", map[string]any{"request_id": requestID})
	}

	// 直接在当前 goroutine 中等待，不使用外层 ctx
	// 这样即使 Agent 执行循环的 context 被取消，我们仍然可以等待用户响应
	select {
	case answers := <-responseChan:
		cleanup()
		askUserLog.Info(context.Background(), "received answer for request", map[string]any{"request_id": requestID, "answers": answers})
		return map[string]any{
			"ok":         true,
			"request_id": requestID,
			"answers":    answers,
			"timestamp":  time.Now().Unix(),
		}, nil
	case <-timeout:
		cleanup()
		askUserLog.Warn(context.Background(), "request timed out after 30 minutes", map[string]any{"request_id": requestID})
		return map[string]any{
			"ok":         false,
			"request_id": requestID,
			"error":      "user response timeout (30 minutes)",
			"timestamp":  time.Now().Unix(),
		}, nil
	case <-ctx.Done():
		// 外层 context 被取消（比如 Agent 执行循环超时）
		// 但我们不清理 pending request，让用户仍然可以响应
		// 启动一个后台 goroutine 来等待用户响应并清理
		askUserLog.Info(context.Background(), "context cancelled, but keeping request alive for user response", map[string]any{"request_id": requestID})
		go func() {
			select {
			case <-responseChan:
				// 用户响应了，但我们已经返回了，所以只需要清理
				askUserLog.Info(context.Background(), "user responded after context cancellation", map[string]any{"request_id": requestID})
			case <-timeout:
				askUserLog.Warn(context.Background(), "request timed out in background", map[string]any{"request_id": requestID})
			}
			cleanup()
		}()
		// 返回一个特殊的结果，告诉 Agent 用户正在响应
		return map[string]any{
			"ok":         false,
			"request_id": requestID,
			"waiting":    true,
			"message":    "等待用户响应中，请稍候...",
			"timestamp":  time.Now().Unix(),
		}, nil
	}
}

// parseQuestions 解析问题列表
func (t *AskUserQuestionTool) parseQuestions(value any) ([]types.Question, error) {
	questionsRaw, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("questions must be an array")
	}

	questions := make([]types.Question, 0, len(questionsRaw))
	for i, qRaw := range questionsRaw {
		qMap, ok := qRaw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("question[%d] must be an object", i)
		}

		q := types.Question{
			Question:    GetStringParam(qMap, "question", ""),
			Header:      GetStringParam(qMap, "header", ""),
			MultiSelect: GetBoolParam(qMap, "multi_select", false),
		}

		// 解析选项
		if optionsRaw, exists := qMap["options"]; exists {
			options, err := t.parseOptions(optionsRaw, i)
			if err != nil {
				return nil, err
			}
			q.Options = options
		}

		questions = append(questions, q)
	}

	return questions, nil
}

// parseOptions 解析选项列表
func (t *AskUserQuestionTool) parseOptions(value any, questionIndex int) ([]types.QuestionOption, error) {
	optionsRaw, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("question[%d].options must be an array", questionIndex)
	}

	options := make([]types.QuestionOption, 0, len(optionsRaw))
	for j, oRaw := range optionsRaw {
		oMap, ok := oRaw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("question[%d].options[%d] must be an object", questionIndex, j)
		}

		opt := types.QuestionOption{
			Label:       GetStringParam(oMap, "label", ""),
			Description: GetStringParam(oMap, "description", ""),
		}
		options = append(options, opt)
	}

	return options, nil
}

// validateQuestion 验证单个问题
func (t *AskUserQuestionTool) validateQuestion(q types.Question, index int) error {
	if q.Question == "" {
		return fmt.Errorf("question[%d].question cannot be empty", index)
	}
	if q.Header == "" {
		return fmt.Errorf("question[%d].header cannot be empty", index)
	}
	if len(q.Header) > 12 {
		return fmt.Errorf("question[%d].header must be at most 12 characters, got %d", index, len(q.Header))
	}
	if len(q.Options) < 2 || len(q.Options) > 4 {
		return fmt.Errorf("question[%d].options must have 2-4 items, got %d", index, len(q.Options))
	}

	for j, opt := range q.Options {
		if opt.Label == "" {
			return fmt.Errorf("question[%d].options[%d].label cannot be empty", index, j)
		}
		if opt.Description == "" {
			return fmt.Errorf("question[%d].options[%d].description cannot be empty", index, j)
		}
	}

	return nil
}

// ReceiveAnswer 接收用户回答（供外部调用）
func (t *AskUserQuestionTool) ReceiveAnswer(requestID string, answers map[string]any) error {
	ch, exists := t.pendingRequests[requestID]
	if !exists {
		return fmt.Errorf("no pending request with ID: %s", requestID)
	}

	select {
	case ch <- answers:
		return nil
	default:
		return fmt.Errorf("response channel is full or closed")
	}
}

func (t *AskUserQuestionTool) Prompt() string {
	return `向用户提出结构化问题，用于澄清需求、确认方案或收集偏好。

使用场景:
- 收集用户偏好或需求
- 澄清模糊的指令
- 获取实施方案的决策
- 提供选择让用户决定方向

参数说明:
- questions: 问题列表（1-4个问题）
  - question: 完整的问题文本，应清晰具体，以问号结尾
  - header: 简短标签（最多12字符），如"Auth method"、"Library"
  - options: 选项列表（2-4个选项）
    - label: 选项标签，1-5个词
    - description: 选项说明
  - multi_select: 是否允许多选（默认false）

使用示例:
{
  "questions": [
    {
      "question": "你希望使用哪种认证方式？",
      "header": "认证方式",
      "options": [
        {"label": "JWT", "description": "基于令牌的无状态认证"},
        {"label": "Session", "description": "基于会话的有状态认证"},
        {"label": "OAuth2", "description": "第三方OAuth2认证"}
      ],
      "multi_select": false
    }
  ]
}

注意事项:
- 用户总是可以选择"Other"来提供自定义输入
- 问题应简洁明了，避免技术术语过多
- 选项描述应解释该选择的含义或影响
- 如果multi_select为true，需要相应地措辞问题`
}
