package security

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

// SecurityPolicy 安全策略
type SecurityPolicy struct {
	// 基本信息
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Enabled     bool   `json:"enabled"`
	Priority    int    `json:"priority"` // 优先级，数值越高优先级越高

	// 作用域
	Scope     PolicyScope  `json:"scope"`
	Target    PolicyTarget `json:"target"`
	Resources []string     `json:"resources"` // 资源列表

	// 规则
	Rules      []PolicyRule      `json:"rules"`
	Conditions []PolicyCondition `json:"conditions"`

	// 动作
	Allow   []string `json:"allow"`   // 允许的动作
	Deny    []string `json:"deny"`    // 拒绝的动作
	Require []string `json:"require"` // 必需的条件

	// 时间限制
	TimeConstraints *TimeConstraints `json:"time_constraints,omitempty"`

	// 环境限制
	EnvironmentConstraints *EnvironmentConstraints `json:"environment_constraints,omitempty"`

	// 处理方式
	Action   PolicyAction   `json:"action"`
	Response PolicyResponse `json:"response"`

	// 元数据
	Tags     []string       `json:"tags"`
	Metadata map[string]any `json:"metadata"`

	// 审计
	AuditEnabled bool       `json:"audit_enabled"`
	AuditLevel   AuditLevel `json:"audit_level"`

	// 创建和更新信息
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedBy string    `json:"created_by"`
	UpdatedBy string    `json:"updated_by"`
}

// PolicyScope 策略作用域
type PolicyScope string

const (
	ScopeGlobal    PolicyScope = "global"    // 全局
	ScopeAgent     PolicyScope = "agent"     // Agent级别
	ScopeWorkflow  PolicyScope = "workflow"  // 工作流级别
	ScopeSession   PolicyScope = "session"   // 会话级别
	ScopeResource  PolicyScope = "resource"  // 资源级别
	ScopeOperation PolicyScope = "operation" // 操作级别
)

// PolicyTarget 策略目标
type PolicyTarget string

const (
	TargetAll     PolicyTarget = "all"     // 所有目标
	TargetUser    PolicyTarget = "user"    // 用户
	TargetAgent   PolicyTarget = "agent"   // Agent
	TargetSystem  PolicyTarget = "system"  // 系统
	TargetNetwork PolicyTarget = "network" // 网络
	TargetData    PolicyTarget = "data"    // 数据
	TargetAPI     PolicyTarget = "api"     // API
)

// PolicyRule 策略规则
type PolicyRule struct {
	ID          string         `json:"id"`
	Type        RuleType       `json:"type"`
	Field       string         `json:"field"`
	Operator    RuleOperator   `json:"operator"`
	Value       any            `json:"value"`
	Description string         `json:"description"`
	Enabled     bool           `json:"enabled"`
	Priority    int            `json:"priority"`
	Metadata    map[string]any `json:"metadata"`
}

// RuleType 规则类型
type RuleType string

const (
	RuleTypeBasic  RuleType = "basic"  // 基础规则
	RuleTypeRegex  RuleType = "regex"  // 正则表达式规则
	RuleTypeScript RuleType = "script" // 脚本规则
	RuleTypeML     RuleType = "ml"     // 机器学习规则
	RuleTypeCustom RuleType = "custom" // 自定义规则
)

// RuleOperator 规则操作符
type RuleOperator string

const (
	OperatorEquals      RuleOperator = "eq"         // 等于
	OperatorNotEquals   RuleOperator = "ne"         // 不等于
	OperatorGreaterThan RuleOperator = "gt"         // 大于
	OperatorGreaterOrEq RuleOperator = "gte"        // 大于等于
	OperatorLessThan    RuleOperator = "lt"         // 小于
	OperatorLessOrEq    RuleOperator = "lte"        // 小于等于
	OperatorContains    RuleOperator = "contains"   // 包含
	OperatorNotContains RuleOperator = "ncontains"  // 不包含
	OperatorIn          RuleOperator = "in"         // 在列表中
	OperatorNotIn       RuleOperator = "nin"        // 不在列表中
	OperatorMatches     RuleOperator = "matches"    // 匹配正则
	OperatorNotMatches  RuleOperator = "nmatches"   // 不匹配正则
	OperatorStartsWith  RuleOperator = "startswith" // 以...开始
	OperatorEndsWith    RuleOperator = "endswith"   // 以...结束
)

