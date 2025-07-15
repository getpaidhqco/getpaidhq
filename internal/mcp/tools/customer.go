package tools

import (
	"payloop/internal/application/dto"
	"payloop/internal/mcp/schema"

	"github.com/mark3labs/mcp-go/mcp"
)

// Customer tool generator
var customerToolGenerator = schema.NewGenerator()

// NewCreateCustomerTool creates a new customer creation tool with schema from DTO
func NewCreateCustomerTool() mcp.Tool {
	tool, err := customerToolGenerator.GenerateToolFromDTO(
		"create_customer",
		"Create a new customer account. Organization ID is automatically extracted from authentication.",
		dto.CreateCustomerInput{},
	)
	if err != nil {
		// Fallback to basic tool if schema generation fails
		return mcp.NewTool("create_customer",
			mcp.WithDescription("Create a new customer account"),
		)
	}
	return tool
}

// NewGetCustomerTool creates a customer retrieval tool
func NewGetCustomerTool() mcp.Tool {
	// Simple ID-based lookup doesn't need complex DTO
	return mcp.NewTool("get_customer",
		mcp.WithDescription("Retrieve a customer by ID"),
		mcp.WithString("customer_id",
			mcp.Required(),
			mcp.Description("Customer ID to retrieve"),
		),
	)
}

// CustomerListFilters represents filters for listing customers
type CustomerListFilters struct {
	Page        int    `json:"page,omitempty" jsonschema:"minimum=1,description=Page number for pagination (default: 1)"`
	Limit       int    `json:"limit,omitempty" jsonschema:"minimum=1,maximum=100,description=Number of items per page (default: 20, max: 100)"`
	Status      string `json:"status,omitempty" jsonschema:"enum=active,enum=inactive,description=Filter by customer status"`
	EmailFilter string `json:"email_filter,omitempty" jsonschema:"description=Filter customers by email address (partial match)"`
}

// NewListCustomersTool creates a customer listing tool with schema
func NewListCustomersTool() mcp.Tool {
	tool, err := customerToolGenerator.GenerateToolFromDTO(
		"list_customers",
		"List customers with optional filtering and pagination",
		CustomerListFilters{},
	)
	if err != nil {
		// Fallback to basic tool if schema generation fails
		return mcp.NewTool("list_customers",
			mcp.WithDescription("List customers with optional filtering and pagination"),
		)
	}
	return tool
}

// NewUpdateCustomerTool creates a customer update tool with schema from DTO
func NewUpdateCustomerTool() mcp.Tool {
	// Create a combined input type for customer ID + update data
	type UpdateCustomerInput struct {
		CustomerID string `json:"customer_id" jsonschema:"required,description=Customer ID to update"`
		dto.UpdateCustomerInput
	}

	tool, err := customerToolGenerator.GenerateToolFromDTO(
		"update_customer", 
		"Update customer information. All fields except customer_id are optional.",
		UpdateCustomerInput{},
	)
	if err != nil {
		// Fallback to basic tool if schema generation fails
		return mcp.NewTool("update_customer",
			mcp.WithDescription("Update customer information"),
		)
	}
	return tool
}

// NewCreatePaymentMethodTool creates a payment method creation tool with schema
func NewCreatePaymentMethodTool() mcp.Tool {
	tool, err := customerToolGenerator.GenerateToolFromDTO(
		"create_payment_method",
		"Add a payment method to a customer account",
		dto.CreatePaymentMethodInput{},
	)
	if err != nil {
		// Fallback to basic tool if schema generation fails
		return mcp.NewTool("create_payment_method",
			mcp.WithDescription("Add a payment method to a customer account"),
		)
	}
	return tool
}

// NewUpdatePaymentMethodTool creates a payment method update tool with schema
func NewUpdatePaymentMethodTool() mcp.Tool {
	// Create a combined input type for payment method ID + update data
	type UpdatePaymentMethodInput struct {
		CustomerID      string `json:"customer_id" jsonschema:"required,description=Customer ID that owns the payment method"`
		PaymentMethodID string `json:"payment_method_id" jsonschema:"required,description=Payment method ID to update"`
		dto.UpdatePaymentMethodInput
	}

	tool, err := customerToolGenerator.GenerateToolFromDTO(
		"update_payment_method",
		"Update payment method information",
		UpdatePaymentMethodInput{},
	)
	if err != nil {
		// Fallback to basic tool if schema generation fails
		return mcp.NewTool("update_payment_method",
			mcp.WithDescription("Update payment method information"),
		)
	}
	return tool
}

// NewGetPaymentMethodTool creates a payment method retrieval tool
func NewGetPaymentMethodTool() mcp.Tool {
	return mcp.NewTool("get_payment_method",
		mcp.WithDescription("Retrieve a payment method by ID"),
		mcp.WithString("payment_method_id",
			mcp.Required(),
			mcp.Description("Payment method ID to retrieve"),
		),
	)
}