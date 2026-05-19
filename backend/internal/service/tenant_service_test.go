package service

import (
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&models.Tenant{},
		&models.TenantUser{},
		&models.TenantQuota{},
		&models.TenantBilling{},
		&models.TenantInvitation{},
		&models.TenantAuditLog{},
		&models.TenantUsageLog{},
		&models.Workflow{},
		&models.WorkflowExecution{},
		&models.SSOConfig{},
		&models.SCIMUser{},
		&models.SCIMGroup{},
		&models.APIAuditLog{},
		&models.ComplianceReport{},
	)
	require.NoError(t, err)

	return db
}

func TestTenantService_CreateTenant(t *testing.T) {
	db := setupTestDB(t)
	service := NewTenantService(db, nil)

	tenant := &models.Tenant{
		Name:         "Test Tenant",
		Code:         "test-tenant",
		Plan:         "free",
		ContactEmail: "test@example.com",
		Status:       "active",
	}

	result, err := service.CreateTenant(tenant, 1)
	require.NoError(t, err)
	assert.NotZero(t, result.ID)
	assert.Equal(t, "Test Tenant", result.Name)
	assert.Equal(t, "test-tenant", result.Code)

	var quota models.TenantQuota
	err = db.Where("tenant_id = ?", result.ID).First(&quota).Error
	require.NoError(t, err)
	assert.Equal(t, 10, quota.MaxUsers)

	var billing models.TenantBilling
	err = db.Where("tenant_id = ?", result.ID).First(&billing).Error
	require.NoError(t, err)
	assert.Equal(t, "free", billing.Plan)
}

func TestTenantService_GetTenant(t *testing.T) {
	db := setupTestDB(t)
	service := NewTenantService(db, nil)

	tenant := &models.Tenant{
		Name:   "Get Test",
		Code:   "get-test",
		Status: "active",
	}
	created, _ := service.CreateTenant(tenant, 1)

	retrieved, err := service.GetTenant(created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, retrieved.ID)
	assert.Equal(t, "Get Test", retrieved.Name)
}

func TestTenantService_GetTenantByCode(t *testing.T) {
	db := setupTestDB(t)
	service := NewTenantService(db, nil)

	tenant := &models.Tenant{
		Name:   "Code Test",
		Code:   "code-test",
		Status: "active",
	}
	_, _ = service.CreateTenant(tenant, 1)

	retrieved, err := service.GetTenantByCode("code-test")
	require.NoError(t, err)
	assert.Equal(t, "Code Test", retrieved.Name)
}

func TestTenantService_ListTenants(t *testing.T) {
	db := setupTestDB(t)
	service := NewTenantService(db, nil)

	for i := 1; i <= 5; i++ {
		tenant := &models.Tenant{
			Name:   "Tenant " + string(rune('0'+i)),
			Code:   "tenant-" + string(rune('0'+i)),
			Status: "active",
		}
		service.CreateTenant(tenant, 1)
	}

	tenants, total, err := service.ListTenants(1, 10, "", "", "")
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, tenants, 5)

	tenants, total, err = service.ListTenants(1, 2, "", "", "")
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, tenants, 2)
}

func TestTenantService_UpdateTenant(t *testing.T) {
	db := setupTestDB(t)
	service := NewTenantService(db, nil)

	tenant := &models.Tenant{
		Name:   "Original Name",
		Code:   "original-code",
		Status: "active",
	}
	created, _ := service.CreateTenant(tenant, 1)

	updates := map[string]interface{}{
		"name": "Updated Name",
	}
	updated, err := service.UpdateTenant(created.ID, updates, 1)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", updated.Name)

	var auditLog models.TenantAuditLog
	err = db.Where("tenant_id = ?", created.ID).First(&auditLog).Error
	require.NoError(t, err)
	assert.Equal(t, "update", auditLog.Action)
}

func TestTenantService_SuspendTenant(t *testing.T) {
	db := setupTestDB(t)
	service := NewTenantService(db, nil)

	tenant := &models.Tenant{
		Name:   "Suspend Test",
		Code:   "suspend-test",
		Status: "active",
	}
	created, _ := service.CreateTenant(tenant, 1)

	updates := map[string]interface{}{
		"status": "suspended",
	}
	updated, err := service.UpdateTenant(created.ID, updates, 1)
	require.NoError(t, err)
	assert.Equal(t, "suspended", updated.Status)
	assert.NotNil(t, updated.SuspendedAt)
}

