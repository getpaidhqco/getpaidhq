package handler

import (
	"reflect"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// EnumSchemaCustomizer renders a `validate:"...,oneof=a b c"` struct tag as an
// OpenAPI `enum` on the field's schema. Fuego's built-in customizer (which
// handles required/min/max) runs first and composes with this one, so a field
// declared `validate:"oneof=draft open paid uncollectible void"` is both
// runtime-validated by the request validator AND rendered as a constrained enum
// in the spec — instead of an opaque `type: string`. This is wired in
// internal/config/server.go via fuego.WithOpenAPIGeneratorSchemaCustomizer.
//
// The same tag works on response-only fields (which the validator never sees) to
// constrain them in the spec, so generated SDK clients get a real enum type for
// status fields rather than a bare string.
func EnumSchemaCustomizer(name string, t reflect.Type, tag reflect.StructTag, schema *openapi3.Schema) error {
	values := oneofValues(tag.Get("validate"))
	if len(values) == 0 {
		return nil
	}
	enum := make([]any, len(values))
	for i, v := range values {
		enum[i] = v
	}
	schema.Enum = enum
	return nil
}

// oneofValues extracts the space-separated values of a `oneof=...` rule from a
// go-playground/validator tag, or nil if the tag has no oneof rule. Rules are
// comma-separated; the oneof values themselves are space-separated and never
// contain commas, so splitting on commas first is safe.
func oneofValues(validateTag string) []string {
	for _, rule := range strings.Split(validateTag, ",") {
		if after, ok := strings.CutPrefix(strings.TrimSpace(rule), "oneof="); ok {
			return strings.Fields(after)
		}
	}
	return nil
}
