package actor

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

// System Actor 系统
// 管理所有 Actor 的生命周期、消息路由和监督
type System struct {
	// 基本信息
	name string

	// Actor 注册表
	actors   map[string]*actorCell
	actorsMu sync.RWMutex

	// 全局邮箱（用于路由消息）
	mailbox chan envelope

	// 死信队列（无法投递的消息）
	deadLetters chan envelope

	// 生命周期控制
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	isRunning atomic.Bool

	// 配置
	config *SystemConfig

	// 统计信息
	stats *SystemStats
}

// SystemConfig 系统配置
type SystemConfig struct {
	// MailboxSize 全局邮箱大小
	MailboxSize int
	// DeadLetterSize 死信队列大小
	DeadLetterSize int
	// DefaultActorMailboxSize 默认 Actor 邮箱大小
	DefaultActorMailboxSize int
	// EnableDeadLetterLogging 是否记录死信
	EnableDeadLetterLogging bool
	// PanicHandler panic 处理函数
	PanicHandler func(actor *PID, msg Message, err interface{})
}

// DefaultSystemConfig 默认系统配置
func DefaultSystemConfig() *SystemConfig {
	return &SystemConfig{
		MailboxSize:             10000,
		DeadLetterSize:          1000,
		DefaultActorMailboxSize: 100,
		EnableDeadLetterLogging: true,
		PanicHandler:            defaultPanicHandler,
	}
}

func defaultPanicHandler(actor *PID, msg Message, err interface{}) {
	log.Printf("[Actor] PANIC in %s processing %s: %v\n%s",
		actor.ID, msg.Kind(), err, debug.Stack())
}

// SystemStats 系统统计
type SystemStats struct {
	TotalActors     int64
	TotalMessages   int64
	DeadLetters     int64
	ProcessedMsgs   int64
	AverageLatency  time.Duration
	StartTime       time.Time
}

// actorCell Actor 单元，包含 Actor 及其运行时状态
type actorCell struct {
	pid      *PID
	actor    Actor
	mailbox  chan envelope
	parent   *PID
	children map[string]*PID
	watchers map[string]*PID // 监控此 Actor 的其他 Actor
	watching map[string]*PID // 此 Actor 监控的其他 Actor

	// 状态
	state    actorState
	stateMu  sync.RWMutex
	restarts int

	// 监督策略
	supervisor SupervisorStrategy

	// 上下文
	ctx    context.Context
	cancel context.CancelFunc
}

type actorState int

const (
	actorStateIdle actorState = iota
	actorStateRunning
	actorStateStopping
	actorStateStopped
	actorStateRestarting
)

// envelope 消息信封
type envelope struct {
	target   *PID
	sender   *PID
	message  Message
	sentAt   time.Time
	response chan Message // 用于 Request/Response
}

// NewSystem 创建新的 Actor 系统
func NewSystem(name string) *System {
	return NewSystemWithConfig(name, DefaultSystemConfig())
}

