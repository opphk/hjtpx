package service

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
	"github.com/hjtpx/hjtpx/pkg/database"
	"gorm.io/gorm"
)

// TenantBillingService 租户计费服务接口
type TenantBillingService interface {
	GenerateMonthlyBill(ctx context.Context, tenantID uint, year, month int) (*model.TenantBilling, error)
	GetBill(ctx context.Context, billID uint) (*model.TenantBilling, error)
	GetBillByNumber(ctx context.Context, billNumber string) (*model.TenantBilling, error)
	ListTenantBills(ctx context.Context, tenantID uint, status string, page, pageSize int) ([]*model.TenantBilling, int64, error)
	ListAllBills(ctx context.Context, status string, page, pageSize int) ([]*model.TenantBilling, int64, error)
	UpdateBillStatus(ctx context.Context, billID uint, status string) error
	CreatePaymentRecord(ctx context.Context, billID uint, amount float64, paymentMethod, transactionID string) (*model.TenantPaymentRecord, error)
	UpdatePaymentStatus(ctx context.Context, paymentID uint, status string, failedReason string) error
	GetPaymentRecord(ctx context.Context, paymentID uint) (*model.TenantPaymentRecord, error)
	ListTenantPayments(ctx context.Context, tenantID uint, page, pageSize int) ([]*model.TenantPaymentRecord, int64, error)
	CalculateUsageCost(ctx context.Context, tenantID uint, year, month int) (float64, error)
}

// tenantBillingService 租户计费服务实现
type tenantBillingService struct {
	db *gorm.DB
}

// NewTenantBillingService 创建租户计费服务实例
func NewTenantBillingService() TenantBillingService {
	return &tenantBillingService{
		db: database.DB,
	}
}

// GenerateMonthlyBill 生成月度账单
func (s *tenantBillingService) GenerateMonthlyBill(ctx context.Context, tenantID uint, year, month int) (*model.TenantBilling, error) {
	var tenant model.Tenant
	if err := s.db.First(&tenant, tenantID).Error; err != nil {
		return nil, err
	}

	periodStart := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 1, 0).Add(-time.Second)

	var existingBill model.TenantBilling
	if err := s.db.Where("tenant_id = ? AND period_start = ?", tenantID, periodStart).First(&existingBill).Error; err == nil {
		return &existingBill, nil
	} else if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	usageCost, err := s.CalculateUsageCost(ctx, tenantID, year, month)
	if err != nil {
		return nil, err
	}

	taxRate := 0.06
	taxAmount := usageCost * taxRate
	discountAmount := 0.0

	if tenant.Plan == "enterprise" {
		discountAmount = usageCost * 0.2
	}

	payableAmount := usageCost + taxAmount - discountAmount

	bill := model.TenantBilling{
		TenantID:      tenantID,
		BillNumber:    generateBillNumber(tenantID, year, month),
		PeriodStart:   periodStart,
		PeriodEnd:     periodEnd,
		TotalAmount:   usageCost,
		TaxAmount:     taxAmount,
		DiscountAmount: discountAmount,
		PayableAmount: payableAmount,
		Status:        "pending",
		Description:   fmt.Sprintf("%d年%d月服务费", year, month),
	}

	if err := s.db.Create(&bill).Error; err != nil {
		return nil, err
	}

	return &bill, nil
}

// GetBill 根据ID获取账单
func (s *tenantBillingService) GetBill(ctx context.Context, billID uint) (*model.TenantBilling, error) {
	var bill model.TenantBilling
	if err := s.db.First(&bill, billID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &bill, nil
}

// GetBillByNumber 根据账单号获取账单
func (s *tenantBillingService) GetBillByNumber(ctx context.Context, billNumber string) (*model.TenantBilling, error) {
	var bill model.TenantBilling
	if err := s.db.Where("bill_number = ?", billNumber).First(&bill).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &bill, nil
}

// ListTenantBills 列出租户账单
func (s *tenantBillingService) ListTenantBills(ctx context.Context, tenantID uint, status string, page, pageSize int) ([]*model.TenantBilling, int64, error) {
	var bills []*model.TenantBilling
	var total int64

	query := s.db.Model(&model.TenantBilling{}).Where("tenant_id = ?", tenantID)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("period_start DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&bills).Error; err != nil {
		return nil, 0, err
	}

	return bills, total, nil
}

// ListAllBills 列出所有账单（管理员）
func (s *tenantBillingService) ListAllBills(ctx context.Context, status string, page, pageSize int) ([]*model.TenantBilling, int64, error) {
	var bills []*model.TenantBilling
	var total int64

	query := s.db.Model(&model.TenantBilling{})
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&bills).Error; err != nil {
		return nil, 0, err
	}

	return bills, total, nil
}

