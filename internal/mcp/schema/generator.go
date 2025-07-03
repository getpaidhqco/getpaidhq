package schema

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/invopop/jsonschema"
	"github.com/mark3labs/mcp-go/mcp"
)

// Generator creates JSON schemas from DTO structs for MCP tools
type Generator struct {
	reflector *jsonschema.Reflector
}

// NewGenerator creates a new schema generator
func NewGenerator() *Generator {
	reflector := &jsonschema.Reflector{
		// Ensure all fields are included even if they have omitempty
		RequiredFromJSONSchemaTags: true,
		// Expand references for better MCP compatibility
		ExpandedStruct: true,
	}

	return &Generator{
		reflector: reflector,
	}
}

// GenerateToolFromDTO creates an MCP tool with JSON schema generated from a DTO struct
func (g *Generator) GenerateToolFromDTO(toolName, description string, dtoType interface{}) (mcp.Tool, error) {
	// Generate schema from the DTO type
	schema := g.reflector.Reflect(dtoType)
	
	// Convert schema to JSON for MCP tool creation
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return mcp.Tool{}, fmt.Errorf("failed to marshal schema: %w", err)
	}

	// Create MCP tool with the generated schema using NewToolWithRawSchema
	tool := mcp.NewToolWithRawSchema(toolName, description, json.RawMessage(schemaBytes))

	return tool, nil
}

// ParseInputToDTO parses MCP request arguments to the target DTO type
func (g *Generator) ParseInputToDTO(args map[string]interface{}, target interface{}) error {
	// Convert args to JSON and then unmarshal to target struct
	argsBytes, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("failed to marshal args: %w", err)
	}

	if err := json.Unmarshal(argsBytes, target); err != nil {
		return fmt.Errorf("failed to unmarshal to DTO: %w", err)
	}

	return nil
}

// ValidateInput validates input against the DTO schema
func (g *Generator) ValidateInput(args map[string]interface{}, dtoType interface{}) error {
	// For now, we'll rely on the unmarshaling to the DTO struct for validation
	// In the future, we could add a separate validation library
	
	// Try to parse to DTO to validate structure
	return g.ParseInputToDTO(args, dtoType)
}

// GetFieldNamesFromDTO extracts field names from a DTO struct for documentation
func (g *Generator) GetFieldNamesFromDTO(dtoType interface{}) []string {
	t := reflect.TypeOf(dtoType)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	var fields []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" && jsonTag != "-" {
			// Extract field name from json tag
			fieldName := jsonTag
			if commaIdx := len(jsonTag); commaIdx > 0 {
				for j, r := range jsonTag {
					if r == ',' {
						commaIdx = j
						break
					}
				}
				fieldName = jsonTag[:commaIdx]
			}
			if fieldName != "" {
				fields = append(fields, fieldName)
			}
		} else {
			fields = append(fields, field.Name)
		}
	}

	return fields
}