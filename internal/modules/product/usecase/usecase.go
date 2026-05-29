package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/tranvux/boilerplate_golang/internal/modules/product/domain"
	"github.com/tranvux/boilerplate_golang/pkg/apperr"
	"github.com/tranvux/boilerplate_golang/pkg/logger"
	"go.uber.org/zap"
)

type productUsecase struct {
	repo domain.ProductRepository
	log  logger.Logger
}

var _ domain.ProductUsecase = (*productUsecase)(nil)

func NewProductUsecase(repo domain.ProductRepository, log logger.Logger) domain.ProductUsecase {
	return &productUsecase{
		repo: repo,
		log:  log,
	}
}

func (u *productUsecase) CreateProduct(ctx context.Context, sku string, name string, price float64, qty int) (*domain.Product, error) {
	// Business validation
	p, err := domain.NewProduct(sku, name, price, qty)
	if err != nil {
		return nil, apperr.NewValidationError("PRODUCT_INVALID", err.Error(), err)
	}

	// Check duplicates
	existing, err := u.repo.GetBySKU(ctx, sku)
	if err != nil && !errors.Is(err, apperr.NewNotFoundError("PRODUCT_NOT_FOUND", "")) {
		// If it's a real database error, wrap it as internal
		if apperr.GetErrorType(err) != apperr.TypeNotFound {
			return nil, apperr.NewInternalError("DATABASE_ERROR", "failed to check product existence", err)
		}
	}
	if existing != nil {
		return nil, apperr.NewConflictError("PRODUCT_SKU_EXISTS", fmt.Sprintf("product with SKU %s already exists", sku), nil)
	}

	if err := u.repo.Create(ctx, p); err != nil {
		return nil, apperr.NewInternalError("DATABASE_ERROR", "failed to save product", err)
	}

	u.log.Info("Product created successfully", zap.Uint64("product_id", p.ID), zap.String("sku", p.SKU))
	return p, nil
}

func (u *productUsecase) GetProduct(ctx context.Context, id uint64) (*domain.Product, error) {
	p, err := u.repo.GetByID(ctx, id)
	if err != nil {
		// Already formatted as AppError if handled properly by repo, but we ensure it here
		if errors.Is(err, apperr.NewNotFoundError("PRODUCT_NOT_FOUND", "")) || apperr.GetErrorType(err) == apperr.TypeNotFound {
			return nil, apperr.NewNotFoundError("PRODUCT_NOT_FOUND", fmt.Sprintf("product with ID %d was not found", id))
		}
		return nil, apperr.NewInternalError("DATABASE_ERROR", "failed to fetch product", err)
	}
	return p, nil
}

func (u *productUsecase) UpdateProduct(ctx context.Context, id uint64, name string, price float64) (*domain.Product, error) {
	p, err := u.GetProduct(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := p.UpdateDetails(name, price); err != nil {
		return nil, apperr.NewValidationError("PRODUCT_INVALID", err.Error(), err)
	}

	if err := u.repo.Update(ctx, p); err != nil {
		return nil, apperr.NewInternalError("DATABASE_ERROR", "failed to update product", err)
	}

	u.log.Info("Product updated successfully", zap.Uint64("product_id", p.ID))
	return p, nil
}

func (u *productUsecase) DeleteProduct(ctx context.Context, id uint64) error {
	_, err := u.GetProduct(ctx, id)
	if err != nil {
		return err
	}

	if err := u.repo.Delete(ctx, id); err != nil {
		return apperr.NewInternalError("DATABASE_ERROR", "failed to delete product", err)
	}

	u.log.Info("Product deleted successfully", zap.Uint64("product_id", id))
	return nil
}

func (u *productUsecase) ListProducts(ctx context.Context, page int, limit int) ([]*domain.Product, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	offset := (page - 1) * limit
	products, err := u.repo.List(ctx, offset, limit)
	if err != nil {
		return nil, apperr.NewInternalError("DATABASE_ERROR", "failed to list products", err)
	}

	return products, nil
}

func (u *productUsecase) AddStock(ctx context.Context, id uint64, qty int) (*domain.Product, error) {
	p, err := u.GetProduct(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := p.AddStock(qty); err != nil {
		return nil, apperr.NewValidationError("PRODUCT_INVALID", err.Error(), err)
	}

	if err := u.repo.Update(ctx, p); err != nil {
		return nil, apperr.NewInternalError("DATABASE_ERROR", "failed to adjust product stock", err)
	}

	u.log.Info("Product stock added successfully", zap.Uint64("product_id", p.ID), zap.Int("added_qty", qty), zap.Int("new_total", p.Quantity))
	return p, nil
}