// PolicyCondition 策略条件
type PolicyCondition struct {
	ID          string            `json:"id"`
	Type        ConditionType     `json:"type"`
	Field       string            `json:"field"`
	Operator    ConditionOperator `json:"operator"`
	Value       any               `json:"value"`
	Logic       ConditionLogic    `json:"logic"` // AND, OR, NOT
	Description string            `json:"description"`
	Enabled     bool              `json:"enabled"`
	Metadata    map[string]any    `json:"metadata"`
}

// ConditionType 条件类型
type ConditionType string

const (
	ConditionTypeStatic   ConditionType = "static"   // 静态条件
	ConditionTypeDynamic  ConditionType = "dynamic"  // 动态条件
	ConditionTypeContext  ConditionType = "context"  // 上下文条件
	ConditionTypeTime     ConditionType = "time"     // 时间条件
	ConditionTypeLocation ConditionType = "location" // 位置条件
	ConditionTypeRisk     ConditionType = "risk"     // 风险条件
)

// ConditionOperator 条件操作符
type ConditionOperator string

const (
	ConditionOperatorExists      ConditionOperator = "exists"    // 存在
	ConditionOperatorNotExists   ConditionOperator = "notexists" // 不存在
	ConditionOperatorEquals      ConditionOperator = "eq"        // 等于
	ConditionOperatorNotEquals   ConditionOperator = "ne"        // 不等于
	ConditionOperatorGreaterThan ConditionOperator = "gt"        // 大于
	ConditionOperatorLessThan    ConditionOperator = "lt"        // 小于
	ConditionOperatorContains    ConditionOperator = "contains"  // 包含
	ConditionOperatorMatches     ConditionOperator = "matches"   // 匹配
)

// ConditionLogic 条件逻辑
type ConditionLogic string

const (
	ConditionLogicAND ConditionLogic = "and" // AND逻辑
	ConditionLogicOR  ConditionLogic = "or"  // OR逻辑
	ConditionLogicNOT ConditionLogic = "not" // NOT逻辑
)

// TimeConstraints 时间约束
type TimeConstraints struct {
	StartDate *time.Time     `json:"start_date,omitempty"`
	EndDate   *time.Time     `json:"end_date,omitempty"`
	StartTime string         `json:"start_time,omitempty"` // HH:MM格式
	EndTime   string         `json:"end_time,omitempty"`   // HH:MM格式
	TimeZone  string         `json:"timezone,omitempty"`
	Weekdays  []int          `json:"weekdays,omitempty"` // 0-6，0为周日
	Duration  *time.Duration `json:"duration,omitempty"` // 最大持续时间
}

// EnvironmentConstraints 环境约束
type EnvironmentConstraints struct {
	AllowedIPs       []string `json:"allowed_ips,omitempty"`
	BlockedIPs       []string `json:"blocked_ips,omitempty"`
	AllowedCountries []string `json:"allowed_countries,omitempty"`
	BlockedCountries []string `json:"blocked_countries,omitempty"`
	AllowedRegions   []string `json:"allowed_regions,omitempty"`
	BlockedRegions   []string `json:"blocked_regions,omitempty"`
	RequiredEnv      []string `json:"required_env,omitempty"`
	BlockedEnv       []string `json:"blocked_env,omitempty"`
	SecurityLevel    string   `json:"security_level,omitempty"`
}

// PolicyAction 策略动作
type PolicyAction string

const (
	ActionAllow      PolicyAction = "allow"      // 允许
	ActionDeny       PolicyAction = "deny"       // 拒绝
	ActionWarn       PolicyAction = "warn"       // 警告
	ActionAudit      PolicyAction = "audit"      // 审计
	ActionQuarantine PolicyAction = "quarantine" // 隔离
	ActionBlock      PolicyAction = "block"      // 阻塞
	ActionRedirect   PolicyAction = "redirect"   // 重定向
	ActionTransform  PolicyAction = "transform"  // 转换
	ActionChallenge  PolicyAction = "challenge"  // 挑战
	ActionStepUp     PolicyAction = "stepup"     // 升级认证
)

