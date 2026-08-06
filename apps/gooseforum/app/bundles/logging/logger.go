package logging

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/leancodebox/GooseForum/app/bundles/closer"
	"github.com/leancodebox/GooseForum/app/bundles/preferences"
	"github.com/leancodebox/GooseForum/app/bundles/setting"
	"go.uber.org/zap"
	"go.uber.org/zap/exp/zapslog"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	LogTypeStdout = "stdout"
	LogTypeFile   = "file"
)

func ErrIf(err error) bool {
	if err != nil {
		slog.Error(err.Error())
		return true
	}
	return false
}

var (
	debug        = setting.IsDebug()
	logType      = preferences.Get("log.type", LogTypeStdout)
	logPath      = preferences.Get("log.path", "./storage/logs/run.log")
	logFormat    = preferences.Get("log.format", "json")
	logErrorPath = preferences.Get("log.errorPath", "./storage/logs/run.error.log")
	maxAge       = preferences.GetInt("log.maxAge", 30)
	maxSize      = preferences.GetInt("log.maxSize", 1024)
	maxBackUps   = preferences.GetInt("log.maxBackUps", 1024)
)

func init() {
	Init()
}

var rootDir = getRootDir()

func getRootDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	// 向上回退 4 层到达项目根目录: logger.go -> logging -> bundles -> app -> project_root
	for range 4 {
		file = filepath.Dir(file)
	}
	return filepath.ToSlash(file)
}

var lumberJackLogger *lumberjack.Logger
var asyncWriter *zapcore.BufferedWriteSyncer
var errorLumberJackLogger *lumberjack.Logger
var errorAsyncWriter *zapcore.BufferedWriteSyncer
var zapLogger *zap.Logger

func newZapCore(ws zapcore.WriteSyncer, level zapcore.Level) zapcore.Core {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
	}
	encoderConfig.EncodeCaller = func(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
		path := caller.File
		if after, ok := strings.CutPrefix(path, rootDir+"/"); ok {
			path = after
		}
		enc.AppendString(fmt.Sprintf("%s:%d", path, caller.Line))
	}

	var encoder zapcore.Encoder
	if logFormat == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	return zapcore.NewCore(encoder, ws, level)
}

// parseLogLevel 解析日志级别配置。explicit 为 false 时按 debug 模式放宽。
func parseLogLevel(raw string, isDebug, explicit bool) (zapcore.Level, error) {
	if explicit {
		var level zapcore.Level
		if err := level.UnmarshalText([]byte(raw)); err != nil {
			return zapcore.InfoLevel, err
		}
		return level, nil
	}
	if isDebug {
		return zapcore.DebugLevel, nil
	}
	return zapcore.InfoLevel, nil
}

// configuredLogLevel 返回配置的日志级别；未显式配置时 debug 模式放宽为 debug。
func configuredLogLevel() zapcore.Level {
	raw := preferences.Get("log.level", "info")
	level, err := parseLogLevel(raw, debug, preferences.IsSet("log.level"))
	if err != nil {
		slog.Warn("invalid log.level, fallback to info", "value", raw, "err", err)
		return zapcore.InfoLevel
	}
	return level
}

func Init() {
	var writeSyncer zapcore.WriteSyncer
	if logType == LogTypeFile {
		// 1. 配置 Lumberjack (分割)
		lumberJackLogger = &lumberjack.Logger{
			Filename:   logPath,
			MaxSize:    maxSize,    // MB
			MaxBackups: maxBackUps, // maxBackUps
			MaxAge:     maxAge,     //days
			Compress:   preferences.GetBool("log.compress", true),
		}

		// 2. 配置 Zap 异步写入器 (Async)
		asyncWriter = &zapcore.BufferedWriteSyncer{
			WS:            zapcore.AddSync(lumberJackLogger),
			Size:          256 * 1024,      // 256KB 缓冲区
			FlushInterval: 3 * time.Second, // 强制刷新间隔
		}
		writeSyncer = asyncWriter
	} else {
		writeSyncer = zapcore.AddSync(os.Stdout)
	}

	// 3. 构建 Zap Core（主日志按配置级别；ERROR/WARNING 单独拆分到错误文件）
	mainCore := newZapCore(writeSyncer, configuredLogLevel())
	zapCore := zapcore.Core(mainCore)
	if logType == LogTypeFile && logErrorPath != "" {
		errorLumberJackLogger = &lumberjack.Logger{
			Filename:   logErrorPath,
			MaxSize:    maxSize,
			MaxBackups: maxBackUps,
			MaxAge:     maxAge,
			Compress:   preferences.GetBool("log.compress", true),
		}
		errorAsyncWriter = &zapcore.BufferedWriteSyncer{
			WS:            zapcore.AddSync(errorLumberJackLogger),
			Size:          256 * 1024,
			FlushInterval: 3 * time.Second,
		}
		zapCore = zapcore.NewTee(mainCore, newZapCore(errorAsyncWriter, zapcore.WarnLevel))
	}
	zapLogger = zap.New(zapCore)

	// 4. 将 Zap 注入 slog (桥接)
	// 这一步是关键：让 slog 使用你配置好的 zap 实例
	logger := slog.New(zapslog.NewHandler(zapLogger.Core(), zapslog.WithCaller(true)))

	// 设置为全局默认
	slog.SetDefault(logger)

	// 注册到全局关闭管理器
	closer.RegisterPriority(closer.PriorityLogger, func() error {
		Shutdown()
		return nil
	})
}

func Shutdown() {
	if zapLogger != nil {
		// 此时 zapLogger 还在工作，先 sync
		_ = zapLogger.Sync()
	}
	if asyncWriter != nil {
		_ = asyncWriter.Stop()
		asyncWriter = nil
	}
	if lumberJackLogger != nil {
		_ = lumberJackLogger.Close()
	}
	if errorAsyncWriter != nil {
		_ = errorAsyncWriter.Stop()
		errorAsyncWriter = nil
	}
	if errorLumberJackLogger != nil {
		_ = errorLumberJackLogger.Close()
	}

	// 恢复默认 logger (保持与 Init 中的 Stdout 配置一致)
	core := newZapCore(zapcore.AddSync(os.Stdout), configuredLogLevel())
	slog.SetDefault(slog.New(zapslog.NewHandler(core, zapslog.WithCaller(true))))
	slog.Info("logging 👋")
	zapLogger = nil
}
