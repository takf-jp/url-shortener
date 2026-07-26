package middleware

import (
	"log/slog"

	"github.com/labstack/echo/v4"
	"github.com/takf-jp/url-shortener/internal/logger"
)

// InjectLogger は request_id 等を含むロガーを context に載せる
func InjectLogger(baseLog *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			log := baseLog
			if reqID, ok := c.Get(contextUserIDKey).(string); ok {
				log = log.With(slog.String("request_id", reqID))
			}
			if userID, ok := c.Get(contextUserIDKey).(int64); ok {
				log = log.With(slog.Int64("user_id", userID))
			}
			ctx := logger.WithLogger(c.Request().Context(), log)
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}
