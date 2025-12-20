package ratelimit

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Config 速率限制配置
type Config struct {
	Enabled       bool
	RequestsPerIP int                       // 每个 IP 的请求限制
	WindowSize    time.Duration             // 时间窗口
	BurstSize     int                       // 突发大小（令牌桶容量）
	KeyFunc       func(*gin.Context) string // 自定义 key 提取函数
	Algorithm     string                    // 算法：token_bucket 或 sliding_window
}

// Middleware 创建速率限制中间件
func Middleware(config Config, limiter Limiter) gin.HandlerFunc {
	if !config.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	// 默认使用 IP 作为 key
	keyFunc := config.KeyFunc
	if keyFunc == nil {
		keyFunc = func(c *gin.Context) string {
			return c.ClientIP()
		}
	}

	return func(c *gin.Context) {
		key := keyFunc(c)

		if !limiter.Allow(key) {
			// 获取限制信息
			info := limiter.GetInfo(key)

			// 设置速率限制响应头
			c.Header("X-RateLimit-Limit", strconv.Itoa(info.Limit))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", strconv.FormatInt(info.ResetAt.Unix(), 10))
			c.Header("Retry-After", strconv.Itoa(int(time.Until(info.ResetAt).Seconds())))

			c.JSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error": gin.H{
					"code":        "rate_limit_exceeded",
					"message":     "Too many requests. Please try again later.",
					"retry_after": int(time.Until(info.ResetAt).Seconds()),
				},
			})
			c.Abort()
			return
		}

		// 设置速率限制响应头
		info := limiter.GetInfo(key)
		c.Header("X-RateLimit-Limit", strconv.Itoa(info.Limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(info.Remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(info.ResetAt.Unix(), 10))

		c.Next()
	}
}

// PerUserMiddleware 基于用户的速率限制
func PerUserMiddleware(config Config, limiter Limiter) gin.HandlerFunc {
	config.KeyFunc = func(c *gin.Context) string {
		// 从上下文获取用户 ID
		userID, exists := c.Get("user_id")
		if !exists {
			return c.ClientIP()
		}
		return fmt.Sprintf("user:%v", userID)
	}
	return Middleware(config, limiter)
}

// PerEndpointMiddleware 基于端点的速率限制
func PerEndpointMiddleware(config Config, limiter Limiter) gin.HandlerFunc {
	config.KeyFunc = func(c *gin.Context) string {
		return fmt.Sprintf("%s:%s", c.ClientIP(), c.FullPath())
	}
	return Middleware(config, limiter)
}

// NewLimiterFromConfig 从配置创建限流器
func NewLimiterFromConfig(config Config) Limiter {
	if config.Algorithm == "sliding_window" {
		return NewSlidingWindowLimiter(config.RequestsPerIP, config.WindowSize)
	}

	// 默认使用令牌桶
	rate := config.RequestsPerIP / int(config.WindowSize.Seconds())
	if rate == 0 {
		rate = 1
	}
	capacity := config.BurstSize
	if capacity == 0 {
		capacity = config.RequestsPerIP
	}

	return NewTokenBucketLimiter(rate, capacity, config.WindowSize)
}
