package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Message struct {
	ID   int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Text string `json:"text"`
}

type Response struct {
	Status  string `json:"status"`
	Message string `json:"message"`
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
	if err := db.AutoMigrate(&Message{}); err != nil {
		return nil, fmt.Errorf("не удалось выполнить миграцию: %w", err)
	}
	return db, nil
}

func (a *App) GetHandler(c echo.Context) error {
	var messages []Message
	if err := a.DB.Find(&messages).Error; err != nil {
		return errorResponse(c, http.StatusBadRequest, err)
	}
	return c.JSON(http.StatusOK, messages)
}

func (a *App) PostHandler(c echo.Context) error {
	var message Message
	if err := c.Bind(&message); err != nil {
		return errorResponse(c, http.StatusBadRequest, err)
	}
	if err := a.DB.Create(&message).Error; err != nil {
		return errorResponse(c, http.StatusBadRequest, err)
	}
	return c.JSON(http.StatusCreated, message)
}

func (a *App) PatchHandler(c echo.Context) error {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, err)
	}
	var payload struct {
		Text string `json:"text"`
	}
	if err := c.Bind(&payload); err != nil {
		return errorResponse(c, http.StatusBadRequest, err)
	}
	var msg Message
	if err := a.DB.First(&msg, id).Error; err != nil {
		return errorResponse(c, http.StatusNotFound, fmt.Errorf("сообщение с ID %d не найдено", id))
	}
	msg.Text = payload.Text
	if err := a.DB.Save(&msg).Error; err != nil {
		return errorResponse(c, http.StatusInternalServerError, err)
	}
	return c.JSON(http.StatusOK, msg)
}

func (a *App) DeleteHandler(c echo.Context) error {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, err)
	}
	if err := a.DB.Delete(&Message{}, id).Error; err != nil {
		return errorResponse(c, http.StatusInternalServerError, err)
	}
	return c.NoContent(http.StatusNoContent)
}
