package sandbox

import (
	"errors"
	"fmt"
	"time"

	"github.com/astercloud/aster/pkg/types"
)

// Factory 沙箱工厂
type Factory struct {
	// 可以添加配置或依赖
}

// NewFactory 创建沙箱工厂
func NewFactory() *Factory {
	return &Factory{}
}

// Create 根据配置创建沙箱
func (f *Factory) Create(config *types.SandboxConfig) (Sandbox, error) {
	if config == nil {
		// 默认使用本地沙箱
		config = &types.SandboxConfig{
			Kind:    types.SandboxKindLocal,
			WorkDir: ".",
		}
	}

	switch config.Kind {
	case types.SandboxKindLocal:
		return NewLocalSandbox(&LocalSandboxConfig{
			WorkDir:         config.WorkDir,
			EnforceBoundary: config.EnforceBoundary,
			AllowPaths:      config.AllowPaths,
			WatchFiles:      config.WatchFiles,
			Settings:        config.Settings,
		})

	case types.SandboxKindDocker:
		return nil, errors.New("docker sandbox not implemented yet")

	case types.SandboxKindK8s:
		return nil, errors.New("k8s sandbox not implemented yet")

	case types.SandboxKindAliyun:
		// 阿里云沙箱需要使用 cloud.NewAliyunSandbox() 直接创建
		return nil, errors.New("aliyun sandbox: use cloud.NewAliyunSandbox() directly")

	case types.SandboxKindVolcengine:
		// 火山引擎沙箱需要使用 cloud.NewVolcengineSandbox() 直接创建
		return nil, errors.New("volcengine sandbox: use cloud.NewVolcengineSandbox() directly")

	case types.SandboxKindRemote:
		// 通用远程沙箱
		if config.Extra == nil {
			return nil, errors.New("remote sandbox requires extra configuration")
		}

		baseURL, _ := config.Extra["base_url"].(string)
		apiKey, _ := config.Extra["api_key"].(string)
		apiSecret, _ := config.Extra["api_secret"].(string)

		timeout := 30 * time.Second
		if t, ok := config.Extra["timeout"].(time.Duration); ok {
			timeout = t
		}

		return NewRemoteSandbox(&RemoteSandboxConfig{
			BaseURL:   baseURL,
			APIKey:    apiKey,
			APISecret: apiSecret,
			WorkDir:   config.WorkDir,
			Timeout:   timeout,
		})

	case types.SandboxKindMock:
		return NewMockSandbox(), nil

	default:
		return nil, fmt.Errorf("unknown sandbox kind: %s", config.Kind)
	}
}
