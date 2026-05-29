package http

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/tranvux/boilerplate_golang/internal/modules/product/domain"
	"github.com/tranvux/boilerplate_golang/pkg/apperr"
	"github.com/tranvux/boilerplate_golang/pkg/response"
)

type ProductHandler struct {
	usecase domain.ProductUsecase
}

func RegisterHandlers(rg *gin.RouterGroup, uc domain.ProductUsecase) {
	h := &ProductHandler{usecase: uc}

	products := rg.Group("/products")
	{
		products.POST("", h.CreateProduct)
		products.GET("", h.ListProducts)
		products.GET("/:id", h.GetProduct)
		products.PUT("/:id", h.UpdateProduct)
		products.DELETE("/:id", h.DeleteProduct)
		products.POST("/:id/stock", h.AddStock)
	}
}

func (h *ProductHandler) CreateProduct(c *gin.Context) {
	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperr.NewValidationError("REQUEST_BIND_FAILED", err.Error(), err))
		return
	}

	p, err := h.usecase.CreateProduct(c.Request.Context(), req.SKU, req.Name, req.Price, req.Quantity)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, MapToProductResponse(p))
}

func (h *ProductHandler) GetProduct(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.Error(c, apperr.NewValidationError("INVALID_PRODUCT_ID", "ID must be a positive integer", err))
		return
	}

	p, err := h.usecase.GetProduct(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, MapToProductResponse(p))
}

func (h *ProductHandler) ListProducts(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil {
		page = 1
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 10
	}

	products, err := h.usecase.ListProducts(c.Request.Context(), page, limit)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, MapToProductListResponse(products))
}

func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.Error(c, apperr.NewValidationError("INVALID_PRODUCT_ID", "ID must be a positive integer", err))
		return
	}

	var req UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperr.NewValidationError("REQUEST_BIND_FAILED", err.Error(), err))
		return
	}

	p, err := h.usecase.UpdateProduct(c.Request.Context(), id, req.Name, req.Price)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, MapToProductResponse(p))
}

func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.Error(c, apperr.NewValidationError("INVALID_PRODUCT_ID", "ID must be a positive integer", err))
		return
	}

	err = h.usecase.DeleteProduct(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, gin.H{"id": id, "deleted": true})
}

func (h *ProductHandler) AddStock(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.Error(c, apperr.NewValidationError("INVALID_PRODUCT_ID", "ID must be a positive integer", err))
		return
	}

	var req AddStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperr.NewValidationError("REQUEST_BIND_FAILED", err.Error(), err))
		return
	}

	p, err := h.usecase.AddStock(c.Request.Context(), id, req.Quantity)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, MapToProductResponse(p))
}
