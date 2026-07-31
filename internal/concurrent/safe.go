package concurrent

import (
	"context"
	"log/slog"
	"runtime/debug"

	"github.com/takf-jp/url-shortener/internal/logger"
)

var semaphore = make(chan struct{}, 100) // 最大100個のゴルーチンを同時に実行可能

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

func GoLimited(ctx context.Context, fn func()) {
	go func() {
		semaphore <- struct{}{} // セマフォを取得
		defer func() {
			<-semaphore // セマフォを解放
		}()

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
