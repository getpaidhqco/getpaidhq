package domain

import (
	"encoding/json"
	"log/slog"
)

// Secret is a string that refuses to leave the process readably. Every
// rendering path — fmt verbs, slog, json.Marshal — prints "[redacted]";
// the plaintext is reachable only through an explicit Reveal() at the
// point of use (constructing a PSP SDK client). Unmarshaling is normal so
// secrets can be parsed INTO from request bodies and decrypted blobs.
type Secret string

const redactedPlaceholder = "[redacted]"

// Reveal returns the plaintext. Call it only at the boundary that needs
// the real value; never store or log the result.
func (s Secret) Reveal() string { return string(s) }

// IsZero reports whether the secret is empty.
func (s Secret) IsZero() bool { return s == "" }

func (s Secret) String() string   { return redactedPlaceholder }
func (s Secret) GoString() string { return redactedPlaceholder } // %#v

func (s Secret) MarshalJSON() ([]byte, error) { return json.Marshal(redactedPlaceholder) }

func (s *Secret) UnmarshalJSON(b []byte) error {
	var v string
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	*s = Secret(v)
	return nil
}

// LogValue keeps slog output redacted even when the Secret is logged directly.
func (s Secret) LogValue() slog.Value { return slog.StringValue(redactedPlaceholder) }

// RevealMap converts a credentials map to plain strings for serialization
// into an encrypted envelope. The caller owns keeping the result off every
// logging/marshaling path.
func RevealMap(m map[string]Secret) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v.Reveal()
	}
	return out
}
