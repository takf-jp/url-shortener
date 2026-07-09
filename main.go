package main

import (
    "net/http"

    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"
)

func main() {
    e := echo.New()

    e.Use(middleware.RequestLogger())
    e.Use(middleware.Recover())

    e.GET("/health", func(c echo.Context) error {
        return c.JSON(http.StatusOK, map[string]string{
            "status": "ok",
        })
    })

    e.Logger.Fatal(e.Start(":8080"))
}
