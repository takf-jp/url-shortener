package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"

	"github.com/takf-jp/url-shortener/db"
	"github.com/takf-jp/url-shortener/handler"
	"github.com/takf-jp/url-shortener/internal/auth"
	"github.com/takf-jp/url-shortener/internal/background"
	"github.com/takf-jp/url-shortener/internal/logger"
	appmiddleware "github.com/takf-jp/url-shortener/internal/middleware"
	"github.com/takf-jp/url-shortener/repository"
	"github.com/takf-jp/url-shortener/service"
)

func main() {
	// Logger
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}
	log := logger.New(env)
	slog.SetDefault(log)

	// アプリ全体の context
	// SIGINT/SIGTERM でキャンセルされる
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	ctx = logger.WithLogger(ctx, log)

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Error("db connect", slog.Any("error", err))
		os.Exit(1)
	}
	defer pool.Close()

	// Repositories
	urlRepo := repository.NewURLRepository(pool)
	userRepo := repository.NewUserRepository(pool)
	clickRepo := repository.NewClickRepository(pool)

	// Auth
	jwtService := auth.NewJWTService(os.Getenv("JWT_SECRET"), time.Hour*24)

	// Services
	shortenService := service.NewShortenService(urlRepo)
	userService := service.NewUserService(userRepo, jwtService)

	// Handlers
	userHandler := handler.NewUserHandler(userService)
	urlHandler := handler.NewShortenHandler(shortenService, urlRepo, clickRepo, "http://localhost:8080")

    e := echo.New()
	e.HideBanner = true
	e.Use(middleware.RequestID())
	e.Use(appmiddleware.InjectLogger(log))
    e.Use(middleware.RequestLogger())
    e.Use(middleware.Recover())

	// 公開エンドポイント
    e.GET("/health", func(c echo.Context) error {
        return c.JSON(200, map[string]string{
            "status": "ok",
        })
    })
	e.POST("/signup", userHandler.Signup)
	e.POST("/login", userHandler.Login)
	e.GET("/:code", urlHandler.Redirect)

	// 認証が必要なエンドポイント
	api := e.Group("")
	api.Use(appmiddleware.JWTAuth(jwtService))
	api.POST("/shorten", urlHandler.Create)
	api.GET("/me/urls", urlHandler.ListMine)

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		log.Info("server starting", slog.String("addr", ":8080"))
		if err := e.Start(":8080"); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error", slog.Any("error", err))
			return err
		}
		return nil
	})

	aggregator := background.NewAggregator(clickRepo, 5*time.Minute)
	g.Go(func() error {
		return aggregator.Run(gCtx)
	})

	// graceful shutdown
	g.Go(func() error {
		<-gCtx.Done()
		log.Info("shutting down server")

		// HTTP サーバを 10秒猶予で停止
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := e.Shutdown(shutdownCtx); err != nil {
			log.Error("server shutdown error", slog.Any("error", err))
			return err
		}
		log.Info("server shutdown completed")
		return nil
	})

	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		log.Error("application error", slog.Any("error", err))
		os.Exit(1)
	}
	log.Info("application exited gracefully")
}