func TestTenantService_AddTenantUser(t *testing.T) {
	db := setupTestDB(t)
	service := NewTenantService(db, nil)

	tenant := &models.Tenant{
		Name:   "User Test",
		Code:   "user-test",
		Status: "active",
	}
	created, _ := service.CreateTenant(tenant, 1)

	user, err := service.AddTenantUser(created.ID, 100, "admin", 1)
	require.NoError(t, err)
	assert.Equal(t, created.ID, user.TenantID)
	assert.Equal(t, uint(100), user.UserID)
	assert.Equal(t, "admin", user.Role)

	var quota models.TenantQuota
	db.Where("tenant_id = ?", created.ID).First(&quota)
	assert.Equal(t, 1, quota.CurrentUsers)
}

func TestTenantService_AddTenantUser_QuotaExceeded(t *testing.T) {
	db := setupTestDB(t)
	service := NewTenantService(db, nil)

	tenant := &models.Tenant{
		Name:   "Quota Test",
		Code:   "quota-test",
		Status: "active",
	}
	created, _ := service.CreateTenant(tenant, 1)

	db.Model(&models.TenantQuota{}).Where("tenant_id = ?", created.ID).Update("max_users", 1)

	_, err := service.AddTenantUser(created.ID, 100, "member", 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "quota exceeded")
}

func TestTenantService_RemoveTenantUser(t *testing.T) {
	db := setupTestDB(t)
	service := NewTenantService(db, nil)

	tenant := &models.Tenant{
		Name:   "Remove User Test",
		Code:   "remove-user-test",
		Status: "active",
	}
	created, _ := service.CreateTenant(tenant, 1)

	_, _ = service.AddTenantUser(created.ID, 100, "admin", 1)

	err := service.RemoveTenantUser(created.ID, 100)
	require.NoError(t, err)

	var user models.TenantUser
	err = db.Where("tenant_id = ? AND user_id = ?", created.ID, 100).First(&user).Error
	require.Error(t, err)
}

func TestTenantService_CheckQuota(t *testing.T) {
	db := setupTestDB(t)
	service := NewTenantService(db, nil)

	tenant := &models.Tenant{
		Name:   "Check Quota Test",
		Code:   "check-quota-test",
		Status: "active",
	}
	created, _ := service.CreateTenant(tenant, 1)

	allowed, msg := service.CheckQuota(created.ID, "user")
	assert.True(t, allowed)
	assert.Empty(t, msg)

	db.Model(&models.TenantQuota{}).Where("tenant_id = ?", created.ID).Update("current_users", 10)

	allowed, msg = service.CheckQuota(created.ID, "user")
	assert.False(t, allowed)
	assert.Contains(t, msg, "quota exceeded")
}

func TestTenantService_UpdateBillingPlan(t *testing.T) {
	db := setupTestDB(t)
	service := NewTenantService(db, nil)

	tenant := &models.Tenant{
		Name:   "Plan Test",
		Code:   "plan-test",
		Plan:   "free",
		Status: "active",
	}
	created, _ := service.CreateTenant(tenant, 1)

	err := service.UpdateBillingPlan(created.ID, "professional", 99.99, 1)
	require.NoError(t, err)

	var updated models.Tenant
	db.First(&updated, created.ID)
	assert.Equal(t, "professional", updated.Plan)

	var quota models.TenantQuota
	db.Where("tenant_id = ?", created.ID).First(&quota)
	assert.Equal(t, 200, quota.MaxUsers)
	assert.True(t, quota.CustomBranding)
	assert.True(t, quota.SSOEnabled)

	var billing models.TenantBilling
	db.Where("tenant_id = ?", created.ID).First(&billing)
	assert.Equal(t, 99.99, billing.Price)
}

func TestTenantService_GetTenantUsageStats(t *testing.T) {
	db := setupTestDB(t)
	service := NewTenantService(db, nil)

	tenant := &models.Tenant{
		Name:   "Usage Test",
		Code:   "usage-test",
		Status: "active",
	}
	created, _ := service.CreateTenant(tenant, 1)

	_, _ = service.AddTenantUser(created.ID, 100, "admin", 1)
	_, _ = service.AddTenantUser(created.ID, 101, "member", 1)

	stats, err := service.GetTenantUsageStats(created.ID)
	require.NoError(t, err)
	assert.NotNil(t, stats)

	usersStats := stats["users"].(map[string]interface{})
	assert.Equal(t, 2, usersStats["used"])
}

