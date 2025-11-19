package security

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// AccessController 访问控制器
type AccessController struct {
	// 核心存储
	roles       map[string]*Role
	permissions map[string]*Permission
	policies    map[string]*AccessPolicy
	users       map[string]*User
	sessions    map[string]*Session

	// 索引
	userRoles       map[string][]string        // userID -> roleIDs
	rolePermissions map[string][]string        // roleID -> permissionIDs
	userPermissions map[string]map[string]bool // userID -> permissionID -> granted

	// 锁
	mu sync.RWMutex

	// 配置
	config *AccessControlConfig

	// 审计
	auditLog AuditLog

	// 缓存
	cache *AccessCache
}

// AccessControlConfig 访问控制配置
type AccessControlConfig struct {
	// 会话配置
	SessionTimeout     time.Duration `json:"session_timeout"`
	MaxSessionsPerUser int           `json:"max_sessions_per_user"`
	EnableSessionCache bool          `json:"enable_session_cache"`

	// 权限配置
	EnablePermissionCache  bool          `json:"enable_permission_cache"`
	PermissionCacheTimeout time.Duration `json:"permission_cache_timeout"`

	// 审计配置
	EnableAudit         bool       `json:"enable_audit"`
	AuditLevel          AuditLevel `json:"audit_level"`
	LoginFailureLockout bool       `json:"login_failure_lockout"`
	MaxFailedAttempts   int        `json:"max_failed_attempts"`

	// 安全配置
	PasswordPolicy   *PasswordPolicy   `json:"password_policy"`
	MFAPolicy        *MFAPolicy        `json:"mfa_policy"`
	IPLockdownPolicy *IPLockdownPolicy `json:"ip_lockdown_policy"`
}

// PasswordPolicy 密码策略
type PasswordPolicy struct {
	MinLength        int           `json:"min_length"`
	RequireUppercase bool          `json:"require_uppercase"`
	RequireLowercase bool          `json:"require_lowercase"`
	RequireNumbers   bool          `json:"require_numbers"`
	RequireSymbols   bool          `json:"require_symbols"`
	MaxAge           time.Duration `json:"max_age"`
	HistoryCount     int           `json:"history_count"`
	PreventReuse     bool          `json:"prevent_reuse"`
}

// MFAPolicy 多因素认证策略
type MFAPolicy struct {
	Enabled         bool        `json:"enabled"`
	RequiredRoles   []string    `json:"required_roles"`
	RequiredActions []string    `json:"required_actions"`
	Methods         []MFAMethod `json:"methods"`
	BackupMethods   []MFAMethod `json:"backup_methods"`
}

// MFAMethod 多因素认证方法
type MFAMethod string

const (
	MFAMethodTOTP      MFAMethod = "totp"      // 时间动态口令
	MFAMethodSMS       MFAMethod = "sms"       // 短信验证
	MFAMethodEmail     MFAMethod = "email"     // 邮件验证
	MFAMethodHardware  MFAMethod = "hardware"  // 硬件令牌
	MFAMethodBiometric MFAMethod = "biometric" // 生物识别
)

// IPLockdownPolicy IP锁定策略
type IPLockdownPolicy struct {
	Enabled         bool     `json:"enabled"`
	AllowedIPs      []string `json:"allowed_ips"`
	BlockedIPs      []string `json:"blocked_ips"`
	TrustedNetworks []string `json:"trusted_networks"`
	RequireVPN      bool     `json:"require_vpn"`
}

// Role 角色
type Role struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Permissions []string               `json:"permissions"`
	Parents     []string               `json:"parents"` // 父角色，继承权限
	Attributes  map[string]interface{} `json:"attributes"`
	Enabled     bool                   `json:"enabled"`
	Priority    int                    `json:"priority"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	CreatedBy   string                 `json:"created_by"`
	UpdatedBy   string                 `json:"updated_by"`
}

// Permission 权限
type Permission struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Resource    string                 `json:"resource"`   // 资源类型
	Action      string                 `json:"action"`     // 操作类型
	Conditions  []PermissionCondition  `json:"conditions"` // 权限条件
	Attributes  map[string]interface{} `json:"attributes"`
	Enabled     bool                   `json:"enabled"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// PermissionCondition 权限条件
