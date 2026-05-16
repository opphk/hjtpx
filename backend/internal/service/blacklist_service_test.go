package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBlacklistService_NewBlacklistService(t *testing.T) {
	service := NewBlacklistService()
	assert.NotNil(t, service)
}

func TestBlacklistConstants(t *testing.T) {
	assert.Equal(t, BlacklistType("ip"), BlacklistTypeIP)
	assert.Equal(t, BlacklistType("user_id"), BlacklistTypeUserID)
	assert.Equal(t, BlacklistType("device_id"), BlacklistTypeDeviceID)
	assert.Equal(t, BlacklistType("phone"), BlacklistTypePhone)
	assert.Equal(t, BlacklistType("email"), BlacklistTypeEmail)

	assert.Equal(t, BlacklistSource("manual"), BlacklistSourceManual)
	assert.Equal(t, BlacklistSource("auto"), BlacklistSourceAuto)
	assert.Equal(t, BlacklistSource("import"), BlacklistSourceImport)

	assert.Equal(t, BlacklistAction("block"), BlacklistActionBlock)
	assert.Equal(t, BlacklistAction("captcha"), BlacklistActionCaptcha)
	assert.Equal(t, BlacklistAction("review"), BlacklistActionReview)

	assert.Equal(t, BlacklistStatus("active"), BlacklistStatusActive)
	assert.Equal(t, BlacklistStatus("expired"), BlacklistStatusExpired)
	assert.Equal(t, BlacklistStatus("unblocked"), BlacklistStatusUnblocked)
}

func TestCreateBlacklistInput(t *testing.T) {
	input := &CreateBlacklistInput{
		Target:         "192.168.1.100",
		Type:           "ip",
		Source:         "manual",
		Reason:         "malicious activity",
		Action:         "block",
		ApplicationIDs: []string{"1", "2"},
		Expiration:     "2025-12-31",
		Note:           "test note",
		CreatedBy:      1,
	}

	assert.Equal(t, "192.168.1.100", input.Target)
	assert.Equal(t, "ip", input.Type)
	assert.Equal(t, "manual", input.Source)
	assert.Equal(t, "malicious activity", input.Reason)
	assert.Equal(t, "block", input.Action)
	assert.Len(t, input.ApplicationIDs, 2)
	assert.Equal(t, "2025-12-31", input.Expiration)
	assert.Equal(t, "test note", input.Note)
	assert.Equal(t, uint(1), input.CreatedBy)
}

func TestUpdateBlacklistInput(t *testing.T) {
	reason := "updated reason"
	action := "captcha"
	expiration := "2025-06-30"

	input := &UpdateBlacklistInput{
		Type:       &reason,
		Reason:     &reason,
		Action:     &action,
		Expiration: &expiration,
	}

	assert.NotNil(t, input.Type)
	assert.NotNil(t, input.Reason)
	assert.NotNil(t, input.Action)
	assert.NotNil(t, input.Expiration)
	assert.Equal(t, "updated reason", *input.Reason)
	assert.Equal(t, "captcha", *input.Action)
}

func TestListBlacklistFilter(t *testing.T) {
	filter := &ListBlacklistFilter{
		Page:          1,
		PageSize:      20,
		Type:          "ip",
		Source:        "manual",
		Status:        "active",
		Keyword:       "test",
		ApplicationID: 1,
	}

	assert.Equal(t, 1, filter.Page)
	assert.Equal(t, 20, filter.PageSize)
	assert.Equal(t, "ip", filter.Type)
	assert.Equal(t, "manual", filter.Source)
	assert.Equal(t, "active", filter.Status)
	assert.Equal(t, "test", filter.Keyword)
	assert.Equal(t, uint(1), filter.ApplicationID)
}

func TestBlacklistSummary(t *testing.T) {
	summary := &BlacklistSummary{
		Total:         100,
		TodayAdded:    10,
		AutoUnblocked: 5,
		TotalBlocked:  100,
	}

	assert.Equal(t, int64(100), summary.Total)
	assert.Equal(t, int64(10), summary.TodayAdded)
	assert.Equal(t, int64(5), summary.AutoUnblocked)
	assert.Equal(t, int64(100), summary.TotalBlocked)
}
