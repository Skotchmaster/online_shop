package transport

import "time"

type LoginResult struct {
	AccessToken  string
	RefreshToken string
	AccessExp    time.Time
	RefreshExp   time.Time
	IsAdmin      bool
}