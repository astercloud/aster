package session

import (
	"context"
	"errors"
	"iter"
	"time"

	"github.com/astercloud/aster/pkg/types"
)

// Session 表示用户与 Agent 之间的一系列交互
// 参考 Google ADK-Go 的 Session 设计
type Session interface {
	// ID 返回会话的唯一标识符
	ID() string

	// AppName 返回应用名称
	AppName() string

	// UserID 返回用户 ID
	UserID() string

	// AgentID 返回关联的 Agent ID
	AgentID() string

	// State 返回会话状态管理器
	State() State

	// Events 返回事件列表
	Events() Events

	// LastUpdateTime 返回最后更新时间
	LastUpdateTime() time.Time

	// Metadata 返回会话元数据
	Metadata() map[string]interface{}
}

// State 定义分层的状态管理接口
// 支持三种作用域: App/User/Temp
type State interface {
	// Get 获取指定 key 的值
	// 如果 key 不存在，返回 ErrStateKeyNotExist
	Get(key string) (interface{}, error)

	// Set 设置 key-value，覆盖已存在的值
	Set(key string, value interface{}) error

	// Delete 删除指定 key
	Delete(key string) error

	// All 返回所有 key-value 的迭代器
	All() iter.Seq2[string, interface{}]

	// Has 检查 key 是否存在
	Has(key string) bool
}

// ReadonlyState 只读状态接口
type ReadonlyState interface {
	Get(key string) (interface{}, error)
	All() iter.Seq2[string, interface{}]
	Has(key string) bool
}

// Events 定义事件列表接口
type Events interface {
	// All 返回所有事件的迭代器，保持顺序
	All() iter.Seq[*Event]

	// Len 返回事件总数
	Len() int

	// At 返回指定索引的事件
	At(i int) *Event

	// Filter 根据条件过滤事件
	Filter(predicate func(*Event) bool) []Event

	// Last 返回最后一个事件
	Last() *Event
}

// Event 表示会话中的一个交互事件
// 扩展自 types.Event，增加更多元数据
type Event struct {
	// 基础字段
	ID           string
	Timestamp    time.Time
	InvocationID string

	// Agent 相关
	AgentID string
	Branch  string // 多 Agent 分支隔离: "agent1.agent2.agent3"
	Author  string // 事件作者 (user/agent/system)

	// 内容
	Content types.Message

	// 动作
	Actions EventActions

	// 长时运行工具 ID 列表
	LongRunningToolIDs []string

	// 元数据
	Metadata map[string]interface{}
}

// EventActions 表示事件附带的动作
type EventActions struct {
	// StateDelta 状态增量更新
	StateDelta map[string]interface{}

	// ArtifactDelta Artifact 版本变化 (filename -> version)
	ArtifactDelta map[string]int64

	// SkipSummarization 是否跳过总结
	SkipSummarization bool

	// TransferToAgent 转移到指定 Agent
	TransferToAgent string

	// Escalate 升级到上级 Agent
	Escalate bool

	// CustomActions 自定义动作
	CustomActions map[string]interface{}
}

// IsFinalResponse 判断是否为最终响应
func (e *Event) IsFinalResponse() bool {
	if e.Actions.SkipSummarization || len(e.LongRunningToolIDs) > 0 {
		return true
	}

	// 检查是否有工具调用
	if e.Content.Role == types.RoleAssistant {
		if len(e.Content.ToolCalls) > 0 {
			return false
		}
	}

	return true
}

// State 作用域前缀常量
const (
	// KeyPrefixApp 应用级状态前缀
	// 跨所有用户和会话共享
	KeyPrefixApp string = "app:"

	// KeyPrefixUser 用户级状态前缀
	// 同一用户的所有会话共享
	KeyPrefixUser string = "user:"

	// KeyPrefixTemp 临时状态前缀
	// 仅当前调用有效，调用结束后丢弃
	KeyPrefixTemp string = "temp:"

	// KeyPrefixSession 会话级状态前缀
	// 当前会话有效
	KeyPrefixSession string = "session:"
)

