package searcher

import "log/slog"

func NewSearchAttrs(query string) []slog.Attr {
	return []slog.Attr{slog.String("query", query)}
}