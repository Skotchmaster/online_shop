package httpserver

import (
	"errors"
	"net/http"

	"github.com/Skotchmaster/online_shop/pkg/logging"
	"github.com/Skotchmaster/online_shop/services/catalog/internal/service"
	"github.com/Skotchmaster/online_shop/services/catalog/internal/transport"
	"github.com/Skotchmaster/online_shop/services/catalog/internal/util"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)
type CatalogHTTP struct {
	Svc *service.CatalogService
}

func (h *CatalogHTTP) GetProduct(c echo.Context) error {
	ctx := c.Request().Context()
	l := logging.FromContext(ctx).With("handler", "product.get_product")
	
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		l.Warn("get_product_failed", "status", 400, "reason", "uuid is not intenger", "error", err)
		return echo.NewHTTPError(http.StatusBadRequest, "uuid is not uuid")
	}

	product, err := h.Svc.GetProduct(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			l.Warn("get_product_failed", "status", 404, "reason", "product with this id dont exist", "error", err)
			return echo.NewHTTPError(http.StatusNotFound, "product with this id dont exist")
		}else {
			l.Error("get_product_failed", "status", 500, "reason", "cannot get product", "error", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "cannot get product")
		}
	}

	return c.JSON(http.StatusOK, product)
}

func (h *CatalogHTTP) GetProducts(c echo.Context) error {
	ctx := c.Request().Context()
	l := logging.FromContext(ctx).With("handler", "product.get_products")

	page := util.ParseIntDefault(c.QueryParam("page"), 1)
	size := util.ParseIntDefault(c.QueryParam("size"), util.DefaultPageSize)

	offset, limit := util.Calculate(page,size)

	total, items, errResp, err := h.Svc.GetProducts(ctx, offset, limit)
	if err != nil {
		l.Error("get_products_error", "status", 500, "reason", errResp, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, errResp)
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

func (h *CatalogHTTP) CreateProduct(c echo.Context) error {
	ctx := c.Request().Context()
	l := logging.FromContext(ctx).With("handler", "create_product")

	var req transport.CreateProductRequest

	if err := c.Bind(&req); err != nil {
		l.Warn("product_create_error", "status", 400, "reason", "invalid body", "error", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid body")
	}

	CreatedProduct, err := h.Svc.CreateProduct(ctx, req)
	if err != nil {
		if errors.Is(err, service.ErrValidation) {
			l.Warn("product_create_error", "status", 400, "reason", "invalid body", "error", err)
			return echo.NewHTTPError(http.StatusBadRequest, "invalid body")
		}
		l.Error("product_create_error", "status", 500, "reason", "cannot add product to db", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "cannot add product to db")
	}

	l.Info("create_product_success")
	return c.JSON(http.StatusCreated, CreatedProduct)
}

func (h *CatalogHTTP) PatchProduct(c echo.Context) error {
	ctx := c.Request().Context()
	l := logging.FromContext(ctx).With("handler", "patch_product")

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		l.Warn("product_patch_error", "status", 400, "reason", "id not a uuid", "error", err)
		return echo.NewHTTPError(http.StatusBadRequest, "id not a uuid")
	}

	var req transport.PatchProductRequest

	if err := c.Bind(&req); err != nil {
		l.Warn("product_patch_error", "status", 400, "reason", "invalid body", "error", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid body")
	}

	prod, err := h.Svc.PatchProduct(ctx, req, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound){
			l.Warn("product_patch_error", "status", 404, "reason", "cannot find product in db", "error", err)
			return echo.NewHTTPError(http.StatusNotFound, "cannot find product in db")
		}
		if errors.Is(err,  service.ErrValidation){
			l.Warn("product_patch_error", "status", 400, "reason", "invalid body", "error", err.Error())
			return echo.NewHTTPError(http.StatusNotFound, "invalid body")
		} else {
			l.Error("product_patch_error", "status", 500, "reason", "cannot add product to db", "error", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "cannot add product to db")
		}
	}

	l.Info("patch_prosuct_success")
	return c.JSON(http.StatusOK, prod)
}

func (h *CatalogHTTP) DeleteProduct(c echo.Context) error {
	ctx := c.Request().Context()
	l := logging.FromContext(ctx).With("handler", "delete_product")

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		l.Warn("product_delete_error", "status", 400, "reason", "id not an uuid", "error", err)
		return echo.NewHTTPError(http.StatusBadRequest, "id not an uuid")
	}
	if err := h.Svc.DeleteProduct(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			l.Warn("product_delete_error", "status", 404, "reason", "product not found", "error", err)
			return echo.NewHTTPError(http.StatusNotFound, "product not found")
		}
		l.Error("product_delete_error", "status", 500, "reason", "cannot delete product from db", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "cannot delete product from db")
	}

	l.Info("delete_product_success")
	return c.NoContent(http.StatusNoContent)
}
