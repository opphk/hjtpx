package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var advancedCryptoService = service.NewAdvancedCryptoService()

type GenerateKeyRequest struct {
	Algorithm string `json:"algorithm"`
	Size      int    `json:"size"`
}

type EncryptRequest struct {
	Plaintext string `json:"plaintext"`
	KeyID     string `json:"key_id"`
}

type DecryptRequest struct {
	Version    int    `json:"version"`
	Algorithm  string `json:"algorithm"`
	Ciphertext string `json:"ciphertext"`
	IV         string `json:"iv"`
	KeyID      string `json:"key_id"`
	Timestamp  int64  `json:"timestamp"`
	Nonce      string `json:"nonce"`
}

type HashRequest struct {
	Data string `json:"data"`
}

func GenerateAdvancedKey(c *gin.Context) {
	req := &GenerateKeyRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request")
		return
	}

	keyResp, err := advancedCryptoService.GenerateKey(c)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, keyResp)
}

func EncryptAdvanced(c *gin.Context) {
	req := &EncryptRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request")
		return
	}

	payload, err := advancedCryptoService.Encrypt(c, req.Plaintext, req.KeyID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, payload)
}

func DecryptAdvanced(c *gin.Context) {
	req := &DecryptRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request")
		return
	}

	payload := &service.EncryptedPayload{
		Version:    req.Version,
		Algorithm:  req.Algorithm,
		Ciphertext: req.Ciphertext,
		IV:         req.IV,
		KeyID:      req.KeyID,
		Timestamp:  req.Timestamp,
		Nonce:      req.Nonce,
	}

	plaintext, err := advancedCryptoService.Decrypt(c, payload)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, map[string]string{
		"plaintext": plaintext,
	})
}

func GenerateQuantumHash(c *gin.Context) {
	req := &HashRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request")
		return
	}

	hash := advancedCryptoService.GenerateQuantumResistantHash([]byte(req.Data))

	response.Success(c, map[string]string{
		"hash": hash,
		"algorithm": "SHA-256",
	})
}

func GetActiveKeys(c *gin.Context) {
	keys, err := advancedCryptoService.GetActiveKeys(c)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, map[string]interface{}{
		"keys": keys,
		"count": len(keys),
	})
}
