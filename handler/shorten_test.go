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
	findFunc func(ctx context.Context, code string) (*repository.URL, error)
}

func (f *fakeURLFinder) FindByCode(ctx context.Context, code string) (*repository.URL, error) {
	return f.findFunc(ctx, code)
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
	h := NewShortenHandler(nil, finder, "http://localhost:8080")

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

func TestShortenHandler_Create_不正URL_400(t *testing.T) {
	h := NewShortenHandler(nil, nil, "http://localhost:8080")

    e := echo.New()
    body := strings.NewReader(`{"url":"not-a-url"}`)
    req := httptest.NewRequest(http.MethodPost, "/shorten", body)
    req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
    rec := httptest.NewRecorder()
    c := e.NewContext(req, rec)

    err := h.Create(c)

	var httpErr *echo.HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected an HTTPError, got %v", err)
	}
	if httpErr.Code != http.StatusBadRequest {
		t.Errorf("expected status code 400, got %d", httpErr.Code)
	}
}
