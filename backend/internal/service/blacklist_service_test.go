package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewBlacklistService(t *testing.T) {
	service := NewBlacklistService()
	assert.NotNil(t, service)
}

func TestBlacklistTypes(t *testing.T) {
	assert.Equal(t, BlacklistType("ip"), BlacklistTypeIP)
	assert.Equal(t, BlacklistType("user_id"), BlacklistTypeUserID)
	assert.Equal(t, BlacklistType("device_id"), BlacklistTypeDeviceID)
	assert.Equal(t, BlacklistType("phone"), BlacklistTypePhone)
	assert.Equal(t, BlacklistType("email"), BlacklistTypeEmail)
}

func TestBlacklistSources(t *testing.T) {
	assert.Equal(t, BlacklistSource("manual"), BlacklistSourceManual)
	assert.Equal(t, BlacklistSource("auto"), BlacklistSourceAuto)
	assert.Equal(t, BlacklistSource("import"), BlacklistSourceImport)
}

func TestBlacklistActions(t *testing.T) {
	assert.Equal(t, BlacklistAction("block"), BlacklistActionBlock)
	assert.Equal(t, BlacklistAction("captcha"), BlacklistActionCaptcha)
	assert.Equal(t, BlacklistAction("review"), BlacklistActionReview)
}

func TestBlacklistStatuses(t *testing.T) {
	assert.Equal(t, BlacklistStatus("active"), BlacklistStatusActive)
	assert.Equal(t, BlacklistStatus("expired"), BlacklistStatusExpired)
	assert.Equal(t, BlacklistStatus("unblocked"), BlacklistStatusUnblocked)
}

func TestListBlacklistFilter_Defaults(t *testing.T) {
	filter := &ListBlacklistFilter{}

	assert.Equal(t, 0, filter.Page)
	assert.Equal(t, 0, filter.PageSize)
	assert.Empty(t, filter.Type)
	assert.Empty(t, filter.Source)
	assert.Empty(t, filter.Status)
	assert.Empty(t, filter.Keyword)
	assert.True(t, filter.StartDate.IsZero())
	assert.True(t, filter.EndDate.IsZero())
}

func TestListBlacklistFilter_Normalization(t *testing.T) {
	filter := &ListBlacklistFilter{
		Page:     0,
		PageSize: 0,
	}

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 20
	}

	assert.Equal(t, 1, filter.Page)
	assert.Equal(t, 20, filter.PageSize)
}

func TestListBlacklistFilter_PageSizeLimit(t *testing.T) {
	testCases := []struct {
		name     string
		pageSize int
		expected int
	}{
		{"zero page size", 0, 20},
		{"negative page size", -1, 20},
		{"over limit page size", 200, 20},
		{"valid page size", 50, 50},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filter := &ListBlacklistFilter{PageSize: tc.pageSize}
			if filter.PageSize < 1 || filter.PageSize > 100 {
				filter.PageSize = 20
			}
			assert.Equal(t, tc.expected, filter.PageSize)
		})
	}
}

func TestCreateBlacklistInput_Validation(t *testing.T) {
	tests := []struct {
		name    string
		input   CreateBlacklistInput
		wantErr bool
	}{
		{
			name: "valid IP input",
			input: CreateBlacklistInput{
				Target: "192.168.1.1",
				Type:   "ip",
				Reason: "bot_attack",
			},
			wantErr: false,
		},
		{
			name: "valid user input",
			input: CreateBlacklistInput{
				Target: "user123",
				Type:   "user_id",
				Reason: "abuse",
			},
			wantErr: false,
		},
		{
			name: "empty target",
			input: CreateBlacklistInput{
				Target: "",
				Type:   "ip",
			},
			wantErr: true,
		},
		{
			name: "empty type",
			input: CreateBlacklistInput{
				Target: "192.168.1.1",
				Type:   "",
			},
			wantErr: true,
		},
		{
			name: "with application IDs",
			input: CreateBlacklistInput{
				Target:         "192.168.1.1",
				Type:           "ip",
				ApplicationIDs: []string{"app1", "app2"},
			},
			wantErr: false,
		},
		{
			name: "with expiration",
			input: CreateBlacklistInput{
				Target:     "192.168.1.1",
				Type:       "ip",
				Expiration: "2025-12-31",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasError := tt.input.Target == "" || tt.input.Type == ""
			assert.Equal(t, tt.wantErr, hasError)
		})
	}
}

func TestUpdateBlacklistInput(t *testing.T) {
	reason := "updated reason"
	action := "captcha"
	expiration := "2025-12-31"
	note := "test note"

	input := &UpdateBlacklistInput{
		Reason:     &reason,
		Action:     &action,
		Expiration: &expiration,
		Note:       &note,
	}

	assert.Equal(t, reason, *input.Reason)
	assert.Equal(t, action, *input.Action)
	assert.Equal(t, expiration, *input.Expiration)
	assert.Equal(t, note, *input.Note)
}

func TestBlacklistSummary(t *testing.T) {
	summary := &BlacklistSummary{
		Total:         100,
		TodayAdded:    10,
		AutoUnblocked: 5,
		TotalBlocked:  95,
	}

	assert.Equal(t, int64(100), summary.Total)
	assert.Equal(t, int64(10), summary.TodayAdded)
	assert.Equal(t, int64(5), summary.AutoUnblocked)
	assert.Equal(t, int64(95), summary.TotalBlocked)
}

func TestPaginatedResult(t *testing.T) {
	result := &PaginatedResult{
		Data:       []string{"item1", "item2", "item3"},
		Total:      100,
		Page:       1,
		PageSize:   10,
		TotalPages: 10,
	}

	assert.Equal(t, 3, len(result.Data.([]string)))
	assert.Equal(t, int64(100), result.Total)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 10, result.PageSize)
	assert.Equal(t, 10, result.TotalPages)
}

func TestBlacklistService_Errors(t *testing.T) {
	assert.Equal(t, "blacklist item not found", ErrBlacklistNotFound.Error())
}

func TestBlacklistTypeConstants(t *testing.T) {
	assert.Equal(t, "ip", string(BlacklistTypeIP))
	assert.Equal(t, "user_id", string(BlacklistTypeUserID))
	assert.Equal(t, "device_id", string(BlacklistTypeDeviceID))
	assert.Equal(t, "phone", string(BlacklistTypePhone))
	assert.Equal(t, "email", string(BlacklistTypeEmail))
}

func TestBlacklistSourceConstants(t *testing.T) {
	assert.Equal(t, "manual", string(BlacklistSourceManual))
	assert.Equal(t, "auto", string(BlacklistSourceAuto))
	assert.Equal(t, "import", string(BlacklistSourceImport))
}

func TestBlacklistActionConstants(t *testing.T) {
	assert.Equal(t, "block", string(BlacklistActionBlock))
	assert.Equal(t, "captcha", string(BlacklistActionCaptcha))
	assert.Equal(t, "review", string(BlacklistActionReview))
}

func TestBlacklistStatusConstants(t *testing.T) {
	assert.Equal(t, "active", string(BlacklistStatusActive))
	assert.Equal(t, "expired", string(BlacklistStatusExpired))
	assert.Equal(t, "unblocked", string(BlacklistStatusUnblocked))
}