// UpdateBillStatus 更新账单状态
func (s *tenantBillingService) UpdateBillStatus(ctx context.Context, billID uint, status string) error {
	return s.db.Model(&model.TenantBilling{}).Where("id = ?", billID).Update("status", status).Error
}

// CreatePaymentRecord 创建支付记录
func (s *tenantBillingService) CreatePaymentRecord(ctx context.Context, billID uint, amount float64, paymentMethod, transactionID string) (*model.TenantPaymentRecord, error) {
	var bill model.TenantBilling
	if err := s.db.First(&bill, billID).Error; err != nil {
		return nil, err
	}

	payment := model.TenantPaymentRecord{
		TenantID:      bill.TenantID,
		BillID:        billID,
		Amount:        amount,
		PaymentMethod: paymentMethod,
		TransactionID: transactionID,
		Status:        "pending",
	}

	if err := s.db.Create(&payment).Error; err != nil {
		return nil, err
	}

	return &payment, nil
}

// UpdatePaymentStatus 更新支付状态
func (s *tenantBillingService) UpdatePaymentStatus(ctx context.Context, paymentID uint, status string, failedReason string) error {
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var payment model.TenantPaymentRecord
	if err := tx.First(&payment, paymentID).Error; err != nil {
		tx.Rollback()
		return err
	}

	payment.Status = status
	payment.FailedReason = failedReason

	if status == "success" {
		payment.PaymentAt = func() *time.Time { t := time.Now(); return &t }()

		if err := tx.Model(&model.TenantBilling{}).Where("id = ?", payment.BillID).Updates(map[string]interface{}{
			"status":              "paid",
			"payment_method":      payment.PaymentMethod,
			"payment_at":          payment.PaymentAt,
			"payment_transaction_id": payment.TransactionID,
		}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Save(&payment).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// GetPaymentRecord 获取支付记录
func (s *tenantBillingService) GetPaymentRecord(ctx context.Context, paymentID uint) (*model.TenantPaymentRecord, error) {
	var payment model.TenantPaymentRecord
	if err := s.db.First(&payment, paymentID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &payment, nil
}

// ListTenantPayments 列出租户支付记录
func (s *tenantBillingService) ListTenantPayments(ctx context.Context, tenantID uint, page, pageSize int) ([]*model.TenantPaymentRecord, int64, error) {
	var payments []*model.TenantPaymentRecord
	var total int64

	query := s.db.Model(&model.TenantPaymentRecord{}).Where("tenant_id = ?", tenantID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&payments).Error; err != nil {
		return nil, 0, err
	}

	return payments, total, nil
}

// CalculateUsageCost 计算使用费用
func (s *tenantBillingService) CalculateUsageCost(ctx context.Context, tenantID uint, year, month int) (float64, error) {
	startDate := fmt.Sprintf("%d-%02d-01", year, month)
	endDate := fmt.Sprintf("%d-%02d-31", year, month)

	var totalUsage int64
	if err := s.db.Model(&model.TenantUsage{}).
		Where("tenant_id = ? AND usage_date >= ? AND usage_date <= ? AND resource_type = ?", tenantID, startDate, endDate, "verification").
		Select("SUM(usage_count)").
		Scan(&totalUsage).Error; err != nil {
		return 0, err
	}

	unitPrice := 0.01
	return float64(totalUsage) * unitPrice, nil
}

// generateBillNumber 生成账单号
func generateBillNumber(tenantID uint, year, month int) string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("BILL%d%04d%02d%06d", tenantID, year, month, rand.Intn(1000000))
}
