// Package apigen contains the OpenAPI client generated from the server's
// committed contract (../../../docs/openapi.yml). Regenerate with `go generate`.
package apigen

//go:generate go run github.com/ogen-go/ogen/cmd/ogen@latest --config ogen.yml --target . --package apigen --clean ../../../docs/openapi.yml
