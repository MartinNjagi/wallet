package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// SafaricomIPs contains the known Safaricom API IP Subnets (Production & Sandbox)
var SafaricomIPs = []string{
	"196.201.214.",
	"196.201.213.",
	"196.201.212.",
	"196.201.211.",
}

// DarajaWebhookGuard unifies URL Secret validation and IP Whitelisting
func DarajaWebhookGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Validate the URL Secret
		expectedSecret := os.Getenv("MPESA_WEBHOOK_SECRET")
		if expectedSecret != "" && c.Param("secret") != expectedSecret {
			logrus.Warnf("WebhookGuard | Invalid secret attempt from IP: %s", c.ClientIP())
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		// 2. Validate the IP Address (IP Whitelisting)
		// Bypass IP check if we are in local development (useful for ngrok/postman)
		if os.Getenv("APP_ENV") == "local" || os.Getenv("APP_ENV") == "development" {
			c.Next()
			return
		}

		clientIP := c.ClientIP()
		isAllowed := false

		for _, subnet := range SafaricomIPs {
			if strings.HasPrefix(clientIP, subnet) {
				isAllowed = true
				break
			}
		}

		if !isAllowed {
			logrus.Warnf("WebhookGuard | Blocked request from non-Safaricom IP: %s", clientIP)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}

		c.Next()
	}
}
