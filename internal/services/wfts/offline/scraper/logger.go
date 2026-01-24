package scraper

import "log/slog"

func NewCrawlAttrs(url string) []slog.Attr {
	return []slog.Attr{slog.String("url", url)}
}

// Переписать через интерфейс для каждого покета, с методом принимающим структуру обязательных полей лога