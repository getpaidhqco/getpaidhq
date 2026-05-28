package lib

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewValidator_RegistersIso4217(t *testing.T) {
	// Two-way check: a valid currency passes, an invalid one fails. If the
	// iso4217 rule silently failed to register (the bug this hardening
	// addresses), the invalid case would pass — `validate:"iso4217"` would
	// be a no-op and any string would be accepted.
	v := NewValidator()

	type dto struct {
		Currency string `validate:"iso4217"`
	}

	require.NoError(t, v.Struct(dto{Currency: "USD"}))
	assert.Error(t, v.Struct(dto{Currency: "XXX"}), "invalid currency must be rejected; if this passes the iso4217 rule didn't register")
}