func TestTenantService_CreateInvitation(t *testing.T) {
	db := setupTestDB(t)
	service := NewTenantService(db, nil)

	tenant := &models.Tenant{
		Name:   "Invite Test",
		Code:   "invite-test",
		Status: "active",
	}
	created, _ := service.CreateTenant(tenant, 1)

	invitation, err := service.CreateInvitation(created.ID, "newuser@example.com", "member", 1)
	require.NoError(t, err)
	assert.NotEmpty(t, invitation.Token)
	assert.Equal(t, "pending", invitation.Status)
	assert.Equal(t, "newuser@example.com", invitation.Email)
}

func TestTenantService_AcceptInvitation(t *testing.T) {
	db := setupTestDB(t)
	service := NewTenantService(db, nil)

	tenant := &models.Tenant{
		Name:   "Accept Test",
		Code:   "accept-test",
		Status: "active",
	}
	created, _ := service.CreateTenant(tenant, 1)

	invitation, _ := service.CreateInvitation(created.ID, "accept@example.com", "member", 1)

	user, err := service.AcceptInvitation(invitation.Token, 200)
	require.NoError(t, err)
	assert.Equal(t, uint(200), user.UserID)
	assert.Equal(t, "member", user.Role)

	var updated models.TenantInvitation
	db.First(&updated, invitation.ID)
	assert.Equal(t, "accepted", updated.Status)
}

func TestTenantService_ApplyTenantScope(t *testing.T) {
	db := setupTestDB(t)
	service := NewTenantService(db, nil)

	tenant := &models.Tenant{
		Name:   "Scope Test",
		Code:   "scope-test",
		Status: "active",
	}
	created, _ := service.CreateTenant(tenant, 1)

	tenant2 := &models.Tenant{
		Name:   "Scope Test 2",
		Code:   "scope-test-2",
		Status: "active",
	}
	created2, _ := service.CreateTenant(tenant2, 1)

	query := db.Model(&models.Tenant{})
	scopedQuery := service.ApplyTenantScope(query, created.ID)

	var results []models.Tenant
	scopedQuery.Find(&results)

	assert.Len(t, results, 1)
	assert.Equal(t, created.ID, results[0].ID)
}

func TestRiskRuleEngineV2_CompileRule(t *testing.T) {
	db := setupTestDB(t)
	engine := NewRiskRuleEngineV2(db)

	rule := &models.RiskRule{
		Name:       "Test Rule",
		Condition:  "ip_frequency > {{threshold}}",
		Params:     `{"condition_type": "ip_frequency", "threshold": 100}`,
		IsEnabled:  true,
	}

	compiled, err := engine.CompileRule(rule)
	require.NoError(t, err)
	assert.NotNil(t, compiled)
	assert.Equal(t, "ip_frequency", compiled.Condition.Type)
	assert.Contains(t, compiled.Condition.Fields, "threshold")
}

func TestRiskRuleEngineV2_EvaluateRules(t *testing.T) {
	db := setupTestDB(t)
	engine := NewRiskRuleEngineV2(db)

	rule := &models.RiskRule{
		Name:       "IP Frequency Rule",
		Condition:  "ip_frequency > {{threshold}}",
		Params:     `{"condition_type": "ip_frequency", "threshold": 100}`,
		IsEnabled:  true,
	}
	engine.CompileRule(rule)

	ctx := &models.RiskContext{
		IPAddress: "192.168.1.1",
	}

	_, _, err := engine.EvaluateRules(ctx)
	require.NoError(t, err)
}