type PermissionCondition struct {
	Type        string      `json:"type"`     // 条件类型
	Field       string      `json:"field"`    // 字段名
	Operator    string      `json:"operator"` // 操作符
	Value       interface{} `json:"value"`    // 条件值
	Description string      `json:"description"`
}

// AccessPolicy 访问策略
type AccessPolicy struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Type        PolicyType        `json:"type"`       // 策略类型
	Effect      PolicyEffect      `json:"effect"`     // 策略效果
	Principal   string            `json:"principal"`  // 主体
	Resource    string            `json:"resource"`   // 资源
	Action      string            `json:"action"`     // 操作
	Conditions  []PolicyCondition `json:"conditions"` // 策略条件
	Priority    int               `json:"priority"`   // 优先级
	Enabled     bool              `json:"enabled"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// PolicyType 策略类型
type PolicyType string

const (
	PolicyTypeAllow PolicyType = "allow" // 允许策略
	PolicyTypeDeny  PolicyType = "deny"  // 拒绝策略
)

// PolicyEffect 策略效果
type PolicyEffect string

const (
	PolicyEffectAllow PolicyEffect = "Allow" // 允许
	PolicyEffectDeny  PolicyEffect = "Deny"  // 拒绝
)

// User 用户
type User struct {
	ID         string                 `json:"id"`
	Username   string                 `json:"username"`
	Email      string                 `json:"email"`
	FullName   string                 `json:"full_name"`
	Roles      []string               `json:"roles"`
	Attributes map[string]interface{} `json:"attributes"`
	Status     UserStatus             `json:"status"`
	Enabled    bool                   `json:"enabled"`
	LastLogin  *time.Time             `json:"last_login"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

// UserStatus 用户状态
type UserStatus string

const (
	UserStatusActive    UserStatus = "active"    // 活跃
	UserStatusInactive  UserStatus = "inactive"  // 非活跃
	UserStatusSuspended UserStatus = "suspended" // 暂停
	UserStatusLocked    UserStatus = "locked"    // 锁定
)

// Session 会话
type Session struct {
	ID           string                 `json:"id"`
	UserID       string                 `json:"user_id"`
	Username     string                 `json:"username"`
	Roles        []string               `json:"roles"`
	Permissions  []string               `json:"permissions"`
	IPAddress    string                 `json:"ip_address"`
	UserAgent    string                 `json:"user_agent"`
	Attributes   map[string]interface{} `json:"attributes"`
	Status       SessionStatus          `json:"status"`
	CreatedAt    time.Time              `json:"created_at"`
	LastActivity time.Time              `json:"last_activity"`
	ExpiresAt    time.Time              `json:"expires_at"`
}

// SessionStatus 会话状态
type SessionStatus string

const (
	SessionStatusActive   SessionStatus = "active"   // 活跃
	SessionStatusExpired  SessionStatus = "expired"  // 已过期
	SessionStatusRevoked  SessionStatus = "revoked"  // 已撤销
	SessionStatusInactive SessionStatus = "inactive" // 非活跃
)

// AccessDecision 访问决策
type AccessDecision struct {
	Allowed      bool                   `json:"allowed"`
	Effect       PolicyEffect           `json:"effect"`
	Reason       string                 `json:"reason"`
	Policies     []string               `json:"policies"`    // 影响决策的策略ID
	Roles        []string               `json:"roles"`       // 相关角色
	Permissions  []string               `json:"permissions"` // 相关权限
	CacheHit     bool                   `json:"cache_hit"`
	DecisionTime time.Duration          `json:"decision_time"`
	EvaluatedAt  time.Time              `json:"evaluated_at"`
	Context      map[string]interface{} `json:"context"`
}

// AccessRequest 访问请求
type AccessRequest struct {
	UserID      string                 `json:"user_id"`
	Username    string                 `json:"username"`
	Resource    string                 `json:"resource"`
	Action      string                 `json:"action"`
	Context     map[string]interface{} `json:"context"`
	IPAddress   string                 `json:"ip_address"`
	UserAgent   string                 `json:"user_agent"`
	SessionID   string                 `json:"session_id"`
	RequestTime time.Time              `json:"request_time"`
}

// AccessCache 访问缓存
type AccessCache struct {
	entries map[string]*CacheEntry
	mu      sync.RWMutex
	ttl     time.Duration
}