// NewSystemWithConfig 使用配置创建 Actor 系统
func NewSystemWithConfig(name string, config *SystemConfig) *System {
	if config == nil {
		config = DefaultSystemConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	s := &System{
		name:        name,
		actors:      make(map[string]*actorCell),
		mailbox:     make(chan envelope, config.MailboxSize),
		deadLetters: make(chan envelope, config.DeadLetterSize),
		ctx:         ctx,
		cancel:      cancel,
		config:      config,
		stats: &SystemStats{
			StartTime: time.Now(),
		},
	}

	s.isRunning.Store(true)

	// 启动消息分发器
	s.wg.Add(1)
	go s.dispatcher()

	// 启动死信处理器
	if config.EnableDeadLetterLogging {
		s.wg.Add(1)
		go s.deadLetterHandler()
	}

	log.Printf("[ActorSystem] %s started", name)
	return s
}

// Name 返回系统名称
func (s *System) Name() string {
	return s.name
}

// Spawn 创建 Actor
func (s *System) Spawn(actor Actor, name string) *PID {
	return s.spawn(actor, name, nil)
}

// SpawnWithProps 使用属性创建 Actor
func (s *System) SpawnWithProps(actor Actor, props *Props) *PID {
	return s.spawnWithProps(actor, props, nil)
}

// spawn 内部创建方法
func (s *System) spawn(actor Actor, name string, parent *PID) *PID {
	props := DefaultProps(name)
	return s.spawnWithProps(actor, props, parent)
}

// spawnWithProps 使用属性创建
func (s *System) spawnWithProps(actor Actor, props *Props, parent *PID) *PID {
	s.actorsMu.Lock()
	defer s.actorsMu.Unlock()

	// 检查名称是否已存在
	if _, exists := s.actors[props.Name]; exists {
		log.Printf("[ActorSystem] Actor %s already exists, returning existing PID", props.Name)
		return s.actors[props.Name].pid
	}

	// 创建 PID
	pid := &PID{
		ID:     props.Name,
		system: s,
	}

	// 创建上下文
	ctx, cancel := context.WithCancel(s.ctx)

	// 确定邮箱大小
	mailboxSize := props.MailboxSize
	if mailboxSize <= 0 {
		mailboxSize = s.config.DefaultActorMailboxSize
	}

	// 创建 Actor 单元
	cell := &actorCell{
		pid:        pid,
		actor:      actor,
		mailbox:    make(chan envelope, mailboxSize),
		parent:     parent,
		children:   make(map[string]*PID),
		watchers:   make(map[string]*PID),
		watching:   make(map[string]*PID),
		state:      actorStateIdle,
		supervisor: props.SupervisorStrategy,
		ctx:        ctx,
		cancel:     cancel,
	}

	// 注册
	s.actors[props.Name] = cell
	atomic.AddInt64(&s.stats.TotalActors, 1)

	// 如果有父 Actor，注册为子 Actor
	if parent != nil {
		if parentCell, ok := s.actors[parent.ID]; ok {
			parentCell.children[props.Name] = pid
		}
	}

	// 启动 Actor 消息循环
	s.wg.Add(1)
	go s.actorLoop(cell)

	// 发送 Started 消息
	s.SendWithSender(pid, &Started{}, nil)

	log.Printf("[ActorSystem] Spawned actor: %s (parent: %v)", props.Name, parent)
	return pid
}

// Send 发送消息（无发送者）
func (s *System) Send(target *PID, msg Message) {
	s.SendWithSender(target, msg, nil)
}

// SendWithSender 发送消息（带发送者）
func (s *System) SendWithSender(target *PID, msg Message, sender *PID) {
	if !s.isRunning.Load() {
		return
	}

	env := envelope{
		target:  target,
		sender:  sender,
		message: msg,
		sentAt:  time.Now(),
	}

	select {
	case s.mailbox <- env:
		atomic.AddInt64(&s.stats.TotalMessages, 1)
	default:
		// 邮箱满，发送到死信队列
		select {
		case s.deadLetters <- env:
			atomic.AddInt64(&s.stats.DeadLetters, 1)
		default:
			log.Printf("[ActorSystem] Both mailbox and dead letter queue full, message dropped: %s -> %s",
				msg.Kind(), target.ID)
		}
	}
}

// Request 同步请求（等待响应）
func (s *System) Request(target *PID, msg Message, timeout time.Duration) (Message, error) {
	if !s.isRunning.Load() {
		return nil, fmt.Errorf("actor system is not running")
	}

	responseChan := make(chan Message, 1)
	env := envelope{
		target:   target,
		sender:   nil,
		message:  msg,
		sentAt:   time.Now(),
		response: responseChan,
	}

	select {
	case s.mailbox <- env:
		atomic.AddInt64(&s.stats.TotalMessages, 1)
	case <-time.After(timeout):
		return nil, &ResponseTimeout{Target: target, Timeout: timeout}
	}

	select {
	case resp := <-responseChan:
		return resp, nil
	case <-time.After(timeout):
		return nil, &ResponseTimeout{Target: target, Timeout: timeout}
	}
}

// Stop 停止 Actor
func (s *System) Stop(pid *PID) {
	s.actorsMu.RLock()
	cell, exists := s.actors[pid.ID]
	s.actorsMu.RUnlock()

	if !exists {
		return
	}

	// 发送 PoisonPill
	s.Send(pid, &PoisonPill{})

	// 等待停止完成
	cell.stateMu.Lock()
	cell.state = actorStateStopping
	cell.stateMu.Unlock()
}

// StopGracefully 优雅停止 Actor（等待处理完当前消息）
func (s *System) StopGracefully(pid *PID, timeout time.Duration) error {
	s.Stop(pid)

	// 等待状态变为 stopped
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		s.actorsMu.RLock()
		cell, exists := s.actors[pid.ID]
		s.actorsMu.RUnlock()

		if !exists {
			return nil
		}

		cell.stateMu.RLock()
		state := cell.state
		cell.stateMu.RUnlock()

		if state == actorStateStopped {
			return nil
		}

		time.Sleep(10 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for actor %s to stop", pid.ID)
}

// Shutdown 关闭整个 Actor 系统
func (s *System) Shutdown() {
	s.ShutdownWithTimeout(30 * time.Second)
}

// ShutdownWithTimeout 带超时的关闭
func (s *System) ShutdownWithTimeout(timeout time.Duration) {
	log.Printf("[ActorSystem] %s shutting down...", s.name)

	s.isRunning.Store(false)

	// 停止所有 Actor
	s.actorsMu.RLock()
	pids := make([]*PID, 0, len(s.actors))
	for _, cell := range s.actors {
		pids = append(pids, cell.pid)
	}
	s.actorsMu.RUnlock()

	for _, pid := range pids {
		s.Stop(pid)
	}

	// 取消上下文
	s.cancel()

	// 等待所有 goroutine 完成
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Printf("[ActorSystem] %s shutdown complete", s.name)
	case <-time.After(timeout):
		log.Printf("[ActorSystem] %s shutdown timeout, forcing exit", s.name)
	}
}

// dispatcher 全局消息分发器
func (s *System) dispatcher() {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		case env := <-s.mailbox:
			s.dispatchMessage(env)
		}
	}
}

