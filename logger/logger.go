package logger

import (
	"context"
	"log/slog"
	"os"
	"sync/atomic"
	"time"

	"github.com/lmittmann/tint"
)

var requestIdValue atomic.Value

type requestIDHandler struct {
	slog.Handler
}

func (h *requestIDHandler) Handle(ctx context.Context, r slog.Record) error {
	if v := requestIdValue.Load(); v != nil {
		if id, ok := v.(string); ok && id != "" {
			r.AddAttrs(slog.String("request_id", id))
		}
	}
	return h.Handler.Handle(ctx, r)
}

func SetRequestID(requestID string) {
	requestIdValue.Store(requestID)
}

func ResetRequestID() {
	requestIdValue.Store("")
}

func SetupLogger() {
	w := os.Stderr

	h := tint.NewHandler(w, &tint.Options{
		Level:      slog.LevelDebug,
		TimeFormat: time.Kitchen,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == "request_id" && a.Value.String() == "" {
				return slog.Attr{}
			}
			return a
		},
	})

	logger := slog.New(&requestIDHandler{Handler: h})
	slog.SetDefault(logger)
}
