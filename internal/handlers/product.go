package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/Skotchmaster/online_shop/internal/models"
)

type Response struct {
	Status  string `json:"status"`
	Message string `json:"product"`
}

type ProductHandler struct {
	DB *gorm.DB
}

func errorResponse(c echo.Context, code int, err error) error {
	return c.JSON(code, Response{
		Status:  "error",
		Message: err.Error(),
	})
}

func InitDB() (*gorm.DB, error) {
	dsn := "host=localhost user=postgres password=root port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("не удалось подключиться к БД: %w", err)
	}
	if err := db.AutoMigrate(&models.Product{}, &models.User{}); err != nil {
		return nil, fmt.Errorf("не удалось выполнить миграцию: %w", err)
	}
	return db, nil
}

func (h *ProductHandler) GetHandler(c echo.Context) error {
	var messages []models.Product
	if err := h.DB.Find(&messages).Error; err != nil {
		return errorResponse(c, http.StatusBadRequest, err)
	}
	return c.JSON(http.StatusOK, messages)
}

func (h *ProductHandler) CreateProduct(c echo.Context) error {
	tok, ok := c.Get("user").(*jwt.Token)
	if !ok {
		return c.JSON(http.StatusBadRequest, "invalid token")
	}

	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return c.JSON(http.StatusBadRequest, "invalid token")
	}

	role := claims["role"].(string)
	if role != "admin" {
		return c.JSON(http.StatusBadRequest, "only admin can create a product")
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

	return c.JSON(http.StatusCreated, prod)
}

func (h *ProductHandler) PatchHandler(c echo.Context) error {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, err)
	}
	var req struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Price       float64 `json:"price"`
	}
	if err := c.Bind(&req); err != nil {
		return errorResponse(c, http.StatusBadRequest, err)
	}
	var prod models.Product
	if err := h.DB.First(&prod, id).Error; err != nil {
		return errorResponse(c, http.StatusNotFound, fmt.Errorf("сообщение с ID %d не найдено", id))
	}
	prod.Name = req.Name
	prod.Description = req.Description
	prod.Price = req.Price
	if err := h.DB.Save(&prod).Error; err != nil {
		return errorResponse(c, http.StatusInternalServerError, err)
	}
	return c.JSON(http.StatusOK, prod)
}

func (h *ProductHandler) DeleteHandler(c echo.Context) error {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, err)
	}
	if err := h.DB.Delete(&models.Product{}, id).Error; err != nil {
		return errorResponse(c, http.StatusInternalServerError, err)
	}
	return c.NoContent(http.StatusNoContent)
}
