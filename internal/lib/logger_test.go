package lib

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPanicf_Panics(t *testing.T) {
	// Panicf previously only logged at error level despite its name, so any
	// caller relying on it to halt control flow was silently passing
	// through. The panic must actually fire, and the panic value must
	// include the formatted message so a recover() sees something useful.
	l := newLogger(Env{Env: "development", LogLevel: "error"})

	assert.PanicsWithValue(
		t,
		"boom user=42",
		func() { l.Panicf("boom user=%d", 42) },
	)
}