// PolicyResponse 策略响应
type PolicyResponse struct {
	Message     string            `json:"message"`
	Code        int               `json:"code"`
	Headers     map[string]string `json:"headers,omitempty"`
	RedirectURL string            `json:"redirect_url,omitempty"`
	Challenge   *ChallengeInfo    `json:"challenge,omitempty"`
	Transform   *TransformInfo    `json:"transform,omitempty"`
	Metadata    map[string]any    `json:"metadata"`
}

// ChallengeInfo 挑战信息
type ChallengeInfo struct {
	Type        string         `json:"type"` // CAPTCHA, MFA, 知识问答等
	Duration    time.Duration  `json:"duration"`
	MaxAttempts int            `json:"max_attempts"`
	Parameters  map[string]any `json:"parameters"`
}

// TransformInfo 转换信息
type TransformInfo struct {
	Type       string         `json:"type"` // 数据脱敏、格式转换等
	Parameters map[string]any `json:"parameters"`
}

// AuditLevel 审计级别
type AuditLevel string

const (
	AuditLevelNone   AuditLevel = "none"   // 无审计
	AuditLevelBasic  AuditLevel = "basic"  // 基础审计
	AuditLevelDetail AuditLevel = "detail" // 详细审计
	AuditLevelFull   AuditLevel = "full"   // 完整审计
)

// PolicyEvaluation 策略评估结果
type PolicyEvaluation struct {
	PolicyID            string          `json:"policy_id"`
	PolicyName          string          `json:"policy_name"`
	Allowed             bool            `json:"allowed"`
	Action              PolicyAction    `json:"action"`
	Reason              string          `json:"reason"`
	Score               float64         `json:"score"` // 风险评分 0-100
	RiskLevel           RiskLevel       `json:"risk_level"`
	Duration            time.Duration   `json:"duration"`
	MatchedRules        []string        `json:"matched_rules"`
	TriggeredConditions []string        `json:"triggered_conditions"`
	Response            *PolicyResponse `json:"response,omitempty"`
	Metadata            map[string]any  `json:"metadata"`
	EvaluatedAt         time.Time       `json:"evaluated_at"`
}

// RiskLevel 风险级别
type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"      // 低风险
	RiskLevelMedium   RiskLevel = "medium"   // 中等风险
	RiskLevelHigh     RiskLevel = "high"     // 高风险
	RiskLevelCritical RiskLevel = "critical" // 严重风险
)

// PolicyRequest 策略请求
type PolicyRequest struct {
	RequestID   string         `json:"request_id"`
	UserID      string         `json:"user_id,omitempty"`
	AgentID     string         `json:"agent_id,omitempty"`
	Action      string         `json:"action"`
	Resource    string         `json:"resource"`
	Context     map[string]any `json:"context"`
	IPAddress   string         `json:"ip_address,omitempty"`
	UserAgent   string         `json:"user_agent,omitempty"`
	Timestamp   time.Time      `json:"timestamp"`
	Environment string         `json:"environment,omitempty"`
	Location    string         `json:"location,omitempty"`
	Metadata    map[string]any `json:"metadata"`
}

// PolicyEngine 策略引擎接口
type PolicyEngine interface {
	// 策略管理
	AddPolicy(policy *SecurityPolicy) error
	UpdatePolicy(policy *SecurityPolicy) error
	DeletePolicy(policyID string) error
	GetPolicy(policyID string) (*SecurityPolicy, error)
	ListPolicies(filters map[string]any) ([]*SecurityPolicy, error)
	EnablePolicy(policyID string) error
	DisablePolicy(policyID string) error

	// 策略评估
	Evaluate(request *PolicyRequest) (*PolicyEvaluation, error)
	EvaluateBatch(requests []*PolicyRequest) ([]*PolicyEvaluation, error)
	EvaluateRealTime(request *PolicyRequest) (*PolicyEvaluation, error)

	// 规则管理
	AddRule(policyID string, rule *PolicyRule) error
	RemoveRule(policyID string, ruleID string) error
	UpdateRule(policyID string, rule *PolicyRule) error

	// 条件管理
	AddCondition(policyID string, condition *PolicyCondition) error
	RemoveCondition(policyID string, conditionID string) error
	UpdateCondition(policyID string, condition *PolicyCondition) error

	// 分析和报告
	AnalyzePolicy(policyID string, timeRange TimeRange) (*PolicyAnalysis, error)
	GenerateReport(reportType ReportType, filters map[string]any) (*PolicyReport, error)

	// 配置和状态
	GetEngineStatus() *EngineStatus
	ReloadPolicies() error
	BackupPolicies() ([]byte, error)
	RestorePolicies(data []byte) error
}

