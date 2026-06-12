package domain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecret_EveryRenderingPathRedacts(t *testing.T) {
	s := Secret("sk_live_supersecret")

	assert.Equal(t, "[redacted]", s.String())
	assert.Equal(t, "[redacted]", fmt.Sprint(s))
	assert.Equal(t, "[redacted]", fmt.Sprintf("%v", s))
	assert.Equal(t, "[redacted]", fmt.Sprintf("%s", s))
	assert.Equal(t, "[redacted]", fmt.Sprintf("%#v", s))

	t.Run("inside a struct", func(t *testing.T) {
		v := struct {
			ApiKey Secret
			Other  string
		}{ApiKey: s, Other: "x"}
		out := fmt.Sprintf("%+v", v)
		assert.NotContains(t, out, "supersecret")
		assert.Contains(t, out, "[redacted]")
	})

	t.Run("json.Marshal", func(t *testing.T) {
		b, err := json.Marshal(map[string]Secret{"secret_key": s})
		require.NoError(t, err)
		assert.NotContains(t, string(b), "supersecret")
		assert.Contains(t, string(b), "[redacted]")
	})

	t.Run("slog", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		logger.Info("charge", "key", s)
		assert.NotContains(t, buf.String(), "supersecret")
		assert.Contains(t, buf.String(), "[redacted]")
	})
}

func TestSecret_RevealAndUnmarshal(t *testing.T) {
	var got struct {
		Key Secret `json:"key"`
	}
	require.NoError(t, json.Unmarshal([]byte(`{"key":"sk_live_abc"}`), &got))
	assert.Equal(t, "sk_live_abc", got.Key.Reveal())
	assert.False(t, got.Key.IsZero())
	assert.True(t, Secret("").IsZero())

	t.Run("RevealMap", func(t *testing.T) {
		m := RevealMap(map[string]Secret{"a": "1", "b": "2"})
		assert.Equal(t, map[string]string{"a": "1", "b": "2"}, m)
	})
}
