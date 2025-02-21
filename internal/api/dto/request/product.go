package request

// CreateProductRequest represents the request to create a product
type CreateProductRequest struct {
	Name        string            `json:"name" binding:"required"`
	Description string            `json:"description"`
	Metadata    map[string]string `json:"metadata"`
}
