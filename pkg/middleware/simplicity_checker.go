package middleware

import (
	"context"
	"log"
	"regexp"
	"strings"
	"sync"
)

// SimplicityCheckerMiddleware 简洁性检查中间件
// 检测过度工程和未请求的功能添加，发出警告但不阻断执行
type SimplicityCheckerMiddleware struct {
	*BaseMiddleware
	config *SimplicityCheckerConfig

	// 会话级统计
	mu              sync.Mutex
	helperCount     int
	interfaceCount  int
	warningsEmitted []SimplicityWarning
}

// SimplicityCheckerConfig 简洁性检查配置
type SimplicityCheckerConfig struct {
	// Enabled 是否启用检测
	Enabled bool

	// MaxHelperFunctions 最大辅助函数创建数（单次会话）
	MaxHelperFunctions int

	// WarnOnPrematureAbstraction 是否警告过早抽象
	WarnOnPrematureAbstraction bool

	// WarnOnUnusedParams 是否警告未使用的参数重命名
	WarnOnUnusedParams bool

	// OnWarning 警告回调（可选）
	OnWarning func(warning SimplicityWarning)
}

// SimplicityWarning 简洁性警告
type SimplicityWarning struct {
	Type    string         `json:"type"`    // 警告类型
	Message string         `json:"message"` // 警告消息
	File    string         `json:"file"`    // 相关文件
	Details map[string]any `json:"details"` // 详细信息
}

// 警告类型常量
const (
	WarningTypeHelperOverflow       = "helper_overflow"
	WarningTypePrematureAbstraction = "premature_abstraction"
	WarningTypeUnusedFeature        = "unused_feature"
	WarningTypeBackwardsCompatHack  = "backwards_compat_hack"
	WarningTypeOverEngineering      = "over_engineering"
)

// 检测模式
var (
	// 辅助函数模式 (Helper, Util, Utils 等)
	helperFuncPattern = regexp.MustCompile(`(?i)func\s+(\w*(?:Helper|Util|Utils|Wrapper)\w*)\s*\(`)

	// 接口定义模式
	interfacePattern = regexp.MustCompile(`type\s+(\w+)\s+interface\s*\{`)

	// 未使用变量重命名模式 (_var)
	unusedVarPattern = regexp.MustCompile(`\b_\w+\s*(?:=|:=)`)

	// 移除注释模式
	removedCommentPattern = regexp.MustCompile(`//\s*(?:removed|deprecated|unused|TODO:\s*remove)`)

	// Feature flag 模式
	featureFlagPattern = regexp.MustCompile(`(?i)(?:feature[_\s]?flag|config[_\s]?option|experimental|toggle)`)

	// 过度配置模式
	overConfigPattern = regexp.MustCompile(`(?i)(?:WithOption|SetConfig|Configure)\s*\(`)
)

// NewSimplicityCheckerMiddleware 创建简洁性检查中间件
func NewSimplicityCheckerMiddleware(config *SimplicityCheckerConfig) *SimplicityCheckerMiddleware {
	if config == nil {
		config = &SimplicityCheckerConfig{
			Enabled:                    true,
			MaxHelperFunctions:         3,
			WarnOnPrematureAbstraction: true,
			WarnOnUnusedParams:         true,
		}
	}

	// 设置默认值
	if config.MaxHelperFunctions <= 0 {
		config.MaxHelperFunctions = 3
	}

	return &SimplicityCheckerMiddleware{
		BaseMiddleware:  NewBaseMiddleware("simplicity_checker", 600),
		config:          config,
		warningsEmitted: make([]SimplicityWarning, 0),
	}
}

// WrapToolCall 包装工具调用，检测简洁性问题
func (m *SimplicityCheckerMiddleware) WrapToolCall(ctx context.Context, req *ToolCallRequest, handler ToolCallHandler) (*ToolCallResponse, error) {
	// 如果禁用，直接执行
	if !m.config.Enabled {
		return handler(ctx, req)
	}

	// 只检测 Write 和 Edit 工具
	switch req.ToolName {
	case "Write", "Edit":
		m.checkToolCall(req)
	}

	// 继续执行原始操作
	return handler(ctx, req)
}

// checkToolCall 检查工具调用是否存在简洁性问题
func (m *SimplicityCheckerMiddleware) checkToolCall(req *ToolCallRequest) {
	// 提取文件路径
	filePath := ""
	if fp, ok := req.ToolInput["file_path"].(string); ok {
		filePath = fp
	}

	// 提取代码内容
	content := ""
	switch req.ToolName {
	case "Write":
		if c, ok := req.ToolInput["content"].(string); ok {
			content = c
		}
	case "Edit":
		if ns, ok := req.ToolInput["new_string"].(string); ok {
			content = ns
		}
	}

	if content == "" {
		return
	}

	// 运行检测规则
	m.checkHelperFunctions(content, filePath)
	m.checkPrematureAbstraction(content, filePath)
	m.checkBackwardsCompatHacks(content, filePath)
	m.checkOverEngineering(content, filePath)
}

