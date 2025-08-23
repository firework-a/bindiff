package logger

import (
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// 全局日志实例
	Log   *zap.Logger
	Sugar *zap.SugaredLogger
)

// LoggerConfig 日志配置
type LoggerConfig struct {
	Level      string `json:"level"`
	OutputPath string `json:"output_path"`
	MaxSize    int    `json:"max_size"` // MB
	MaxAge     int    `json:"max_age"`  // days
	MaxBackups int    `json:"max_backups"`
	Compress   bool   `json:"compress"`
}

// InitLogger 初始化日志系统
func InitLogger(config LoggerConfig) error {
	// 解析日志级别
	level, err := zapcore.ParseLevel(config.Level)
	if err != nil {
		return fmt.Errorf("invalid log level: %w", err)
	}

	// 创建编码器配置
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 创建核心配置
	var cores []zapcore.Core

	// 控制台输出
	consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
	consoleCore := zapcore.NewCore(
		consoleEncoder,
		zapcore.AddSync(os.Stdout),
		level,
	)
	cores = append(cores, consoleCore)

	// 文件输出（如果配置了输出路径）
	if config.OutputPath != "" {
		// 确保日志目录存在
		if err := os.MkdirAll(filepath.Dir(config.OutputPath), 0755); err != nil {
			return fmt.Errorf("failed to create log directory: %w", err)
		}

		// 创建文件输出
		fileWriter, err := os.OpenFile(config.OutputPath,
			os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}

		fileEncoder := zapcore.NewJSONEncoder(encoderConfig)
		fileCore := zapcore.NewCore(
			fileEncoder,
			zapcore.AddSync(fileWriter),
			level,
		)
		cores = append(cores, fileCore)
	}

	// 创建日志器
	core := zapcore.NewTee(cores...)
	Log = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	Sugar = Log.Sugar()

	return nil
}

// Close 关闭日志器
func Close() {
	if Log != nil {
		Log.Sync()
	}
}

// WithField 添加字段
func WithField(key string, value interface{}) *zap.Logger {
	if Log == nil {
		return zap.NewNop()
	}
	return Log.With(zap.Any(key, value))
}

// WithFields 添加多个字段
func WithFields(fields map[string]interface{}) *zap.Logger {
	if Log == nil {
		return zap.NewNop()
	}

	zapFields := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}

	return Log.With(zapFields...)
}

// 便捷方法
func Debug(msg string, fields ...zap.Field) {
	if Log != nil {
		Log.Debug(msg, fields...)
	}
}

func Info(msg string, fields ...zap.Field) {
	if Log != nil {
		Log.Info(msg, fields...)
	}
}

func Warn(msg string, fields ...zap.Field) {
	if Log != nil {
		Log.Warn(msg, fields...)
	}
}

func Error(msg string, fields ...zap.Field) {
	if Log != nil {
		Log.Error(msg, fields...)
	}
}

func Fatal(msg string, fields ...zap.Field) {
	if Log != nil {
		Log.Fatal(msg, fields...)
	}
}

// 格式化日志方法
func Debugf(template string, args ...interface{}) {
	if Sugar != nil {
		Sugar.Debugf(template, args...)
	}
}

func Infof(template string, args ...interface{}) {
	if Sugar != nil {
		Sugar.Infof(template, args...)
	}
}

func Warnf(template string, args ...interface{}) {
	if Sugar != nil {
		Sugar.Warnf(template, args...)
	}
}

func Errorf(template string, args ...interface{}) {
	if Sugar != nil {
		Sugar.Errorf(template, args...)
	}
}

func Fatalf(template string, args ...interface{}) {
	if Sugar != nil {
		Sugar.Fatalf(template, args...)
	}
}

// Performance 性能日志记录器
type Performance struct {
	logger *zap.Logger
}

// NewPerformance 创建性能日志记录器
func NewPerformance() *Performance {
	return &Performance{
		logger: Log.Named("performance"),
	}
}

// LogOperation 记录操作性能
func (p *Performance) LogOperation(operation string, duration int64, size int64, fields ...zap.Field) {
	allFields := []zap.Field{
		zap.String("operation", operation),
		zap.Int64("duration_ms", duration),
		zap.Int64("size_bytes", size),
	}
	allFields = append(allFields, fields...)

	if p.logger != nil {
		p.logger.Info("operation_completed", allFields...)
	}
}

// LogMemoryUsage 记录内存使用情况
func (p *Performance) LogMemoryUsage(operation string, memoryMB float64) {
	if p.logger != nil {
		p.logger.Info("memory_usage",
			zap.String("operation", operation),
			zap.Float64("memory_mb", memoryMB))
	}
}
