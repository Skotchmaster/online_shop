package handlers

import (
	"net/http"
	"strconv"

	"github.com/Skotchmaster/online_shop/internal/service/search"
	"github.com/Skotchmaster/online_shop/internal/util"
	"github.com/elastic/go-elasticsearch/v9"
	"github.com/labstack/echo/v4"
)

type SearchHandler struct {
	ES    *elasticsearch.Client
	Index string
}

func NewSearchHandler(es *elasticsearch.Client, index string) *SearchHandler {
	return &SearchHandler{
		ES:    es,
		Index: index,
	}
}

func (h *SearchHandler) Handler(c echo.Context) error {
	q := c.QueryParam("q")
	if q == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "query error")
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	size, _ := strconv.Atoi(c.QueryParam("size"))
	from, size := util.Calculate(page, size)

	ctx := c.Request().Context()

	total, products, err := search.Search(ctx, h.ES, h.Index, q, from, size)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}
	return c.JSON(http.StatusOK, echo.Map{"total": total, "products": products})
}
