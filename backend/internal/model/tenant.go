package model

import (
	"time"

	"gorm.io/gorm"
)

// Tenant 租户模型
type Tenant struct {
	gorm.Model
	Name            string         `gorm:"size:255;not null;uniqueIndex:idx_tenant_name" json:"name"`
	Domain          string         `gorm:"size:255;uniqueIndex:idx_tenant_domain" json:"domain"`
	Subdomain       string         `gorm:"size:100;uniqueIndex:idx_tenant_subdomain" json:"subdomain"`
	Description     string         `gorm:"type:text" json:"description,omitempty"`
	Status          string         `gorm:"size:20;default:active;index:idx_tenant_status" json:"status"`
	Plan            string         `gorm:"size:50;default:free" json:"plan"`
	Email           string         `gorm:"size:255" json:"email"`
	Phone           string         `gorm:"size:20" json:"phone"`
	Address         string         `gorm:"type:text" json:"address,omitempty"`
	LogoURL         string         `gorm:"size:500" json:"logo_url,omitempty"`
	Industry        string         `gorm:"size:100" json:"industry,omitempty"`
	Timezone        string         `gorm:"size:50;default:Asia/Shanghai" json:"timezone"`
	Currency        string         `gorm:"size:10;default:CNY" json:"currency"`
	Language        string         `gorm:"size:10;default:zh-CN" json:"language"`
	CreatedBy       uint           `json:"created_by"`
	UpdatedBy       uint           `json:"updated_by"`
	ExpiresAt       *time.Time     `json:"expires_at,omitempty"`
	IsDeleted       bool           `gorm:"default:false" json:"is_deleted"`
	TenantUsers     []TenantUser   `gorm:"foreignKey:TenantID" json:"tenant_users,omitempty"`
	TenantQuotas    []TenantQuota  `gorm:"foreignKey:TenantID" json:"tenant_quotas,omitempty"`
	TenantBillings  []TenantBilling `gorm:"foreignKey:TenantID" json:"tenant_billings,omitempty"`
}

func (Tenant) TableName() string {
	return "tenants"
}

// TenantUser 租户用户关联模型
type TenantUser struct {
	gorm.Model
	TenantID    uint      `gorm:"not null;index:idx_tenant_user_tenant;index:idx_tenant_user_unique" json:"tenant_id"`
	UserID      uint      `gorm:"not null;index:idx_tenant_user_user;index:idx_tenant_user_unique" json:"user_id"`
	Role        string    `gorm:"size:50;default:member" json:"role"` // owner, admin, member
	Status      string    `gorm:"size:20;default:active" json:"status"`
	JoinedAt    time.Time `json:"joined_at"`
	InvitedBy   uint      `json:"invited_by"`
	Tenant      Tenant    `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
}

func (TenantUser) TableName() string {
	return "tenant_users"
}

// TenantQuota 租户配额模型
type TenantQuota struct {
	gorm.Model
	TenantID          uint      `gorm:"not null;index:idx_tenant_quota_tenant" json:"tenant_id"`
	ResourceType      string    `gorm:"size:100;not null;index:idx_tenant_quota_resource" json:"resource_type"`
	Limit             int64     `gorm:"default:0" json:"limit"`
	Used              int64     `gorm:"default:0" json:"used"`
	Remaining         int64     `gorm:"default:0" json:"remaining"`
	WarningThreshold  float64   `gorm:"default:80" json:"warning_threshold"`
	HardLimit         bool      `gorm:"default:true" json:"hard_limit"`
	PeriodType        string    `gorm:"size:20;default:monthly" json:"period_type"` // daily, weekly, monthly, yearly
	ResetAt           time.Time `json:"reset_at"`
	LastConsumedAt    *time.Time `json:"last_consumed_at,omitempty"`
	IsActive          bool      `gorm:"default:true" json:"is_active"`
	Tenant            Tenant    `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
}

func (TenantQuota) TableName() string {
	return "tenant_quotas"
}

