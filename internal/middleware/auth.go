package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/takf-jp/url-shortener/internal/auth"
)

const contextUserIDKey = "userID"

func JWTAuth(jwtService *auth.JWTService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			header := c.Request().Header.Get("Authorization")
			if header == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing Authorization header")
			}

			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid Authorization header format")
			}

			claims, err := jwtService.Verify(parts[1])
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
			}

			c.Set(contextUserIDKey, claims.UserID)
			return next(c)
		}
	}
}

// Context から userID を取り出すヘルパ
func GetUserIDFromContext(c echo.Context) (int64, bool) {
	userID, ok := c.Get(contextUserIDKey).(int64)
	return userID, ok
}