// 错误定义
var (
	ErrStateKeyNotExist = errors.New("state key does not exist")
	ErrSessionNotFound  = errors.New("session not found")
	ErrInvalidStateKey  = errors.New("invalid state key")
)

// Service 定义 Session 服务接口
type Service interface {
	// Create 创建新会话
	Create(ctx context.Context, req *CreateRequest) (Session, error)

	// Get 获取会话
	Get(ctx context.Context, req *GetRequest) (Session, error)

	// Update 更新会话
	Update(ctx context.Context, req *UpdateRequest) error

	// Delete 删除会话
	Delete(ctx context.Context, sessionID string) error

	// List 列出会话
	List(ctx context.Context, req *ListRequest) ([]*Session, error)

	// AppendEvent 添加事件
	AppendEvent(ctx context.Context, sessionID string, event *Event) error

	// GetEvents 获取事件列表
	GetEvents(ctx context.Context, sessionID string, filter *EventFilter) ([]Event, error)

	// UpdateState 更新状态
	UpdateState(ctx context.Context, sessionID string, delta map[string]interface{}) error
}

// CreateRequest 创建会话请求
type CreateRequest struct {
	AppName  string
	UserID   string
	AgentID  string
	Metadata map[string]interface{}
}

// GetRequest 获取会话请求
type GetRequest struct {
	AppName   string
	UserID    string
	SessionID string
}

// UpdateRequest 更新会话请求
type UpdateRequest struct {
	SessionID string
	Metadata  map[string]interface{}
}

// ListRequest 列出会话请求
type ListRequest struct {
	AppName string
	UserID  string
	Limit   int
	Offset  int
}

// EventFilter 事件过滤器
type EventFilter struct {
	AgentID   string
	Branch    string
	Author    string
	StartTime *time.Time
	EndTime   *time.Time
	Limit     int
	Offset    int
}

// ListOptions 列表选项（用于数据库实现）
type ListOptions struct {
	AppName string
	Limit   int
	Offset  int
}

// EventOptions 事件查询选项（用于数据库实现）
type EventOptions struct {
	InvocationID string
	Branch       string
	Limit        int
	Offset       int
}

// SessionData Session 接口的具体实现（用于数据库服务）
type SessionData struct {
	ID             string
	AppName        string
	UserID         string
	AgentID        string
	CreatedAt      time.Time
	LastUpdateTime time.Time
	Metadata       map[string]interface{}
}

// NewEvent 创建新事件
func NewEvent(invocationID string) *Event {
	return &Event{
		ID:           generateEventID(),
		InvocationID: invocationID,
		Timestamp:    time.Now(),
		Actions: EventActions{
			StateDelta:    make(map[string]interface{}),
			ArtifactDelta: make(map[string]int64),
			CustomActions: make(map[string]interface{}),
		},
		Metadata: make(map[string]interface{}),
	}
}

// generateEventID 生成事件 ID
func generateEventID() string {
	// 实现 ID 生成逻辑
	return "evt_" + time.Now().Format("20060102150405")
}

// IsAppKey 判断是否为应用级 key
func IsAppKey(key string) bool {
	return len(key) > len(KeyPrefixApp) && key[:len(KeyPrefixApp)] == KeyPrefixApp
}

// IsUserKey 判断是否为用户级 key
func IsUserKey(key string) bool {
	return len(key) > len(KeyPrefixUser) && key[:len(KeyPrefixUser)] == KeyPrefixUser
}

// IsTempKey 判断是否为临时 key
func IsTempKey(key string) bool {
	return len(key) > len(KeyPrefixTemp) && key[:len(KeyPrefixTemp)] == KeyPrefixTemp
}

// IsSessionKey 判断是否为会话级 key
func IsSessionKey(key string) bool {
	return len(key) > len(KeyPrefixSession) && key[:len(KeyPrefixSession)] == KeyPrefixSession
}
