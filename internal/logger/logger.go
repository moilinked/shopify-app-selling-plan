package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// Log 是全局日志实例，线程安全。
// 使用前需调用 Init 初始化；零值会静默丢弃所有输出。
var Log zerolog.Logger

const defaultLogDir = "./logs"

// Init 初始化全局日志。
//   - logLevel: 最低输出等级，支持 trace/debug/info/warn/error/fatal。
//
// 日志文件按 ISO 周轮转，文件名格式: app-{年}-W{周}.log。
func Init(logLevel string) error {
	level, err := parseLevel(logLevel)
	if err != nil {
		return err
	}
	zerolog.SetGlobalLevel(level)
	zerolog.TimeFieldFormat = time.RFC3339

	// 确保日志目录存在
	if err := os.MkdirAll(defaultLogDir, 0755); err != nil {
		return fmt.Errorf("create log dir %s: %w", defaultLogDir, err)
	}

	fileWriter := &WeeklyRotateWriter{Dir: defaultLogDir, Prefix: "app"}

	// 控制台输出人可读格式，便于开发调试
	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: "2006-01-02 15:04:05",
	}

	// 同时写入控制台和文件
	multi := io.MultiWriter(consoleWriter, fileWriter)
	Log = zerolog.New(multi).With().Timestamp().Caller().Logger()

	return nil
}

// parseLevel 将字符串日志等级转换为 zerolog.Level。
func parseLevel(s string) (zerolog.Level, error) {
	switch strings.TrimSpace(strings.ToLower(s)) {
	case "trace":
		return zerolog.TraceLevel, nil
	case "debug":
		return zerolog.DebugLevel, nil
	case "info", "":
		return zerolog.InfoLevel, nil
	case "warn", "warning":
		return zerolog.WarnLevel, nil
	case "error":
		return zerolog.ErrorLevel, nil
	case "fatal":
		return zerolog.FatalLevel, nil
	default:
		return zerolog.InfoLevel, fmt.Errorf("unknown log level: %s", s)
	}
}
