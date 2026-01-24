package model

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"time"
)

type Logger struct {
	log 		*slog.Logger
}

func NewLogger(log *slog.Logger) *Logger {
	return &Logger{
		log: log,
	}
}

func (l *Logger) logf(slogAttrs []slog.Attr, level slog.Level, format string, args ...any) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	r := slog.NewRecord(time.Now(), level, fmt.Sprintf(format, args...), pcs[0])
	r.AddAttrs(slogAttrs...)
	l.log.Handler().Handle(context.Background(), r)
}

func (l *Logger) Infof(slogAttrs []slog.Attr, format string, args ...any) {
	l.logf(slogAttrs, slog.LevelDebug, format, args...)
}

func (l *Logger) Debugf(slogAttrs []slog.Attr, format string, args ...any) {
	l.logf(slogAttrs, slog.LevelDebug, format, args...)
}

func (l *Logger) Errorf(slogAttrs []slog.Attr, format string, args ...any) {
	l.logf(slogAttrs, slog.LevelError, format, args...)
}