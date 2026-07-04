package controllers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"strconv"
	"strings"
	"wallet/data"
	"wallet/models"
)

// Controller holds the injected dependencies for authentication logic
type Controller struct {
	Config   *data.AppConfig
	DB       *gorm.DB
	Redis    *redis.Client
	Ctx      context.Context
	S3Client *s3.Client // <-- NEW
}

// SendJSON is a convenience wrapper for handlers
func SendJSON(ctx *gin.Context, resp data.APIResponse) {
	ctx.JSON(resp.Status, resp)
}

// deref safely dereferences a *string
func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
func Float64Ptr(v float64) *float64 {
	return &v
}

func StringPtr(v string) *string {
	return &v
}

// LogAudit creates an audit log entry. If tx == nil, it defaults to ctr.DB.
func (ctr *Controller) LogAudit(tx *gorm.DB, params data.AuditLogParams) error {
	var oldDataStr, newDataStr *string

	if params.OldData != nil {
		if oldBytes, err := json.Marshal(params.OldData); err == nil {
			str := string(oldBytes)
			oldDataStr = &str
		}
	}

	if params.NewData != nil {
		if newBytes, err := json.Marshal(params.NewData); err == nil {
			str := string(newBytes)
			newDataStr = &str
		}
	}

	audit := models.AuditLog{
		UserID:          params.UserID,
		Username:        params.Username,
		Action:          params.Action,
		OldData:         oldDataStr,
		NewData:         newDataStr,
		PerformedBy:     params.PerformedBy,
		PerformedByName: params.PerformedByName,
		IPAddress:       &params.IPAddress,
	}

	// ✅ Use ctr.DB if no tx provided
	if tx == nil {
		tx = ctr.DB
	}

	return tx.Create(&audit).Error
}

// getClientID securely extracts the client_id from the context session.
func getClientID(ctx *gin.Context) (uint, bool) {
	val, exists := ctx.Get("client_id")
	if !exists {
		return 0, false
	}
	return val.(uint), true
}

func normalizePhoneStr(phone string) string {
	clean := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, phone)
	return clean
}

// Helper: Generates a cryptographically secure random string
func generateSecureToken(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		panic(err) // rand.Read should never fail on a healthy OS
	}
	return hex.EncodeToString(b)
}

// Helper for parsing pagination queries safely
func getPaginationParams(ctx *gin.Context) (int, int, int) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	return page, pageSize, offset
}
