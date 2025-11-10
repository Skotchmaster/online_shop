package tokens

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"
)

func AccessClaimsFromToken(TokenStr string, AccessSecret []byte) (*AccessClaims, error) {
	var claims AccessClaims
	tkn, err := jwt.ParseWithClaims(TokenStr, &claims, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected sign method")
		}
		return AccessSecret, nil
	})
	if err != nil || !tkn.Valid {
		return nil, err
	}
	return &claims, nil
}