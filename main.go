package main

import (
	"context"
	"log/slog"
	"os"
	"time"

    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"

	"github.com/takf-jp/url-shortener/db"
	"github.com/takf-jp/url-shortener/handler"
	"github.com/takf-jp/url-shortener/internal/auth"
	"github.com/takf-jp/url-shortener/internal/logger"
	appmiddleware "github.com/takf-jp/url-shortener/internal/middleware"
	"github.com/takf-jp/url-shortener/repository"
	"github.com/takf-jp/url-shortener/service"
)

func main() {
	ctx := context.Background()

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}
	log := logger.New(env)
	slog.SetDefault(log)
	log.Info("app starting", "env", env)

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Error("DATABASE_URL environment variable is not set")
		os.Exit(1)
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Error("JWT_SECRET environment variable is not set")
		os.Exit(1)
	}

	pool, err := db.New(ctx, dsn)
	if err != nil {
		log.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Repositories
	urlRepo := repository.NewURLRepository(pool)
	userRepo := repository.NewUserRepository(pool)

	// Auth
	jwtService := auth.NewJWTService(jwtSecret, time.Hour*24)

	// Services
	shortenService := service.NewShortenService(urlRepo)
	userService := service.NewUserService(userRepo, jwtService)

	// Handlers
	userHandler := handler.NewUserHandler(userService)
	urlHandler := handler.NewShortenHandler(shortenService, urlRepo, "http://localhost:8080")

    e := echo.New()
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

    e.Logger.Fatal(e.Start(":8080"))
}
