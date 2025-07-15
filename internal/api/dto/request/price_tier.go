package request

// CreatePriceTierRequest represents the request to create a price tier
type CreatePriceTierRequest struct {
	Tier        int     `json:"tier" binding:"required,gt=0"`
	FromQty     int     `json:"from_qty" binding:"required,gt=0"`
	ToQty       *int    `json:"to_qty" binding:"omitempty,gtfield=FromQty"`
	UnitPrice   int64   `json:"unit_price" binding:"required,gte=0"`
	Description string  `json:"description" binding:"omitempty,max=255"`
}