package csrf

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

type Config struct {
	CookieName string
	HeaderName string
	FormField  string

	CookiePath string
	Domain     string
	Secure     bool
	SameSite   http.SameSite
	MaxAge     time.Duration

	EnforceSameOrigin bool

	SkipPaths []string
}

func DefaultConfig() Config {
	return Config{
		CookieName:        "XSRF-TOKEN",
		HeaderName:        "X-CSRF-Token",
		FormField:         "csrf_token",
		CookiePath:        "/",
		Secure:            false, // включить в проде
		SameSite:          http.SameSiteLaxMode,
		MaxAge:            24 * time.Hour,
		EnforceSameOrigin: true,
	}
}

func Middleware(cfg Config) echo.MiddlewareFunc {
	def := DefaultConfig()
	if cfg.CookieName == "" { cfg.CookieName = def.CookieName }
	if cfg.HeaderName == "" { cfg.HeaderName = def.HeaderName }
	if cfg.FormField == ""  { cfg.FormField  = def.FormField }
	if cfg.CookiePath == "" { cfg.CookiePath = def.CookiePath }
	if cfg.SameSite == 0    { cfg.SameSite   = def.SameSite }
	if cfg.MaxAge == 0      { cfg.MaxAge     = def.MaxAge }

	skip := map[string]struct{}{}
	for _, p := range cfg.SkipPaths { skip[p] = struct{}{} }

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			res := c.Response()

			if _, ok := skip[req.URL.Path]; ok {
				return next(c)
			}

			token := readCookie(req, cfg.CookieName)
			if token == "" {
				var err error
				token, err = newToken(32)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "failed to create CSRF token")
				}
				setCSRFCookie(c, cfg, token)
			} else {
				setCSRFCookie(c, cfg, token)
			}

			method := strings.ToUpper(req.Method)
			if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
				res.Header().Set(cfg.HeaderName, token)
				return next(c)
			}

			if cfg.EnforceSameOrigin {
				if !sameOrigin(req) {
					return echo.NewHTTPError(http.StatusForbidden, "invalid origin")
				}
			}

			provided := req.Header.Get(cfg.HeaderName)
			if provided == "" {
				if err := req.ParseForm(); err == nil {
					provided = req.FormValue(cfg.FormField)
				}
			}
			if !secureCompare(token, provided) {
				return echo.NewHTTPError(http.StatusForbidden, "invalid CSRF token")
			}

			c.Set("csrf_token", token)

			return next(c)
		}
	}
}

func newToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil { return "", err }
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func setCSRFCookie(c echo.Context, cfg Config, token string) {
	c.SetCookie(&http.Cookie{
		Name:     cfg.CookieName,
		Value:    token,
		Path:     cfg.CookiePath,
		Domain:   cfg.Domain,
		Secure:   cfg.Secure,
		HttpOnly: false,
		MaxAge:   int(cfg.MaxAge.Seconds()),
		SameSite: cfg.SameSite,
	})
}

func readCookie(req *http.Request, name string) string {
	c, err := req.Cookie(name)
	if err != nil { return "" }
	return c.Value
}

func secureCompare(a, b string) bool {
	if len(a) == 0 || len(a) != len(b) { return false }
	var v byte
	for i := 0; i < len(a); i++ { v |= a[i] ^ b[i] }
	return v == 0
}

func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		ref := r.Header.Get("Referer")
		if ref == "" { return false }
		origin = ref
	}
	u, err := url.Parse(origin)
	if err != nil { return false }
	return strings.EqualFold(u.Scheme, schemeOf(r)) && strings.EqualFold(u.Host, r.Host)
}

func schemeOf(r *http.Request) string {
	if r.Header.Get("X-Forwarded-Proto") != "" {
		return r.Header.Get("X-Forwarded-Proto")
	}
	if r.TLS != nil { return "https" }
	return "http"
}