func TestWorkflowEngine_ExecuteWorkflow(t *testing.T) {
	db := setupTestDB(t)
	engine := NewWorkflowEngine(db)

	workflow := &Workflow{
		ID:   "test-workflow",
		Name: "Test Workflow",
		Steps: []WorkflowStep{
			{
				ID:   "step1",
				Name: "Test Step",
				Type: "action",
				Config: map[string]interface{}{
					"type":   "allow",
					"reason": "test",
				},
			},
		},
	}

	err := engine.CreateWorkflow(workflow)
	require.NoError(t, err)

	event := &WorkflowEvent{
		WorkflowID: "test-workflow",
		Type:      "test",
		Data:      map[string]interface{}{"test": "data"},
		Timestamp: time.Now(),
	}

	engine.eventQueue <- event

	time.Sleep(100 * time.Millisecond)

	var execution models.WorkflowExecution
	err = db.Where("workflow_id = ?", "test-workflow").First(&execution).Error
	require.NoError(t, err)
	assert.Contains(t, []string{"running", "completed", "failed"}, execution.Status)
}

func TestSSOUser(t *testing.T) {
	user := &SSOUser{
		ID:          "user123",
		Email:       "user@example.com",
		Username:    "testuser",
		FirstName:   "Test",
		LastName:    "User",
		DisplayName: "Test User",
		Groups:      []string{"admin", "users"},
	}

	assert.Equal(t, "user123", user.ID)
	assert.Equal(t, "user@example.com", user.Email)
	assert.Len(t, user.Groups, 2)
}

func TestOAuth2Provider_InitiateAuth(t *testing.T) {
	provider := NewOAuth2Provider(
		"client123",
		"secret",
		"https://auth.example.com/oauth/authorize",
		"https://auth.example.com/oauth/token",
		"https://app.example.com/callback",
		"openid,profile,email",
	)

	url, err := provider.InitiateAuth()
	require.NoError(t, err)
	assert.Contains(t, url, "client_id=client123")
	assert.Contains(t, url, "redirect_uri=https://app.example.com/callback")
}

func TestOIDCProvider(t *testing.T) {
	provider := &OIDCProvider{
		config: &models.SSOConfig{
			AuthorizationURL: "https://auth.example.com/authorize",
			ClientID:        "client123",
		},
	}

	url, err := provider.InitiateAuth()
	require.NoError(t, err)
	assert.Contains(t, url, "client_id=client123")
	assert.Contains(t, url, "response_type=code")
}

func TestSCIMService_CreateSCIMUser(t *testing.T) {
	db := setupTestDB(t)
	service := NewSCIMService(db)

	user := &SSOUser{
		ID:          "ext-user-123",
		Email:       "scim@example.com",
		Username:    "scimuser",
		FirstName:   "SCIM",
		LastName:    "User",
		DisplayName: "SCIM User",
		Groups:      []string{"engineering"},
	}

	scimUser, err := service.CreateSCIMUser(1, user)
	require.NoError(t, err)
	assert.Equal(t, "ext-user-123", scimUser.ExternalID)
	assert.Equal(t, "scim@example.com", scimUser.Email)
	assert.Equal(t, "synced", scimUser.SyncStatus)
}

func TestAPIAuditService_LogAPIRequest(t *testing.T) {
	db := setupTestDB(t)
	service := NewAPIAuditService(db)

	log := &models.APIAuditLog{
		TenantID:       1,
		ApplicationID:  1,
		UserID:         1,
		Method:         "POST",
		Endpoint:       "/api/v1/verify",
		ResponseStatus: 200,
		Latency:        50,
		IPAddress:      "192.168.1.1",
	}

	err := service.LogAPIRequest(log)
	require.NoError(t, err)
	assert.NotZero(t, log.ID)
}

func TestAPIAuditService_GetAuditLogs(t *testing.T) {
	db := setupTestDB(t)
	service := NewAPIAuditService(db)

	for i := 0; i < 5; i++ {
		log := &models.APIAuditLog{
			TenantID:       1,
			Method:         "GET",
			Endpoint:       "/api/v1/resource",
			ResponseStatus: 200,
			Latency:        int64(50 + i*10),
		}
		service.LogAPIRequest(log)
	}

	logs, total, err := service.GetAuditLogs(1, 1, 10, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, logs, 5)
}

func TestComplianceService_CreateReport(t *testing.T) {
	db := setupTestDB(t)
	service := NewComplianceService(db)

	startDate := time.Now().AddDate(0, -1, 0)
	endDate := time.Now()

	report, err := service.CreateReport(1, "gdpr", startDate, endDate)
	require.NoError(t, err)
	assert.Equal(t, "gdpr", report.ReportType)
	assert.Equal(t, "pending", report.Status)
}

