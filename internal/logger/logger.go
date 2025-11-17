package logger

import (
	"io"
	"log/slog"
)

func New(application string, out []io.Writer) *slog.Logger {
	writer := io.MultiWriter(out...)
	h := slog.NewJSONHandler(writer, nil)
	log := slog.New(h)

	log = log.With(slog.Group("application", "name", application))

	return log
}