// CacheEntry 缓存条目
type CacheEntry struct {
	decision  *AccessDecision
	expiredAt time.Time
	createdAt time.Time
}

// NewAccessController 创建访问控制器
func NewAccessController(config *AccessControlConfig, auditLog AuditLog) *AccessController {
	if config == nil {
		config = &AccessControlConfig{
			SessionTimeout:         time.Hour * 24,
			MaxSessionsPerUser:     5,
			EnableSessionCache:     true,
			EnablePermissionCache:  true,
			PermissionCacheTimeout: time.Minute * 15,
			EnableAudit:            true,
			AuditLevel:             AuditLevelBasic,
			LoginFailureLockout:    true,
			MaxFailedAttempts:      5,
		}
	}

	ac := &AccessController{
		roles:           make(map[string]*Role),
		permissions:     make(map[string]*Permission),
		policies:        make(map[string]*AccessPolicy),
		users:           make(map[string]*User),
		sessions:        make(map[string]*Session),
		userRoles:       make(map[string][]string),
		rolePermissions: make(map[string][]string),
		userPermissions: make(map[string]map[string]bool),
		config:          config,
		auditLog:        auditLog,
	}

	// 初始化缓存
	if config.EnablePermissionCache {
		ac.cache = &AccessCache{
			entries: make(map[string]*CacheEntry),
			ttl:     config.PermissionCacheTimeout,
		}
	}

	// 启动清理协程
	go ac.startCleanupWorker()

	return ac
}

// CreateUser 创建用户
func (ac *AccessController) CreateUser(user *User) error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if user.ID == "" {
		return fmt.Errorf("user ID is required")
	}

	if _, exists := ac.users[user.ID]; exists {
		return fmt.Errorf("user %s already exists", user.ID)
	}

	if user.Username == "" {
		return fmt.Errorf("username is required")
	}

	// 检查用户名是否已存在
	for _, existingUser := range ac.users {
		if existingUser.Username == user.Username {
			return fmt.Errorf("username %s already exists", user.Username)
		}
	}

	// 设置默认值
	if user.Status == "" {
		user.Status = UserStatusActive
	}
	user.Enabled = true
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	ac.users[user.ID] = user

	// 记录审计日志
	if ac.config.EnableAudit && ac.auditLog != nil {
		_ = ac.auditLog.LogEvent(AuditEvent{
			Type:      AuditTypeUserCreated,
			UserID:    user.ID,
			Timestamp: time.Now(),
			Message:   fmt.Sprintf("User %s (%s) created", user.ID, user.Username),
		})
	}

	return nil
}

// GetUser 获取用户
func (ac *AccessController) GetUser(userID string) (*User, error) {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	user, exists := ac.users[userID]
	if !exists {
		return nil, fmt.Errorf("user %s not found", userID)
	}

	return user, nil
}

// UpdateUser 更新用户
func (ac *AccessController) UpdateUser(user *User) error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	existing, exists := ac.users[user.ID]
	if !exists {
		return fmt.Errorf("user %s not found", user.ID)
	}

	user.CreatedAt = existing.CreatedAt
	user.UpdatedAt = time.Now()

	ac.users[user.ID] = user

	// 记录审计日志
	if ac.config.EnableAudit && ac.auditLog != nil {
		_ = ac.auditLog.LogEvent(AuditEvent{
			Type:      AuditTypeUserUpdated,
			UserID:    user.ID,
			Timestamp: time.Now(),
			Message:   fmt.Sprintf("User %s (%s) updated", user.ID, user.Username),
		})
	}

	return nil
}

// DeleteUser 删除用户
func (ac *AccessController) DeleteUser(userID string) error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	user, exists := ac.users[userID]
	if !exists {
		return fmt.Errorf("user %s not found", userID)
	}

	delete(ac.users, userID)
	delete(ac.userRoles, userID)
	delete(ac.userPermissions, userID)

	// 删除用户的所有会话
	for sessionID, session := range ac.sessions {
		if session.UserID == userID {
			delete(ac.sessions, sessionID)
		}
	}

	// 记录审计日志
	if ac.config.EnableAudit && ac.auditLog != nil {
		_ = ac.auditLog.LogEvent(AuditEvent{
			Type:      AuditTypeUserDeleted,
			UserID:    userID,
			Timestamp: time.Now(),
			Message:   fmt.Sprintf("User %s (%s) deleted", user.ID, user.Username),
		})
	}

	return nil
}