func TestComplianceService_GenerateReport(t *testing.T) {
	db := setupTestDB(t)
	service := NewComplianceService(db)

	startDate := time.Now().AddDate(0, -1, 0)
	endDate := time.Now()

	report, _ := service.CreateReport(1, "gdpr", startDate, endDate)

	err := service.GenerateReport(report.ID)
	require.NoError(t, err)

	var updated models.ComplianceReport
	db.First(&updated, report.ID)
	assert.Equal(t, "completed", updated.Status)
	assert.NotEmpty(t, updated.FilePath)
}

func TestConditionEvaluator_Threshold(t *testing.T) {
	db := setupTestDB(t)
	engine := NewRiskRuleEngineV2(db)

	ctx := &models.RiskContext{
		FailureCount: 5,
	}

	params := map[string]interface{}{
		"metric":     "failure_count",
		"threshold":  3.0,
		"operator":   "gt",
	}

	result := engine.evalThreshold(ctx, params)
	assert.True(t, result)

	params["threshold"] = 10.0
	result = engine.evalThreshold(ctx, params)
	assert.False(t, result)
}

func TestConditionEvaluator_TimeWindow(t *testing.T) {
	db := setupTestDB(t)
	engine := NewRiskRuleEngineV2(db)

	ctx := &models.RiskContext{}

	params := map[string]interface{}{
		"start_hour": 0.0,
		"end_hour":   24.0,
	}

	result := engine.evalTimeWindow(ctx, params)
	assert.False(t, result)

	params["start_hour"] = 10.0
	params["end_hour"] = 12.0

	currentHour := time.Now().Hour()
	if currentHour >= 10 && currentHour <= 12 {
		result = engine.evalTimeWindow(ctx, params)
		assert.False(t, result)
	} else {
		result = engine.evalTimeWindow(ctx, params)
		assert.True(t, result)
	}
}

func TestCalculatePathEfficiency(t *testing.T) {
	db := setupTestDB(t)
	engine := NewRiskRuleEngineV2(db)

	traceData := []models.TracePoint{
		{X: 0, Y: 0, T: 0},
		{X: 100, Y: 0, T: 1},
		{X: 200, Y: 0, T: 2},
	}

	efficiency := engine.calculatePathEfficiency(traceData)
	assert.Greater(t, efficiency, 0.5)

	traceData = []models.TracePoint{
		{X: 0, Y: 0, T: 0},
		{X: 50, Y: 100, T: 1},
		{X: 0, Y: 200, T: 2},
		{X: 50, Y: 300, T: 3},
		{X: 100, Y: 0, T: 4},
	}

	efficiency = engine.calculatePathEfficiency(traceData)
	assert.Less(t, efficiency, 1.0)
}

func TestGetQuotaForPlan(t *testing.T) {
	plans := []string{"free", "starter", "professional", "enterprise"}

	for _, plan := range plans {
		quota := getQuotaForPlan(plan)
		assert.NotNil(t, quota)
		assert.Contains(t, quota, "max_users")
		assert.Contains(t, quota, "max_applications")
	}

	quota := getQuotaForPlan("unknown")
	assert.NotNil(t, quota)
	assert.Equal(t, 10, quota["max_users"])
}

func TestGenerateSecureToken(t *testing.T) {
	token1 := generateSecureToken(32)
	token2 := generateSecureToken(32)

	assert.Len(t, token1, 32)
	assert.Len(t, token2, 32)
	assert.NotEqual(t, token1, token2)

	for _, c := range token1 {
		assert.True(t, c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9')
	}
}

func TestCalculatePercentage(t *testing.T) {
	result := calculatePercentage(50, 100)
	assert.Equal(t, 50.0, result)

	result = calculatePercentage(25, 100)
	assert.Equal(t, 25.0, result)

	result = calculatePercentage(100, 0)
	assert.Equal(t, 0.0, result)
}

func TestToFloat64(t *testing.T) {
	assert.Equal(t, 123.0, toFloat64(123))
	assert.Equal(t, 123.0, toFloat64(int64(123)))
	assert.Equal(t, 123.5, toFloat64(123.5))
	assert.Equal(t, 123.0, toFloat64("123"))
	assert.Equal(t, 0.0, toFloat64(nil))
}
