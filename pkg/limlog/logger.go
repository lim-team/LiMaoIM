package limlog

import (
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var logger *zap.Logger
var errorLogger *zap.Logger
var warnLogger *zap.Logger
var testLogger *zap.Logger
var atom = zap.NewAtomicLevel()

// TestMode TestMode
var TestMode = false

func init() {

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(newEncoderConfig()),
		zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout)),
		atom,
	)
	testLogger = zap.New(core)

	infoWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   "info.log",
		MaxSize:    500, // megabytes
		MaxBackups: 3,
		MaxAge:     28, // days
	})
	core = zapcore.NewCore(
		zapcore.NewJSONEncoder(newEncoderConfig()),
		zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(infoWriter)),
		atom,
	)
	logger = zap.New(core)

	errorWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   "error.log",
		MaxSize:    500, // megabytes
		MaxBackups: 3,
		MaxAge:     28, // days
	})
	core = zapcore.NewCore(
		zapcore.NewJSONEncoder(newEncoderConfig()),
		zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(errorWriter)),
		zap.ErrorLevel,
	)
	errorLogger = zap.New(core)

	warnWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   "warn.log",
		MaxSize:    500, // megabytes
		MaxBackups: 3,
		MaxAge:     28, // days
	})
	core = zapcore.NewCore(
		zapcore.NewJSONEncoder(newEncoderConfig()),
		zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(warnWriter)),
		zap.WarnLevel,
	)
	warnLogger = zap.New(core)

}

// SetLevel 设置日志登录
func SetLevel(l zapcore.Level) {
	atom.SetLevel(l)
}

func newEncoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		// Keys can be anything except the empty string.
		TimeKey:       "time",
		LevelKey:      "level",
		NameKey:       "logger",
		CallerKey:     "linenum",
		MessageKey:    "msg",
		StacktraceKey: "stacktrace",
		LineEnding:    zapcore.DefaultLineEnding,
		EncodeLevel:   zapcore.LowercaseLevelEncoder, // 小写编码器
		EncodeCaller:  zapcore.FullCallerEncoder,     // 全路径编码器
		EncodeName:    zapcore.FullNameEncoder,
		EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format("2006-01-02 15:04:05"))
		},
		EncodeDuration: func(d time.Duration, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendInt64(int64(d) / 1000000)
		},
	}
}
func timeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
}

// Info Info
func Info(msg string, fields ...zap.Field) {
	if TestMode {
		testLogger.Info(msg, fields...)
		return
	}
	logger.Info(msg, fields...)
}

// Debug Debug
func Debug(msg string, fields ...zap.Field) {
	if TestMode {
		testLogger.Debug(msg, fields...)
		return
	}
	logger.Debug(msg, fields...)

}

// Error Error
func Error(msg string, fields ...zap.Field) {
	if TestMode {
		testLogger.Error(msg, fields...)
		return
	}
	errorLogger.Error(msg, fields...)
}

// Warn Warn
func Warn(msg string, fields ...zap.Field) {
	if TestMode {
		testLogger.Warn(msg, fields...)
		return
	}
	warnLogger.Warn(msg, fields...)
}

// Log Log
type Log interface {
	Info(msg string, fields ...zap.Field)
	Debug(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
}

// LIMLog TLog
type LIMLog struct {
	prefix string // 日志前缀
}

// NewLIMLog NewLIMLog
func NewLIMLog(prefix string) *LIMLog {

	return &LIMLog{prefix: prefix}
}

// Info Info
func (t *LIMLog) Info(msg string, fields ...zap.Field) {
	Info(fmt.Sprintf("【%s】%s", t.prefix, msg), fields...)
}

// Debug Debug
func (t *LIMLog) Debug(msg string, fields ...zap.Field) {
	Debug(fmt.Sprintf("【%s】%s", t.prefix, msg), fields...)
}

// Error Error
func (t *LIMLog) Error(msg string, fields ...zap.Field) {
	Error(fmt.Sprintf("【%s】%s", t.prefix, msg), fields...)
}

// Warn Warn
func (t *LIMLog) Warn(msg string, fields ...zap.Field) {
	Warn(fmt.Sprintf("【%s】%s", t.prefix, msg), fields...)
}
