package router

import (
	"context"
	"errors"
	"fmt"

	"github.com/astercloud/aster/pkg/types"
)

// Priority 模型路由优先级/偏好
// 目前只是简单的枚举，后续可以扩展为更复杂的策略。
type Priority string

const (
	// PriorityLatency 优先低延迟
	PriorityLatency Priority = "latency"
	// PriorityCost 优先低成本
	PriorityCost Priority = "cost"
	// PriorityQuality 优先高质量
	PriorityQuality Priority = "quality"
)

// RouteIntent 描述一次调用的“意图”，用于帮助 Router 选择模型。
// 设计尽量保持简单，只做轻量级路由，不做复杂编排。
type RouteIntent struct {
	// Task 任务类型，比如 "chat"、"summarize"、"code"、"analysis" 等。
	Task string `json:"task,omitempty"`
	// Priority 路由偏好：延迟/成本/质量。
	Priority Priority `json:"priority,omitempty"`
	// TemplateID 可选，对应当前 Agent 使用的模板 ID。
	TemplateID string `json:"template_id,omitempty"`
	// Metadata 预留扩展字段，比如调用方、业务场景等。
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Router 抽象的模型路由器。
// 输入 RouteIntent，输出具体的 ModelConfig。
type Router interface {
	SelectModel(ctx context.Context, intent *RouteIntent) (*types.ModelConfig, error)
}

// StaticRouteEntry 静态路由条目——最简单的实现方式。
// 匹配逻辑很保守：只根据 Task + Priority 精确匹配。
type StaticRouteEntry struct {
	Task     string             `json:"task,omitempty"`
	Priority Priority           `json:"priority,omitempty"`
	Model    *types.ModelConfig `json:"model"`
}

// StaticRouter 一个内存中的静态路由表实现。
// 用来满足当前“简单可控”的需求，后续可替换为更高级的实现。
type StaticRouter struct {
	defaultModel *types.ModelConfig
	routes       []StaticRouteEntry
}

// NewStaticRouter 创建一个静态路由器。
// defaultModel 为兜底模型，当没有任何条目匹配时使用。
func NewStaticRouter(defaultModel *types.ModelConfig, routes []StaticRouteEntry) *StaticRouter {
	return &StaticRouter{
		defaultModel: defaultModel,
		routes:       routes,
	}
}

// SelectModel 根据 RouteIntent 选择模型。
// 匹配规则：
//  1. 先找 Task + Priority 都匹配的条目。
//  2. 如果找不到，再找 Task 匹配但 Priority 为空的条目。
//  3. 否则返回 defaultModel（如果存在）。
func (r *StaticRouter) SelectModel(_ context.Context, intent *RouteIntent) (*types.ModelConfig, error) {
	if intent == nil {
		if r.defaultModel != nil {
			return r.defaultModel, nil
		}
		return nil, errors.New("route intent is nil and no default model configured")
	}

	// 1. Task + Priority 精确匹配
	for _, entry := range r.routes {
		if entry.Model == nil {
			continue
		}
		if entry.Task == intent.Task && entry.Priority == intent.Priority {
			return entry.Model, nil
		}
	}

	// 2. 只根据 Task 匹配（Priority 为空）
	for _, entry := range r.routes {
		if entry.Model == nil {
			continue
		}
		if entry.Task == intent.Task && entry.Priority == "" {
			return entry.Model, nil
		}
	}

	// 3. 兜底
	if r.defaultModel != nil {
		return r.defaultModel, nil
	}

	return nil, fmt.Errorf("no route matched for task=%q priority=%q and no default model configured", intent.Task, intent.Priority)
}
