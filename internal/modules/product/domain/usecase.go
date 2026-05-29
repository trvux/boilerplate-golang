package domain

import (
	"context"
)

type ProductUsecase interface {
	CreateProduct(ctx context.Context, sku string, name string, price float64, qty int) (*Product, error)
	GetProduct(ctx context.Context, id uint64) (*Product, error)
	UpdateProduct(ctx context.Context, id uint64, name string, price float64) (*Product, error)
	DeleteProduct(ctx context.Context, id uint64) error
	ListProducts(ctx context.Context, page int, limit int) ([]*Product, error)
	AddStock(ctx context.Context, id uint64, qty int) (*Product, error)
}
