package security

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// AuditType 审计类型
type AuditType string

const (
	// 用户相关审计事件
	AuditTypeUserCreated  AuditType = "user_created"
	AuditTypeUserUpdated  AuditType = "user_updated"
	AuditTypeUserDeleted  AuditType = "user_deleted"
	AuditTypeUserLogin    AuditType = "user_login"
	AuditTypeUserLogout   AuditType = "user_logout"
	AuditTypeUserLocked   AuditType = "user_locked"
	AuditTypeUserUnlocked AuditType = "user_unlocked"

	// 角色和权限相关审计事件
	AuditTypeRoleCreated       AuditType = "role_created"
	AuditTypeRoleUpdated       AuditType = "role_updated"
	AuditTypeRoleDeleted       AuditType = "role_deleted"
	AuditTypePermissionCreated AuditType = "permission_created"
	AuditTypePermissionUpdated AuditType = "permission_updated"
	AuditTypePermissionDeleted AuditType = "permission_deleted"
	AuditTypeRoleAssigned      AuditType = "role_assigned"
	AuditTypeRoleRevoked       AuditType = "role_revoked"

	// 会话相关审计事件
	AuditTypeSessionCreated AuditType = "session_created"
	AuditTypeSessionUpdated AuditType = "session_updated"
	AuditTypeSessionDeleted AuditType = "session_deleted"
	AuditTypeSessionExpired AuditType = "session_expired"

	// 访问相关审计事件
	AuditTypeAccessChecked AuditType = "access_checked"
	AuditTypeAccessGranted AuditType = "access_granted"
	AuditTypeAccessDenied  AuditType = "access_denied"

	// 策略相关审计事件
	AuditTypePolicyCreated AuditType = "policy_created"
	AuditTypePolicyUpdated AuditType = "policy_updated"
	AuditTypePolicyDeleted AuditType = "policy_deleted"
	AuditTypeUnauthorized  AuditType = "unauthorized"
	AuditTypeForbidden     AuditType = "forbidden"

	// 安全相关审计事件
	AuditTypeSecurityAlert      AuditType = "security_alert"
	AuditTypeSecurityViolation  AuditType = "security_violation"
	AuditTypeSuspiciousActivity AuditType = "suspicious_activity"
	AuditTypeAttackDetected     AuditType = "attack_detected"
	AuditTypeDataBreach         AuditType = "data_breach"

	// 系统相关审计事件
	AuditTypeSystemStarted        AuditType = "system_started"
	AuditTypeSystemShutdown       AuditType = "system_shutdown"
	AuditTypeConfigurationChanged AuditType = "configuration_changed"
	AuditTypeError                AuditType = "error"
)

