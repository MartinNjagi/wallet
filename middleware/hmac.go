package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"github.com/gin-gonic/gin"
	"io"
)

func VerifySignature(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, _ := io.ReadAll(c.Request.Body)
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body)) // restore for binding

		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		expected := hex.EncodeToString(mac.Sum(nil))
		incoming := c.Request.Header.Get("X-Signature")
		if !hmac.Equal([]byte(incoming), []byte(expected)) {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid signature"})
			return
		}
		c.Next()
	}
}
