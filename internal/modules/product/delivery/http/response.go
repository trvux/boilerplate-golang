package http

import (
	"time"

	"github.com/tranvux/boilerplate_golang/internal/modules/product/domain"
)

type ProductResponse struct {
	ID        uint64    `json:"id"`
	SKU       string    `json:"sku"`
	Name      string    `json:"name"`
	Price     float64   `json:"price"`
	Quantity  int       `json:"quantity"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func MapToProductResponse(p *domain.Product) ProductResponse {
	return ProductResponse{
		ID:        p.ID,
		SKU:       p.SKU,
		Name:      p.Name,
		Price:     p.Price,
		Quantity:  p.Quantity,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

func MapToProductListResponse(products []*domain.Product) []ProductResponse {
	res := make([]ProductResponse, len(products))
	for i, p := range products {
		res[i] = MapToProductResponse(p)
	}
	return res
}