// TimeRange 时间范围
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// PolicyAnalysis 策略分析结果
type PolicyAnalysis struct {
	PolicyID         string                 `json:"policy_id"`
	Period           TimeRange              `json:"period"`
	TotalRequests    int64                  `json:"total_requests"`
	AllowedRequests  int64                  `json:"allowed_requests"`
	DeniedRequests   int64                  `json:"denied_requests"`
	AverageScore     float64                `json:"average_score"`
	RiskDistribution map[RiskLevel]int64    `json:"risk_distribution"`
	ActionStats      map[PolicyAction]int64 `json:"action_stats"`
	TopViolators     []string               `json:"top_violators"`
	Recommendations  []string               `json:"recommendations"`
}

// ReportType 报告类型
type ReportType string

const (
	ReportTypeSummary    ReportType = "summary"    // 摘要报告
	ReportTypeDetail     ReportType = "detail"     // 详细报告
	ReportTypeViolation  ReportType = "violation"  // 违规报告
	ReportTypeTrend      ReportType = "trend"      // 趋势报告
	ReportTypeCompliance ReportType = "compliance" // 合规报告
)

// PolicyReport 策略报告
type PolicyReport struct {
	ID          string         `json:"id"`
	Type        ReportType     `json:"type"`
	Title       string         `json:"title"`
	Period      TimeRange      `json:"period"`
	GeneratedAt time.Time      `json:"generated_at"`
	GeneratedBy string         `json:"generated_by"`
	Content     map[string]any `json:"content"`
	Format      ReportFormat   `json:"format"`
}

// ReportFormat 报告格式
type ReportFormat string

const (
	ReportFormatJSON ReportFormat = "json" // JSON格式
	ReportFormatCSV  ReportFormat = "csv"  // CSV格式
	ReportFormatPDF  ReportFormat = "pdf"  // PDF格式
	ReportFormatHTML ReportFormat = "html" // HTML格式
)

// EngineStatus 引擎状态
type EngineStatus struct {
	Status            string         `json:"status"`
	Version           string         `json:"version"`
	Uptime            time.Duration  `json:"uptime"`
	PolicyCount       int            `json:"policy_count"`
	ActivePolicyCount int            `json:"active_policy_count"`
	TotalEvaluations  int64          `json:"total_evaluations"`
	AverageLatency    time.Duration  `json:"average_latency"`
	ErrorRate         float64        `json:"error_rate"`
	LastReload        time.Time      `json:"last_reload"`
	MemoryUsage       map[string]any `json:"memory_usage"`
	CPUUsage          map[string]any `json:"cpu_usage"`
}

// BasicPolicyEngine 基础策略引擎实现
type BasicPolicyEngine struct {
	policies map[string]*SecurityPolicy
	mu       sync.RWMutex
	config   *EngineConfig
	auditLog AuditLog
}

// EngineConfig 引擎配置
type EngineConfig struct {
	EnableCaching     bool          `json:"enable_caching"`
	CacheTimeout      time.Duration `json:"cache_timeout"`
	EnableMetrics     bool          `json:"enable_metrics"`
	EnableAudit       bool          `json:"enable_audit"`
	MaxConcurrentEval int           `json:"max_concurrent_eval"`
	DefaultAction     PolicyAction  `json:"default_action"`
}

// NewBasicPolicyEngine 创建基础策略引擎
func NewBasicPolicyEngine(config *EngineConfig, auditLog AuditLog) *BasicPolicyEngine {
	if config == nil {
		config = &EngineConfig{
			EnableCaching:     true,
			CacheTimeout:      time.Minute * 5,
			EnableMetrics:     true,
			EnableAudit:       true,
			MaxConcurrentEval: 100,
			DefaultAction:     ActionDeny,
		}
	}

	return &BasicPolicyEngine{
		policies: make(map[string]*SecurityPolicy),
		config:   config,
		auditLog: auditLog,
	}
}

