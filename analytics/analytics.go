package analytics

import (
	"net/http"
	"net/url"
	"os"

	"github.com/labstack/echo/v4"
)

const endpoint = "https://www.google-analytics.com/collect"

// Middleware ...
func Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			r := c.Request()

			data := url.Values{}
			data.Set("v", "1")
			data.Set("tid", os.Getenv("GOOGLE_ANALYTICS_ID"))
			data.Set("ds", "api")
			data.Set("uid", c.Response().Header().Get(echo.HeaderXRequestID))
			data.Set("uip", c.RealIP())
			data.Set("ua", r.UserAgent())
			data.Set("dh", r.Host)
			data.Set("dp", r.URL.Path)
			data.Set("t", "pageview")

			_, err := http.PostForm(endpoint, data)
			if err != nil {
				return err
			}

			return next(c)
		}
	}
}
