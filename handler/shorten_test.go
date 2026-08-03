package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/takf-jp/url-shortener/repository"
)

type fakeURLFinder struct {
	findFunc       func(ctx context.Context, code string) (*repository.URL, error)
	findByUserFunc func(ctx context.Context, userID int64) ([]*repository.URL, error)
}

type fakeClickRecorder struct{}

func (f *fakeClickRecorder) Insert(ctx context.Context, urlID int64, ip string, userAgent string) error {
	return nil
}

func (f *fakeURLFinder) FindByCode(ctx context.Context, code string) (*repository.URL, error) {
	return f.findFunc(ctx, code)
}

func (f *fakeURLFinder) FindByUserID(ctx context.Context, userID int64) ([]*repository.URL, error) {
	return f.findByUserFunc(ctx, userID)
}

func TestShortenHandler_Redirect_見つかった場合_302(t *testing.T) {
	finder := &fakeURLFinder{
		findFunc: func(ctx context.Context, code string) (*repository.URL, error) {
			if code == "abc123" {
				return &repository.URL{Code: code, OriginalURL: "https://example.com"}, nil
			}
			return nil, repository.ErrNotFound
		},
	}
	h := NewShortenHandler(nil, finder, &fakeClickRecorder{}, "http://localhost:8080")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("code")
	c.SetParamValues("abc123")

	if err := h.Redirect(c); err != nil {
		t.Fatalf("Redirect returned error: %v", err)
	}

	if rec.Code != http.StatusFound {
		t.Errorf("expected status code 302, got %d", rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "https://example.com" {
		t.Errorf("expected Location header 'https://example.com', got '%s'", got)
	}
}

func TestShortenHandler_Redirect_DBエラー_500(t *testing.T) {
	finder := &fakeURLFinder{
		findFunc: func(ctx context.Context, code string) (*repository.URL, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewShortenHandler(nil, finder, &fakeClickRecorder{}, "http://localhost:8080")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("code")
	c.SetParamValues("abc123")

	err := h.Redirect(c)

	var httpErr *echo.HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected an HTTPError, got %v", err)
	}
	if httpErr.Code != http.StatusInternalServerError {
		t.Errorf("expected status code 500, got %d", httpErr.Code)
	}
}

func TestShortenHandler_Create_不正URL_400(t *testing.T) {
	h := NewShortenHandler(nil, nil, &fakeClickRecorder{}, "http://localhost:8080")

	e := echo.New()
	body := strings.NewReader(`{"url":"not-a-url"}`)
	req := httptest.NewRequest(http.MethodPost, "/shorten", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userID", int64(42)) // 認証済みとして扱い、URL 検証まで到達させる

	err := h.Create(c)

	var httpErr *echo.HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected an HTTPError, got %v", err)
	}
	if httpErr.Code != http.StatusBadRequest {
		t.Errorf("expected status code 400, got %d", httpErr.Code)
	}
}

func TestShortenHandler_Create_不正URLスキーム_400(t *testing.T) {
	h := NewShortenHandler(nil, nil, &fakeClickRecorder{}, "http://localhost:8080")

	e := echo.New()
	body := strings.NewReader(`{"url":"javascript:alert(document.cookie)"}`)
	req := httptest.NewRequest(http.MethodPost, "/shorten", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userID", int64(42)) // 認証済みとして扱い、URL 検証まで到達させる

	err := h.Create(c)

	var httpErr *echo.HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected an HTTPError, got %v", err)
	}
	if httpErr.Code != http.StatusBadRequest {
		t.Errorf("expected status code 400, got %d", httpErr.Code)
	}
}

func TestShortenHandler_Create_認証なし_401(t *testing.T) {
	h := NewShortenHandler(nil, nil, &fakeClickRecorder{}, "http://localhost:8080")

	e := echo.New()
	body := strings.NewReader(`{"url":"https://example.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/shorten", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec) // userID をセットしない = 未認証

	err := h.Create(c)

	var httpErr *echo.HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected an HTTPError, got %v", err)
	}
	if httpErr.Code != http.StatusUnauthorized {
		t.Errorf("expected status code 401, got %d", httpErr.Code)
	}
}

func TestShortenHandler_ListMine_正常系(t *testing.T) {
	finder := &fakeURLFinder{
		findByUserFunc: func(ctx context.Context, userID int64) ([]*repository.URL, error) {
			return []*repository.URL{
				{ID: 1, Code: "abc123", OriginalURL: "https://go.dev/"},
			}, nil
		},
	}
	h := NewShortenHandler(nil, finder, &fakeClickRecorder{}, "http://localhost:8080")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/me/urls", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userID", int64(42)) // Middlewareのかわりに context に userID をセット

	if err := h.ListMine(c); err != nil {
		t.Fatalf("ListMine returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status code 200, got %d", rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "abc123") {
		t.Errorf("expected response body to contain 'abc123', got '%s'", rec.Body.String())
	}
}

func TestShortenHandler_ListMine_認証なし_401(t *testing.T) {
	finder := &fakeURLFinder{
		findByUserFunc: func(ctx context.Context, userID int64) ([]*repository.URL, error) {
			return []*repository.URL{
				{ID: 1, Code: "abc123", OriginalURL: "https://go.dev/"},
			}, nil
		},
	}
	h := NewShortenHandler(nil, finder, &fakeClickRecorder{}, "http://localhost:8080")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/me/urls", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.ListMine(c)
	var httpErr *echo.HTTPError
	if !errors.As(err, &httpErr) || httpErr.Code != http.StatusUnauthorized {
		t.Fatalf("expected an 401, got %v", err)
	}
}
