package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Unit tests for the price-tier request parsers (decimalOrZero, toDomainTiers).
//
// Note on the malformed-decimal path: every PriceTierRequest decimal field is
// tagged `validate:"omitempty,numeric"`, and the set of strings accepted by the
// `numeric` validator is a subset of those decimal.NewFromString can parse. So
// any value that would make decimalOrZero return an error is also rejected by
// the validator first (HTTP 400) and never reaches toDomainTiers. The handler's
// "invalid tier value" branch is therefore unreachable from HTTP, and the parse
// error paths can only be exercised here. The HTTP-boundary behaviour (valid
// tiers round-tripping, malformed tiers rejected at validation) is covered by
// the TestProductHandler_*Tiers* tests in product_handler_test.go.

func TestDecimalOrZero(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		in      string
		want    string // canonical decimal string; only asserted when wantErr is false
		wantErr bool
	}{
		{name: "empty string short-circuits to zero", in: "", want: "0"},
		{name: "explicit zero", in: "0", want: "0"},
		{name: "positive integer", in: "100", want: "100"},
		{name: "decimal fraction", in: "1.5", want: "1.5"},
		{name: "sub-cent precision retained", in: "0.001", want: "0.001"},
		{name: "negative value", in: "-5", want: "-5"},
		{name: "leading plus normalized", in: "+5", want: "5"},
		{name: "scientific notation expanded", in: "1e3", want: "1000"},
		{name: "non-numeric garbage", in: "abc", wantErr: true},
		{name: "multiple decimal points", in: "1.2.3", wantErr: true},
		{name: "surrounding whitespace not trimmed", in: " 1 ", wantErr: true},
		{name: "comma decimal separator", in: "1,5", wantErr: true},
		{name: "lone decimal point", in: ".", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := decimalOrZero(tt.in)
			if tt.wantErr {
				require.Error(t, err, "decimalOrZero(%q) should error", tt.in)
				assert.True(t, got.IsZero(), "value should be zero on error")
				return
			}
			require.NoError(t, err, "decimalOrZero(%q) should not error", tt.in)
			assert.Equal(t, tt.want, got.String(), "decimalOrZero(%q)", tt.in)
		})
	}
}

func TestToDomainTiers(t *testing.T) {
	t.Parallel()

	t.Run("nil input returns nil", func(t *testing.T) {
		t.Parallel()
		got, err := toDomainTiers(nil)
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("empty slice returns nil", func(t *testing.T) {
		t.Parallel()
		got, err := toDomainTiers([]PriceTierRequest{})
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("single tier parses every decimal field and passes flat amount through", func(t *testing.T) {
		t.Parallel()
		got, err := toDomainTiers([]PriceTierRequest{
			{FromValue: "0", ToValue: "10.5", PerUnitAmount: "2.5", FlatAmount: 100},
		})
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "0", got[0].FromValue.String())
		assert.Equal(t, "10.5", got[0].ToValue.String())
		assert.Equal(t, "2.5", got[0].PerUnitAmount.String())
		assert.Equal(t, int64(100), got[0].FlatAmount)
	})

	t.Run("empty to_value yields the unbounded (zero) last tier", func(t *testing.T) {
		t.Parallel()
		got, err := toDomainTiers([]PriceTierRequest{
			{FromValue: "10", ToValue: "", PerUnitAmount: "1.25"},
		})
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.True(t, got[0].ToValue.IsZero(), "empty to_value must mean unbounded (zero)")
	})

	t.Run("empty from_value defaults to zero", func(t *testing.T) {
		t.Parallel()
		got, err := toDomainTiers([]PriceTierRequest{
			{FromValue: "", ToValue: "10", PerUnitAmount: "1"},
		})
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.True(t, got[0].FromValue.IsZero(), "empty from_value must default to zero")
	})

	t.Run("preserves order and index mapping across multiple tiers", func(t *testing.T) {
		t.Parallel()
		got, err := toDomainTiers([]PriceTierRequest{
			{FromValue: "0", ToValue: "10", PerUnitAmount: "5", FlatAmount: 1},
			{FromValue: "10", ToValue: "20", PerUnitAmount: "4", FlatAmount: 2},
			{FromValue: "20", ToValue: "", PerUnitAmount: "3", FlatAmount: 3},
		})
		require.NoError(t, err)
		require.Len(t, got, 3)
		assert.Equal(t, "5", got[0].PerUnitAmount.String())
		assert.Equal(t, "4", got[1].PerUnitAmount.String())
		assert.Equal(t, "3", got[2].PerUnitAmount.String())
		assert.Equal(t, []int64{1, 2, 3}, []int64{got[0].FlatAmount, got[1].FlatAmount, got[2].FlatAmount})
	})

	t.Run("overlapping tiers parse without a validation error", func(t *testing.T) {
		t.Parallel()
		// toDomainTiers is a pure parser: it does not reject overlapping or
		// out-of-order bands. This documents that contract (range validation,
		// if any, lives elsewhere).
		got, err := toDomainTiers([]PriceTierRequest{
			{FromValue: "0", ToValue: "100", PerUnitAmount: "1"},
			{FromValue: "50", ToValue: "150", PerUnitAmount: "1"},
		})
		require.NoError(t, err)
		assert.Len(t, got, 2)
	})

	// Error branches — each malformed field exercises a distinct return path in
	// toDomainTiers. Only reachable at the unit level (see file-level note).
	errTests := []struct {
		name string
		in   PriceTierRequest
	}{
		{name: "malformed from_value propagates error", in: PriceTierRequest{FromValue: "abc", ToValue: "10", PerUnitAmount: "1"}},
		{name: "malformed to_value propagates error", in: PriceTierRequest{FromValue: "0", ToValue: "1.2.3", PerUnitAmount: "1"}},
		{name: "malformed per_unit_amount propagates error", in: PriceTierRequest{FromValue: "0", ToValue: "10", PerUnitAmount: "x"}},
	}
	for _, tt := range errTests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := toDomainTiers([]PriceTierRequest{tt.in})
			require.Error(t, err)
			assert.Nil(t, got, "no partial slice should be returned on parse error")
		})
	}

	t.Run("error in a later tier still fails the whole parse", func(t *testing.T) {
		t.Parallel()
		got, err := toDomainTiers([]PriceTierRequest{
			{FromValue: "0", ToValue: "10", PerUnitAmount: "1"},
			{FromValue: "10", ToValue: "20", PerUnitAmount: "nope"},
		})
		require.Error(t, err)
		assert.Nil(t, got)
	})
}
