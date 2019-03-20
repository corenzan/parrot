package main

import (
	"net/http"
	"os"

	"github.com/corenzan/parrot/flickr"
	"github.com/corenzan/parrot/instagram"
	"github.com/corenzan/parrot/twitter"

	raven "github.com/getsentry/raven-go"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func handle(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!")
}

func main() {
	raven.SetDSN("")

	e := echo.New()

	e.HTTPErrorHandler = func(err error, c echo.Context) {
		if e, ok := err.(*echo.HTTPError); ok {
			c.NoContent(e.Code)
			return
		}
		raven.SetHttpContext(raven.NewHttp(c.Request()))
		raven.CaptureError(err, nil)
		e.Logger.Error(err)
		c.NoContent(http.StatusInternalServerError)
	}

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(middleware.Static("public/"))

	flickr.Route(e.Group("/flickr"))
	instagram.Route(e.Group("/instagram"))
	twitter.Route(e.Group("/twitter"))

	e.Logger.Fatal(e.Start(":" + os.Getenv("PORT")))
}
