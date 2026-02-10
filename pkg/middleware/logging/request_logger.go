package loggingmw

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/Skotchmaster/online_shop/pkg/logging"
)

func RequestLogger(base *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			rid := c.Response().Header().Get(echo.HeaderXRequestID)
			if rid == "" {
				rid = c.Request().Header.Get(echo.HeaderXRequestID)
			}
			if rid != "" {
				c.Response().Header().Set(echo.HeaderXRequestID, rid)
			}

			l := base.With(
				"method", c.Request().Method,
				"path", c.Path(),
				"url", c.Request().URL.Path,
				"remote_ip", c.RealIP(),
				"user_agent", c.Request().UserAgent(),
			)
			if rid != "" {
				l = l.With("request_id", rid)
				c.Response().Header().Set(echo.HeaderXRequestID, rid)
			}

			req := c.Request().WithContext(logging.IntoContext(c.Request().Context(), l))
			c.SetRequest(req)

			err := next(c)
			dur := time.Since(start)
			status := c.Response().Status

			if err != nil {
				if he, ok := err.(*echo.HTTPError); ok {
					status = he.Code
				} else {
					status = http.StatusInternalServerError
				}
			}

			switch {
			case err != nil || status >= 500:
				l.Error("request completed", "status", status, "duration_ms", dur.Milliseconds(), "error", errStr(err))
			case status >= 400:
				l.Warn("request completed", "status", status, "duration_ms", dur.Milliseconds())
			default:
				l.Info("request completed", "status", status, "duration_ms", dur.Milliseconds(), "bytes", c.Response().Size)
			}
			return err
		}
	}
}

func errStr(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("%v", err)
}
