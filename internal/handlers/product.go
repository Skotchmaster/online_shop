package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"github.com/Skotchmaster/online_shop/internal/logging"
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
	ctx := c.Request().Context()
	l := logging.FromContext(ctx).With("handler", "product.get_product")
	
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		l.Error("get_product_failed", "status", 400, "reason", "id is not intenger", "error", err)
		return echo.NewHTTPError(http.StatusBadRequest, "id is not integer")
	}

	product := models.Product{}
	if err := h.DB.Where("ID=?", id).First(&product).Error; err != nil {
		l.Error("get_product_failed", "status", 500, "reason", "cannot add product to db", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "cannot add product to db")
	}

	return c.JSON(http.StatusOK, product)
}

func (h *ProductHandler) GetProducts(c echo.Context) error {
	ctx := c.Request().Context()
	l := logging.FromContext(ctx).With("handler", "product.get_products")

	page := parseIntDefault(c.QueryParam("page"), 1)
	size := parseIntDefault(c.QueryParam("size"), util.DefaultPageSize)

	offset, limit := util.Calculate(page,size)

	var total int64
	if err := h.DB.Model(models.Product{}).Count(&total); err != nil{
		l.Error("get_products_error", "status", 500, "reason", "cannot count total products", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "cannot count total products")
	}

	var items []models.Product
	if err := h.DB.Model(&models.Product{}).Order("id ASC").Offset(offset).Limit(limit).Find(&items); err != nil {
		l.Error("get_products_error", "status", 500, "reason", "cannot get products with offset", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "cannot get products with offset")
	}

	l.Info("get_products_success")
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
	ctx := c.Request().Context()
	l := logging.FromContext(ctx).With("handler", "create_product")

	var req struct {
		Name        string  `gorm:"not null"                  json:"name"`
		Description string  `gorm:"not null"                  json:"description"`
		Price       float64 `gorm:"not null"                  json:"price"`
		Count       uint    `json:"count"`
	}

	if err := c.Bind(&req); err != nil {
		l.Error("product_create_error", "status", 400, "reason", "invalid body", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "invalid body")
	}

	prod := models.Product{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Count:       req.Count,
	}

	if err := h.DB.Create(&prod).Error; err != nil {
		l.Error("product_create_error", "status", 500, "reason", "cannot add product to db", "error", err)
		return echo.NewHTTPError(http.StatusBadRequest, "cannot add product to db")
	}

	event := map[string]interface{}{
		"type":      "product_created",
		"productID": prod.ID,
		"name":      prod.Name,
	}

	h.publish(c, event)
	l.Info("create_product_success")
	return c.JSON(http.StatusCreated, prod)
}

func (h *ProductHandler) PatchProduct(c echo.Context) error {
	ctx := c.Request().Context()
	l := logging.FromContext(ctx).With("handler", "patch_product")

	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		l.Error("product_patch_error", "status", 400, "reason", "id not a string", "error", err)
		return echo.NewHTTPError(http.StatusBadRequest, "id not a string")
	}

	var req struct {
		Name        string  `gorm:"not null"                  json:"name"`
		Description string  `gorm:"not null"                  json:"description"`
		Price       float64 `gorm:"not null"                  json:"price"`
		Count       uint    `json:"count"`
	}

	if err := c.Bind(&req); err != nil {
		l.Error("product_patch_error", "status", 400, "reason", "invalid body", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "invalid body")
	}

	var prod models.Product
	if err := h.DB.First(&prod, id).Error; err != nil {
		l.Warn("product_patch_error", "status", 404, "reason", "cannot find product in db", "error", err)
		return echo.NewHTTPError(http.StatusNotFound, "cannot find product in db")
	}

	prod.Name = req.Name
	prod.Description = req.Description
	prod.Price = req.Price
	prod.Count = req.Count

	if err := h.DB.Save(&prod).Error; err != nil {
		l.Error("product_patch_error", "status", 500, "reason", "cannot add product to db", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "cannot add product to db")
	}

	event := map[string]interface{}{
		"type":      "product_updated",
		"productID": prod.ID,
		"name":      prod.Name,
	}

	h.publish(c, event)
	l.Info("patch_prosuct_success")
	return c.JSON(http.StatusOK, prod)
}

func (h *ProductHandler) DeleteProduct(c echo.Context) error {
	ctx := c.Request().Context()
	l := logging.FromContext(ctx).With("handler", "delete_product")

	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		l.Warn("product_delete_error", "status", 400, "reason", "id not an integer", "error", err)
		return echo.NewHTTPError(http.StatusBadRequest, "id not an integer")
	}
	if err := h.DB.Delete(&models.Product{}, id).Error; err != nil {
		l.Error("product_delete_error", "status", 500, "reason", "cannot delete product from db", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "cannot delete product from db")
	}

	event := map[string]interface{}{
		"type":      "product_deleted",
		"productID": id,
	}

	h.publish(c, event)
	l.Info("delete_product_success")
	return c.NoContent(http.StatusNoContent)
}