// dispatchMessage 分发单条消息
func (s *System) dispatchMessage(env envelope) {
	s.actorsMu.RLock()
	cell, exists := s.actors[env.target.ID]
	s.actorsMu.RUnlock()

	if !exists {
		// Actor 不存在，发送到死信
		select {
		case s.deadLetters <- env:
			atomic.AddInt64(&s.stats.DeadLetters, 1)
		default:
		}
		return
	}

	// 投递到 Actor 邮箱
	select {
	case cell.mailbox <- env:
	default:
		// Actor 邮箱满，背压处理
		log.Printf("[ActorSystem] Actor %s mailbox full, message queued to dead letter", env.target.ID)
		select {
		case s.deadLetters <- env:
			atomic.AddInt64(&s.stats.DeadLetters, 1)
		default:
		}
	}
}

// actorLoop Actor 消息处理循环
func (s *System) actorLoop(cell *actorCell) {
	defer s.wg.Done()
	defer s.cleanupActor(cell)

	cell.stateMu.Lock()
	cell.state = actorStateRunning
	cell.stateMu.Unlock()

	for {
		select {
		case <-cell.ctx.Done():
			return
		case env := <-cell.mailbox:
			s.processMessage(cell, env)

			// 检查是否收到 PoisonPill
			if _, ok := env.message.(*PoisonPill); ok {
				return
			}
		}
	}
}

// processMessage 处理单条消息
func (s *System) processMessage(cell *actorCell, env envelope) {
	// panic 恢复
	defer func() {
		if r := recover(); r != nil {
			if s.config.PanicHandler != nil {
				s.config.PanicHandler(cell.pid, env.message, r)
			}
			// 触发监督策略
			s.handleFailure(cell, env.message, r)
		}
	}()

	// 创建上下文，支持 Request/Response
	ctx := &Context{
		Self:     cell.pid,
		Sender:   env.sender,
		Parent:   cell.parent,
		Children: s.getChildrenPIDs(cell),
		system:   s,
		ctx:      cell.ctx,
		message:  env.message,
	}

	// 如果是 Request 模式，设置响应通道
	if env.response != nil {
		ctx.responseChan = env.response
	}

	// 处理系统消息
	switch msg := env.message.(type) {
	case *PoisonPill:
		// 发送 Stopping 消息
		cell.actor.Receive(ctx, &Stopping{})
		return

	case *Watch:
		cell.watchers[msg.Watcher.ID] = msg.Watcher
		return

	case *Unwatch:
		delete(cell.watchers, msg.Watcher.ID)
		return
	}

	// 处理用户消息
	cell.actor.Receive(ctx, env.message)
	atomic.AddInt64(&s.stats.ProcessedMsgs, 1)
}

// handleFailure 处理 Actor 失败
func (s *System) handleFailure(cell *actorCell, msg Message, err interface{}) {
	supervisor := cell.supervisor
	if supervisor == nil && cell.parent != nil {
		// 使用父 Actor 的监督策略
		s.actorsMu.RLock()
		if parentCell, ok := s.actors[cell.parent.ID]; ok {
			supervisor = parentCell.supervisor
		}
		s.actorsMu.RUnlock()
	}

	if supervisor == nil {
		supervisor = DefaultSupervisorStrategy()
	}

	directive := supervisor.HandleFailure(s, cell.pid, msg, err)
	s.applyDirective(cell, directive)
}

