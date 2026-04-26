package internal

import (
	"log/slog"
	"os"
)

func NewLogger(isProduction bool) *slog.Logger {
	var level slog.Level
	var handler slog.Handler

	if isProduction {
		level = slog.LevelInfo
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		})
	} else {
		level = slog.LevelDebug
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		})
	}

	return slog.New(handler)
}
