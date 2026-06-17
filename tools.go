//go:build tools

package main

import (
	// goose library — imported here to keep it as a direct (non-indirect) dependency
	// so integration tests can reference it without it being pruned by go mod tidy.
	_ "github.com/pressly/goose/v3"
)
