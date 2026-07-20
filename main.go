package main

import (
	"context"
	"log"
	"os"

    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"

	"github.com/takf-jp/url-shortener/db"
	"github.com/takf-jp/url-shortener/handler"
	"github.com/takf-jp/url-shortener/repository"
	"github.com/takf-jp/url-shortener/service"
)

func main() {
	ctx := context.Background()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	pool, err := db.New(ctx, dsn)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	urlRepo := repository.NewURLRepository(pool)
	shortenService := service.NewShortenService(urlRepo)
	handler := handler.NewShortenHandler(shortenService, urlRepo, "http://localhost:8080")

    e := echo.New()
    e.Use(middleware.RequestLogger())
    e.Use(middleware.Recover())

    e.GET("/health", func(c echo.Context) error {
        return c.JSON(200, map[string]string{
            "status": "ok",
        })
    })
	e.POST("/shorten", handler.Create)
	e.GET("/:code", handler.Redirect)

    e.Logger.Fatal(e.Start(":8080"))
}