// AuditEvent 审计事件
type AuditEvent struct {
	ID         string         `json:"id"`
	Type       AuditType      `json:"type"`
	Timestamp  time.Time      `json:"timestamp"`
	Severity   AuditSeverity  `json:"severity"`
	Category   AuditCategory  `json:"category"`
	UserID     string         `json:"user_id,omitempty"`
	Username   string         `json:"username,omitempty"`
	AgentID    string         `json:"agent_id,omitempty"`
	SessionID  string         `json:"session_id,omitempty"`
	Resource   string         `json:"resource,omitempty"`
	Action     string         `json:"action,omitempty"`
	ObjectID   string         `json:"object_id,omitempty"`
	ObjectType string         `json:"object_type,omitempty"`
	IPAddress  string         `json:"ip_address,omitempty"`
	UserAgent  string         `json:"user_agent,omitempty"`
	Location   string         `json:"location,omitempty"`
	Result     AuditResult    `json:"result,omitempty"`
	Message    string         `json:"message"`
	Details    string         `json:"details,omitempty"`
	Duration   time.Duration  `json:"duration,omitempty"`
	RequestID  string         `json:"request_id,omitempty"`
	TraceID    string         `json:"trace_id,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	RiskScore  float64        `json:"risk_score,omitempty"`
	Tags       []string       `json:"tags,omitempty"`
}

// AuditSeverity 审计严重级别
type AuditSeverity string

const (
	AuditSeverityInfo     AuditSeverity = "info"     // 信息
	AuditSeverityLow      AuditSeverity = "low"      // 低风险
	AuditSeverityMedium   AuditSeverity = "medium"   // 中等风险
	AuditSeverityHigh     AuditSeverity = "high"     // 高风险
	AuditSeverityCritical AuditSeverity = "critical" // 严重
)

// AuditCategory 审计类别
type AuditCategory string

const (
	AuditCategoryAuthentication AuditCategory = "authentication" // 认证
	AuditCategoryAuthorization  AuditCategory = "authorization"  // 授权
	AuditCategoryAccess         AuditCategory = "access"         // 访问
	AuditCategoryConfiguration  AuditCategory = "configuration"  // 配置
	AuditCategorySecurity       AuditCategory = "security"       // 安全
	AuditCategorySystem         AuditCategory = "system"         // 系统
	AuditCategoryData           AuditCategory = "data"           // 数据
	AuditCategoryNetwork        AuditCategory = "network"        // 网络
)

// AuditResultStatus 审计结果状态
type AuditResultStatus string

const (
	AuditResultSuccess AuditResultStatus = "success" // 成功
	AuditResultFailure AuditResultStatus = "failure" // 失败
	AuditResultError   AuditResultStatus = "error"   // 错误
	AuditResultPartial AuditResultStatus = "partial" // 部分
)

// AuditLog 审计日志接口
type AuditLog interface {
	// 基础操作
	LogEvent(event AuditEvent) error
	LogEventAsync(event AuditEvent) error
	LogEvents(events []AuditEvent) error

	// 查询操作
	QueryEvents(ctx context.Context, query *AuditQuery) (*AuditResult, error)
	GetEvent(eventID string) (*AuditEvent, error)
	GetEventsByUser(userID string, limit int) ([]*AuditEvent, error)
	GetEventsByType(eventType AuditType, limit int) ([]*AuditEvent, error)
	GetEventsByTimeRange(start, end time.Time, limit int) ([]*AuditEvent, error)

	// 统计操作
	GetStatistics(ctx context.Context, filters *AuditFilters) (*AuditStatistics, error)
	GetEventSummary(timeRange TimeRange) (*EventSummary, error)

	// 管理操作
	ArchiveEvents(ctx context.Context, before time.Time) (int64, error)
	PurgeEvents(ctx context.Context, before time.Time) (int64, error)
	ExportEvents(ctx context.Context, query *AuditQuery, format ExportFormat) ([]byte, error)

	// 配置和状态
	GetConfiguration() *AuditConfiguration
	UpdateConfiguration(config *AuditConfiguration) error
	GetStatus() *AuditLogStatus
	Close() error
}

// AuditQuery 审计查询
type AuditQuery struct {
	// 时间范围
	TimeRange *TimeRange `json:"time_range,omitempty"`

	// 过滤条件
	Types      []AuditType         `json:"types,omitempty"`
	Severities []AuditSeverity     `json:"severities,omitempty"`
	Categories []AuditCategory     `json:"categories,omitempty"`
	Users      []string            `json:"users,omitempty"`
	Resources  []string            `json:"resources,omitempty"`
	Actions    []string            `json:"actions,omitempty"`
	Results    []AuditResultStatus `json:"results,omitempty"`

	// 文本搜索
	SearchText string `json:"search_text,omitempty"`

	// 分页和排序
	Limit     int    `json:"limit"`
	Offset    int    `json:"offset"`
	OrderBy   string `json:"order_by"`   // 排序字段
	OrderDesc bool   `json:"order_desc"` // 是否降序

	// 元数据过滤
	MetadataFilters map[string]any `json:"metadata_filters,omitempty"`

	// 风险评分范围
	RiskScoreMin *float64 `json:"risk_score_min,omitempty"`
	RiskScoreMax *float64 `json:"risk_score_max,omitempty"`
}

// AuditResult 审计查询结果
type AuditResult struct {
	Events    []*AuditEvent `json:"events"`
	Total     int64         `json:"total"`
	Offset    int           `json:"offset"`
	Limit     int           `json:"limit"`
	HasMore   bool          `json:"has_more"`
	QueryTime time.Duration `json:"query_time"`
}

// AuditFilters 审计过滤条件
type AuditFilters struct {
	TimeRange  *TimeRange      `json:"time_range,omitempty"`
	Types      []AuditType     `json:"types,omitempty"`
	Users      []string        `json:"users,omitempty"`
	Resources  []string        `json:"resources,omitempty"`
	Severities []AuditSeverity `json:"severities,omitempty"`
}

// AuditStatistics 审计统计
type AuditStatistics struct {
	TimeRange      TimeRange           `json:"time_range"`
	TotalEvents    int64               `json:"total_events"`
	EventsByType   map[AuditType]int64 `json:"events_by_type"`
	EventsByUser   map[string]int64    `json:"events_by_user"`
	EventsByHour   map[int]int64       `json:"events_by_hour"` // 小时统计
	EventsByDay    map[string]int64    `json:"events_by_day"`  // 日期统计
	TopUsers       []UserStat          `json:"top_users"`
	TopResources   []ResourceStat      `json:"top_resources"`
	TopActions     []ActionStat        `json:"top_actions"`
	SecurityEvents SecurityEventStats  `json:"security_events"`
	AccessEvents   AccessEventStats    `json:"access_events"`
	GeneratedAt    time.Time           `json:"generated_at"`
}

// UserStat 用户统计
type UserStat struct {
	UserID       string    `json:"user_id"`
	Username     string    `json:"username"`
	EventCount   int64     `json:"event_count"`
	LastActivity time.Time `json:"last_activity"`
}

// ResourceStat 资源统计
type ResourceStat struct {
	Resource   string `json:"resource"`
	EventCount int64  `json:"event_count"`
}

// ActionStat 操作统计
type ActionStat struct {
	Action     string `json:"action"`
	EventCount int64  `json:"event_count"`
}

// SecurityEventStats 安全事件统计
type SecurityEventStats struct {
	TotalAlerts     int64 `json:"total_alerts"`
	TotalViolations int64 `json:"total_violations"`
	TotalAttacks    int64 `json:"total_attacks"`
	HighRiskEvents  int64 `json:"high_risk_events"`
	CriticalEvents  int64 `json:"critical_events"`
}

// AccessEventStats 访问事件统计
type AccessEventStats struct {
	TotalAccessChecks int64 `json:"total_access_checks"`
	AccessGranted     int64 `json:"access_granted"`
	AccessDenied      int64 `json:"access_denied"`
	Unauthorized      int64 `json:"unauthorized"`
	Forbidden         int64 `json:"forbidden"`
}

// EventSummary 事件摘要
type EventSummary struct {
	TimeRange       TimeRange       `json:"time_range"`
	TotalEvents     int64           `json:"total_events"`
	KeyMetrics      map[string]any  `json:"key_metrics"`
	Trends          []TrendData     `json:"trends"`
	Alerts          []SecurityAlert `json:"alerts"`
	Recommendations []string        `json:"recommendations"`
}

// TrendData 趋势数据
type TrendData struct {
	Timestamp time.Time `json:"timestamp"`
	Value     int64     `json:"value"`
	Label     string    `json:"label,omitempty"`
}

// SecurityAlert 安全警报
type SecurityAlert struct {
	ID          string         `json:"id"`
	Type        AlertType      `json:"type"`
	Severity    AuditSeverity  `json:"severity"`
	Message     string         `json:"message"`
	Description string         `json:"description"`
	Events      []string       `json:"events"` // 相关事件ID
	DetectedAt  time.Time      `json:"detected_at"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// AlertType 警报类型
type AlertType string

const (
	AlertTypeSuspiciousLogin     AlertType = "suspicious_login"
	AlertTypeBruteForceAttack    AlertType = "brute_force_attack"
	AlertTypePrivilegeEscalation AlertType = "privilege_escalation"
	AlertTypeDataAccessAnomaly   AlertType = "data_access_anomaly"
	AlertTypeUnauthorizedAccess  AlertType = "unauthorized_access"
)

// ExportFormat 导出格式
type ExportFormat string

const (
	ExportFormatJSON ExportFormat = "json" // JSON格式
	ExportFormatCSV  ExportFormat = "csv"  // CSV格式
	ExportFormatXML  ExportFormat = "xml"  // XML格式
	ExportFormatPDF  ExportFormat = "pdf"  // PDF格式
)

// AuditConfiguration 审计配置
type AuditConfiguration struct {
	// 存储配置
	StorageType   StorageType   `json:"storage_type"`
	StoragePath   string        `json:"storage_path,omitempty"`
	MaxFileSize   int64         `json:"max_file_size"`
	MaxFileAge    time.Duration `json:"max_file_age"`
	Compression   bool          `json:"compression"`
	Encryption    bool          `json:"encryption"`
	EncryptionKey string        `json:"encryption_key,omitempty"`

	// 缓存配置
	EnableCache  bool          `json:"enable_cache"`
	CacheSize    int           `json:"cache_size"`
	CacheTimeout time.Duration `json:"cache_timeout"`

	// 索引配置
	EnableIndexing bool     `json:"enable_indexing"`
	IndexFields    []string `json:"index_fields"`

	// 性能配置
	WorkerPoolSize int           `json:"worker_pool_size"`
	BatchSize      int           `json:"batch_size"`
	FlushInterval  time.Duration `json:"flush_interval"`

	// 保留策略
	RetentionPolicy *RetentionPolicy `json:"retention_policy"`

	// 实时监控
	EnableRealTime  bool           `json:"enable_real_time"`
	AlertThresholds map[string]int `json:"alert_thresholds"`

	// 安全配置
	EnableSignature bool   `json:"enable_signature"`
	SignatureKey    string `json:"signature_key,omitempty"`
	EnableHash      bool   `json:"enable_hash"`
	HashAlgorithm   string `json:"hash_algorithm,omitempty"`
}

// StorageType 存储类型
type StorageType string

const (
	StorageTypeMemory   StorageType = "memory"   // 内存存储
	StorageTypeFile     StorageType = "file"     // 文件存储
	StorageTypeDatabase StorageType = "database" // 数据库存储
	StorageTypeElastic  StorageType = "elastic"  // Elasticsearch
)

// RetentionPolicy 保留策略
type RetentionPolicy struct {
	EnableAutoArchive bool          `json:"enable_auto_archive"`
	ArchiveAfter      time.Duration `json:"archive_after"`
	EnableAutoPurge   bool          `json:"enable_auto_purge"`
	PurgeAfter        time.Duration `json:"purge_after"`
	MinRetention      time.Duration `json:"min_retention"`
	MaxRetention      time.Duration `json:"max_retention"`
}

// AuditLogStatus 审计日志状态
type AuditLogStatus struct {
	Status           string         `json:"status"`
	Version          string         `json:"version"`
	Uptime           time.Duration  `json:"uptime"`
	TotalEvents      int64          `json:"total_events"`
	EventsPerSecond  float64        `json:"events_per_second"`
	StorageSize      int64          `json:"storage_size"`
	LastEventTime    time.Time      `json:"last_event_time"`
	ErrorCount       int64          `json:"error_count"`
	LastError        string         `json:"last_error,omitempty"`
	WorkerPoolStatus map[string]any `json:"worker_pool_status"`
	MemoryUsage      map[string]any `json:"memory_usage"`
}

// InMemoryAuditLog 内存审计日志实现
type InMemoryAuditLog struct {
	events    []AuditEvent
	mu        sync.RWMutex
	config    *AuditConfiguration
	status    *AuditLogStatus
	eventChan chan AuditEvent
	workers   int
	done      chan struct{}
}

// NewInMemoryAuditLog 创建内存审计日志
func NewInMemoryAuditLog(config *AuditConfiguration) *InMemoryAuditLog {
	if config == nil {
		config = &AuditConfiguration{
			StorageType:    StorageTypeMemory,
			EnableCache:    true,
			CacheSize:      10000,
			CacheTimeout:   time.Minute * 5,
			WorkerPoolSize: 5,
			BatchSize:      100,
			FlushInterval:  time.Second * 10,
		}
	}

	al := &InMemoryAuditLog{
		events:    make([]AuditEvent, 0),
		config:    config,
		status:    &AuditLogStatus{},
		eventChan: make(chan AuditEvent, config.BatchSize*10),
		workers:   config.WorkerPoolSize,
		done:      make(chan struct{}),
	}

	// 启动工作协程
	for range al.workers {
		go al.eventWorker()
	}

	return al
}

// LogEvent 记录事件
func (al *InMemoryAuditLog) LogEvent(event AuditEvent) error {
	// 设置默认值
	if event.ID == "" {
		event.ID = generateEventID()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	if event.Severity == "" {
		event.Severity = AuditSeverityInfo
	}

	// 同步写入
	al.mu.Lock()
	al.events = append(al.events, event)
	al.status.TotalEvents++
	al.status.LastEventTime = time.Now()
	al.mu.Unlock()

	return nil
}

// LogEventAsync 异步记录事件
func (al *InMemoryAuditLog) LogEventAsync(event AuditEvent) error {
	select {
	case al.eventChan <- event:
		return nil
	case <-time.After(time.Second):
		return errors.New("audit log buffer full, event dropped")
	}
}

// LogEvents 批量记录事件
func (al *InMemoryAuditLog) LogEvents(events []AuditEvent) error {
	for _, event := range events {
		if err := al.LogEventAsync(event); err != nil {
			return err
		}
	}
	return nil
}

// eventWorker 事件处理工作协程
func (al *InMemoryAuditLog) eventWorker() {
	for {
		select {
		case event := <-al.eventChan:
			if err := al.LogEvent(event); err != nil {
				al.mu.Lock()
				al.status.ErrorCount++
				al.status.LastError = err.Error()
				al.mu.Unlock()
			}
		case <-al.done:
			return
		}
	}
}

// QueryEvents 查询事件
func (al *InMemoryAuditLog) QueryEvents(ctx context.Context, query *AuditQuery) (*AuditResult, error) {
	start := time.Now()

	al.mu.RLock()
	defer al.mu.RUnlock()

	var filteredEvents []*AuditEvent
	for i := range al.events {
		event := &al.events[i]
		if al.matchesQuery(event, query) {
			filteredEvents = append(filteredEvents, event)
		}
	}

	// 应用排序
	if query.OrderBy != "" {
		al.sortEvents(filteredEvents, query.OrderBy, query.OrderDesc)
	}

	// 应用分页
	total := int64(len(filteredEvents))
	offset := query.Offset
	if offset < 0 {
		offset = 0
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 100
	}

	var events []*AuditEvent
	if offset < len(filteredEvents) {
		end := offset + limit
		if end > len(filteredEvents) {
			end = len(filteredEvents)
		}
		events = filteredEvents[offset:end]
	} else {
		events = []*AuditEvent{}
	}

	return &AuditResult{
		Events:    events,
		Total:     total,
		Offset:    offset,
		Limit:     limit,
		HasMore:   int64(offset+limit) < total,
		QueryTime: time.Since(start),
	}, nil
}

// matchesQuery 检查事件是否匹配查询条件
func (al *InMemoryAuditLog) matchesQuery(event *AuditEvent, query *AuditQuery) bool {
	// 时间范围过滤
	if query.TimeRange != nil {
		if event.Timestamp.Before(query.TimeRange.Start) || event.Timestamp.After(query.TimeRange.End) {
			return false
		}
	}

	// 类型过滤
	if len(query.Types) > 0 && !containsAuditType(query.Types, event.Type) {
		return false
	}

	// 严重级别过滤
	if len(query.Severities) > 0 && !containsAuditSeverity(query.Severities, event.Severity) {
		return false
	}

	// 用户过滤
	if len(query.Users) > 0 && !containsString(query.Users, event.UserID) {
		return false
	}

	// 资源过滤
	if len(query.Resources) > 0 && !containsString(query.Resources, event.Resource) {
		return false
	}

	// 操作过滤
	if len(query.Actions) > 0 && !containsString(query.Actions, event.Action) {
		return false
	}

	// 文本搜索
	if query.SearchText != "" {
		if !containsSearchText(event, query.SearchText) {
			return false
		}
	}

	return true
}

// sortEvents 排序事件
func (al *InMemoryAuditLog) sortEvents(events []*AuditEvent, orderBy string, desc bool) {
	// 简化排序实现，实际应用中应该使用更高效的排序算法
	if len(events) <= 1 {
		return
	}

	switch orderBy {
	case "timestamp":
		if desc {
			for i := range len(events) - 1 {
				for j := i + 1; j < len(events); j++ {
					if events[i].Timestamp.Before(events[j].Timestamp) {
						events[i], events[j] = events[j], events[i]
					}
				}
			}
		} else {
			for i := range len(events) - 1 {
				for j := i + 1; j < len(events); j++ {
					if events[i].Timestamp.After(events[j].Timestamp) {
						events[i], events[j] = events[j], events[i]
					}
				}
			}
		}
	}
}

// GetEvent 获取事件
func (al *InMemoryAuditLog) GetEvent(eventID string) (*AuditEvent, error) {
	al.mu.RLock()
	defer al.mu.RUnlock()

	for i := range al.events {
		if al.events[i].ID == eventID {
			return &al.events[i], nil
		}
	}

	return nil, fmt.Errorf("event %s not found", eventID)
}

// GetEventsByUser 根据用户获取事件
func (al *InMemoryAuditLog) GetEventsByUser(userID string, limit int) ([]*AuditEvent, error) {
	query := &AuditQuery{
		Users:     []string{userID},
		Limit:     limit,
		OrderBy:   "timestamp",
		OrderDesc: true,
	}

	result, err := al.QueryEvents(context.Background(), query)
	if err != nil {
		return nil, err
	}

	return result.Events, nil
}

// GetEventsByType 根据类型获取事件
func (al *InMemoryAuditLog) GetEventsByType(eventType AuditType, limit int) ([]*AuditEvent, error) {
	query := &AuditQuery{
		Types:     []AuditType{eventType},
		Limit:     limit,
		OrderBy:   "timestamp",
		OrderDesc: true,
	}

	result, err := al.QueryEvents(context.Background(), query)
	if err != nil {
		return nil, err
	}

	return result.Events, nil
}

// GetEventsByTimeRange 根据时间范围获取事件
func (al *InMemoryAuditLog) GetEventsByTimeRange(start, end time.Time, limit int) ([]*AuditEvent, error) {
	query := &AuditQuery{
		TimeRange: &TimeRange{Start: start, End: end},
		Limit:     limit,
		OrderBy:   "timestamp",
		OrderDesc: true,
	}

	result, err := al.QueryEvents(context.Background(), query)
	if err != nil {
		return nil, err
	}

	return result.Events, nil
}

// GetStatistics 获取统计信息
func (al *InMemoryAuditLog) GetStatistics(ctx context.Context, filters *AuditFilters) (*AuditStatistics, error) {
	al.mu.RLock()
	defer al.mu.RUnlock()

	stats := &AuditStatistics{
		TimeRange:    TimeRange{Start: time.Now(), End: time.Now()},
		EventsByType: make(map[AuditType]int64),
		EventsByUser: make(map[string]int64),
		EventsByHour: make(map[int]int64),
		EventsByDay:  make(map[string]int64),
		GeneratedAt:  time.Now(),
	}

	var timeRange *TimeRange
	if filters != nil && filters.TimeRange != nil {
		timeRange = filters.TimeRange
		stats.TimeRange = *timeRange
	}

	totalEvents := int64(0)
	for i := range al.events {
		event := &al.events[i]

		// 应用时间过滤
		if timeRange != nil {
			if event.Timestamp.Before(timeRange.Start) || event.Timestamp.After(timeRange.End) {
				continue
			}
		}

		// 应用其他过滤条件
		if filters != nil {
			if len(filters.Types) > 0 && !containsAuditType(filters.Types, event.Type) {
				continue
			}
			if len(filters.Users) > 0 && !containsString(filters.Users, event.UserID) {
				continue
			}
		}

		totalEvents++
		stats.EventsByType[event.Type]++
		stats.EventsByUser[event.UserID]++

		// 小时统计
		hour := event.Timestamp.Hour()
		stats.EventsByHour[hour]++

		// 日期统计
		day := event.Timestamp.Format("2006-01-02")
		stats.EventsByDay[day]++
	}

	stats.TotalEvents = totalEvents

	return stats, nil
}

// GetEventSummary 获取事件摘要
func (al *InMemoryAuditLog) GetEventSummary(timeRange TimeRange) (*EventSummary, error) {
	query := &AuditQuery{
		TimeRange: &timeRange,
	}

	result, err := al.QueryEvents(context.Background(), query)
	if err != nil {
		return nil, err
	}

	summary := &EventSummary{
		TimeRange:   timeRange,
		TotalEvents: result.Total,
		KeyMetrics:  make(map[string]any),
		Trends:      []TrendData{},
		Alerts:      []SecurityAlert{},
	}

	// 生成趋势数据
	trendData := make(map[string]int64)
	for _, event := range result.Events {
		hour := event.Timestamp.Truncate(time.Hour)
		key := hour.Format("2006-01-02 15:00")
		trendData[key]++
	}

	for timestamp, count := range trendData {
		t, _ := time.Parse("2006-01-02 15:00", timestamp)
		summary.Trends = append(summary.Trends, TrendData{
			Timestamp: t,
			Value:     count,
			Label:     timestamp,
		})
	}

	// 设置关键指标
	summary.KeyMetrics["events_per_hour"] = float64(result.Total) / timeRange.End.Sub(timeRange.Start).Hours()
	summary.KeyMetrics["unique_users"] = len(getUniqueUsers(result.Events))

	return summary, nil
}

// ArchiveEvents 归档事件
func (al *InMemoryAuditLog) ArchiveEvents(ctx context.Context, before time.Time) (int64, error) {
	al.mu.Lock()
	defer al.mu.Unlock()

	var archivedCount int64
	var remainingEvents []AuditEvent

	for _, event := range al.events {
		if event.Timestamp.Before(before) {
			archivedCount++
		} else {
			remainingEvents = append(remainingEvents, event)
		}
	}

	al.events = remainingEvents
	// 更新TotalEvents计数
	al.status.TotalEvents = int64(len(remainingEvents))

	return archivedCount, nil
}

// PurgeEvents 清理事件
func (al *InMemoryAuditLog) PurgeEvents(ctx context.Context, before time.Time) (int64, error) {
	return al.ArchiveEvents(ctx, before)
}

// ExportEvents 导出事件
func (al *InMemoryAuditLog) ExportEvents(ctx context.Context, query *AuditQuery, format ExportFormat) ([]byte, error) {
	result, err := al.QueryEvents(ctx, query)
	if err != nil {
		return nil, err
	}

	switch format {
	case ExportFormatJSON:
		return json.MarshalIndent(result.Events, "", "  ")
	case ExportFormatCSV:
		return al.exportToCSV(result.Events)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// exportToCSV 导出为CSV格式
func (al *InMemoryAuditLog) exportToCSV(events []*AuditEvent) ([]byte, error) {
	// 简化的CSV导出实现
	var csv string
	csv += "ID,Timestamp,Type,Severity,UserID,Message\n"

	var csvSb834 strings.Builder
	for _, event := range events {
		csvSb834.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,\"%s\"\n",
			event.ID,
			event.Timestamp.Format(time.RFC3339),
			event.Type,
			event.Severity,
			event.UserID,
			event.Message,
		))
	}
	csv += csvSb834.String()

	return []byte(csv), nil
}

// GetConfiguration 获取配置
func (al *InMemoryAuditLog) GetConfiguration() *AuditConfiguration {
	return al.config
}

// UpdateConfiguration 更新配置
func (al *InMemoryAuditLog) UpdateConfiguration(config *AuditConfiguration) error {
	al.config = config
	return nil
}

// GetStatus 获取状态
func (al *InMemoryAuditLog) GetStatus() *AuditLogStatus {
	al.mu.RLock()
	defer al.mu.RUnlock()

	al.status.Status = "running"
	al.status.Version = "1.0.0"
	al.status.Uptime = time.Since(time.Now().Add(-time.Hour)) // 示例

	if al.status.TotalEvents > 0 {
		al.status.EventsPerSecond = float64(al.status.TotalEvents) / al.status.Uptime.Seconds()
	}

	return al.status
}

// Close 关闭审计日志
func (al *InMemoryAuditLog) Close() error {
	close(al.done)
	close(al.eventChan)
	return nil
}

// 辅助函数
func generateEventID() string {
	return fmt.Sprintf("audit_%d", time.Now().UnixNano())
}

func containsAuditType(slice []AuditType, item AuditType) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func containsAuditSeverity(slice []AuditSeverity, item AuditSeverity) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func containsSearchText(event *AuditEvent, searchText string) bool {
	searchLower := strings.ToLower(searchText)
	return strings.Contains(strings.ToLower(event.Message), searchLower) ||
		strings.Contains(strings.ToLower(event.Details), searchLower)
}

func getUniqueUsers(events []*AuditEvent) []string {
	userSet := make(map[string]bool)
	for _, event := range events {
		if event.UserID != "" {
			userSet[event.UserID] = true
		}
	}

	var users []string
	for user := range userSet {
		users = append(users, user)
	}
	return users
}
