package http

type CreateProductRequest struct {
	SKU      string  `json:"sku" binding:"required,min=3,max=50"`
	Name     string  `json:"name" binding:"required,min=2,max=255"`
	Price    float64 `json:"price" binding:"required,gt=0"`
	Quantity int     `json:"quantity" binding:"required,gte=0"`
}

type UpdateProductRequest struct {
	Name  string  `json:"name" binding:"required,min=2,max=255"`
	Price float64 `json:"price" binding:"required,gt=0"`
}

type AddStockRequest struct {
	Quantity int `json:"quantity" binding:"required,gt=0"`
}
