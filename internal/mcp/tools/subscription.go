package tools

import (
	"payloop/internal/application/dto"
	"payloop/internal/mcp/schema"

	"github.com/mark3labs/mcp-go/mcp"
)

// Subscription tool generator
var subscriptionToolGenerator = schema.NewGenerator()

// NewCreateSubscriptionTool creates a new subscription creation tool with schema from DTO
func NewCreateSubscriptionTool() mcp.Tool {
	tool, err := subscriptionToolGenerator.GenerateToolFromDTO(
		"create_subscription",
		"Create a new subscription for a customer. Organization ID is automatically extracted from authentication.",
		dto.CreateSubscriptionInput{},
	)
	if err != nil {
		// Fallback to basic tool if schema generation fails
		return mcp.NewTool("create_subscription",
			mcp.WithDescription("Create a new subscription for a customer"),
		)
	}
	return tool
}

// NewGetSubscriptionTool creates a subscription retrieval tool
func NewGetSubscriptionTool() mcp.Tool {
	return mcp.NewTool("get_subscription",
		mcp.WithDescription("Retrieve a subscription by ID"),
		mcp.WithString("subscription_id",
			mcp.Required(),
			mcp.Description("Subscription ID to retrieve"),
		),
	)
}

// NewListSubscriptionsTool creates a subscription listing tool with schema
func NewListSubscriptionsTool() mcp.Tool {
	tool, err := subscriptionToolGenerator.GenerateToolFromDTO(
		"list_subscriptions",
		"List subscriptions with optional filtering, sorting, and pagination",
		dto.SubscriptionListFilters{},
	)
	if err != nil {
		// Fallback to basic tool if schema generation fails
		return mcp.NewTool("list_subscriptions",
			mcp.WithDescription("List subscriptions with optional filtering, sorting, and pagination"),
		)
	}
	return tool
}

// NewUpdateSubscriptionTool creates a subscription update tool with schema
func NewUpdateSubscriptionTool() mcp.Tool {
	tool, err := subscriptionToolGenerator.GenerateToolFromDTO(
		"update_subscription",
		"Update subscription information such as status and metadata",
		dto.UpdateSubscriptionInput{},
	)
	if err != nil {
		// Fallback to basic tool if schema generation fails
		return mcp.NewTool("update_subscription",
			mcp.WithDescription("Update subscription information such as status and metadata"),
		)
	}
	return tool
}

// NewPauseSubscriptionTool creates a subscription pause tool with schema
func NewPauseSubscriptionTool() mcp.Tool {
	tool, err := subscriptionToolGenerator.GenerateToolFromDTO(
		"pause_subscription",
		"Pause an active subscription temporarily",
		dto.PauseSubscriptionInput{},
	)
	if err != nil {
		// Fallback to basic tool if schema generation fails
		return mcp.NewTool("pause_subscription",
			mcp.WithDescription("Pause an active subscription temporarily"),
		)
	}
	return tool
}

// NewResumeSubscriptionTool creates a subscription resume tool with schema
func NewResumeSubscriptionTool() mcp.Tool {
	tool, err := subscriptionToolGenerator.GenerateToolFromDTO(
		"resume_subscription",
		"Resume a paused subscription",
		dto.ResumeSubscriptionInput{},
	)
	if err != nil {
		// Fallback to basic tool if schema generation fails
		return mcp.NewTool("resume_subscription",
			mcp.WithDescription("Resume a paused subscription"),
		)
	}
	return tool
}

// NewCancelSubscriptionTool creates a subscription cancellation tool with schema
func NewCancelSubscriptionTool() mcp.Tool {
	tool, err := subscriptionToolGenerator.GenerateToolFromDTO(
		"cancel_subscription",
		"Cancel a subscription either immediately or at the end of the current billing period",
		dto.CancelSubscriptionInput{},
	)
	if err != nil {
		// Fallback to basic tool if schema generation fails
		return mcp.NewTool("cancel_subscription",
			mcp.WithDescription("Cancel a subscription either immediately or at the end of the current billing period"),
		)
	}
	return tool
}

// NewChangePlanTool creates a subscription plan change tool with schema
func NewChangePlanTool() mcp.Tool {
	tool, err := subscriptionToolGenerator.GenerateToolFromDTO(
		"change_subscription_plan",
		"Change a subscription's plan to a different product variant and price with proration options",
		dto.ChangePlanInput{},
	)
	if err != nil {
		// Fallback to basic tool if schema generation fails
		return mcp.NewTool("change_subscription_plan",
			mcp.WithDescription("Change a subscription's plan to a different product variant and price"),
		)
	}
	return tool
}

// NewGetSubscriptionPaymentsTool creates a tool to list payments for a subscription
func NewGetSubscriptionPaymentsTool() mcp.Tool {
	// Simple ID-based lookup with pagination
	return mcp.NewTool("get_subscription_payments",
		mcp.WithDescription("List all payments for a subscription"),
		mcp.WithString("subscription_id",
			mcp.Required(),
			mcp.Description("Subscription ID to list payments for"),
		),
		mcp.WithNumber("page",
			mcp.Description("Page number for pagination (default: 1)"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Number of items per page (default: 20, max: 100)"),
		),
	)
}

// NewGetSubscriptionUsageTool creates a tool to get usage data for a subscription
func NewGetSubscriptionUsageTool() mcp.Tool {
	return mcp.NewTool("get_subscription_usage",
		mcp.WithDescription("Get usage data for a subscription within a date range"),
		mcp.WithString("subscription_id",
			mcp.Required(),
			mcp.Description("Subscription ID to get usage for"),
		),
		mcp.WithString("start_date",
			mcp.Description("Start date for usage data (ISO 8601 format, e.g., 2024-01-01)"),
		),
		mcp.WithString("end_date",
			mcp.Description("End date for usage data (ISO 8601 format, e.g., 2024-01-31)"),
		),
	)
}