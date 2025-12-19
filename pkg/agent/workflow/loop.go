package workflow

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/astercloud/aster/pkg/session"
	"github.com/astercloud/aster/pkg/stream"
)

// LoopAgent 循环执行子 Agent 直到满足终止条件
// 参考 Google ADK-Go 的 LoopAgent 设计
//
// 使用场景:
// - 代码迭代优化直到满足质量要求
// - 多轮对话直到用户满意
// - 任务重试直到成功或达到最大次数
type LoopAgent struct {
	name          string
	subAgents     []Agent
	maxIterations uint
	shouldStop    StopCondition
}

// StopCondition 停止条件函数
// 返回 true 表示应该停止循环
type StopCondition func(event *session.Event) bool

// LoopConfig LoopAgent 配置
type LoopConfig struct {
	// Name Agent 名称
	Name string

	// SubAgents 子 Agent 列表（按顺序执行）
	SubAgents []Agent

	// MaxIterations 最大迭代次数（0 表示无限制，需要依赖 StopCondition）
	MaxIterations uint

	// StopCondition 自定义停止条件（可选）
	// 如果未提供，默认检查 event.Actions.Escalate
	StopCondition StopCondition
}

// NewLoopAgent 创建循环 Agent
func NewLoopAgent(cfg LoopConfig) (*LoopAgent, error) {
	if cfg.Name == "" {
		return nil, errors.New("agent name is required")
	}

	if len(cfg.SubAgents) == 0 {
		return nil, errors.New("at least one sub-agent is required")
	}

	if cfg.MaxIterations == 0 && cfg.StopCondition == nil {
		return nil, errors.New("either MaxIterations or StopCondition must be specified")
	}

	// 默认停止条件：检查 Escalate 标志
	stopCondition := cfg.StopCondition
	if stopCondition == nil {
		stopCondition = func(event *session.Event) bool {
			return event != nil && event.Actions.Escalate
		}
	}

	return &LoopAgent{
		name:          cfg.Name,
		subAgents:     cfg.SubAgents,
		maxIterations: cfg.MaxIterations,
		shouldStop:    stopCondition,
	}, nil
}

// Name 返回 Agent 名称
func (a *LoopAgent) Name() string {
	return a.name
}

// Execute 循环执行子 Agent
func (a *LoopAgent) Execute(ctx context.Context, message string) *stream.Reader[*session.Event] {
	reader, writer := stream.Pipe[*session.Event](10)

	go func() {
		defer writer.Close()

		iteration := uint(0)

		for {
			// 检查最大迭代次数
			if a.maxIterations > 0 {
				if iteration >= a.maxIterations {
					return
				}
				iteration++
			}

			// 顺序执行所有子 Agent
			shouldExit := false
			for i, subAgent := range a.subAgents {
				branch := fmt.Sprintf("%s.%s.iter%d", a.name, subAgent.Name(), iteration)

				subReader := subAgent.Execute(ctx, message)
				for {
					event, err := subReader.Recv()
					if err != nil {
						if errors.Is(err, io.EOF) {
							break
						}
						// 传递错误
						writer.Send(nil, err)
						return
					}

					// 丰富事件信息
					enrichedEvent := a.enrichEvent(event, branch, iteration, i)

					// 传递事件
					if writer.Send(enrichedEvent, nil) {
						return // 客户端取消
					}

					// 检查停止条件
					if a.shouldStop(enrichedEvent) {
						shouldExit = true
						break
					}
				}

				if shouldExit {
					return
				}

				// 检查上下文取消
				if ctx.Err() != nil {
					writer.Send(nil, ctx.Err())
					return
				}
			}

			// 如果没有设置最大迭代次数且没有触发停止条件，继续循环
			// 否则在上面的检查中会退出
		}
	}()

	return reader
}

// enrichEvent 丰富事件信息
func (a *LoopAgent) enrichEvent(event *session.Event, branch string, iteration uint, index int) *session.Event {
	if event == nil {
		return nil
	}

	// 更新 Branch 信息
	event.Branch = branch

	// 添加循环执行的元数据
	if event.Metadata == nil {
		event.Metadata = make(map[string]any)
	}
	event.Metadata["loop_iteration"] = iteration
	event.Metadata["loop_agent"] = a.name
	event.Metadata["sub_agent_index"] = index

	return event
}

// SubAgents 返回所有子 Agent
func (a *LoopAgent) SubAgents() []Agent {
	return append([]Agent{}, a.subAgents...)
}

// CurrentIteration 返回当前迭代次数（仅用于监控）
func (a *LoopAgent) MaxIterations() uint {
	return a.maxIterations
}
