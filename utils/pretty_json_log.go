package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
)

func NewPrettyJSONHandler(w io.Writer, opts *slog.HandlerOptions) slog.Handler {
	return &prettyJSONHandler{
		w:     w,
		opts:  opts,
		attrs: []slog.Attr{},
	}
}

type prettyJSONHandler struct {
	w     io.Writer
	opts  *slog.HandlerOptions
	attrs []slog.Attr
}

func (h *prettyJSONHandler) Enabled(ctx context.Context, level slog.Level) bool {
	if h.opts.Level != nil {
		return level >= h.opts.Level.Level()
	}
	return true
}

func (h *prettyJSONHandler) Handle(ctx context.Context, r slog.Record) error {
	var buf bytes.Buffer
	buf.WriteString("\n")
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")

	m := make(map[string]interface{})
	m["time"] = r.Time
	m["level"] = r.Level.String()
	m["msg"] = r.Message

	// Add handler attributes
	for _, attr := range h.attrs {
		m[attr.Key] = attr.Value.Any()
	}

	// Add record attributes
	r.Attrs(func(a slog.Attr) bool {
		m[a.Key] = a.Value.Any()
		return true
	})

	if err := enc.Encode(m); err != nil {
		return err
	}
	_, err := h.w.Write(buf.Bytes())
	return err
}

func (h *prettyJSONHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandler := *h
	newHandler.attrs = append(newHandler.attrs, attrs...)
	return &newHandler
}

func (h *prettyJSONHandler) WithGroup(name string) slog.Handler {
	return h
}
