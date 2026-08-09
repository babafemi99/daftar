package observability

import (
	"io"
	"log/slog"
	"strings"

	"github.com/babafemi99/daftar/backend/internal/cfg"
)

// NewLogger constructs Daftar's process-wide structured logger. Container
// runtimes collect its stdout stream; Daftar never writes rotating log files.
func NewLogger(writer io.Writer, config cfg.Config) *slog.Logger {
	options := &slog.HandlerOptions{Level: level(config.Logging.Level)}
	var handler slog.Handler
	if strings.EqualFold(config.Logging.Format, "json") {
		handler = slog.NewJSONHandler(writer, options)
	} else {
		handler = slog.NewTextHandler(writer, options)
	}

	return slog.New(handler).With(
		"service", config.ServiceName,
		"environment", config.Environment,
	)
}

func level(value string) slog.Level {
	switch strings.ToLower(value) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
