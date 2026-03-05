package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// Log is the application-wide logger. It is safe for concurrent use.
// Call Init before using it; the zero value silently discards output.
var Log zerolog.Logger

const defaultLogDir = "./logs"

// Init sets up the global logger.
//   - logLevel: minimum level to emit ("debug", "info", "warn", "error").
//
// Log files are always written to ./logs (weekly rotation, JSON format).
// In Docker, mount a volume to ./logs to persist on the host.
func Init(logLevel string) error {
	level, err := parseLevel(logLevel)
	if err != nil {
		return err
	}
	zerolog.SetGlobalLevel(level)
	zerolog.TimeFieldFormat = time.RFC3339

	if err := os.MkdirAll(defaultLogDir, 0755); err != nil {
		return fmt.Errorf("create log dir %s: %w", defaultLogDir, err)
	}

	fileWriter := &WeeklyRotateWriter{Dir: defaultLogDir, Prefix: "app"}

	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: "2006-01-02 15:04:05",
	}

	multi := io.MultiWriter(consoleWriter, fileWriter)
	Log = zerolog.New(multi).With().Timestamp().Caller().Logger()

	return nil
}

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