// applyDirective 应用监督指令
func (s *System) applyDirective(cell *actorCell, directive Directive) {
	switch directive {
	case DirectiveResume:
		// 继续运行，不做处理
		log.Printf("[ActorSystem] Actor %s resumed after failure", cell.pid.ID)

	case DirectiveRestart:
		cell.restarts++
		cell.stateMu.Lock()
		cell.state = actorStateRestarting
		cell.stateMu.Unlock()

		// 发送 Restarting 消息
		ctx := &Context{Self: cell.pid, system: s, ctx: cell.ctx}
		cell.actor.Receive(ctx, &Restarting{})

		// 重新发送 Started 消息
		s.Send(cell.pid, &Started{})

		cell.stateMu.Lock()
		cell.state = actorStateRunning
		cell.stateMu.Unlock()

		log.Printf("[ActorSystem] Actor %s restarted (total restarts: %d)", cell.pid.ID, cell.restarts)

	case DirectiveStop:
		s.Stop(cell.pid)

	case DirectiveEscalate:
		if cell.parent != nil {
			// 将失败上报给父 Actor
			s.Send(cell.parent, &Terminated{Who: cell.pid})
		}
		s.Stop(cell.pid)
	}
}

// cleanupActor 清理 Actor
func (s *System) cleanupActor(cell *actorCell) {
	cell.stateMu.Lock()
	cell.state = actorStateStopped
	cell.stateMu.Unlock()

	// 发送 Stopped 消息
	ctx := &Context{Self: cell.pid, system: s, ctx: context.Background()}
	cell.actor.Receive(ctx, &Stopped{})

	// 通知所有监控者
	for _, watcher := range cell.watchers {
		s.Send(watcher, &Terminated{Who: cell.pid})
	}

	// 停止所有子 Actor
	for _, child := range cell.children {
		s.Stop(child)
	}

	// 从注册表中移除
	s.actorsMu.Lock()
	delete(s.actors, cell.pid.ID)
	s.actorsMu.Unlock()

	// 从父 Actor 中移除
	if cell.parent != nil {
		s.actorsMu.RLock()
		if parentCell, ok := s.actors[cell.parent.ID]; ok {
			delete(parentCell.children, cell.pid.ID)
		}
		s.actorsMu.RUnlock()
	}

	// 取消上下文
	cell.cancel()

	atomic.AddInt64(&s.stats.TotalActors, -1)
	log.Printf("[ActorSystem] Actor %s stopped", cell.pid.ID)
}

// getChildrenPIDs 获取子 Actor PID 列表
func (s *System) getChildrenPIDs(cell *actorCell) []*PID {
	pids := make([]*PID, 0, len(cell.children))
	for _, pid := range cell.children {
		pids = append(pids, pid)
	}
	return pids
}

// deadLetterHandler 死信处理器
func (s *System) deadLetterHandler() {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		case env := <-s.deadLetters:
			log.Printf("[DeadLetter] Message %s to %s from %v could not be delivered",
				env.message.Kind(), env.target.ID, env.sender)
		}
	}
}

// Stats 获取统计信息
func (s *System) Stats() *SystemStats {
	return &SystemStats{
		TotalActors:   atomic.LoadInt64(&s.stats.TotalActors),
		TotalMessages: atomic.LoadInt64(&s.stats.TotalMessages),
		DeadLetters:   atomic.LoadInt64(&s.stats.DeadLetters),
		ProcessedMsgs: atomic.LoadInt64(&s.stats.ProcessedMsgs),
		StartTime:     s.stats.StartTime,
	}
}

// GetActor 获取 Actor（用于调试）
func (s *System) GetActor(name string) (*PID, bool) {
	s.actorsMu.RLock()
	defer s.actorsMu.RUnlock()

	if cell, ok := s.actors[name]; ok {
		return cell.pid, true
	}
	return nil, false
}

// ListActors 列出所有 Actor（用于调试）
func (s *System) ListActors() []*PID {
	s.actorsMu.RLock()
	defer s.actorsMu.RUnlock()

	pids := make([]*PID, 0, len(s.actors))
	for _, cell := range s.actors {
		pids = append(pids, cell.pid)
	}
	return pids
}
