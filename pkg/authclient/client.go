package authclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(authServiceURL string) *Client {
	return &Client{
		baseURL: authServiceURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

type RefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	AccessExp    int64  `json:"access_exp"`
	RefreshExp   int64  `json:"refresh_exp"`
	IsAdmin      bool   `json:"is_admin"`
}

func (c *Client) RefreshTokens(ctx context.Context, refreshToken, accessToken string) (*RefreshResponse, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"auth/refresh",
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.AddCookie(&http.Cookie{
		Name:  "refreshToken",
		Value: refreshToken,
	})
	req.AddCookie(&http.Cookie{
		Name:  "accessToken",
		Value: accessToken,
	})

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh failed with status: %d", resp.StatusCode)
	}

	var result RefreshResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}