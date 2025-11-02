package tests

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Skotchmaster/online_shop/internal/models"
	"github.com/stretchr/testify/require"
)

func createProducts(env *testEnv) {
	env.DB.Create(&models.Product{Name:"p1", Description:"d1", Price:10, Count:100})
	env.DB.Create(&models.Product{Name:"p2", Description:"d1", Price:10, Count:100})
	env.DB.Create(&models.Product{Name:"p3", Description:"d1", Price:10, Count:100})
}

func TestGetCart(t *testing.T) {
	env := newTestEnv(t)

	accessToken, refreshToken := login(t, env)
	ck_r := &http.Cookie{Name: "refreshToken", Value: refreshToken, Path: "/"}
	ck_a := &http.Cookie{Name: "accessToken", Value: accessToken, Path: "/"}

	createProducts(env)

	env.DB.Create(&models.CartItem{UserID: 1, ProductID: 2, Quantity: 3})

	rec, _, c := env.doJSONRequest(http.MethodGet, "/api/cart", nil, ck_r, ck_a)
	require.NoError(t, env.C.GetCart(c))
	require.Equal(t, http.StatusOK, rec.Code)

	var Resp []models.CartItem
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &Resp))

	event := consumeNextEvent(t, "cart_events", func() {
		rec2, _, c2 := env.doJSONRequest(http.MethodGet, "/api/cart", nil, ck_r, ck_a)
		require.NoError(t, env.C.GetCart(c2))
		require.Equal(t, http.StatusOK, rec2.Code)
	})
	require.Equal(t, "get_cart", event["type"])
	require.EqualValues(t, 1, event["userID"])

	require.Len(t, Resp, 1)
	require.EqualValues(t, 1, Resp[0].UserID)
	require.EqualValues(t, 2, Resp[0].ProductID)
	require.EqualValues(t, 3, Resp[0].Quantity)
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
	createProducts(env)

	rec, _, c := env.doJSONRequest(http.MethodPost, "/api/cart", load, ck_r, ck_a)
	require.NoError(t, env.C.AddToCart(c))
	require.Equal(t, http.StatusOK, rec.Code)

	var Resp models.CartItem
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &Resp))

	event := consumeNextEvent(t, "cart_events", func() {
		rec2, _, c2 := env.doJSONRequest(http.MethodPost, "/api/cart", load, ck_r, ck_a)
		require.NoError(t, env.C.AddToCart(c2))
		require.Equal(t, http.StatusOK, rec2.Code)
	})
	require.Equal(t, "add_cart_items", event["type"])
	require.EqualValues(t, 1, event["userID"])
	require.EqualValues(t, 3, event["productID"])

	require.EqualValues(t, 1, Resp.UserID)
	require.EqualValues(t, 2, Resp.Quantity)
	require.EqualValues(t, 3, Resp.ProductID)

}

func TestDeleteOneFromCart(t *testing.T) {
	env := newTestEnv(t)

	accessToken, refreshToken := login(t, env)
	ck_r := &http.Cookie{Name: "refreshToken", Value: refreshToken, Path: "/"}
	ck_a := &http.Cookie{Name: "accessToken", Value: accessToken, Path: "/"}
	createProducts(env)	
	test_item := models.CartItem{
		UserID:    1,
		ProductID: 1,
		Quantity:  2,
	}
	env.DB.Create(&test_item)

	event := consumeNextEvent(t, "cart_events", func() {
		rec, _, c := env.doJSONRequest(http.MethodDelete, "/api/cart/1", "", ck_r, ck_a)
		c.SetParamNames("id")
		c.SetParamValues("1")
		require.NoError(t, env.C.DeleteOneFromCart(c))
		require.Equal(t, http.StatusOK, rec.Code)

		var Resp models.CartItem
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &Resp))
	})

	require.Equal(t, "one_elem_deleted", event["type"])
	require.EqualValues(t, 1, event["new_quantity"])
	require.EqualValues(t, 1, event["id"])
	require.EqualValues(t, 1, event["userID"])

	event2 := consumeNextEvent(t, "cart_events", func() {
		rec2, _, c2 := env.doJSONRequest(http.MethodDelete, "/api/cart/1", "", ck_r, ck_a)
		c2.SetParamNames("id")
		c2.SetParamValues("1")
		require.NoError(t, env.C.DeleteOneFromCart(c2))
		require.Equal(t, http.StatusOK, rec2.Code)

		var Resp models.CartItem
		require.NoError(t, json.Unmarshal(rec2.Body.Bytes(), &Resp))
	})

	require.Equal(t, "cart_item_deleted", event2["type"])
	require.EqualValues(t, 1, event2["userID"])
	require.EqualValues(t, 1, event2["deleted_item"])
}

