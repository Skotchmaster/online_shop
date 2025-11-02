package httpserver

import (
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	echo "github.com/labstack/echo/v4"
)

func newProxy(target, stripPrefix string) (echo.HandlerFunc, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, err
	}

	baseTransport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 60 * time.Second,
		}).DialContext,
		MaxIdleConns:          200,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	p := httputil.NewSingleHostReverseProxy(u)
	p.Transport = baseTransport

	origDirector := p.Director
	p.Director = func(req *http.Request) {
		originalHost := req.Host
		originalProto := "http"
		if req.TLS != nil {
			originalProto = "https"
		} else if xf := req.Header.Get("X-Forwarded-Proto"); xf != "" {
			originalProto = xf
		}

		origDirector(req)

		if stripPrefix != "" && strings.HasPrefix(req.URL.Path, stripPrefix) {
			req.URL.Path = strings.TrimPrefix(req.URL.Path, stripPrefix)
			if rp := req.URL.RawPath; rp != "" && strings.HasPrefix(rp, stripPrefix) {
				req.URL.RawPath = strings.TrimPrefix(rp, stripPrefix)
			}
		}

		if req.Header.Get("X-Forwarded-Proto") == "" {
			req.Header.Set("X-Forwarded-Proto", originalProto)
		}
		if req.Header.Get("X-Forwarded-Host") == "" && originalHost != "" {
			req.Header.Set("X-Forwarded-Host", originalHost)
		}
	}

	p.FlushInterval = 100 * time.Millisecond

	return func(c echo.Context) error {
		p.ServeHTTP(c.Response(), c.Request())
		return nil
	}, nil
}
