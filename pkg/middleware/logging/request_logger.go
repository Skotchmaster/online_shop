package loggingmw

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/Skotchmaster/online_shop/pkg/logging"
)

func RequestLogger(base *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			rid := c.Request().Header.Get(echo.HeaderXRequestID)

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

			start := time.Now()
			err := next(c)
			dur := time.Since(start)
			status := c.Response().Status

			if err != nil {
				c.Echo().HTTPErrorHandler(err, c)
				status = c.Response().Status
			}

			switch {
			case err != nil || status >= 500:
				l.Error("request completed", "status", status, "duration_ms", dur.Milliseconds(), "error", errStr(err))
			case status >= 400:
				l.Warn("request completed", "status", status, "duration_ms", dur.Milliseconds())
			default:
				l.Info("request completed", "status", status, "duration_ms", dur.Milliseconds(), "bytes", c.Response().Size)
			}
			return nil
		}
	}
}

func errStr(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("%v", err)
}
