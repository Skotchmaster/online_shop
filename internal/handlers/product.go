package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/Skotchmaster/online_shop/internal/models"
	"github.com/Skotchmaster/online_shop/internal/mykafka"
)

type Response struct {
	Status  string `json:"status"`
	Message string `json:"product"`
}

type ProductHandler struct {
	DB       *gorm.DB
	Producer *mykafka.Producer
}

func errorResponse(c echo.Context, code int, err error) error {
	return c.JSON(code, Response{
		Status:  "error",
		Message: err.Error(),
	})
}

func InitDB() (*gorm.DB, error) {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("не удалось подключиться к БД: %w", err)
	}
	if err := db.AutoMigrate(&models.Product{}, &models.User{}, &models.RefreshToken{}, &models.CartItem{}); err != nil {
		return nil, fmt.Errorf("не удалось выполнить миграцию: %w", err)
	}
	return db, nil
}

func IsAdmin(c echo.Context, err string) error {
	tok, ok := c.Get("user").(*jwt.Token)
	if !ok {
		return c.JSON(http.StatusBadRequest, "invalid token")
	}

	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return c.JSON(http.StatusBadRequest, "invalid token")
	}

	role := claims["role"]
	if role != "admin" {
		return c.JSON(http.StatusBadRequest, err)
	}

	return nil

}

func (h *ProductHandler) CreateProduct(c echo.Context) error {
	if err := IsAdmin(c, "only admin can create a product"); err != nil {
		return c.JSON(http.StatusBadRequest, err)
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

	prod := models.Product{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Count:       req.Count,
	}

	if err := h.DB.Create(&prod); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	event := map[string]interface{}{
		"type":      "product_created",
		"productID": prod.ID,
		"name":      prod.Name,
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	if err := h.Producer.PublishEvent(
		ctx,
		"product_events",
		fmt.Sprint(prod.ID),
		event,
	); err != nil {
		c.Logger().Errorf("Kafka publish error: %v", err)
	}

	return c.JSON(http.StatusCreated, prod)
}

func (h *ProductHandler) PatchProduct(c echo.Context) error {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, err)
	}

	if err := IsAdmin(c, "only admin can patch the product"); err != nil {
		return c.JSON(http.StatusBadRequest, err)
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
	if err := h.DB.First(&prod, id); err != nil {
		return c.JSON(http.StatusNotFound, err)
	}

	prod.Name = req.Name
	prod.Description = req.Description
	prod.Price = req.Price
	prod.Count = req.Count

	if err := h.DB.Save(&prod); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	event := map[string]interface{}{
		"type":      "product_updated",
		"productID": prod.ID,
		"name":      prod.Name,
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	if err := h.Producer.PublishEvent(
		ctx,
		"product_events",
		fmt.Sprint(prod.ID),
		event,
	); err != nil {
		c.Logger().Errorf("Kafka publish error: %v", err)
	}

	return c.JSON(http.StatusOK, prod)
}

func (h *ProductHandler) DeleteProduct(c echo.Context) error {
	if err := IsAdmin(c, "only admin can delete a product"); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}
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

	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	if err := h.Producer.PublishEvent(
		ctx,
		"product_event",
		fmt.Sprint(id),
		event,
	); err != nil {
		c.Logger().Errorf("kafka publish error: %v", err)
	}
	return c.NoContent(http.StatusNoContent)
}