// AddPolicy 添加策略
func (bpe *BasicPolicyEngine) AddPolicy(policy *SecurityPolicy) error {
	bpe.mu.Lock()
	defer bpe.mu.Unlock()

	if policy.ID == "" {
		return fmt.Errorf("policy ID is required")
	}

	if _, exists := bpe.policies[policy.ID]; exists {
		return fmt.Errorf("policy %s already exists", policy.ID)
	}

	// 设置默认值
	if policy.Version == "" {
		policy.Version = "1.0.0"
	}
	if policy.Priority == 0 {
		policy.Priority = 100
	}
	if policy.CreatedAt.IsZero() {
		policy.CreatedAt = time.Now()
	}
	if policy.UpdatedAt.IsZero() {
		policy.UpdatedAt = time.Now()
	}

	bpe.policies[policy.ID] = policy

	// 记录审计日志
	if bpe.config.EnableAudit && bpe.auditLog != nil {
		_ = bpe.auditLog.LogEvent(AuditEvent{
			Type:      AuditTypePolicyCreated,
			UserID:    policy.CreatedBy,
			Timestamp: time.Now(),
			Message:   fmt.Sprintf("Policy %s (%s) created", policy.ID, policy.Name),
			Metadata: map[string]any{
				"policy_id":     policy.ID,
				"policy_name":   policy.Name,
				"policy_scope":  policy.Scope,
				"policy_target": policy.Target,
			},
		})
	}

	return nil
}

// UpdatePolicy 更新策略
func (bpe *BasicPolicyEngine) UpdatePolicy(policy *SecurityPolicy) error {
	bpe.mu.Lock()
	defer bpe.mu.Unlock()

	if policy.ID == "" {
		return fmt.Errorf("policy ID is required")
	}

	existing, exists := bpe.policies[policy.ID]
	if !exists {
		return fmt.Errorf("policy %s not found", policy.ID)
	}

	// 更新时间戳
	policy.UpdatedAt = time.Now()
	policy.CreatedAt = existing.CreatedAt // 保持创建时间
	policy.CreatedBy = existing.CreatedBy // 保持创建者

	bpe.policies[policy.ID] = policy

	// 记录审计日志
	if bpe.config.EnableAudit && bpe.auditLog != nil {
		_ = bpe.auditLog.LogEvent(AuditEvent{
			Type:      AuditTypePolicyUpdated,
			UserID:    policy.UpdatedBy,
			Timestamp: time.Now(),
			Message:   fmt.Sprintf("Policy %s (%s) updated", policy.ID, policy.Name),
			Metadata: map[string]any{
				"policy_id":      policy.ID,
				"policy_name":    policy.Name,
				"policy_version": policy.Version,
			},
		})
	}

	return nil
}

// DeletePolicy 删除策略
func (bpe *BasicPolicyEngine) DeletePolicy(policyID string) error {
	bpe.mu.Lock()
	defer bpe.mu.Unlock()

	_, exists := bpe.policies[policyID]
	if !exists {
		return fmt.Errorf("policy %s not found", policyID)
	}

	delete(bpe.policies, policyID)

	// 记录审计日志
	if bpe.config.EnableAudit && bpe.auditLog != nil {
		_ = bpe.auditLog.LogEvent(AuditEvent{
			Type:      AuditTypePolicyDeleted,
			Timestamp: time.Now(),
			Message:   fmt.Sprintf("Policy %s deleted", policyID),
			Metadata: map[string]any{
				"policy_id":         policyID,
				"deleted_policy_id": policyID,
			},
		})
	}

	return nil
}

// GetPolicy 获取策略
func (bpe *BasicPolicyEngine) GetPolicy(policyID string) (*SecurityPolicy, error) {
	bpe.mu.RLock()
	defer bpe.mu.RUnlock()

	policy, exists := bpe.policies[policyID]
	if !exists {
		return nil, fmt.Errorf("policy %s not found", policyID)
	}

	return policy, nil
}

// ListPolicies 列出策略
func (bpe *BasicPolicyEngine) ListPolicies(filters map[string]any) ([]*SecurityPolicy, error) {
	bpe.mu.RLock()
	defer bpe.mu.RUnlock()

	var policies []*SecurityPolicy
	for _, policy := range bpe.policies {
		if bpe.matchesFilters(policy, filters) {
			policies = append(policies, policy)
		}
	}

	return policies, nil
}

