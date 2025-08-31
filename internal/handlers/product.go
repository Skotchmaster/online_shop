package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"github.com/Skotchmaster/online_shop/internal/models"
	"github.com/Skotchmaster/online_shop/internal/mykafka"
	"github.com/Skotchmaster/online_shop/internal/util"
)

type Response struct {
	Status  string `json:"status"`
	Message string `json:"product"`
}

type ProductHandler struct {
	DB        *gorm.DB
	Producer  *mykafka.Producer
	JWTSecret []byte
}

func errorResponse(c echo.Context, code int, err error) error {
	return c.JSON(code, Response{
		Status:  "error",
		Message: err.Error(),
	})
}

func parseIntDefault(s string, def int) int {
	if s == "" { return def }
	if v, err := strconv.Atoi(s); err == nil { return v }
	return def
}

func (h *ProductHandler) publish(c echo.Context, event map[string]any) {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()
	if err := h.Producer.PublishEvent(ctx, "product_events", fmt.Sprint(event["userID"]), event); err != nil {
		c.Logger().Errorf("Kafka publish error: %v", err)
	}
}

func (h *ProductHandler) GetProduct(c echo.Context) error {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, err)
	}

	product := models.Product{}
	if err := h.DB.Where("ID=?", id).First(&product).Error; err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	return c.JSON(http.StatusOK, product)
}

func (h *ProductHandler) GetProducts(c echo.Context) error {
	page := parseIntDefault(c.QueryParam("page"), 1)
	size := parseIntDefault(c.QueryParam("size"), util.DefaultPageSize)

	offset, limit := util.Calculate(page,size)

	var total int64
	if err := h.DB.Model(models.Product{}).Count(&total); err != nil{
		return c.JSON(http.StatusInternalServerError, err)
	}

	var items []models.Product
	if err := h.DB.Model(&models.Product{}).Order("id ASC").Offset(offset).Limit(limit).Find(&items); err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": items,
		"meta": map[string]any{
			"page":        page,
			"size":        limit,
			"total":       total,
			"total_pages": (total + int64(limit) - 1) / int64(limit),
			"has_prev":    page > 1,
			"has_next":    int64(offset+limit) < total,
		},
	})
}

func (h *ProductHandler) CreateProduct(c echo.Context) error {
	var req struct {
		Name        string  `gorm:"not null"                  json:"name"`
		Description string  `gorm:"not null"                  json:"description"`
		Price       float64 `gorm:"not null"                  json:"price"`
		Count       uint    `json:"count"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	prod := models.Product{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Count:       req.Count,
	}

	if err := h.DB.Create(&prod).Error; err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	event := map[string]interface{}{
		"type":      "product_created",
		"productID": prod.ID,
		"name":      prod.Name,
	}

	h.publish(c, event)

	return c.JSON(http.StatusCreated, prod)
}

func (h *ProductHandler) PatchProduct(c echo.Context) error {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, err)
	}

	var req struct {
		Name        string  `gorm:"not null"                  json:"name"`
		Description string  `gorm:"not null"                  json:"description"`
		Price       float64 `gorm:"not null"                  json:"price"`
		Count       uint    `json:"count"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	var prod models.Product
	if err := h.DB.First(&prod, id).Error; err != nil {
		return c.JSON(http.StatusNotFound, err)
	}

	prod.Name = req.Name
	prod.Description = req.Description
	prod.Price = req.Price
	prod.Count = req.Count

	if err := h.DB.Save(&prod).Error; err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	event := map[string]interface{}{
		"type":      "product_updated",
		"productID": prod.ID,
		"name":      prod.Name,
	}

	h.publish(c, event)

	return c.JSON(http.StatusOK, prod)
}

func (h *ProductHandler) DeleteProduct(c echo.Context) error {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, err)
	}
	if err := h.DB.Delete(&models.Product{}, id).Error; err != nil {
		return errorResponse(c, http.StatusInternalServerError, err)
	}

	event := map[string]interface{}{
		"type":      "product_deleted",
		"productID": id,
	}

	h.publish(c, event)

	return c.NoContent(http.StatusNoContent)
}
