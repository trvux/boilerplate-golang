package domain

import (
	"context"
)

type ProductRepository interface {
	Create(ctx context.Context, product *Product) error
	GetByID(ctx context.Context, id uint64) (*Product, error)
	GetBySKU(ctx context.Context, sku string) (*Product, error)
	Update(ctx context.Context, product *Product) error
	Delete(ctx context.Context, id uint64) error
	List(ctx context.Context, offset int, limit int) ([]*Product, error)
}
