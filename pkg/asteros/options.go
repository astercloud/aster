package asteros

import (
	"github.com/astercloud/aster/pkg/core"
)

// Options AsterOS 配置选项
type Options struct {
	// 基本配置
	Name string // AsterOS 实例名称
	Port int    // HTTP 服务端口，默认 8080

	// 核心组件
	Pool *core.Pool // Pool 实例（必需）

	// 自动发现
	AutoDiscover bool // 是否自动发现和注册资源，默认 true

	// API 配置
	APIPrefix  string // API 路径前缀，默认 ""
	EnableCORS bool   // 是否启用 CORS，默认 true

	// 安全配置
	EnableAuth bool   // 是否启用认证，默认 false
	APIKey     string // API Key（如果启用认证）

	// 监控配置
	EnableMetrics bool // 是否启用 Prometheus 指标，默认 true
	EnableHealth  bool // 是否启用健康检查，默认 true

	// 日志配置
	EnableLogging bool   // 是否启用请求日志，默认 true
	LogLevel      string // 日志级别：debug, info, warn, error，默认 info
}

// DefaultOptions 返回默认配置
func DefaultOptions() *Options {
	return &Options{
		Name:          "AsterOS",
		Port:          8080,
		AutoDiscover:  true,
		APIPrefix:     "",
		EnableCORS:    true,
		EnableAuth:    false,
		EnableMetrics: true,
		EnableHealth:  true,
		EnableLogging: true,
		LogLevel:      "info",
	}
}

// Validate 验证配置
func (o *Options) Validate() error {
	if o.Pool == nil {
		return ErrPoolRequired
	}
	if o.Port <= 0 || o.Port > 65535 {
		return ErrInvalidPort
	}
	return nil
}
