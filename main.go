package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/sync/errgroup"

	"github.com/takf-jp/url-shortener/db"
	"github.com/takf-jp/url-shortener/handler"
	"github.com/takf-jp/url-shortener/internal/auth"
	"github.com/takf-jp/url-shortener/internal/background"
	"github.com/takf-jp/url-shortener/internal/logger"
	appmiddleware "github.com/takf-jp/url-shortener/internal/middleware"
	"github.com/takf-jp/url-shortener/repository"
	"github.com/takf-jp/url-shortener/service"
)

type Config struct {
	DatabaseURL string
	JWTSecret   string
	BaseURL     string
	Port        string
}

func LoadConfig() (*Config, error) {
	c := &Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		BaseURL:     os.Getenv("BASE_URL"),
		Port:        os.Getenv("PORT"),
	}
	var missing []string
	if c.DatabaseURL == "" {
		missing = append(missing, "DATABASE_URL")
	}
	if c.JWTSecret == "" {
		missing = append(missing, "JWT_SECRET")
	}
	if c.BaseURL == "" {
		c.BaseURL = "http://localhost:8080"
	}
	if c.Port == "" {
		c.Port = "8080"
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required env: %v", missing) // まとめて報告できる
	}
	return c, nil
}

func main() {
	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}
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

	gormDB, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("db connect", slog.Any("error", err))
		os.Exit(1)
	}

	// Repositories
	urlRepo := repository.NewURLRepository(gormDB)
	userRepo := repository.NewUserRepository(gormDB)
	clickRepo := repository.NewClickRepository(gormDB)

	// Auth
	jwtService := auth.NewJWTService(cfg.JWTSecret, time.Hour*24)

	// Services
	shortenService := service.NewShortenService(urlRepo)
	userService := service.NewUserService(userRepo, jwtService)

	// Handlers
	userHandler := handler.NewUserHandler(userService)
	urlHandler := handler.NewShortenHandler(shortenService, urlRepo, clickRepo, cfg.BaseURL)

	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.RequestID())
	e.Use(appmiddleware.InjectLogger(log))
	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())

	// 公開エンドポイント
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]any{
			"status":     "ok",
			"goroutines": runtime.NumGoroutine(),
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
	port := ":" + cfg.Port

	g.Go(func() error {
		log.Info("server starting", slog.String("addr", port))
		if err := e.Start(port); err != nil && !errors.Is(err, http.ErrServerClosed) {
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