// matchesFilters 检查策略是否匹配过滤条件
func (bpe *BasicPolicyEngine) matchesFilters(policy *SecurityPolicy, filters map[string]any) bool {
	if len(filters) == 0 {
		return true
	}

	for key, value := range filters {
		switch key {
		case "enabled":
			if policy.Enabled != value.(bool) {
				return false
			}
		case "scope":
			if policy.Scope != PolicyScope(value.(string)) {
				return false
			}
		case "target":
			if policy.Target != PolicyTarget(value.(string)) {
				return false
			}
		case "tags":
			tags := value.([]string)
			if !containsAny(policy.Tags, tags) {
				return false
			}
		}
	}

	return true
}

// containsAny 检查数组是否包含任意一个元素
func containsAny(slice []string, elements []string) bool {
	for _, elem := range elements {
		for _, item := range slice {
			if item == elem {
				return true
			}
		}
	}
	return false
}

// Evaluate 评估策略
func (bpe *BasicPolicyEngine) Evaluate(request *PolicyRequest) (*PolicyEvaluation, error) {
	start := time.Now()

	bpe.mu.RLock()
	policies := bpe.getMatchingPolicies(request)
	bpe.mu.RUnlock()

	if len(policies) == 0 {
		// 没有匹配的策略，使用默认动作
		return &PolicyEvaluation{
			Allowed:     bpe.config.DefaultAction == ActionAllow,
			Action:      bpe.config.DefaultAction,
			Reason:      "No matching policies found",
			Score:       50.0,
			RiskLevel:   RiskLevelMedium,
			Duration:    time.Since(start),
			EvaluatedAt: time.Now(),
		}, nil
	}

	// 按优先级排序策略
	sortedPolicies := bpe.sortPoliciesByPriority(policies)

	// 评估策略
	for _, policy := range sortedPolicies {
		if !policy.Enabled {
			continue
		}

		eval, err := bpe.evaluatePolicy(policy, request)
		if err != nil {
			continue // 跳过评估失败的策略
		}

		if eval.Action != ActionAllow {
			// 找到第一个拒绝或需要特殊处理的策略
			eval.Duration = time.Since(start)
			return eval, nil
		}
	}

	// 所有策略都允许
	return &PolicyEvaluation{
		Allowed:     true,
		Action:      ActionAllow,
		Reason:      "All matching policies allow the request",
		Score:       0.0,
		RiskLevel:   RiskLevelLow,
		Duration:    time.Since(start),
		EvaluatedAt: time.Now(),
	}, nil
}

// getMatchingPolicies 获取匹配的策略
func (bpe *BasicPolicyEngine) getMatchingPolicies(request *PolicyRequest) []*SecurityPolicy {
	var policies []*SecurityPolicy
	for _, policy := range bpe.policies {
		if bpe.policyMatchesRequest(policy, request) {
			policies = append(policies, policy)
		}
	}
	return policies
}

// policyMatchesRequest 检查策略是否匹配请求
func (bpe *BasicPolicyEngine) policyMatchesRequest(policy *SecurityPolicy, request *PolicyRequest) bool {
	// 检查作用域
	if !bpe.scopeMatches(policy.Scope, request) {
		return false
	}

	// 检查目标
	if !bpe.targetMatches(policy.Target, request) {
		return false
	}

	// 检查资源
	if len(policy.Resources) > 0 && !containsPolicy(policy.Resources, request.Resource) {
		return false
	}

	// 检查时间约束
	if !bpe.timeConstraintsMatch(policy.TimeConstraints, request.Timestamp) {
		return false
	}

	// 检查环境约束
	if !bpe.environmentConstraintsMatch(policy.EnvironmentConstraints, request) {
		return false
	}

	return true
}

// scopeMatches 检查作用域是否匹配
func (bpe *BasicPolicyEngine) scopeMatches(scope PolicyScope, request *PolicyRequest) bool {
	switch scope {
	case ScopeGlobal:
		return true
	case ScopeAgent:
		return request.AgentID != ""
	case ScopeSession:
		return request.Context["session_id"] != nil
	case ScopeResource:
		return request.Resource != ""
	case ScopeOperation:
		return request.Action != ""
	default:
		return true
	}
}

