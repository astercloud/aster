package context

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// InjectorConfig Context 注入器配置
type InjectorConfig struct {
	EnableDateTime     bool   // 注入日期时间
	EnableLocation     bool   // 注入位置信息
	EnableSessionState bool   // 注入 Session 状态
	EnableUserInfo     bool   // 注入用户信息
	EnableSystemInfo   bool   // 注入系统信息
	CustomContext      string // 自定义上下文
	
	// 可选的数据提供者
	LocationProvider     LocationProvider
	SessionStateProvider SessionStateProvider
	UserInfoProvider     UserInfoProvider
}

// ContextInjector Context 自动注入器
type ContextInjector struct {
	config InjectorConfig
}

// NewContextInjector 创建 Context 注入器
func NewContextInjector(config InjectorConfig) *ContextInjector {
	return &ContextInjector{
		config: config,
	}
}

// InjectContext 注入上下文到系统提示词
func (ci *ContextInjector) InjectContext(ctx context.Context, systemPrompt string) string {
	var additions []string

	// 注入日期时间
	if ci.config.EnableDateTime {
		additions = append(additions, ci.buildDateTimeContext())
	}

	// 注入位置信息
	if ci.config.EnableLocation && ci.config.LocationProvider != nil {
		if locCtx := ci.buildLocationContext(ctx); locCtx != "" {
			additions = append(additions, locCtx)
		}
	}

	// 注入 Session 状态
	if ci.config.EnableSessionState && ci.config.SessionStateProvider != nil {
		if stateCtx := ci.buildSessionStateContext(ctx); stateCtx != "" {
			additions = append(additions, stateCtx)
		}
	}

	// 注入用户信息
	if ci.config.EnableUserInfo && ci.config.UserInfoProvider != nil {
		if userCtx := ci.buildUserInfoContext(ctx); userCtx != "" {
			additions = append(additions, userCtx)
		}
	}

	// 注入系统信息
	if ci.config.EnableSystemInfo {
		additions = append(additions, ci.buildSystemInfoContext())
	}

	// 注入自定义上下文
	if ci.config.CustomContext != "" {
		additions = append(additions, ci.config.CustomContext)
	}

	// 如果没有需要注入的内容，直接返回原始提示词
	if len(additions) == 0 {
		return systemPrompt
	}

	// 构建完整的系统提示词
	var builder strings.Builder
	builder.WriteString(systemPrompt)
	
	if !strings.HasSuffix(systemPrompt, "\n") {
		builder.WriteString("\n")
	}
	
	builder.WriteString("\n## Context Information\n\n")
	builder.WriteString(strings.Join(additions, "\n\n"))

	return builder.String()
}

// buildDateTimeContext 构建日期时间上下文
func (ci *ContextInjector) buildDateTimeContext() string {
	now := time.Now()
	
	var builder strings.Builder
	builder.WriteString("### Current Date and Time\n\n")
	builder.WriteString(fmt.Sprintf("- Date: %s\n", now.Format("2006-01-02")))
	builder.WriteString(fmt.Sprintf("- Time: %s\n", now.Format("15:04:05")))
	builder.WriteString(fmt.Sprintf("- Day of Week: %s\n", now.Weekday().String()))
	builder.WriteString(fmt.Sprintf("- Timezone: %s\n", now.Location().String()))
	
	return builder.String()
}

