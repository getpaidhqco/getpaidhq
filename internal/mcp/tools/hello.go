package tools

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// NewHelloWorldTool creates a new hello world tool
func NewHelloWorldTool() mcp.Tool {
	return mcp.NewTool("hello_world",
		mcp.WithDescription("Say hello to someone"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the person to greet"),
		),
	)
}
