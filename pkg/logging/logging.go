package logging

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"time"
)

// Level 日志级别
type Level string

const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

// LogRecord 标准化日志记录结构
type LogRecord struct {
	Timestamp time.Time      `json:"ts"`
	Level     Level          `json:"level"`
	Message   string         `json:"message"`
	Fields    map[string]any `json:"fields,omitempty"`
}

// Transport 日志输出通道接口
// 设计参考: 常见的多目标日志管道实现, 但尽量保持简单
type Transport interface {
	// Name 返回 transport 名称(用于调试)
	Name() string
	// Log 写入一条日志记录
	Log(ctx context.Context, rec *LogRecord) error
	// Flush 刷新缓冲(如果有)
	Flush(ctx context.Context) error
}

// Logger 聚合多个 Transport, 提供统一的日志接口
type Logger struct {
	mu         sync.RWMutex
	level      Level
	transports []Transport
}

// NewLogger 创建 Logger 实例
func NewLogger(level Level, transports ...Transport) *Logger {
	return &Logger{
		level:      level,
		transports: transports,
	}
}

// SetLevel 设置日志级别
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// AddTransport 动态添加 transport
func (l *Logger) AddTransport(t Transport) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.transports = append(l.transports, t)
}

// log 内部通用日志函数
func (l *Logger) log(ctx context.Context, level Level, msg string, fields map[string]any) {
	if !l.enabled(level) {
		return
	}

	rec := &LogRecord{
		Timestamp: time.Now(),
		Level:     level,
		Message:   msg,
		Fields:    fields,
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	for _, t := range l.transports {
		_ = t.Log(ctx, rec)
	}
}

func (l *Logger) enabled(level Level) bool {
	// 简单的级别优先级比较: debug < info < warn < error
	order := map[Level]int{
		LevelDebug: 1,
		LevelInfo:  2,
		LevelWarn:  3,
		LevelError: 4,
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	return order[level] >= order[l.level]
}

// Debug 记录调试日志
func (l *Logger) Debug(ctx context.Context, msg string, fields map[string]any) {
	l.log(ctx, LevelDebug, msg, fields)
}

// Info 记录信息日志
func (l *Logger) Info(ctx context.Context, msg string, fields map[string]any) {
	l.log(ctx, LevelInfo, msg, fields)
}

// Warn 记录警告日志
func (l *Logger) Warn(ctx context.Context, msg string, fields map[string]any) {
	l.log(ctx, LevelWarn, msg, fields)
}

// Error 记录错误日志
func (l *Logger) Error(ctx context.Context, msg string, fields map[string]any) {
	l.log(ctx, LevelError, msg, fields)
}

// Flush 刷新所有 transports
func (l *Logger) Flush(ctx context.Context) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	for _, t := range l.transports {
		_ = t.Flush(ctx)
	}
}

// =========================
// Stdout Transport
// =========================

// StdoutTransport 将日志记录以 JSON 行的形式写到 stdout
type StdoutTransport struct {
	mu      sync.Mutex
	encoder *json.Encoder
}

// NewStdoutTransport 创建 StdoutTransport
func NewStdoutTransport() *StdoutTransport {
	return &StdoutTransport{
		encoder: json.NewEncoder(os.Stdout),
	}
}

func (t *StdoutTransport) Name() string { return "stdout" }

func (t *StdoutTransport) Log(ctx context.Context, rec *LogRecord) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.encoder.Encode(rec)
}

func (t *StdoutTransport) Flush(ctx context.Context) error {
	// stdout 无需显式刷新
	return nil
}

// =========================
// File Transport
// =========================

// FileTransport 将日志记录以 JSON 行写入到指定文件
type FileTransport struct {
	mu      sync.Mutex
	file    *os.File
	encoder *json.Encoder
}

// NewFileTransport 创建 FileTransport, path 为日志文件路径
func NewFileTransport(path string) (*FileTransport, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}

	return &FileTransport{
		file:    f,
		encoder: json.NewEncoder(f),
	}, nil
}

func (t *FileTransport) Name() string { return "file" }

func (t *FileTransport) Log(ctx context.Context, rec *LogRecord) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.encoder.Encode(rec)
}

func (t *FileTransport) Flush(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.file.Sync()
}

// Close 关闭底层文件
func (t *FileTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.file.Close()
}

// =========================
// 默认 Logger
// =========================

// Default 是一个可选的全局 Logger, 方便快速集成。
var Default = NewLogger(LevelInfo, NewStdoutTransport())

// Helper 函数, 便于直接调用 logging.Info(ctx, ...)
func Debug(ctx context.Context, msg string, fields map[string]any) {
	Default.Debug(ctx, msg, fields)
}

func Info(ctx context.Context, msg string, fields map[string]any) {
	Default.Info(ctx, msg, fields)
}

func Warn(ctx context.Context, msg string, fields map[string]any) {
	Default.Warn(ctx, msg, fields)
}

func Error(ctx context.Context, msg string, fields map[string]any) {
	Default.Error(ctx, msg, fields)
}

func Flush(ctx context.Context) {
	Default.Flush(ctx)
}