// buildLocationContext 构建位置上下文
func (ci *ContextInjector) buildLocationContext(ctx context.Context) string {
	location, err := ci.config.LocationProvider.GetLocation(ctx)
	if err != nil || location == nil {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("### Location Information\n\n")
	
	if location.City != "" {
		builder.WriteString(fmt.Sprintf("- City: %s\n", location.City))
	}
	if location.Country != "" {
		builder.WriteString(fmt.Sprintf("- Country: %s\n", location.Country))
	}
	if location.Timezone != "" {
		builder.WriteString(fmt.Sprintf("- Timezone: %s\n", location.Timezone))
	}
	if location.Language != "" {
		builder.WriteString(fmt.Sprintf("- Language: %s\n", location.Language))
	}

	return builder.String()
}

// buildSessionStateContext 构建 Session 状态上下文
func (ci *ContextInjector) buildSessionStateContext(ctx context.Context) string {
	state, err := ci.config.SessionStateProvider.GetSessionState(ctx)
	if err != nil || state == nil {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("### Session State\n\n")
	
	if state.SessionID != "" {
		builder.WriteString(fmt.Sprintf("- Session ID: %s\n", state.SessionID))
	}
	if state.MessageCount > 0 {
		builder.WriteString(fmt.Sprintf("- Message Count: %d\n", state.MessageCount))
	}
	if state.Duration > 0 {
		builder.WriteString(fmt.Sprintf("- Session Duration: %s\n", state.Duration.String()))
	}
	
	// 添加自定义状态
	if len(state.CustomState) > 0 {
		builder.WriteString("- Custom State:\n")
		for key, value := range state.CustomState {
			builder.WriteString(fmt.Sprintf("  - %s: %v\n", key, value))
		}
	}

	return builder.String()
}

// buildUserInfoContext 构建用户信息上下文
func (ci *ContextInjector) buildUserInfoContext(ctx context.Context) string {
	userInfo, err := ci.config.UserInfoProvider.GetUserInfo(ctx)
	if err != nil || userInfo == nil {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("### User Information\n\n")
	
	if userInfo.Name != "" {
		builder.WriteString(fmt.Sprintf("- Name: %s\n", userInfo.Name))
	}
	if userInfo.PreferredLanguage != "" {
		builder.WriteString(fmt.Sprintf("- Preferred Language: %s\n", userInfo.PreferredLanguage))
	}
	if userInfo.Timezone != "" {
		builder.WriteString(fmt.Sprintf("- Timezone: %s\n", userInfo.Timezone))
	}
	
	// 添加用户偏好
	if len(userInfo.Preferences) > 0 {
		builder.WriteString("- Preferences:\n")
		for key, value := range userInfo.Preferences {
			builder.WriteString(fmt.Sprintf("  - %s: %v\n", key, value))
		}
	}

	return builder.String()
}

// buildSystemInfoContext 构建系统信息上下文
func (ci *ContextInjector) buildSystemInfoContext() string {
	var builder strings.Builder
	builder.WriteString("### System Information\n\n")
	builder.WriteString("- Platform: Aster AI Agent Framework\n")
	builder.WriteString("- Version: 1.0.0\n")
	
	return builder.String()
}

// LocationProvider 位置信息提供者接口
type LocationProvider interface {
	GetLocation(ctx context.Context) (*LocationInfo, error)
}

// LocationInfo 位置信息
type LocationInfo struct {
	City     string
	Country  string
	Timezone string
	Language string
	Latitude float64
	Longitude float64
}

// SessionStateProvider Session 状态提供者接口
type SessionStateProvider interface {
	GetSessionState(ctx context.Context) (*SessionState, error)
}

// SessionState Session 状态
type SessionState struct {
	SessionID    string
	MessageCount int
	Duration     time.Duration
	CustomState  map[string]interface{}
}

// UserInfoProvider 用户信息提供者接口
type UserInfoProvider interface {
	GetUserInfo(ctx context.Context) (*UserInfo, error)
}

// UserInfo 用户信息
type UserInfo struct {
	UserID            string
	Name              string
	PreferredLanguage string
	Timezone          string
	Preferences       map[string]interface{}
}

// SimpleLocationProvider 简单的位置提供者（基于配置）
type SimpleLocationProvider struct {
	location *LocationInfo
}

// NewSimpleLocationProvider 创建简单位置提供者
func NewSimpleLocationProvider(location *LocationInfo) *SimpleLocationProvider {
	return &SimpleLocationProvider{
		location: location,
	}
}

// GetLocation 获取位置信息
func (slp *SimpleLocationProvider) GetLocation(ctx context.Context) (*LocationInfo, error) {
	return slp.location, nil
}

// SimpleSessionStateProvider 简单的 Session 状态提供者
type SimpleSessionStateProvider struct {
	state *SessionState
}

// NewSimpleSessionStateProvider 创建简单 Session 状态提供者
func NewSimpleSessionStateProvider(state *SessionState) *SimpleSessionStateProvider {
	return &SimpleSessionStateProvider{
		state: state,
	}
}

// GetSessionState 获取 Session 状态
func (sssp *SimpleSessionStateProvider) GetSessionState(ctx context.Context) (*SessionState, error) {
	return sssp.state, nil
}

// SimpleUserInfoProvider 简单的用户信息提供者
type SimpleUserInfoProvider struct {
	userInfo *UserInfo
}

// NewSimpleUserInfoProvider 创建简单用户信息提供者
func NewSimpleUserInfoProvider(userInfo *UserInfo) *SimpleUserInfoProvider {
	return &SimpleUserInfoProvider{
		userInfo: userInfo,
	}
}

// GetUserInfo 获取用户信息
func (suip *SimpleUserInfoProvider) GetUserInfo(ctx context.Context) (*UserInfo, error) {
	return suip.userInfo, nil
}
