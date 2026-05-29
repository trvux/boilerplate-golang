package domain

import (
	"errors"
	"time"
)

var (
	ErrSKURequired       = errors.New("product SKU is required")
	ErrNameRequired      = errors.New("product name is required")
	ErrInvalidPrice      = errors.New("product price must be greater than zero")
	ErrInsufficientStock = errors.New("insufficient stock for requested operation")
)

type Product struct {
	ID        uint64
	SKU       string
	Name      string
	Price     float64
	Quantity  int
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewProduct(sku string, name string, price float64, qty int) (*Product, error) {
	p := &Product{
		SKU:       sku,
		Name:      name,
		Price:     price,
		Quantity:  qty,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := p.Validate(); err != nil {
		return nil, err
	}

	return p, nil
}

func (p *Product) Validate() error {
	if p.SKU == "" {
		return ErrSKURequired
	}
	if p.Name == "" {
		return ErrNameRequired
	}
	if p.Price <= 0 {
		return ErrInvalidPrice
	}
	if p.Quantity < 0 {
		return errors.New("product quantity cannot be negative")
	}
	return nil
}

func (p *Product) UpdateDetails(name string, price float64) error {
	p.Name = name
	p.Price = price
	p.UpdatedAt = time.Now()
	return p.Validate()
}

func (p *Product) DeductStock(qty int) error {
	if qty < 0 {
		return errors.New("deduction quantity cannot be negative")
	}
	if p.Quantity < qty {
		return ErrInsufficientStock
	}
	p.Quantity -= qty
	p.UpdatedAt = time.Now()
	return nil
}

func (p *Product) AddStock(qty int) error {
	if qty < 0 {
		return errors.New("addition quantity cannot be negative")
	}
	p.Quantity += qty
	p.UpdatedAt = time.Now()
	return nil
}