// targetMatches 检查目标是否匹配
func (bpe *BasicPolicyEngine) targetMatches(target PolicyTarget, request *PolicyRequest) bool {
	switch target {
	case TargetAll:
		return true
	case TargetUser:
		return request.UserID != ""
	case TargetAgent:
		return request.AgentID != ""
	case TargetSystem:
		return request.Environment == "system"
	case TargetNetwork:
		return request.IPAddress != ""
	case TargetData:
		return strings.HasPrefix(request.Resource, "data/")
	case TargetAPI:
		return strings.HasPrefix(request.Resource, "api/")
	default:
		return true
	}
}

// timeConstraintsMatch 检查时间约束是否匹配
func (bpe *BasicPolicyEngine) timeConstraintsMatch(constraints *TimeConstraints, timestamp time.Time) bool {
	if constraints == nil {
		return true
	}

	// 检查日期范围
	if constraints.StartDate != nil && timestamp.Before(*constraints.StartDate) {
		return false
	}
	if constraints.EndDate != nil && timestamp.After(*constraints.EndDate) {
		return false
	}

	// TODO: 实现更复杂的时间约束检查
	return true
}

// environmentConstraintsMatch 检查环境约束是否匹配
func (bpe *BasicPolicyEngine) environmentConstraintsMatch(constraints *EnvironmentConstraints, request *PolicyRequest) bool {
	if constraints == nil {
		return true
	}

	// 检查IP地址
	if len(constraints.AllowedIPs) > 0 && !containsPolicy(constraints.AllowedIPs, request.IPAddress) {
		return false
	}
	if len(constraints.BlockedIPs) > 0 && containsPolicy(constraints.BlockedIPs, request.IPAddress) {
		return false
	}

	return true
}

// sortPoliciesByPriority 按优先级排序策略
func (bpe *BasicPolicyEngine) sortPoliciesByPriority(policies []*SecurityPolicy) []*SecurityPolicy {
	// 简单的冒泡排序，实际应用中可以使用更高效的排序算法
	sorted := make([]*SecurityPolicy, len(policies))
	copy(sorted, policies)

	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i].Priority < sorted[j].Priority {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	return sorted
}

// evaluatePolicy 评估单个策略
func (bpe *BasicPolicyEngine) evaluatePolicy(policy *SecurityPolicy, request *PolicyRequest) (*PolicyEvaluation, error) {
	eval := &PolicyEvaluation{
		PolicyID:    policy.ID,
		PolicyName:  policy.Name,
		Allowed:     true,
		Action:      ActionAllow,
		Score:       0.0,
		RiskLevel:   RiskLevelLow,
		EvaluatedAt: time.Now(),
	}

	// 检查动作
	if len(policy.Allow) > 0 && !containsPolicy(policy.Allow, request.Action) {
		eval.Allowed = false
		eval.Action = ActionDeny
		eval.Reason = fmt.Sprintf("Action %s is not in allowed list", request.Action)
		return eval, nil
	}

	if len(policy.Deny) > 0 && containsPolicy(policy.Deny, request.Action) {
		eval.Allowed = false
		eval.Action = ActionDeny
		eval.Reason = fmt.Sprintf("Action %s is in deny list", request.Action)
		eval.Score = 90.0
		eval.RiskLevel = RiskLevelHigh
		return eval, nil
	}

	// 评估规则
	ruleScore, matchedRules := bpe.evaluateRules(policy.Rules, request)
	eval.MatchedRules = matchedRules
	eval.Score += ruleScore

	// 评估条件
	conditionScore, triggeredConditions := bpe.evaluateConditions(policy.Conditions, request)
	eval.TriggeredConditions = triggeredConditions
	eval.Score += conditionScore

	// 确定最终动作和风险级别
	eval.Action = policy.Action
	if eval.Action == ActionAllow {
		eval.Allowed = true
	} else {
		eval.Allowed = false
	}

	eval.RiskLevel = bpe.determineRiskLevel(eval.Score)

	return eval, nil
}

// evaluateRules 评估规则
func (bpe *BasicPolicyEngine) evaluateRules(rules []PolicyRule, request *PolicyRequest) (float64, []string) {
	var score float64
	var matchedRules []string

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		if bpe.evaluateRule(rule, request) {
			score += 10.0 // 每个匹配的规则增加10分风险
			matchedRules = append(matchedRules, rule.ID)
		}
	}

	return score, matchedRules
}

