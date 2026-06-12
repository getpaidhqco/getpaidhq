package port

// SecretCipher encrypts small secrets (PSP credentials) for storage on their
// owning row. Implementations bind the ciphertext to (orgId, id) so an
// envelope copied onto another row or org fails to decrypt.
type SecretCipher interface {
	// Encrypt seals plaintext into an opaque envelope string safe to store.
	Encrypt(orgId, id string, plaintext []byte) (string, error)
	// Decrypt opens an envelope produced by Encrypt for the same (orgId, id).
	Decrypt(orgId, id string, envelope string) ([]byte, error)
}
