package search

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/Skotchmaster/online_shop/internal/models"
	"github.com/elastic/go-elasticsearch/v9"
	"github.com/labstack/echo/v4"
)

func Search(ctx context.Context, es *elasticsearch.Client, index, query string, from, size int) (int64, []models.Product, error) {
	body := map[string]interface{}{
		"query": map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":     query,
				"fields":    []string{"name^2", "description"},
				"fuzziness": "AUTO",
			},
		},
		"from": from,
		"size": size,
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return 0, nil, echo.NewHTTPError(http.StatusBadRequest, "search error: %w", err)
	}

	res, err := es.Search(
		es.Search.WithContext(ctx),
		es.Search.WithIndex(index),
		es.Search.WithBody(&buf),
	)

	if err != nil {
		return 0, nil, echo.NewHTTPError(http.StatusBadRequest, "search error: %w", err)
	}

	defer res.Body.Close()
	if res.IsError() {
		return 0, nil, echo.NewHTTPError(http.StatusBadRequest, "search error: %w", err)
	}

	var r struct {
		Hits struct {
			Total struct{ Value int64 }             `json:"total"`
			Hits  []struct{ Source models.Product } `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return 0, nil, err
	}

	prods := make([]models.Product, len(r.Hits.Hits))
	for i, hit := range r.Hits.Hits {
		prods[i] = hit.Source
	}
	return r.Hits.Total.Value, prods, nil
}
