package handler

import (
	"net/http"

	"electronic-digital-signature/internal/app/dto"
	"electronic-digital-signature/internal/infra/crypto"
	"electronic-digital-signature/internal/infra/encryption"

	"github.com/gin-gonic/gin"
)

type IdentityHandler struct {
	role        string
	email       string
	publicKey   string
	fingerprint string
}

func NewIdentityHandler(role, email, publicKey, fingerprint string) *IdentityHandler {
	return &IdentityHandler{
		role:        role,
		email:       email,
		publicKey:   publicKey,
		fingerprint: fingerprint,
	}
}

func (h *IdentityHandler) GetIdentity(ctx *gin.Context) {
	respondSuccess(ctx, http.StatusOK, dto.IdentityResponse{
		Role:                 h.role,
		Email:                h.email,
		PublicKeyPEM:         h.publicKey,
		PublicKeyFingerprint: h.fingerprint,
		SignatureAlgorithm:   crypto.ECDSASHA256Algorithm,
		EncryptionAlgorithm:  encryption.AESGCMAlgorithm,
		KeyTransport:         encryption.PlaintextDemoKey,
	})
}
