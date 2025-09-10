package handlers

import (
	"net/http"
	"strconv"

	"github.com/Skotchmaster/online_shop/internal/service/search"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type SearchHandler struct {
	DB *gorm.DB
}

func (h *SearchHandler) Search(c echo.Context) error {
	q := c.QueryParam("q")
	page, _ := strconv.Atoi(c.QueryParam("page"))
	size, _ := strconv.Atoi(c.QueryParam("size"))
	if size <= 0 { size = 20 }
	if page < 1 { page = 1 }
	offset := (page - 1) * size

	res, err := search.Search(h.DB, q, offset, size)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "search failed")
	}
	return c.JSON(http.StatusOK, echo.Map{
		"total":    res.Total,
		"products": res.Items,
		"page":     page,
		"size":     size,
		"query":    q,
	})
}