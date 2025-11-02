package middleware

import (
	"github.com/labstack/echo/v4"
	ecM "github.com/labstack/echo/v4/middleware"
)

func Common() []echo.MiddlewareFunc {
	return []echo.MiddlewareFunc{
		ecM.Recover(),
		ecM.RequestID(),
		ecM.Logger(),
		ecM.Secure(),
	}
}
