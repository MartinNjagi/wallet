package models

import "time"

// ClientBillingConfig holds the wallet configurations for a specific tenant.
type ClientBillingConfig struct {
	ID                     uint    `gorm:"column:id;primaryKey;autoIncrement"`
	ClientID               uint    `gorm:"uniqueIndex;not null"`
	BaseSmsRate            float64 `gorm:"default:1.0"`  // Cost in KES per 1 SMS Credit
	RefundOnFailedDelivery bool    `gorm:"default:true"` // Toggle for automatic refunds on failure
}

func (ClientBillingConfig) TableName() string {
	return "client_billing_config"
}

type AuditLog struct {
	ID              uint      `gorm:"column:id;primaryKey;autoIncrement"`
	UserID          uint      `gorm:"column:user_id;index;not null"` // ID of user being modified
	Username        string    `gorm:"column:username"`
	Action          string    `gorm:"column:action;not null"`    // e.g., "soft_delete", "update", "create"
	OldData         *string   `gorm:"column:old_data;type:json"` // JSON of old values
	NewData         *string   `gorm:"column:new_data;type:json"` // JSON of new values
	PerformedBy     *uint     `gorm:"column:performed_by"`       // ID of admin who performed action
	PerformedByName *string   `gorm:"column:performed_by_name"`
	IPAddress       *string   `gorm:"column:ip_address"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}
