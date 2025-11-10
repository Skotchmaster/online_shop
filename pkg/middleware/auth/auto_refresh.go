package middleware

import (
	"errors"
	"net/http"
	"time"

	"github.com/Skotchmaster/online_shop/pkg/authclient"
	jwthelp "github.com/Skotchmaster/online_shop/pkg/jwt"
	"github.com/Skotchmaster/online_shop/pkg/tokens"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

type AutoRefreshMiddleware struct {
	JWTSecret  []byte
	AuthClient *authclient.Client
}

func NewAutoRefreshMiddleware(secret []byte, authClient *authclient.Client) *AutoRefreshMiddleware {
	return &AutoRefreshMiddleware{
		JWTSecret:  secret,
		AuthClient: authClient,
	}
}

type ValidatorFunc func(claims *tokens.AccessClaims) error

func (m *AutoRefreshMiddleware) RequireAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return m.requireAuthWithValidator(next, nil)
}

func (m *AutoRefreshMiddleware) RequireAdmin(next echo.HandlerFunc) echo.HandlerFunc {
	return m.requireAuthWithValidator(next, func(claims *tokens.AccessClaims) error {
		if claims.Role != "admin" {
			return echo.NewHTTPError(http.StatusForbidden, "admin access required")
		}
		return nil
	})
}

func (m *AutoRefreshMiddleware) requireAuthWithValidator(next echo.HandlerFunc, validator ValidatorFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		accessCookie, err := c.Cookie("accessToken")
		if err != nil || accessCookie.Value == "" {
			return echo.NewHTTPError(http.StatusUnauthorized, "missing access token")
		}

		claims, err := tokens.AccessClaimsFromToken(accessCookie.Value, m.JWTSecret)

		if err == nil && claims != nil {
			if validator != nil {
				if validationErr := validator(claims); validationErr != nil {
					return validationErr
				}
			}
			
			setUserContext(c, claims)
			return next(c)
		}

		if !errors.Is(err, jwt.ErrTokenExpired) {
			clearAuthCookies(c)
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid access token")
		}

		refreshCookie, rErr := c.Cookie("refreshToken")
		if rErr != nil || refreshCookie.Value == "" {
			clearAuthCookies(c)
			return echo.NewHTTPError(http.StatusUnauthorized, "refresh token missing")
		}

		ctx := c.Request().Context()
		refreshResp, refErr := m.AuthClient.RefreshTokens(
			ctx,
			refreshCookie.Value,
			accessCookie.Value,
		)
		if refErr != nil {
			clearAuthCookies(c)
			return echo.NewHTTPError(http.StatusUnauthorized, "refresh failed: "+refErr.Error())
		}

		c.SetCookie(jwthelp.CreateCookie(
			"accessToken",
			refreshResp.AccessToken,
			"/",
			time.Unix(refreshResp.AccessExp, 0),
		))
		c.SetCookie(jwthelp.CreateCookie(
			"refreshToken",
			refreshResp.RefreshToken,
			"/",
			time.Unix(refreshResp.RefreshExp, 0),
		))

		newClaims, pErr := tokens.AccessClaimsFromToken(refreshResp.AccessToken, m.JWTSecret)
		if pErr != nil || newClaims == nil {
			clearAuthCookies(c)
			return echo.NewHTTPError(http.StatusUnauthorized, "new access token invalid")
		}

		if validator != nil {
			if validationErr := validator(newClaims); validationErr != nil {
				clearAuthCookies(c)
				return validationErr
			}
		}
		
		setUserContext(c, newClaims)

		return next(c)
	}
}

func clearAuthCookies(c echo.Context) {
	c.SetCookie(jwthelp.DeleteCookie("accessToken", "/"))
	c.SetCookie(jwthelp.DeleteCookie("refreshToken", "/"))
}

func setUserContext(c echo.Context, claims *tokens.AccessClaims) {
	c.Set("user_id", claims.Subject)
	c.Set("role", claims.Role)
}