// TenantBilling 租户账单模型
type TenantBilling struct {
	gorm.Model
	TenantID        uint       `gorm:"not null;index:idx_tenant_billing_tenant" json:"tenant_id"`
	BillNumber      string     `gorm:"size:50;uniqueIndex:idx_bill_number" json:"bill_number"`
	PeriodStart     time.Time  `json:"period_start"`
	PeriodEnd       time.Time  `json:"period_end"`
	TotalAmount     float64    `gorm:"default:0" json:"total_amount"`
	TaxAmount       float64    `gorm:"default:0" json:"tax_amount"`
	DiscountAmount  float64    `gorm:"default:0" json:"discount_amount"`
	PayableAmount   float64    `gorm:"default:0" json:"payable_amount"`
	Status          string     `gorm:"size:20;default:pending;index:idx_bill_status" json:"status"` // pending, paid, overdue, canceled
	PaymentMethod   string     `gorm:"size:50" json:"payment_method,omitempty"`
	PaymentAt       *time.Time `json:"payment_at,omitempty"`
	PaymentTransactionID string `gorm:"size:100" json:"payment_transaction_id,omitempty"`
	InvoiceNumber   string     `gorm:"size:50" json:"invoice_number,omitempty"`
	Description     string     `gorm:"type:text" json:"description,omitempty"`
	Tenant          Tenant     `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
}

func (TenantBilling) TableName() string {
	return "tenant_billings"
}

// TenantPaymentRecord 租户支付记录模型
type TenantPaymentRecord struct {
	gorm.Model
	TenantID        uint       `gorm:"not null;index:idx_payment_tenant" json:"tenant_id"`
	BillID          uint       `gorm:"index:idx_payment_bill" json:"bill_id"`
	Amount          float64    `gorm:"default:0" json:"amount"`
	PaymentMethod   string     `gorm:"size:50;not null" json:"payment_method"` // wechat, alipay, bank, stripe
	TransactionID   string     `gorm:"size:100;uniqueIndex:idx_payment_transaction" json:"transaction_id"`
	Status          string     `gorm:"size:20;default:pending;index:idx_payment_status" json:"status"` // pending, success, failed, refunded
	RefundAmount    float64    `gorm:"default:0" json:"refund_amount"`
	RefundReason    string     `gorm:"type:text" json:"refund_reason,omitempty"`
	FailedReason    string     `gorm:"type:text" json:"failed_reason,omitempty"`
	PaymentAt       *time.Time `json:"payment_at,omitempty"`
	Tenant          Tenant     `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Billing         *TenantBilling `gorm:"foreignKey:BillID" json:"billing,omitempty"`
}

func (TenantPaymentRecord) TableName() string {
	return "tenant_payment_records"
}

// TenantFeature 租户功能开关模型
type TenantFeature struct {
	gorm.Model
	TenantID    uint   `gorm:"not null;index:idx_tenant_feature_tenant" json:"tenant_id"`
	FeatureKey  string `gorm:"size:100;not null;index:idx_tenant_feature_key" json:"feature_key"`
	IsEnabled   bool   `gorm:"default:false" json:"is_enabled"`
	Config      string `gorm:"type:text" json:"config,omitempty"` // JSON配置
	Description string `gorm:"type:text" json:"description,omitempty"`
	Tenant      Tenant `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
}

func (TenantFeature) TableName() string {
	return "tenant_features"
}

// TenantUsage 租户使用量统计模型
type TenantUsage struct {
	gorm.Model
	TenantID        uint      `gorm:"not null;index:idx_tenant_usage_tenant" json:"tenant_id"`
	ResourceType    string    `gorm:"size:100;not null;index:idx_tenant_usage_resource" json:"resource_type"`
	UsageDate       string    `gorm:"size:10;not null;index:idx_tenant_usage_date" json:"usage_date"` // YYYY-MM-DD
	UsageCount      int64     `gorm:"default:0" json:"usage_count"`
	UsageAmount     float64   `gorm:"default:0" json:"usage_amount"`
	Unit            string    `gorm:"size:20" json:"unit"`
	Tenant          Tenant    `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
}

func (TenantUsage) TableName() string {
	return "tenant_usages"
}

// TenantInvitation 租户邀请模型
type TenantInvitation struct {
	gorm.Model
	TenantID    uint       `gorm:"not null;index:idx_invitation_tenant" json:"tenant_id"`
	Email       string     `gorm:"size:255;not null;index:idx_invitation_email" json:"email"`
	Token       string     `gorm:"size:100;uniqueIndex:idx_invitation_token" json:"token"`
	Role        string     `gorm:"size:50;default:member" json:"role"`
	Status      string     `gorm:"size:20;default:pending;index:idx_invitation_status" json:"status"` // pending, accepted, expired, revoked
	ExpiresAt   time.Time  `json:"expires_at"`
	AcceptedAt  *time.Time `json:"accepted_at,omitempty"`
	InvitedBy   uint       `json:"invited_by"`
	Tenant      Tenant     `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
}

func (TenantInvitation) TableName() string {
	return "tenant_invitations"
}
