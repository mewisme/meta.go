package logging

import (
	"context"
	"log/slog"
	"strings"
)

const Redacted = "<redacted>"

var sensitiveKeyFragments = []string{
	"authorization",
	"cookie",
	"credential",
	"dtsg",
	"password",
	"secret",
	"session",
	"token",
	"totp",
	"xs",
}

// IsSensitiveKey reports whether a structured log key should be redacted.
func IsSensitiveKey(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	for _, fragment := range sensitiveKeyFragments {
		if key == fragment || strings.Contains(key, fragment) {
			return true
		}
	}
	return false
}

// Secret creates an explicitly redacted attribute.
func Secret(key string, _ any) slog.Attr { return slog.String(key, Redacted) }

// RedactAttr recursively redacts sensitive structured attributes.
func RedactAttr(attr slog.Attr) slog.Attr {
	if IsSensitiveKey(attr.Key) {
		return slog.String(attr.Key, Redacted)
	}
	if attr.Value.Kind() == slog.KindGroup {
		attrs := attr.Value.Group()
		redacted := make([]slog.Attr, 0, len(attrs))
		for _, child := range attrs {
			redacted = append(redacted, RedactAttr(child))
		}
		return slog.Group(attr.Key, attrsToAny(redacted)...)
	}
	return attr
}

func attrsToAny(attrs []slog.Attr) []any {
	values := make([]any, len(attrs))
	for i := range attrs {
		values[i] = attrs[i]
	}
	return values
}

type redactingHandler struct{ next slog.Handler }

// NewRedactingHandler wraps next and redacts sensitive keys before emission.
func NewRedactingHandler(next slog.Handler) slog.Handler {
	if next == nil {
		next = slog.Default().Handler()
	}
	return redactingHandler{next: next}
}

func (h redactingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h redactingHandler) Handle(ctx context.Context, record slog.Record) error {
	clean := slog.NewRecord(record.Time, record.Level, record.Message, record.PC)
	record.Attrs(func(attr slog.Attr) bool {
		clean.AddAttrs(RedactAttr(attr))
		return true
	})
	return h.next.Handle(ctx, clean)
}

func (h redactingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clean := make([]slog.Attr, len(attrs))
	for i := range attrs {
		clean[i] = RedactAttr(attrs[i])
	}
	return redactingHandler{next: h.next.WithAttrs(clean)}
}

func (h redactingHandler) WithGroup(name string) slog.Handler {
	return redactingHandler{next: h.next.WithGroup(name)}
}

// RedactLogger wraps logger so sensitive structured fields are redacted.
func RedactLogger(logger *slog.Logger) *slog.Logger {
	if logger == nil {
		logger = slog.Default()
	}
	return slog.New(NewRedactingHandler(logger.Handler()))
}
