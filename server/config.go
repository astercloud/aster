package server

import (
	"time"

	aster "github.com/astercloud/aster"
)

// Config holds all configuration for the aster production server
type Config struct {
	Host string
	Port int
	Mode string // "development" or "production"

	CORS          CORSConfig
	Auth          AuthConfig
	RateLimit     RateLimitConfig
	Logging       LoggingConfig
	Observability ObservabilityConfig
	Database      DatabaseConfig
	Redis         RedisConfig

	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration

	TLS TLSConfig
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	Enabled          bool
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	APIKey APIKeyConfig
	JWT    JWTConfig
}

// APIKeyConfig holds API key authentication settings
type APIKeyConfig struct {
	Enabled    bool
	HeaderName string
	Keys       []string
}

// JWTConfig holds JWT authentication settings
type JWTConfig struct {
	Enabled       bool
	Secret        string
	Issuer        string
	Audience      string
	TokenDuration int // seconds
	Expiry        int // seconds (alias for TokenDuration)
	ExpiryMinutes int
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Enabled       bool
	RequestsPerIP int
	WindowSize    time.Duration
	BurstSize     int
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level      string
	Format     string
	Output     string
	Structured bool
}

// ObservabilityConfig holds observability settings
type ObservabilityConfig struct {
	Enabled     bool
	Metrics     MetricsConfig
	Tracing     TracingConfig
	HealthCheck HealthCheckConfig
}

// MetricsConfig holds metrics configuration
type MetricsConfig struct {
	Enabled  bool
	Endpoint string
	Port     int
}

// TracingConfig holds tracing configuration
type TracingConfig struct {
	Enabled        bool
	ServiceName    string
	ServiceVersion string
	Environment    string
	OTLPEndpoint   string  // OTLP endpoint (e.g., "localhost:4318")
	OTLPInsecure   bool    // Use insecure connection
	SamplingRate   float64 // 0.0 - 1.0
}

// HealthCheckConfig holds health check settings
type HealthCheckConfig struct {
	Enabled  bool
	Endpoint string
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Driver       string
	DSN          string
	MaxOpenConns int
	MaxIdleConns int
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Enabled  bool
	Address  string
	Password string
	DB       int
}

// TLSConfig holds TLS configuration
type TLSConfig struct {
	Enabled  bool
	CertFile string
	KeyFile  string
}

// DefaultConfig returns a default configuration for development
func DefaultConfig() *Config {
	return &Config{
		Host: "0.0.0.0",
		Port: 8080,
		Mode: "development",
		CORS: CORSConfig{
			Enabled:          true,
			AllowOrigins:     []string{"*"},
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-API-Key"},
			ExposeHeaders:    []string{"Content-Length"},
			AllowCredentials: true,
			MaxAge:           86400,
		},
		Auth: AuthConfig{
			APIKey: APIKeyConfig{
				Enabled:    false, // Disabled for local development
				HeaderName: "X-API-Key",
				Keys:       []string{"dev-key-12345"},
			},
			JWT: JWTConfig{
				Enabled:       false,
				Secret:        "change-this-secret",
				Issuer:        "aster",
				Audience:      "aster-api",
				ExpiryMinutes: 60,
			},
		},
		RateLimit: RateLimitConfig{
			Enabled:       true,
			RequestsPerIP: 100,
			WindowSize:    time.Minute,
			BurstSize:     20,
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "json",
			Output:     "stdout",
			Structured: true,
		},
		Observability: ObservabilityConfig{
			Enabled: true,
			Metrics: MetricsConfig{
				Enabled:  true,
				Endpoint: "/metrics",
				Port:     9090,
			},
			Tracing: TracingConfig{
				Enabled:        false,
				ServiceName:    "aster",
				ServiceVersion: aster.Version,
				Environment:    "development",
				OTLPEndpoint:   "localhost:4318",
				OTLPInsecure:   true,
				SamplingRate:   1.0,
			},
			HealthCheck: HealthCheckConfig{
				Enabled:  true,
				Endpoint: "/health",
			},
		},
		Database: DatabaseConfig{
			Driver:       "sqlite",
			DSN:          ".data/aster.db",
			MaxOpenConns: 25,
			MaxIdleConns: 5,
		},
		Redis: RedisConfig{
			Enabled: false,
		},
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
		TLS: TLSConfig{
			Enabled: false,
		},
	}
}

// ProductionConfig returns a configuration suitable for production
func ProductionConfig() *Config {
	config := DefaultConfig()
	config.Mode = "production"
	config.CORS.AllowOrigins = []string{"https://yourdomain.com"}
	config.Auth.APIKey.Keys = []string{} // Must be set via env
	config.Auth.JWT.Enabled = true
	config.RateLimit.RequestsPerIP = 1000
	config.Observability.Tracing.Enabled = true
	config.Database.Driver = "postgres"
	config.Database.DSN = "postgres://user:pass@localhost/aster"
	config.Redis.Enabled = true
	return config
}
