package tests

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Skotchmaster/online_shop/internal/models"
	"github.com/stretchr/testify/require"
)

func TestGetProduct(t *testing.T) {
	env := newTestEnv(t)

	accessToken, refreshToken := login(t, env)
	ck_r := &http.Cookie{Name: "refreshToken", Value: refreshToken, Path: "/"}
	ck_a := &http.Cookie{Name: "accessToken", Value: accessToken, Path: "/"}

	test_product := models.Product{
		Name:        "test_name",
		Description: "test_description",
		Price:       1,
		Count:       1,
	}

	env.DB.Create(&test_product)

	rec, _, c := env.doJSONRequest(http.MethodGet, "/api/product/1", "", ck_r, ck_a)
	c.SetParamNames("id")
	c.SetParamValues("1")
	require.NoError(t, env.P.GetProduct(c))
	require.Equal(t, http.StatusOK, rec.Code)

	var Resp models.Product
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &Resp))
	require.Equal(t, test_product.ID, Resp.ID)
	require.Equal(t, test_product.Name, Resp.Name)
	require.Equal(t, test_product.Description, Resp.Description)
	require.Equal(t, test_product.Price, Resp.Price)
	require.Equal(t, test_product.Count, Resp.Count)
}

func TestCreateProduct(t *testing.T) {
	env := newTestEnv(t)

	accessToken, refreshToken := login_admin(t, env)
	ck_r := &http.Cookie{Name: "refreshToken", Value: refreshToken, Path: "/"}
	ck_a := &http.Cookie{Name: "accessToken", Value: accessToken, Path: "/"}

	test_product := models.Product{
		Name:        "test_name",
		Description: "test_description",
		Price:       1,
		Count:       1,
	}

	var Resp models.Product
	event := consumeNextEvent(t, "product_events", func() {
		rec, _, c := env.doJSONRequest(http.MethodPost, "/api/product", test_product, ck_r, ck_a)
		require.NoError(t, env.P.CreateProduct(c))
		require.Equal(t, http.StatusCreated, rec.Code)
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &Resp))
	})
	require.Equal(t, "product_created", event["type"])
	require.EqualValues(t, 1, event["productID"])
	require.Equal(t, "test_name", event["name"])
}

func TestPatchProduct(t *testing.T) {
	env := newTestEnv(t)

	accessToken, refreshToken := login_admin(t, env)
	ck_r := &http.Cookie{Name: "refreshToken", Value: refreshToken, Path: "/"}
	ck_a := &http.Cookie{Name: "accessToken", Value: accessToken, Path: "/"}

	test_product := models.Product{
		Name:        "test_name",
		Description: "test_description",
		Price:       1,
		Count:       1,
	}

	env.DB.Create(&test_product)

	load_map := models.Product{
		Name:        "test_name_1",
		Description: "test_description_1",
		Price:       2,
		Count:       2,
	}

	rec, _, c := env.doJSONRequest(http.MethodPatch, "/api/product/1", load_map, ck_r, ck_a)
	c.SetParamNames("id")
	c.SetParamValues("1")
	require.NoError(t, env.P.PatchProduct(c))
	require.Equal(t, http.StatusOK, rec.Code)

	var Resp models.Product
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &Resp))
	require.EqualValues(t, 1, Resp.ID)
	require.Equal(t, "test_name_1", Resp.Name)
	require.Equal(t, "test_description_1", Resp.Description)
	require.EqualValues(t, 2, Resp.Price)
	require.EqualValues(t, 2, Resp.Count)

	event := consumeNextEvent(t, "product_events", func() {
		rec2, _, c2 := env.doJSONRequest(http.MethodDelete, "/api/product/1", load_map, ck_r, ck_a)
		c2.SetParamNames("id")
		c2.SetParamValues("1")
		require.NoError(t, env.P.PatchProduct(c2))
		require.Equal(t, http.StatusOK, rec2.Code)
	})
	require.Equal(t, "product_updated", event["type"])
	require.EqualValues(t, 1, event["productID"])
	require.Equal(t, "test_name_1", event["name"])
}

func TestDeleteProduct(t *testing.T) {
	env := newTestEnv(t)

	accessToken, refreshToken := login_admin(t, env)
	ck_r := &http.Cookie{Name: "refreshToken", Value: refreshToken, Path: "/"}
	ck_a := &http.Cookie{Name: "accessToken", Value: accessToken, Path: "/"}

	test_product := models.Product{
		Name:        "test_name",
		Description: "test_description",
		Price:       1,
		Count:       1,
	}

	env.DB.Create(&test_product)
	event := consumeNextEvent(t, "product_events", func() {
		rec, _, c := env.doJSONRequest(http.MethodDelete, "/api/product/1", nil, ck_r, ck_a)
		c.SetParamNames("id")
		c.SetParamValues("1")
		require.NoError(t, env.P.DeleteProduct(c))
		require.Equal(t, http.StatusNoContent, rec.Code)
	})
	require.Equal(t, "product_deleted", event["type"])
	require.EqualValues(t, 1, event["productID"])
}
