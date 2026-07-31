package concurrent

import (
	"context"
	"log/slog"
	"runtime/debug"

	"github.com/takf-jp/url-shortener/internal/logger"
)

func Go(ctx context.Context, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.FromContext(ctx).Error("panic recovered",
					slog.Any("panic", r),
					slog.String("stack", string(debug.Stack())),
				)
			}
		}()
		fn()
	}()
}
