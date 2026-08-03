package handler

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/takf-jp/url-shortener/internal/concurrent"
	"github.com/takf-jp/url-shortener/internal/logger"
	"github.com/takf-jp/url-shortener/internal/middleware"
	"github.com/takf-jp/url-shortener/repository"
	"github.com/takf-jp/url-shortener/service"
)

// URLFinder は Redirect が必要とする「コードから URL を引く」振る舞いだけを表す。
// 具体型ではなくこれに依存させることで、テストで fake を差し込める（依存性逆転）。
type URLFinder interface {
	FindByCode(ctx context.Context, code string) (*repository.URL, error)
	FindByUserID(ctx context.Context, userID int64) ([]*repository.URL, error)
}

type ClickRecorder interface {
	Insert(ctx context.Context, urlID int64, ip string, userAgent string) error
}

type ShortenHandler struct {
	shortenService *service.ShortenService
	urlRepo        URLFinder
	clickRepo      ClickRecorder
	baseURL        string
}

func NewShortenHandler(shortenService *service.ShortenService, urlRepo URLFinder, clickRepo ClickRecorder, baseURL string) *ShortenHandler {
	return &ShortenHandler{
		shortenService: shortenService,
		urlRepo:        urlRepo,
		clickRepo:      clickRepo,
		baseURL:        baseURL,
	}
}

type shortenRequest struct {
	OriginalURL string `json:"url"`
}

type shortenResponse struct {
	Code         string `json:"code"`
	OriginalURL  string `json:"original_url"`
	ShortenedURL string `json:"shortened_url"`
}

func (h *ShortenHandler) Create(c echo.Context) error {
	uid, ok := middleware.GetUserIDFromContext(c)

	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}
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

	u, err := h.shortenService.Shorten(c.Request().Context(), &uid, req.OriginalURL)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to shorten url")
	}

	return c.JSON(http.StatusOK, shortenResponse{
		Code:         u.Code,
		OriginalURL:  u.OriginalURL,
		ShortenedURL: h.baseURL + "/" + u.Code,
	})
}

func (h *ShortenHandler) Redirect(c echo.Context) error {
	ctx := c.Request().Context()
	log := logger.FromContext(ctx)

	code := c.Param("code")
	u, err := h.urlRepo.FindByCode(ctx, code)
	if err != nil {
		// ErrNotFoundの場合は404を返す, それ以外のエラーは500を返す
		return echo.NewHTTPError(http.StatusNotFound, "url not found")
	}
	// ★ 非同期でクリック記録
	// 重要: ctx をそのまま渡すと、 c.Request().Context() が
	// レスポンス返却後にキャンセルされて中途半端に終わる可能性。
	// → 切り離した context を作る
	backgroundCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	backgroundCtx = logger.WithLogger(backgroundCtx, log)

	ip := c.RealIP()
	userAgent := c.Request().UserAgent()

	concurrent.Go(backgroundCtx, func() {
		defer cancel()
		if err := h.clickRepo.Insert(backgroundCtx, u.ID, ip, userAgent); err != nil {
			log.Error("failed to record click", "error", err)
			return
		}
		log.Debug("click recorded", slog.Int64("url_id", u.ID))
	})

	return c.Redirect(http.StatusFound, u.OriginalURL)
}

func (h *ShortenHandler) ListMine(c echo.Context) error {
	uid, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	urls, err := h.urlRepo.FindByUserID(c.Request().Context(), uid)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list urls")
	}
	return c.JSON(http.StatusOK, urls)
}
