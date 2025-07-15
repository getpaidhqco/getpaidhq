package tools

import (
	"payloop/internal/application/dto"
	"payloop/internal/mcp/schema"

	"github.com/mark3labs/mcp-go/mcp"
)

// Product tool generator
var productToolGenerator = schema.NewGenerator()

// NewCreateProductTool creates a new product creation tool with schema from DTO
func NewCreateProductTool() mcp.Tool {
	tool, err := productToolGenerator.GenerateToolFromDTO(
		"create_product",
		"Create a new product in the catalog. Organization ID is automatically extracted from authentication.",
		dto.CreateProductInput{},
	)
	if err != nil {
		// Fallback to basic tool if schema generation fails
		return mcp.NewTool("create_product",
			mcp.WithDescription("Create a new product in the catalog"),
		)
	}
	return tool
}

// NewGetProductTool creates a product retrieval tool
func NewGetProductTool() mcp.Tool {
	return mcp.NewTool("get_product",
		mcp.WithDescription("Retrieve a product by ID"),
		mcp.WithString("product_id",
			mcp.Required(),
			mcp.Description("Product ID to retrieve"),
		),
	)
}

// NewListProductsTool creates a product listing tool with schema
func NewListProductsTool() mcp.Tool {
	tool, err := productToolGenerator.GenerateToolFromDTO(
		"list_products",
		"List products with optional pagination",
		dto.ProductListFilters{},
	)
	if err != nil {
		// Fallback to basic tool if schema generation fails
		return mcp.NewTool("list_products",
			mcp.WithDescription("List products with optional pagination"),
		)
	}
	return tool
}

// NewUpdateProductTool creates a product update tool with schema from DTO
func NewUpdateProductTool() mcp.Tool {
	// Create a combined input type for product ID + update data
	type UpdateProductInput struct {
		ProductID string `json:"product_id" jsonschema:"required,description=Product ID to update"`
		dto.UpdateProductInput
	}

	tool, err := productToolGenerator.GenerateToolFromDTO(
		"update_product",
		"Update product information. All fields except product_id are optional.",
		UpdateProductInput{},
	)
	if err != nil {
		// Fallback to basic tool if schema generation fails
		return mcp.NewTool("update_product",
			mcp.WithDescription("Update product information"),
		)
	}
	return tool
}

// NewDeleteProductTool creates a product deletion tool
func NewDeleteProductTool() mcp.Tool {
	return mcp.NewTool("delete_product",
		mcp.WithDescription("Delete a product from the catalog"),
		mcp.WithString("product_id",
			mcp.Required(),
			mcp.Description("Product ID to delete"),
		),
	)
}

// NewCreateVariantTool creates a variant creation tool with schema
func NewCreateVariantTool() mcp.Tool {
	tool, err := productToolGenerator.GenerateToolFromDTO(
		"create_variant",
		"Create a new product variant",
		dto.CreateVariantInput{},
	)
	if err != nil {
		// Fallback to basic tool if schema generation fails
		return mcp.NewTool("create_variant",
			mcp.WithDescription("Create a new product variant"),
		)
	}
	return tool
}

// NewGetVariantTool creates a variant retrieval tool
func NewGetVariantTool() mcp.Tool {
	return mcp.NewTool("get_variant",
		mcp.WithDescription("Retrieve a product variant by ID"),
		mcp.WithString("variant_id",
			mcp.Required(),
			mcp.Description("Variant ID to retrieve"),
		),
	)
}

// NewListVariantsTool creates a variant listing tool with schema
func NewListVariantsTool() mcp.Tool {
	tool, err := productToolGenerator.GenerateToolFromDTO(
		"list_variants",
		"List variants for a product with optional pagination",
		dto.VariantListFilters{},
	)
	if err != nil {
		// Fallback to basic tool if schema generation fails
		return mcp.NewTool("list_variants",
			mcp.WithDescription("List variants for a product with optional pagination"),
		)
	}
	return tool
}

// NewUpdateVariantTool creates a variant update tool with schema
func NewUpdateVariantTool() mcp.Tool {
	// Create a combined input type for variant ID + update data
	type UpdateVariantInput struct {
		VariantID string `json:"variant_id" jsonschema:"required,description=Variant ID to update"`
		dto.UpdateVariantInput
	}

	tool, err := productToolGenerator.GenerateToolFromDTO(
		"update_variant",
		"Update variant information. All fields except variant_id are optional.",
		UpdateVariantInput{},
	)
	if err != nil {
		// Fallback to basic tool if schema generation fails
		return mcp.NewTool("update_variant",
			mcp.WithDescription("Update variant information"),
		)
	}
	return tool
}

// NewDeleteVariantTool creates a variant deletion tool
func NewDeleteVariantTool() mcp.Tool {
	return mcp.NewTool("delete_variant",
		mcp.WithDescription("Delete a product variant"),
		mcp.WithString("variant_id",
			mcp.Required(),
			mcp.Description("Variant ID to delete"),
		),
	)
}

// NewCreatePriceTool creates a price creation tool with comprehensive schema
func NewCreatePriceTool() mcp.Tool {
	tool, err := productToolGenerator.GenerateToolFromDTO(
		"create_price",
		"Create a new price for a product variant. Supports all pricing models including usage-based billing, tiered pricing, and subscription billing.",
		dto.CreatePriceInput{},
	)
	if err != nil {
		// Fallback to basic tool if schema generation fails
		return mcp.NewTool("create_price",
			mcp.WithDescription("Create a new price for a product variant"),
		)
	}
	return tool
}

// NewGetPriceTool creates a price retrieval tool
func NewGetPriceTool() mcp.Tool {
	return mcp.NewTool("get_price",
		mcp.WithDescription("Retrieve a price by ID"),
		mcp.WithString("price_id",
			mcp.Required(),
			mcp.Description("Price ID to retrieve"),
		),
	)
}

// NewListPricesTool creates a price listing tool with schema
func NewListPricesTool() mcp.Tool {
	tool, err := productToolGenerator.GenerateToolFromDTO(
		"list_prices",
		"List prices for a variant with optional pagination",
		dto.PriceListFilters{},
	)
	if err != nil {
		// Fallback to basic tool if schema generation fails
		return mcp.NewTool("list_prices",
			mcp.WithDescription("List prices for a variant with optional pagination"),
		)
	}
	return tool
}

// NewUpdatePriceTool creates a price update tool with schema
func NewUpdatePriceTool() mcp.Tool {
	// Create a combined input type for price ID + update data
	type UpdatePriceInput struct {
		PriceID string `json:"price_id" jsonschema:"required,description=Price ID to update"`
		dto.UpdatePriceInput
	}

	tool, err := productToolGenerator.GenerateToolFromDTO(
		"update_price",
		"Update price information. All fields except price_id are optional.",
		UpdatePriceInput{},
	)
	if err != nil {
		// Fallback to basic tool if schema generation fails
		return mcp.NewTool("update_price",
			mcp.WithDescription("Update price information"),
		)
	}
	return tool
}

// NewDeletePriceTool creates a price deletion tool
func NewDeletePriceTool() mcp.Tool {
	return mcp.NewTool("delete_price",
		mcp.WithDescription("Delete a price"),
		mcp.WithString("price_id",
			mcp.Required(),
			mcp.Description("Price ID to delete"),
		),
	)
}