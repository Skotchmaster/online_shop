package tests

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Skotchmaster/online_shop/internal/models"
	"github.com/stretchr/testify/require"
)

func TestGetCart(t *testing.T) {
	env := newTestEnv(t)

	accessToken, refreshToken := login(t, env)
	ck_r := &http.Cookie{Name: "refreshToken", Value: refreshToken, Path: "/"}
	ck_a := &http.Cookie{Name: "accessToken", Value: accessToken, Path: "/"}

	env.DB.Create(&models.CartItem{UserID: 1, ProductID: 2, Quantity: 3})

	rec, _, c := env.doJSONRequest(http.MethodGet, "/api/cart", nil, ck_r, ck_a)
	require.NoError(t, env.C.GetCart(c))
	require.Equal(t, http.StatusOK, rec.Code)

	var resp []models.CartItem
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp, 1)
	require.Equal(t, uint(1), resp[0].UserID)
	require.Equal(t, uint(2), resp[0].ProductID)
	require.Equal(t, uint(3), resp[0].Quantity)
}

func TestAddToCart(t *testing.T) {
	env := newTestEnv(t)

	accessToken, refreshToken := login(t, env)
	ck_r := &http.Cookie{Name: "refreshToken", Value: refreshToken, Path: "/"}
	ck_a := &http.Cookie{Name: "accessToken", Value: accessToken, Path: "/"}

	load := map[string]uint{
		"quantity":   2,
		"product_id": 3,
	}

	rec, _, c := env.doJSONRequest(http.MethodPost, "/api/cart", load, ck_r, ck_a)
	require.NoError(t, env.C.AddToCart(c))
	require.Equal(t, http.StatusOK, rec.Code)

	var Resp models.CartItem
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &Resp))
	require.Equal(t, uint(1), Resp.UserID)
	require.Equal(t, uint(2), Resp.Quantity)
	require.Equal(t, uint(3), Resp.ProductID)

}

func TestDeleteOneFromCart(t *testing.T) {
	env := newTestEnv(t)

	accessToken, refreshToken := login(t, env)
	ck_r := &http.Cookie{Name: "refreshToken", Value: refreshToken, Path: "/"}
	ck_a := &http.Cookie{Name: "accessToken", Value: accessToken, Path: "/"}

	test_item := models.CartItem{
		UserID:    1,
		ProductID: 1,
		Quantity:  2,
	}
	env.DB.Create(&test_item)

	rec, _, c := env.doJSONRequest(http.MethodDelete, "/api/cart/1", "", ck_r, ck_a)
	c.SetParamNames("id")
	c.SetParamValues("1")
	require.NoError(t, env.C.DeleteOneFromCart(c))
	require.Equal(t, http.StatusOK, rec.Code)

	var Resp models.CartItem
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &Resp))
	require.Equal(t, uint(1), Resp.Quantity)
}

func TestDeleteAllFromCart(t *testing.T) {
	env := newTestEnv(t)

	accessToken, refreshToken := login(t, env)
	ck_r := &http.Cookie{Name: "refreshToken", Value: refreshToken, Path: "/"}
	ck_a := &http.Cookie{Name: "accessToken", Value: accessToken, Path: "/"}

	test_item := models.CartItem{
		UserID:    1,
		ProductID: 1,
		Quantity:  10,
	}
	env.DB.Create(&test_item)

	rec, _, c := env.doJSONRequest(http.MethodDelete, "/api/cart/1", ck_r, ck_a)
	c.SetParamNames("id")
	c.SetParamValues("1")
	require.NoError(t, env.C.DeleteAllFromCart(c))
	require.Equal(t, http.StatusOK, rec.Code)

	var Resp []models.CartItem
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &Resp))
	require.Len(t, Resp, 0)

}

func TestMakeOrder(t *testing.T) {
	env := newTestEnv(t)
	accessToken, refreshToken := login(t, env)
	ck_r := &http.Cookie{Name: "refreshToken", Value: refreshToken, Path: "/"}
	ck_a := &http.Cookie{Name: "accessToken", Value: accessToken, Path: "/"}

	test_item := models.CartItem{
		UserID:    1,
		ProductID: 1,
		Quantity:  10,
	}

	test_product := models.Product{
		Name:        "test_name",
		Description: "test_description",
		Price:       10,
		Count:       1,
	}
	env.DB.Create(&test_item)
	env.DB.Create(&test_product)

	rec, _, c := env.doJSONRequest(http.MethodPost, "/api/cart/order", nil, ck_a, ck_r)
	require.NoError(t, env.C.MakeOrder(c))
	require.Equal(t, http.StatusOK, rec.Code)

	type OrderResponse struct {
		OrderID uint               `json:"order_id"`
		Total   float64            `json:"total"`
		Status  string             `json:"status"`
		Items   []models.OrderItem `json:"items"`
	}

	OrderItem := models.OrderItem{
		OrderID:   1,
		UserID:    1,
		ProductID: 1,
		Quantity:  10,
	}

	var Resp OrderResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &Resp))
	require.Equal(t, uint(1), Resp.OrderID)
	require.Equal(t, float64(100), Resp.Total)
	require.Equal(t, "new", Resp.Status)
	require.Equal(t, OrderItem, Resp.Items[0])
}
