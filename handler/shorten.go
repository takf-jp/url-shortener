package handler

import (
	"context"
	"net/http"
	"net/url"

	"github.com/labstack/echo/v4"
	"github.com/takf-jp/url-shortener/repository"
	"github.com/takf-jp/url-shortener/service"
)

// URLFinder は Redirect が必要とする「コードから URL を引く」振る舞いだけを表す。
// 具体型ではなくこれに依存させることで、テストで fake を差し込める（依存性逆転）。
type URLFinder interface {
	FindByCode(ctx context.Context, code string) (*repository.URL, error)
}

type ShortenHandler struct {
	shortenService *service.ShortenService
	repo           URLFinder
	baseURL        string
}

func NewShortenHandler(shortenService *service.ShortenService, repo URLFinder, baseURL string) *ShortenHandler {
	return &ShortenHandler{
		shortenService: shortenService,
		repo:           repo,
		baseURL:        baseURL,
	}
}

type shortenRequest struct {
	OriginalURL string `json:"url"`
}

type shortenResponse struct {
	Code string `json:"code"`
	OriginalURL string `json:"original_url"`
	ShortenedURL string `json:"shortened_url"`
}

func (h *ShortenHandler) Create(c echo.Context) error {
	var req shortenRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.OriginalURL == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "url is required")
	}
	if _, err := url.ParseRequestURI(req.OriginalURL); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid url format")
	}

	u, err := h.shortenService.Shorten(c.Request().Context(), req.OriginalURL)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to shorten url")
	}

	return c.JSON(http.StatusOK, shortenResponse{
		Code: u.Code,
		OriginalURL: u.OriginalURL,
		ShortenedURL: h.baseURL + "/" + u.Code,
	})
}

func (h *ShortenHandler) Redirect(c echo.Context) error {
	code := c.Param("code")
	u, err := h.repo.FindByCode(c.Request().Context(), code)
	if err != nil {
		// ErrNotFoundの場合は404を返す, それ以外のエラーは500を返す
		return echo.NewHTTPError(http.StatusNotFound, "url not found")
	}
	return c.Redirect(http.StatusFound, u.OriginalURL)
}
