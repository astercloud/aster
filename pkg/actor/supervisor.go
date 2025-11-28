package actor

import (
	"time"
)

// Directive 监督指令
type Directive int

const (
	// DirectiveResume 恢复 Actor，继续处理消息
	DirectiveResume Directive = iota
	// DirectiveRestart 重启 Actor
	DirectiveRestart
	// DirectiveStop 停止 Actor
	DirectiveStop
	// DirectiveEscalate 上报给父 Actor 处理
	DirectiveEscalate
)

// String 返回指令名称
func (d Directive) String() string {
	switch d {
	case DirectiveResume:
		return "Resume"
	case DirectiveRestart:
		return "Restart"
	case DirectiveStop:
		return "Stop"
	case DirectiveEscalate:
		return "Escalate"
	default:
		return "Unknown"
	}
}

// SupervisorStrategy 监督策略接口
type SupervisorStrategy interface {
	// HandleFailure 处理 Actor 失败
	// 返回应该采取的指令
	HandleFailure(system *System, child *PID, msg Message, err interface{}) Directive
}

// ============== 内置监督策略 ==============

// OneForOneStrategy 一对一策略
// 只重启失败的 Actor，不影响其他子 Actor
type OneForOneStrategy struct {
	MaxRestarts   int           // 最大重启次数
	WithinDuration time.Duration // 时间窗口
	Decider       Decider       // 决策函数

	// 内部状态
	restartWindow []time.Time
}

// Decider 决策函数类型
type Decider func(err interface{}) Directive

// NewOneForOneStrategy 创建一对一策略
func NewOneForOneStrategy(maxRestarts int, within time.Duration, decider Decider) *OneForOneStrategy {
	if decider == nil {
		decider = DefaultDecider
	}
	return &OneForOneStrategy{
		MaxRestarts:    maxRestarts,
		WithinDuration: within,
		Decider:        decider,
		restartWindow:  make([]time.Time, 0),
	}
}

// HandleFailure 实现 SupervisorStrategy
func (s *OneForOneStrategy) HandleFailure(system *System, child *PID, msg Message, err interface{}) Directive {
	directive := s.Decider(err)

	if directive == DirectiveRestart {
		// 检查重启次数限制
		now := time.Now()
		cutoff := now.Add(-s.WithinDuration)

		// 清理过期的重启记录
		validRestarts := make([]time.Time, 0)
		for _, t := range s.restartWindow {
			if t.After(cutoff) {
				validRestarts = append(validRestarts, t)
			}
		}
		s.restartWindow = validRestarts

		// 检查是否超过限制
		if len(s.restartWindow) >= s.MaxRestarts {
			return DirectiveStop
		}

		// 记录本次重启
		s.restartWindow = append(s.restartWindow, now)
	}

	return directive
}

// AllForOneStrategy 全部重启策略
// 当一个子 Actor 失败时，重启所有子 Actor
type AllForOneStrategy struct {
	MaxRestarts    int
	WithinDuration time.Duration
	Decider        Decider

	restartWindow []time.Time
}

// NewAllForOneStrategy 创建全部重启策略
func NewAllForOneStrategy(maxRestarts int, within time.Duration, decider Decider) *AllForOneStrategy {
	if decider == nil {
		decider = DefaultDecider
	}
	return &AllForOneStrategy{
		MaxRestarts:    maxRestarts,
		WithinDuration: within,
		Decider:        decider,
		restartWindow:  make([]time.Time, 0),
	}
}

// HandleFailure 实现 SupervisorStrategy
func (s *AllForOneStrategy) HandleFailure(system *System, child *PID, msg Message, err interface{}) Directive {
	directive := s.Decider(err)

	if directive == DirectiveRestart {
		now := time.Now()
		cutoff := now.Add(-s.WithinDuration)

		validRestarts := make([]time.Time, 0)
		for _, t := range s.restartWindow {
			if t.After(cutoff) {
				validRestarts = append(validRestarts, t)
			}
		}
		s.restartWindow = validRestarts

		if len(s.restartWindow) >= s.MaxRestarts {
			return DirectiveStop
		}

		s.restartWindow = append(s.restartWindow, now)

		// 这里不直接重启所有子 Actor，而是返回指令
		// 实际的全部重启逻辑需要在 System 层面处理
	}

	return directive
}

// ExponentialBackoffStrategy 指数退避策略
// 重启间隔逐渐增加
type ExponentialBackoffStrategy struct {
	InitialDelay time.Duration
	MaxDelay     time.Duration
	MaxRestarts  int
	Decider      Decider

	currentDelay time.Duration
	restartCount int
}

// NewExponentialBackoffStrategy 创建指数退避策略
func NewExponentialBackoffStrategy(initialDelay, maxDelay time.Duration, maxRestarts int, decider Decider) *ExponentialBackoffStrategy {
	if decider == nil {
		decider = DefaultDecider
	}
	return &ExponentialBackoffStrategy{
		InitialDelay: initialDelay,
		MaxDelay:     maxDelay,
		MaxRestarts:  maxRestarts,
		Decider:      decider,
		currentDelay: initialDelay,
	}
}