// AssignRole 为用户分配角色
func (ac *AccessController) AssignRole(userID, roleID string) error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	user, exists := ac.users[userID]
	if !exists {
		return fmt.Errorf("user %s not found", userID)
	}

	role, exists := ac.roles[roleID]
	if !exists {
		return fmt.Errorf("role %s not found", roleID)
	}

	// 添加到用户的角色列表
	if !contains(user.Roles, roleID) {
		user.Roles = append(user.Roles, roleID)
		user.UpdatedAt = time.Now()
	}

	// 更新用户角色索引
	if _, exists := ac.userRoles[userID]; !exists {
		ac.userRoles[userID] = []string{}
	}
	if !contains(ac.userRoles[userID], roleID) {
		ac.userRoles[userID] = append(ac.userRoles[userID], roleID)
	}

	// 清除用户权限缓存
	delete(ac.userPermissions, userID)

	// 记录审计日志
	if ac.config.EnableAudit && ac.auditLog != nil {
		_ = ac.auditLog.LogEvent(AuditEvent{
			Type:      AuditTypeRoleAssigned,
			UserID:    userID,
			Timestamp: time.Now(),
			Message:   fmt.Sprintf("Role %s assigned to user %s", role.Name, user.Username),
			Metadata: map[string]interface{}{
				"role_id":   roleID,
				"role_name": role.Name,
			},
		})
	}

	return nil
}

// RevokeRole 撤销用户角色
func (ac *AccessController) RevokeRole(userID, roleID string) error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	user, exists := ac.users[userID]
	if !exists {
		return fmt.Errorf("user %s not found", userID)
	}

	// 从用户的角色列表中移除
	user.Roles = removeString(user.Roles, roleID)
	user.UpdatedAt = time.Now()

	// 更新用户角色索引
	if userRoles, exists := ac.userRoles[userID]; exists {
		ac.userRoles[userID] = removeString(userRoles, roleID)
	}

	// 清除用户权限缓存
	delete(ac.userPermissions, userID)

	// 记录审计日志
	if ac.config.EnableAudit && ac.auditLog != nil {
		_ = ac.auditLog.LogEvent(AuditEvent{
			Type:      AuditTypeRoleRevoked,
			UserID:    userID,
			Timestamp: time.Now(),
			Message:   fmt.Sprintf("Role %s revoked from user %s", roleID, user.Username),
			Metadata: map[string]interface{}{
				"role_id": roleID,
			},
		})
	}

	return nil
}

// CheckPermission 检查用户权限
func (ac *AccessController) CheckPermission(userID, resource, action string, context map[string]interface{}) (*AccessDecision, error) {
	start := time.Now()

	// 检查缓存
	cacheKey := ac.generateCacheKey(userID, resource, action, context)
	if ac.cache != nil {
		if entry := ac.cache.get(cacheKey); entry != nil {
			decision := entry.decision
			decision.CacheHit = true
			decision.DecisionTime = time.Since(start)
			return decision, nil
		}
	}

	ac.mu.RLock()
	defer ac.mu.RUnlock()

	user, exists := ac.users[userID]
	if !exists {
		return &AccessDecision{
			Allowed:      false,
			Effect:       PolicyEffectDeny,
			Reason:       "User not found",
			DecisionTime: time.Since(start),
			EvaluatedAt:  time.Now(),
			Context:      context,
		}, nil
	}

	if !user.Enabled || user.Status != UserStatusActive {
		return &AccessDecision{
			Allowed:      false,
			Effect:       PolicyEffectDeny,
			Reason:       "User is disabled or inactive",
			DecisionTime: time.Since(start),
			EvaluatedAt:  time.Now(),
			Context:      context,
		}, nil
	}

	decision := &AccessDecision{
		Allowed:      false,
		Effect:       PolicyEffectDeny,
		Reason:       "Access denied",
		Roles:        user.Roles,
		DecisionTime: time.Since(start),
		EvaluatedAt:  time.Now(),
		Context:      context,
	}

	// 1. 检查显式策略
	policyDecision := ac.evaluatePolicies(userID, resource, action, context)
	if policyDecision != nil {
		decision.Allowed = policyDecision.Allowed
		decision.Effect = policyDecision.Effect
		decision.Reason = policyDecision.Reason
		decision.Policies = policyDecision.Policies
	} else {
		// 2. 检查基于角色的权限
		permissionIDs := ac.getUserPermissions(userID)
		hasPermission := ac.checkPermissionList(permissionIDs, resource, action, context)

		decision.Allowed = hasPermission
		if hasPermission {
			decision.Effect = PolicyEffectAllow
			decision.Reason = "Permission granted via role-based access control"
			decision.Permissions = permissionIDs
		} else {
			decision.Reason = "No matching permissions found"
		}
	}

	// 缓存决策结果
	if ac.cache != nil {
		ac.cache.set(cacheKey, decision, ac.cache.ttl)
	}

	// 记录审计日志
	if ac.config.EnableAudit && ac.auditLog != nil {
		_ = ac.auditLog.LogEvent(AuditEvent{
			Type:      AuditTypeAccessChecked,
			UserID:    userID,
			Timestamp: time.Now(),
			Message:   fmt.Sprintf("Access check for %s:%s - %s", resource, action, decision.Effect),
			Metadata: map[string]interface{}{
				"resource":      resource,
				"action":        action,
				"allowed":       decision.Allowed,
				"decision_time": decision.DecisionTime,
			},
		})
	}

	return decision, nil
}

