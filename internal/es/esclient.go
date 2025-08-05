package es

import (
	"io"
	"log"
	"net/http"

	"github.com/Skotchmaster/online_shop/internal/config"
	"github.com/elastic/go-elasticsearch/v9"
	"github.com/labstack/echo/v4"
)

func NewClient(cfg *config.Config) (*elasticsearch.Client, error) {
	log.Printf("Connecting to Elasticsearch at: %s", cfg.ES_URL)
	log.Printf("Using credentials: %s:%s", cfg.ES_USER, cfg.ES_PASSWORD)

	esCfg := elasticsearch.Config{
		Addresses: []string{cfg.ES_URL},
		Username:  cfg.ES_USER,
		Password:  cfg.ES_PASSWORD,
	}

	client, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		log.Printf("Failed to create Elasticsearch client: %v", err)
		return nil, echo.NewHTTPError(http.StatusBadRequest, err)
	}

	res, err := client.Info()
	if err != nil {
		log.Printf("Failed to get Elasticsearch info: %v", err)
		return nil, echo.NewHTTPError(http.StatusBadRequest, err)
	}

	defer res.Body.Close()
	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		log.Printf("Elasticsearch error response: %s", body)
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Elasticsearch error: "+res.Status())
	}

	log.Println("Successfully connected to Elasticsearch")
	return client, nil
}
