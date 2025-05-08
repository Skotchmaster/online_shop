package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/Skotchmaster/project_for_t_bank/internal/models"
)

type Response struct {
	Status  string `json:"status"`
	Message string `json:"product"`
}

type App struct {
	DB *gorm.DB
}

func errorResponse(c echo.Context, code int, err error) error {
	return c.JSON(code, Response{
		Status:  "error",
		Message: err.Error(),
	})
}

func initDB() (*gorm.DB, error) {
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

func (a *App) GetHandler(c echo.Context) error {
	var messages []models.Product
	if err := a.DB.Find(&messages).Error; err != nil {
		return errorResponse(c, http.StatusBadRequest, err)
	}
	return c.JSON(http.StatusOK, messages)
}

func (a *App) PostHandler(c echo.Context) error {
	var product models.Product
	if err := c.Bind(&product); err != nil {
		return errorResponse(c, http.StatusBadRequest, err)
	}
	if err := a.DB.Create(&product).Error; err != nil {
		return errorResponse(c, http.StatusBadRequest, err)
	}
	return c.JSON(http.StatusCreated, product)
}

func (a *App) PatchHandler(c echo.Context) error {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, err)
	}
	var payload struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Price       float64 `json:"price"`
	}
	if err := c.Bind(&payload); err != nil {
		return errorResponse(c, http.StatusBadRequest, err)
	}
	var prod models.Product
	if err := a.DB.First(&prod, id).Error; err != nil {
		return errorResponse(c, http.StatusNotFound, fmt.Errorf("сообщение с ID %d не найдено", id))
	}
	prod.Name = payload.Name
	prod.Description = payload.Description
	prod.Price = payload.Price
	if err := a.DB.Save(&prod).Error; err != nil {
		return errorResponse(c, http.StatusInternalServerError, err)
	}
	return c.JSON(http.StatusOK, prod)
}

func (a *App) DeleteHandler(c echo.Context) error {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, err)
	}
	if err := a.DB.Delete(&models.Product{}, id).Error; err != nil {
		return errorResponse(c, http.StatusInternalServerError, err)
	}
	return c.NoContent(http.StatusNoContent)
}