// checkHelperFunctions 检测辅助函数泛滥
func (m *SimplicityCheckerMiddleware) checkHelperFunctions(content, filePath string) {
	matches := helperFuncPattern.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, match := range matches {
		m.helperCount++
		funcName := match[1]

		if m.helperCount > m.config.MaxHelperFunctions {
			m.emitWarning(SimplicityWarning{
				Type:    WarningTypeHelperOverflow,
				Message: "创建了过多的辅助函数，考虑简化设计或合并功能",
				File:    filePath,
				Details: map[string]any{
					"function_name": funcName,
					"helper_count":  m.helperCount,
					"max_allowed":   m.config.MaxHelperFunctions,
				},
			})
		}
	}
}

// checkPrematureAbstraction 检测过早抽象
func (m *SimplicityCheckerMiddleware) checkPrematureAbstraction(content, filePath string) {
	if !m.config.WarnOnPrematureAbstraction {
		return
	}

	matches := interfacePattern.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, match := range matches {
		m.interfaceCount++
		interfaceName := match[1]

		// 如果单次会话创建了多个接口，可能是过早抽象
		if m.interfaceCount > 2 {
			m.emitWarning(SimplicityWarning{
				Type:    WarningTypePrematureAbstraction,
				Message: "创建了较多接口定义，确保每个接口都有明确的用途和多个实现",
				File:    filePath,
				Details: map[string]any{
					"interface_name":  interfaceName,
					"interface_count": m.interfaceCount,
				},
			})
		}
	}
}

// checkBackwardsCompatHacks 检测向后兼容 hack
func (m *SimplicityCheckerMiddleware) checkBackwardsCompatHacks(content, filePath string) {
	if !m.config.WarnOnUnusedParams {
		return
	}

	warnings := make([]string, 0)

	// 检测未使用变量重命名
	if unusedVarPattern.MatchString(content) {
		warnings = append(warnings, "发现未使用变量重命名模式 (_var)")
	}

	// 检测移除注释
	if removedCommentPattern.MatchString(content) {
		warnings = append(warnings, "发现'// removed'或类似注释，考虑直接删除代码")
	}

	if len(warnings) > 0 {
		m.emitWarning(SimplicityWarning{
			Type:    WarningTypeBackwardsCompatHack,
			Message: "发现向后兼容 hack 模式，考虑直接删除未使用的代码",
			File:    filePath,
			Details: map[string]any{
				"patterns_found": warnings,
			},
		})
	}
}

// checkOverEngineering 检测过度工程
func (m *SimplicityCheckerMiddleware) checkOverEngineering(content, filePath string) {
	warnings := make([]string, 0)

	// 检测 feature flag 模式
	if featureFlagPattern.MatchString(content) {
		warnings = append(warnings, "发现 feature flag 或配置选项模式")
	}

	// 检测过度配置模式
	configMatches := overConfigPattern.FindAllString(content, -1)
	if len(configMatches) > 3 {
		warnings = append(warnings, "发现大量配置方法，可能过度设计")
	}

	// 检测代码行数（粗略估计）
	lines := strings.Count(content, "\n")
	if lines > 200 {
		warnings = append(warnings, "单次添加超过 200 行代码，考虑拆分")
	}

	if len(warnings) > 0 {
		m.emitWarning(SimplicityWarning{
			Type:    WarningTypeOverEngineering,
			Message: "检测到可能的过度工程迹象",
			File:    filePath,
			Details: map[string]any{
				"patterns_found": warnings,
				"lines_added":    lines,
			},
		})
	}
}

// emitWarning 发出警告
func (m *SimplicityCheckerMiddleware) emitWarning(warning SimplicityWarning) {
	m.warningsEmitted = append(m.warningsEmitted, warning)

	// 记录日志
	log.Printf("[SimplicityChecker] WARNING [%s] %s (file: %s)",
		warning.Type, warning.Message, warning.File)

	// 调用回调
	if m.config.OnWarning != nil {
		m.config.OnWarning(warning)
	}
}

// GetWarnings 获取所有已发出的警告
func (m *SimplicityCheckerMiddleware) GetWarnings() []SimplicityWarning {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]SimplicityWarning{}, m.warningsEmitted...)
}

// Reset 重置会话统计
func (m *SimplicityCheckerMiddleware) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.helperCount = 0
	m.interfaceCount = 0
	m.warningsEmitted = make([]SimplicityWarning, 0)
}

// OnAgentStart Agent 启动时重置统计
func (m *SimplicityCheckerMiddleware) OnAgentStart(ctx context.Context, agentID string) error {
	m.Reset()
	return nil
}
