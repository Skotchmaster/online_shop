package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/google/uuid"
)

func CreateCookie(name string, value string, path string, exp_time time.Time) *http.Cookie {
	cookie := &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		Expires:  exp_time,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}

	return cookie
}

func DeleteCookie(name, path string) *http.Cookie {
  return &http.Cookie{
    Name:     name,
    Value:    "",
    Path:     path,
    Expires:  time.Unix(0, 0),
    MaxAge:   -1,
    HttpOnly: true,
    Secure:   true,
    SameSite: http.SameSiteLaxMode,
  }
}

func sha256Hex(s string) string {
  sum := sha256.Sum256([]byte(s))
  return hex.EncodeToString(sum[:])
}

func newJTI() string { return uuid.NewString() }