// evaluatePolicies 评估策略
func (ac *AccessController) evaluatePolicies(userID, resource, action string, context map[string]interface{}) *AccessDecision {
	var applicablePolicies []*AccessPolicy

	// 收集适用的策略
	for _, policy := range ac.policies {
		if !policy.Enabled {
			continue
		}

		if ac.policyApplies(policy, userID, resource, action, context) {
			applicablePolicies = append(applicablePolicies, policy)
		}
	}

	if len(applicablePolicies) == 0 {
		return nil
	}

	// 按优先级排序
	sortedPolicies := ac.sortPoliciesByPriority(applicablePolicies)

	// 应用策略
	for _, policy := range sortedPolicies {
		if ac.evaluatePolicyConditions(policy.Conditions, userID, resource, action, context) {
			return &AccessDecision{
				Allowed:  policy.Effect == PolicyEffectAllow,
				Effect:   policy.Effect,
				Reason:   fmt.Sprintf("Policy %s (%s) applies", policy.ID, policy.Name),
				Policies: []string{policy.ID},
			}
		}
	}

	return nil
}

// policyApplies 检查策略是否适用
func (ac *AccessController) policyApplies(policy *AccessPolicy, userID, resource, action string, context map[string]interface{}) bool {
	// 检查主体
	if policy.Principal != userID && policy.Principal != "*" {
		return false
	}

	// 检查资源
	if policy.Resource != "*" && policy.Resource != resource {
		return false
	}

	// 检查操作
	if policy.Action != "*" && policy.Action != action {
		return false
	}

	return true
}

// evaluatePolicyConditions 评估策略条件
func (ac *AccessController) evaluatePolicyConditions(conditions []PolicyCondition, userID, resource, action string, context map[string]interface{}) bool {
	for _, condition := range conditions {
		if !ac.evaluateCondition(condition, userID, resource, action, context) {
			return false
		}
	}
	return true
}

// evaluateCondition 评估单个条件
func (ac *AccessController) evaluateCondition(condition PolicyCondition, userID, resource, action string, context map[string]interface{}) bool {
	// 简化的条件评估
	switch condition.Field {
	case "user.id":
		return ac.compareValues(userID, string(condition.Operator), condition.Value)
	case "resource":
		return ac.compareValues(resource, string(condition.Operator), condition.Value)
	case "action":
		return ac.compareValues(action, string(condition.Operator), condition.Value)
	default:
		// 从context中获取值
		if val, exists := context[condition.Field]; exists {
			return ac.compareValues(val, string(condition.Operator), condition.Value)
		}
		return false
	}
}

// compareValues 比较值
func (ac *AccessController) compareValues(actual interface{}, operator string, expected interface{}) bool {
	actualStr := fmt.Sprintf("%v", actual)
	expectedStr := fmt.Sprintf("%v", expected)

	switch operator {
	case "eq":
		return actualStr == expectedStr
	case "ne":
		return actualStr != expectedStr
	case "in":
		return ac.valueInList(actualStr, expected)
	default:
		return false
	}
}