// HandleFailure 实现 SupervisorStrategy
func (s *ExponentialBackoffStrategy) HandleFailure(system *System, child *PID, msg Message, err interface{}) Directive {
	directive := s.Decider(err)

	if directive == DirectiveRestart {
		if s.restartCount >= s.MaxRestarts {
			return DirectiveStop
		}

		// 等待退避时间
		time.Sleep(s.currentDelay)

		// 增加下次延迟
		s.currentDelay *= 2
		if s.currentDelay > s.MaxDelay {
			s.currentDelay = s.MaxDelay
		}

		s.restartCount++
	}

	return directive
}

// Reset 重置退避状态
func (s *ExponentialBackoffStrategy) Reset() {
	s.currentDelay = s.InitialDelay
	s.restartCount = 0
}

// ============== 默认策略和决策器 ==============

// DefaultDecider 默认决策器
// 对所有错误采取重启策略
func DefaultDecider(err interface{}) Directive {
	return DirectiveRestart
}

// StoppingDecider 停止决策器
// 对所有错误采取停止策略
func StoppingDecider(err interface{}) Directive {
	return DirectiveStop
}

// EscalatingDecider 上报决策器
// 对所有错误采取上报策略
func EscalatingDecider(err interface{}) Directive {
	return DirectiveEscalate
}

// ResumingDecider 恢复决策器
// 对所有错误采取恢复策略（忽略错误继续运行）
func ResumingDecider(err interface{}) Directive {
	return DirectiveResume
}

// DefaultSupervisorStrategy 默认监督策略
// 允许 3 次重启在 1 分钟内
func DefaultSupervisorStrategy() SupervisorStrategy {
	return NewOneForOneStrategy(3, time.Minute, DefaultDecider)
}

// StrictSupervisorStrategy 严格监督策略
// 任何失败都停止 Actor
func StrictSupervisorStrategy() SupervisorStrategy {
	return NewOneForOneStrategy(0, time.Second, StoppingDecider)
}

// LenientSupervisorStrategy 宽松监督策略
// 允许更多重启
func LenientSupervisorStrategy() SupervisorStrategy {
	return NewOneForOneStrategy(10, 5*time.Minute, DefaultDecider)
}

// ============== 组合策略 ==============

// CompositeStrategy 组合策略
// 根据错误类型选择不同的策略
type CompositeStrategy struct {
	strategies map[string]SupervisorStrategy
	fallback   SupervisorStrategy
}

// NewCompositeStrategy 创建组合策略
func NewCompositeStrategy(fallback SupervisorStrategy) *CompositeStrategy {
	if fallback == nil {
		fallback = DefaultSupervisorStrategy()
	}
	return &CompositeStrategy{
		strategies: make(map[string]SupervisorStrategy),
		fallback:   fallback,
	}
}

// RegisterStrategy 注册特定错误类型的策略
func (s *CompositeStrategy) RegisterStrategy(errType string, strategy SupervisorStrategy) {
	s.strategies[errType] = strategy
}

// HandleFailure 实现 SupervisorStrategy
func (s *CompositeStrategy) HandleFailure(system *System, child *PID, msg Message, err interface{}) Directive {
	// 尝试匹配错误类型
	if e, ok := err.(error); ok {
		errType := e.Error()
		if strategy, found := s.strategies[errType]; found {
			return strategy.HandleFailure(system, child, msg, err)
		}
	}

	// 使用回退策略
	return s.fallback.HandleFailure(system, child, msg, err)
}

// ============== 监督树辅助 ==============

// SupervisorConfig 监督配置
type SupervisorConfig struct {
	Strategy SupervisorStrategy
	Children []ChildSpec
}

// ChildSpec 子 Actor 规格
type ChildSpec struct {
	Name    string
	Factory func() Actor
	Props   *Props
}

// SupervisorActor 监督者 Actor
// 用于构建监督树
type SupervisorActor struct {
	config   *SupervisorConfig
	children map[string]*PID
}

// NewSupervisorActor 创建监督者 Actor
func NewSupervisorActor(config *SupervisorConfig) *SupervisorActor {
	return &SupervisorActor{
		config:   config,
		children: make(map[string]*PID),
	}
}

// Receive 处理消息
func (s *SupervisorActor) Receive(ctx *Context, msg Message) {
	switch m := msg.(type) {
	case *Started:
		// 启动所有子 Actor
		for _, spec := range s.config.Children {
			child := spec.Factory()
			props := spec.Props
			if props == nil {
				props = DefaultProps(spec.Name)
			}
			props.SupervisorStrategy = s.config.Strategy
			pid := ctx.SpawnWithProps(child, props)
			s.children[spec.Name] = pid
		}

	case *Terminated:
		// 子 Actor 终止，根据策略处理
		delete(s.children, m.Who.ID)

	case *Stopping:
		// 停止所有子 Actor
		for _, pid := range s.children {
			ctx.Stop(pid)
		}
	}
}

// GetChild 获取子 Actor
func (s *SupervisorActor) GetChild(name string) *PID {
	return s.children[name]
}
