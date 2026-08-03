package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/takf-jp/url-shortener/internal/auth"
)

// test用 : 呼ばれたら called を true にして 200 を返すだけのハンドラ
func newSpyHandler(called *bool) echo.HandlerFunc {
	return func(c echo.Context) error {
		*called = true
		return c.String(http.StatusOK, "ok")
	}
}

func TestJWTAuth_ヘッダなし_401(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", time.Hour)

	called := false
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/me/urls", nil) // Authorization ヘッダなし
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := JWTAuth(jwtService)(newSpyHandler(&called))
	err := h(c)

	var httpErr *echo.HTTPError
	if !errors.As(err, &httpErr) || httpErr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %v", err)
	}
	if called {
		t.Error("next handler should not be called when Authorization header is missing")
	}
}

func TestJWTAuth_Bearer形式違反_401(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", time.Hour)

	called := false
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/me/urls", nil)
	req.Header.Set("Authorization", "InvalidFormatToken") // Bearer 形式違反
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := JWTAuth(jwtService)(newSpyHandler(&called))
	err := h(c)

	var httpErr *echo.HTTPError
	if !errors.As(err, &httpErr) || httpErr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %v", err)
	}
	if called {
		t.Error("next handler should not be called when Authorization header format is invalid")
	}
}

func TestJWTAuth_不正なトークンで401(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", time.Hour)

	called := false
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/me/urls", nil)
	req.Header.Set("Authorization", "Bearer invalidtoken") // 不正なトークン
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := JWTAuth(jwtService)(newSpyHandler(&called))
	err := h(c)

	var httpErr *echo.HTTPError
	if !errors.As(err, &httpErr) || httpErr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %v", err)
	}
	if called {
		t.Error("next handler should not be called when token is invalid")
	}
}

func TestJWTAuth_正常系(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", time.Hour)

	// 正常なトークンを発行
	token, err := jwtService.Issue(42)
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}

	called := false
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/me/urls", nil)
	req.Header.Set("Authorization", "Bearer "+token) // 正常なトークン
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := JWTAuth(jwtService)(newSpyHandler(&called))(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if !called {
		t.Error("next handler should be called when token is valid")
	}

	// userID が context にセットされていることを確認
	userID, ok := GetUserIDFromContext(c)
	if !ok || userID != 42 {
		t.Errorf("expected userID 42 in context, got %d (ok=%v)", userID, ok)
	}
}
