package repository

import (
	"context"
	"errors"
	"time"

	"github.com/tranvux/boilerplate_golang/internal/modules/product/domain"
	"github.com/tranvux/boilerplate_golang/pkg/apperr"
	"github.com/tranvux/boilerplate_golang/pkg/database"
	"gorm.io/gorm"
)

type gormProduct struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement"`
	SKU       string    `gorm:"type:varchar(50);uniqueIndex;not null"`
	Name      string    `gorm:"type:varchar(255);not null"`
	Price     float64   `gorm:"type:decimal(10,2);not null"`
	Quantity  int       `gorm:"type:integer;not null"`
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

func (gormProduct) TableName() string {
	return "products"
}

type postgresProductRepository struct {
	db *database.PostgresDB
}

var _ domain.ProductRepository = (*postgresProductRepository)(nil)

func NewPostgresProductRepository(db *database.PostgresDB) domain.ProductRepository {
	return &postgresProductRepository{
		db: db,
	}
}

func toDomain(gp *gormProduct) *domain.Product {
	return &domain.Product{
		ID:        gp.ID,
		SKU:       gp.SKU,
		Name:      gp.Name,
		Price:     gp.Price,
		Quantity:  gp.Quantity,
		CreatedAt: gp.CreatedAt,
		UpdatedAt: gp.UpdatedAt,
	}
}

func fromDomain(dp *domain.Product) *gormProduct {
	return &gormProduct{
		ID:        dp.ID,
		SKU:       dp.SKU,
		Name:      dp.Name,
		Price:     dp.Price,
		Quantity:  dp.Quantity,
		CreatedAt: dp.CreatedAt,
		UpdatedAt: dp.UpdatedAt,
	}
}

func (r *postgresProductRepository) Create(ctx context.Context, p *domain.Product) error {
	gp := fromDomain(p)
	err := r.db.WithContext(ctx).Create(gp).Error
	if err != nil {
		return err
	}
	p.ID = gp.ID
	return nil
}

func (r *postgresProductRepository) GetByID(ctx context.Context, id uint64) (*domain.Product, error) {
	var gp gormProduct
	err := r.db.WithContext(ctx).First(&gp, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NewNotFoundError("PRODUCT_NOT_FOUND", "product not found")
		}
		return nil, err
	}
	return toDomain(&gp), nil
}

func (r *postgresProductRepository) GetBySKU(ctx context.Context, sku string) (*domain.Product, error) {
	var gp gormProduct
	err := r.db.WithContext(ctx).Where("sku = ?", sku).First(&gp).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NewNotFoundError("PRODUCT_NOT_FOUND", "product not found")
		}
		return nil, err
	}
	return toDomain(&gp), nil
}

func (r *postgresProductRepository) Update(ctx context.Context, p *domain.Product) error {
	gp := fromDomain(p)
	err := r.db.WithContext(ctx).Save(gp).Error
	return err
}

func (r *postgresProductRepository) Delete(ctx context.Context, id uint64) error {
	err := r.db.WithContext(ctx).Delete(&gormProduct{}, id).Error
	return err
}

func (r *postgresProductRepository) List(ctx context.Context, offset int, limit int) ([]*domain.Product, error) {
	var gps []gormProduct
	err := r.db.WithContext(ctx).Limit(limit).Offset(offset).Order("id ASC").Find(&gps).Error
	if err != nil {
		return nil, err
	}

	products := make([]*domain.Product, len(gps))
	for i := range gps {
		products[i] = toDomain(&gps[i])
	}
	return products, nil
}