// valueInList 检查值是否在列表中
func (ac *AccessController) valueInList(value string, list interface{}) bool {
	switch list := list.(type) {
	case []string:
		for _, item := range list {
			if item == value {
				return true
			}
		}
	case []interface{}:
		for _, item := range list {
			if fmt.Sprintf("%v", item) == value {
				return true
			}
		}
	}
	return false
}

// sortPoliciesByPriority 按优先级排序策略
func (ac *AccessController) sortPoliciesByPriority(policies []*AccessPolicy) []*AccessPolicy {
	// 简单排序，优先级数值越小优先级越高
	sorted := make([]*AccessPolicy, len(policies))
	copy(sorted, policies)

	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i].Priority > sorted[j].Priority {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	return sorted
}

// getUserPermissions 获取用户的所有权限
func (ac *AccessController) getUserPermissions(userID string) []string {
	// 检查缓存
	if permissions, exists := ac.userPermissions[userID]; exists {
		var permissionIDs []string
		for permID := range permissions {
			permissionIDs = append(permissionIDs, permID)
		}
		return permissionIDs
	}

	permissionSet := make(map[string]bool)
	userRoles := ac.userRoles[userID]

	// 获取角色的权限
	for _, roleID := range userRoles {
		rolePerms := ac.rolePermissions[roleID]
		for _, permID := range rolePerms {
			permissionSet[permID] = true
		}

		// 递归获取父角色权限
		role, exists := ac.roles[roleID]
		if exists {
			parentPermissions := ac.getParentRolePermissions(role)
			for _, permID := range parentPermissions {
				permissionSet[permID] = true
			}
		}
	}

	// 缓存结果
	ac.userPermissions[userID] = permissionSet

	var permissionIDs []string
	for permID := range permissionSet {
		permissionIDs = append(permissionIDs, permID)
	}

	return permissionIDs
}

// getParentRolePermissions 获取父角色权限
func (ac *AccessController) getParentRolePermissions(role *Role) []string {
	var permissions []string

	for _, parentID := range role.Parents {
		if parentRole, exists := ac.roles[parentID]; exists {
			// 获取父角色的直接权限
			permissions = append(permissions, parentRole.Permissions...)

			// 递归获取祖先角色的权限
			ancestorPermissions := ac.getParentRolePermissions(parentRole)
			permissions = append(permissions, ancestorPermissions...)
		}
	}

	return permissions
}

// checkPermissionList 检查权限列表
func (ac *AccessController) checkPermissionList(permissionIDs []string, resource, action string, context map[string]interface{}) bool {
	for _, permID := range permissionIDs {
		permission, exists := ac.permissions[permID]
		if !exists || !permission.Enabled {
			continue
		}

		// 检查资源和操作匹配
		if (permission.Resource == "*" || permission.Resource == resource) &&
			(permission.Action == "*" || permission.Action == action) {

			// 检查权限条件
			if ac.checkPermissionConditions(permission.Conditions, context) {
				return true
			}
		}
	}

	return false
}

// checkPermissionConditions 检查权限条件
func (ac *AccessController) checkPermissionConditions(conditions []PermissionCondition, context map[string]interface{}) bool {
	for _, condition := range conditions {
		if !ac.checkPermissionCondition(condition, context) {
			return false
		}
	}
	return true
}

// checkPermissionCondition 检查单个权限条件
func (ac *AccessController) checkPermissionCondition(condition PermissionCondition, context map[string]interface{}) bool {
	value, exists := context[condition.Field]
	if !exists {
		return false
	}

	return ac.compareValues(value, condition.Operator, condition.Value)
}