func TestDeleteAllFromCart(t *testing.T) {
	env := newTestEnv(t)

	accessToken, refreshToken := login(t, env)
	ck_r := &http.Cookie{Name: "refreshToken", Value: refreshToken, Path: "/"}
	ck_a := &http.Cookie{Name: "accessToken", Value: accessToken, Path: "/"}

	createProducts(env)
	test_item := models.CartItem{
		UserID:    1,
		ProductID: 1,
		Quantity:  10,
	}
	env.DB.Create(&test_item)

	var Resp []models.CartItem
	event := consumeNextEvent(t, "cart_events", func() {
		rec, _, c := env.doJSONRequest(http.MethodDelete, "/api/cart/1", nil, ck_r, ck_a)
		c.SetParamNames("id")
		c.SetParamValues("1")
		require.NoError(t, env.C.DeleteAllFromCart(c))
		require.Equal(t, http.StatusOK, rec.Code)
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &Resp))
		require.Len(t, Resp, 0)
	})
	require.Equal(t, "cart_item_deleted", event["type"])
	require.EqualValues(t, 1, event["userID"])
	require.EqualValues(t, 1, event["deleted_item"])
	remaining, ok := event["remaining"].([]interface{})
	require.True(t, ok)
	require.Len(t, remaining, 0)
}

func TestMakeOrder(t *testing.T) {
	env := newTestEnv(t)

	accessToken, refreshToken := login(t, env)
	ck_r := &http.Cookie{Name: "refreshToken", Value: refreshToken, Path: "/"}
	ck_a := &http.Cookie{Name: "accessToken", Value: accessToken, Path: "/"}

	createProducts(env)

	env.DB.Create(&models.CartItem{UserID: 1, ProductID: 1, Quantity: 10})
	env.DB.Create(&models.Product{
		Name:        "test_name",
		Description: "test_description",
		Price:       10,
		Count:       999,
	})

	type OrderResponse struct {
		OrderID uint               `json:"order_id"`
		Total   float64            `json:"total"`
		Status  string             `json:"status"`
		Items   []models.OrderItem `json:"items"`
	}
	var resp OrderResponse

	event := consumeNextEvent(t, "cart_events", func() {
		rec, _, c := env.doJSONRequest(http.MethodPost, "/api/cart/order", nil, ck_a, ck_r)
		require.NoError(t, env.C.MakeOrder(c))
		require.Equal(t, http.StatusOK, rec.Code)

		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.Greater(t, resp.OrderID, uint(0))
		require.EqualValues(t, 100, resp.Total)
		require.Equal(t, "new", resp.Status)

		require.Len(t, resp.Items, 1)
		it := resp.Items[0]
		require.EqualValues(t, resp.OrderID, it.OrderID)
		require.EqualValues(t, 1, it.UserID)
		require.EqualValues(t, 1, it.ProductID)
		require.EqualValues(t, 10, it.Quantity)
	})

	require.Equal(t, "order_created", event["type"])
	require.EqualValues(t, 1, event["userID"])
	require.EqualValues(t, resp.OrderID, event["orderID"])

	items, ok := event["items"].([]interface{})
	require.True(t, ok)
	require.Len(t, items, 1)

	var remaining []models.CartItem
	require.NoError(t, env.DB.Where("user_id = ?", 1).Find(&remaining).Error)
	require.Len(t, remaining, 0)
}
