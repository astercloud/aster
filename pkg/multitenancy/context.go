package multitenancy

import (
	"context"
	"errors"
)

// contextKey 用于在上下文中存储多租户信息的键类型
type contextKey string

const (
	orgIDKey    contextKey = "org_id"
	tenantIDKey contextKey = "tenant_id"
)

var (
	// ErrNoOrgID 上下文中未找到组织 ID
	ErrNoOrgID = errors.New("no organization ID found in context")
	// ErrNoTenantID 上下文中未找到租户 ID
	ErrNoTenantID = errors.New("no tenant ID found in context")
)

// WithOrgID 将组织 ID 添加到上下文中
// 组织 ID 用于多租户场景下的顶层隔离
func WithOrgID(ctx context.Context, orgID string) context.Context {
	return context.WithValue(ctx, orgIDKey, orgID)
}

// GetOrgID 从上下文中获取组织 ID
// 如果上下文中没有组织 ID，返回 ErrNoOrgID 错误
func GetOrgID(ctx context.Context) (string, error) {
	orgID, ok := ctx.Value(orgIDKey).(string)
	if !ok || orgID == "" {
		return "", ErrNoOrgID
	}
	return orgID, nil
}

// MustGetOrgID 从上下文中获取组织 ID，如果不存在则 panic
// 仅在确定上下文中一定存在组织 ID 时使用
func MustGetOrgID(ctx context.Context) string {
	orgID, err := GetOrgID(ctx)
	if err != nil {
		panic(err)
	}
	return orgID
}

// WithTenantID 将租户 ID 添加到上下文中
// 租户 ID 用于组织内部的二级隔离
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantIDKey, tenantID)
}

// GetTenantID 从上下文中获取租户 ID
// 如果上下文中没有租户 ID，返回 ErrNoTenantID 错误
func GetTenantID(ctx context.Context) (string, error) {
	tenantID, ok := ctx.Value(tenantIDKey).(string)
	if !ok || tenantID == "" {
		return "", ErrNoTenantID
	}
	return tenantID, nil
}

// MustGetTenantID 从上下文中获取租户 ID，如果不存在则 panic
// 仅在确定上下文中一定存在租户 ID 时使用
func MustGetTenantID(ctx context.Context) string {
	tenantID, err := GetTenantID(ctx)
	if err != nil {
		panic(err)
	}
	return tenantID
}

// GetOrgIDOrDefault 从上下文中获取组织 ID，如果不存在则返回默认值
func GetOrgIDOrDefault(ctx context.Context, defaultVal string) string {
	orgID, err := GetOrgID(ctx)
	if err != nil {
		return defaultVal
	}
	return orgID
}

// GetTenantIDOrDefault 从上下文中获取租户 ID，如果不存在则返回默认值
func GetTenantIDOrDefault(ctx context.Context, defaultVal string) string {
	tenantID, err := GetTenantID(ctx)
	if err != nil {
		return defaultVal
	}
	return tenantID
}