// CreateSession 创建会话
func (ac *AccessController) CreateSession(userID, ipAddress, userAgent string) (*Session, error) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	user, exists := ac.users[userID]
	if !exists {
		return nil, fmt.Errorf("user %s not found", userID)
	}

	if !user.Enabled || user.Status != UserStatusActive {
		return nil, fmt.Errorf("user %s is not active", userID)
	}

	// 检查会话数量限制
	if ac.config.MaxSessionsPerUser > 0 {
		activeSessions := 0
		for _, session := range ac.sessions {
			if session.UserID == userID && session.Status == SessionStatusActive {
				activeSessions++
			}
		}
		if activeSessions >= ac.config.MaxSessionsPerUser {
			return nil, fmt.Errorf("user %s has reached maximum sessions limit", userID)
		}
	}

	sessionID := ac.generateSessionID(userID, ipAddress, userAgent)
	expiresAt := time.Now().Add(ac.config.SessionTimeout)

	session := &Session{
		ID:           sessionID,
		UserID:       userID,
		Username:     user.Username,
		Roles:        user.Roles,
		Permissions:  ac.getUserPermissions(userID),
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		Status:       SessionStatusActive,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		ExpiresAt:    expiresAt,
	}

	ac.sessions[sessionID] = session

	// 更新用户最后登录时间
	now := time.Now()
	user.LastLogin = &now
	user.UpdatedAt = now

	// 记录审计日志
	if ac.config.EnableAudit && ac.auditLog != nil {
		_ = ac.auditLog.LogEvent(AuditEvent{
			Type:      AuditTypeSessionCreated,
			UserID:    userID,
			Timestamp: time.Now(),
			Message:   fmt.Sprintf("Session %s created for user %s", sessionID, user.Username),
			Metadata: map[string]interface{}{
				"session_id": sessionID,
				"ip_address": ipAddress,
				"user_agent": userAgent,
			},
		})
	}

	return session, nil
}

// GetSession 获取会话
func (ac *AccessController) GetSession(sessionID string) (*Session, error) {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	session, exists := ac.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	// 检查会话是否过期
	if time.Now().After(session.ExpiresAt) {
		session.Status = SessionStatusExpired
		return nil, fmt.Errorf("session %s has expired", sessionID)
	}

	return session, nil
}

// DeleteSession 删除会话
func (ac *AccessController) DeleteSession(sessionID string) error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	session, exists := ac.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	session.Status = SessionStatusRevoked
	delete(ac.sessions, sessionID)

	// 记录审计日志
	if ac.config.EnableAudit && ac.auditLog != nil {
		_ = ac.auditLog.LogEvent(AuditEvent{
			Type:      AuditTypeSessionDeleted,
			UserID:    session.UserID,
			Timestamp: time.Now(),
			Message:   fmt.Sprintf("Session %s revoked for user %s", sessionID, session.Username),
			Metadata: map[string]interface{}{
				"session_id": sessionID,
			},
		})
	}

	return nil
}

// generateSessionID 生成会话ID
func (ac *AccessController) generateSessionID(userID, ipAddress, userAgent string) string {
	data := fmt.Sprintf("%s:%s:%s:%d", userID, ipAddress, userAgent, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// generateCacheKey 生成缓存键
func (ac *AccessController) generateCacheKey(userID, resource, action string, context map[string]interface{}) string {
	data := fmt.Sprintf("%s:%s:%s:%v", userID, resource, action, context)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// startCleanupWorker 启动清理工作协程
func (ac *AccessController) startCleanupWorker() {
	ticker := time.NewTicker(time.Minute * 10)
	defer ticker.Stop()

	for range ticker.C {
		ac.cleanupExpiredSessions()
		ac.cleanupExpiredCache()
	}
}

// cleanupExpiredSessions 清理过期会话
func (ac *AccessController) cleanupExpiredSessions() {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	now := time.Now()
	for sessionID, session := range ac.sessions {
		if now.After(session.ExpiresAt) {
			session.Status = SessionStatusExpired
			delete(ac.sessions, sessionID)
		}
	}
}

// cleanupExpiredCache 清理过期缓存
func (ac *AccessController) cleanupExpiredCache() {
	if ac.cache == nil {
		return
	}

	ac.cache.mu.Lock()
	defer ac.cache.mu.Unlock()

	now := time.Now()
	for key, entry := range ac.cache.entries {
		if now.After(entry.expiredAt) {
			delete(ac.cache.entries, key)
		}
	}
}

// AccessCache 缓存方法
func (cache *AccessCache) get(key string) *CacheEntry {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	entry, exists := cache.entries[key]
	if !exists || time.Now().After(entry.expiredAt) {
		return nil
	}

	return entry
}

func (cache *AccessCache) set(key string, decision *AccessDecision, ttl time.Duration) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	cache.entries[key] = &CacheEntry{
		decision:  decision,
		expiredAt: time.Now().Add(ttl),
		createdAt: time.Now(),
	}
}

// 辅助函数
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func removeString(slice []string, item string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}
