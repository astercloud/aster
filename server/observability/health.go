package observability

import (
	"context"
	"maps"
	"sync"
	"time"
)

// HealthStatus 健康状态
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// HealthCheck 健康检查接口
type HealthCheck interface {
	Name() string
	Check(ctx context.Context) error
}

// HealthInfo 健康信息
type HealthInfo struct {
	Status       HealthStatus           `json:"status"`
	Version      string                 `json:"version"`
	Uptime       time.Duration          `json:"uptime"`
	Timestamp    time.Time              `json:"timestamp"`
	Checks       map[string]CheckResult `json:"checks,omitempty"`
	Dependencies map[string]bool        `json:"dependencies,omitempty"`
}

// CheckResult 检查结果
type CheckResult struct {
	Status  HealthStatus `json:"status"`
	Message string       `json:"message,omitempty"`
	Error   string       `json:"error,omitempty"`
	Latency string       `json:"latency,omitempty"`
}

// HealthChecker 健康检查器
type HealthChecker struct {
	mu        sync.RWMutex
	checks    map[string]HealthCheck
	startTime time.Time
	version   string
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(version string) *HealthChecker {
	return &HealthChecker{
		checks:    make(map[string]HealthCheck),
		startTime: time.Now(),
		version:   version,
	}
}

// RegisterCheck 注册健康检查
func (h *HealthChecker) RegisterCheck(check HealthCheck) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checks[check.Name()] = check
}

// Check 执行所有健康检查
func (h *HealthChecker) Check(ctx context.Context) *HealthInfo {
	h.mu.RLock()
	checks := make(map[string]HealthCheck, len(h.checks))
	maps.Copy(checks, h.checks)
	h.mu.RUnlock()

	info := &HealthInfo{
		Status:    HealthStatusHealthy,
		Version:   h.version,
		Uptime:    time.Since(h.startTime),
		Timestamp: time.Now(),
		Checks:    make(map[string]CheckResult),
	}

	// 并发执行所有检查
	var wg sync.WaitGroup
	resultChan := make(chan struct {
		name   string
		result CheckResult
	}, len(checks))

	for name, check := range checks {
		wg.Add(1)
		go func(n string, c HealthCheck) {
			defer wg.Done()

			start := time.Now()
			err := c.Check(ctx)
			latency := time.Since(start)

			result := CheckResult{
				Latency: latency.String(),
			}

			if err != nil {
				result.Status = HealthStatusUnhealthy
				result.Error = err.Error()
			} else {
				result.Status = HealthStatusHealthy
				result.Message = "OK"
			}

			resultChan <- struct {
				name   string
				result CheckResult
			}{n, result}
		}(name, check)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集结果
	for r := range resultChan {
		info.Checks[r.name] = r.result

		// 如果有任何检查失败，整体状态降级
		if r.result.Status == HealthStatusUnhealthy {
			info.Status = HealthStatusDegraded
		}
	}

	return info
}

// Uptime 返回运行时间
func (h *HealthChecker) Uptime() time.Duration {
	return time.Since(h.startTime)
}

// 内置健康检查实现

// StoreHealthCheck Store 健康检查
type StoreHealthCheck struct {
	name      string
	checkFunc func(context.Context) error
}

// NewStoreHealthCheck 创建 Store 健康检查
func NewStoreHealthCheck(name string, checkFunc func(context.Context) error) *StoreHealthCheck {
	return &StoreHealthCheck{
		name:      name,
		checkFunc: checkFunc,
	}
}

func (c *StoreHealthCheck) Name() string {
	return c.name
}

func (c *StoreHealthCheck) Check(ctx context.Context) error {
	if c.checkFunc != nil {
		return c.checkFunc(ctx)
	}
	return nil
}

// SimpleHealthCheck 简单健康检查
type SimpleHealthCheck struct {
	name    string
	checker func() error
}

// NewSimpleHealthCheck 创建简单健康检查
func NewSimpleHealthCheck(name string, checker func() error) *SimpleHealthCheck {
	return &SimpleHealthCheck{
		name:    name,
		checker: checker,
	}
}

func (c *SimpleHealthCheck) Name() string {
	return c.name
}

func (c *SimpleHealthCheck) Check(ctx context.Context) error {
	if c.checker != nil {
		return c.checker()
	}
	return nil
}
