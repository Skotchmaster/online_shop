package tokens

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"
)

func RefreshClaimsFromToken(TokenStr string, RefreshSecret []byte) (*RefreshClaims, error) {
	var claims RefreshClaims
	tkn, err := jwt.ParseWithClaims(TokenStr, &claims, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected sign method")
		}
		return RefreshSecret, nil
	})
	if err != nil || !tkn.Valid {
		return nil, err
	}
	return &claims, nil
}