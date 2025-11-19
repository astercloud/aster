package asteros

import "errors"

var (
	// 配置错误
	ErrCosmosRequired = errors.New("cosmos is required")
	ErrInvalidPort    = errors.New("invalid port number")

	// 资源错误
	ErrAgentNotFound    = errors.New("agent not found")
	ErrStarsNotFound    = errors.New("stars not found")
	ErrWorkflowNotFound = errors.New("workflow not found")
	ErrResourceExists   = errors.New("resource already exists")

	// Interface 错误
	ErrInterfaceNotFound = errors.New("interface not found")
	ErrInterfaceExists   = errors.New("interface already exists")

	// 运行时错误
	ErrNotRunning     = errors.New("asteros is not running")
	ErrAlreadyRunning = errors.New("asteros is already running")
)
