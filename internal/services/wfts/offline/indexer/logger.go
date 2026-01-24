package indexer

import "log/slog"

func NewIndexAttrs(url string) []slog.Attr {
	return []slog.Attr{slog.String("url", url)}
}

func NewQueryAttr(query string) []slog.Attr {
	return []slog.Attr{slog.String("query", query)}
}