// evaluateRule 评估单个规则
func (bpe *BasicPolicyEngine) evaluateRule(rule PolicyRule, request *PolicyRequest) bool {
	// 从请求中获取字段值
	value, exists := bpe.getFieldValue(request, rule.Field)
	if !exists {
		return false
	}

	// 根据操作符进行匹配
	switch rule.Operator {
	case OperatorEquals:
		return bpe.compareValues(value, rule.Value) == 0
	case OperatorNotEquals:
		return bpe.compareValues(value, rule.Value) != 0
	case OperatorContains:
		return bpe.stringContains(value, rule.Value)
	case OperatorIn:
		return bpe.valueInList(value, rule.Value)
	case OperatorMatches:
		return bpe.regexMatch(value, rule.Value)
	default:
		return false
	}
}

// getFieldValue 从请求中获取字段值
func (bpe *BasicPolicyEngine) getFieldValue(request *PolicyRequest, field string) (any, bool) {
	switch field {
	case "user_id":
		return request.UserID, request.UserID != ""
	case "agent_id":
		return request.AgentID, request.AgentID != ""
	case "action":
		return request.Action, request.Action != ""
	case "resource":
		return request.Resource, request.Resource != ""
	case "ip_address":
		return request.IPAddress, request.IPAddress != ""
	case "user_agent":
		return request.UserAgent, request.UserAgent != ""
	case "environment":
		return request.Environment, request.Environment != ""
	case "location":
		return request.Location, request.Location != ""
	default:
		// 从context中获取
		if val, exists := request.Context[field]; exists {
			return val, true
		}
		return nil, false
	}
}

// compareValues 比较值
func (bpe *BasicPolicyEngine) compareValues(a, b any) int {
	// 简化的值比较，实际实现应该处理更多类型
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)

	if aStr < bStr {
		return -1
	} else if aStr > bStr {
		return 1
	}
	return 0
}

// stringContains 字符串包含检查
func (bpe *BasicPolicyEngine) stringContains(a, b any) bool {
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)
	return strings.Contains(aStr, bStr)
}

// valueInList 值是否在列表中
func (bpe *BasicPolicyEngine) valueInList(value, list any) bool {
	valStr := fmt.Sprintf("%v", value)

	switch list := list.(type) {
	case []any:
		for _, item := range list {
			if fmt.Sprintf("%v", item) == valStr {
				return true
			}
		}
	case []string:
		for _, item := range list {
			if item == valStr {
				return true
			}
		}
	}

	return false
}

// regexMatch 正则匹配
func (bpe *BasicPolicyEngine) regexMatch(value, pattern any) bool {
	valStr := fmt.Sprintf("%v", value)
	patternStr := fmt.Sprintf("%v", pattern)

	matched, err := regexp.MatchString(patternStr, valStr)
	if err != nil {
		return false
	}

	return matched
}

// evaluateConditions 评估条件
func (bpe *BasicPolicyEngine) evaluateConditions(conditions []PolicyCondition, request *PolicyRequest) (float64, []string) {
	var score float64
	var triggeredConditions []string

	for _, condition := range conditions {
		if !condition.Enabled {
			continue
		}

		if bpe.evaluateCondition(condition, request) {
			score += 5.0 // 每个匹配的条件增加5分风险
			triggeredConditions = append(triggeredConditions, condition.ID)
		}
	}

	return score, triggeredConditions
}

// evaluateCondition 评估单个条件
func (bpe *BasicPolicyEngine) evaluateCondition(condition PolicyCondition, request *PolicyRequest) bool {
	// 简化的条件评估
	value, exists := bpe.getFieldValue(request, condition.Field)
	if !exists {
		return false
	}

	switch condition.Operator {
	case ConditionOperatorExists:
		return exists
	case ConditionOperatorEquals:
		return bpe.compareValues(value, condition.Value) == 0
	default:
		return false
	}
}

// determineRiskLevel 确定风险级别
func (bpe *BasicPolicyEngine) determineRiskLevel(score float64) RiskLevel {
	if score < 20 {
		return RiskLevelLow
	} else if score < 50 {
		return RiskLevelMedium
	} else if score < 80 {
		return RiskLevelHigh
	}
	return RiskLevelCritical
}

// containsPolicy 检查字符串是否在数组中
func containsPolicy(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
