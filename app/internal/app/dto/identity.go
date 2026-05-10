package dto

type IdentityResponse struct {
	Role                 string `json:"role"`
	Email                string `json:"email"`
	PublicKeyPEM         string `json:"public_key_pem"`
	PublicKeyFingerprint string `json:"public_key_fingerprint"`
	SignatureAlgorithm   string `json:"signature_algorithm"`
	EncryptionAlgorithm  string `json:"encryption_algorithm"`
	KeyTransport         string `json:"key_transport"`
}
