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

	rec, _, c := env.doJSONRequest(http.MethodGet, "/api/product/1", test_product, ck_r, ck_a)
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

	rec, _, c := env.doJSONRequest(http.MethodPost, "/api/product", nil, ck_r, ck_a)
	require.NoError(t, env.P.CreateProduct(c))
	require.Equal(t, http.StatusCreated, rec.Code)

	var Resp models.Product
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &Resp))
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
	require.Equal(t, uint(1), Resp.ID)
	require.Equal(t, "test_name_1", Resp.Name)
	require.Equal(t, "test_description_1", Resp.Description)
	require.Equal(t, float64(2), Resp.Price)
	require.Equal(t, uint(2), Resp.Count)
}

func TestDeletProduct(t *testing.T) {
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
	rec, _, c := env.doJSONRequest(http.MethodDelete, "/api/product/1", nil, ck_r, ck_a)
	c.SetParamNames("id")
	c.SetParamValues("1")
	require.NoError(t, env.P.DeleteProduct(c))
	require.Equal(t, http.StatusNoContent, rec.Code)

}
