package es

import (
	"net/http"

	"github.com/Skotchmaster/online_shop/internal/config"
	"github.com/elastic/go-elasticsearch/v9"
	"github.com/labstack/echo/v4"
)

func NewClient(cfg *config.Config) (*elasticsearch.Client, error) {
	esCfg := elasticsearch.Config{
		Addresses: []string{cfg.ES_URL},
		Username:  cfg.ES_USER,
		Password:  cfg.ES_PASSWORD,
	}

	client, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, err)
	}

	res, err := client.Info()
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, err)
	}

	defer res.Body.Close()
	if res.IsError() {
		return nil, echo.NewHTTPError(http.StatusBadRequest, err)
	}

	return client, nil
}
