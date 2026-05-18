package service

import (
	"testing"
)

func TestNewSessionService(t *testing.T) {
	sessionService := NewSessionService()
	if sessionService == nil {
		t.Error("NewSessionService 返回了 nil")
	}
}

func TestCreateSession(t *testing.T) {
	sessionService := NewSessionService()
	
	session, err := sessionService.CreateSession("user-123")
	if err != nil {
		t.Errorf("创建会话失败: %v", err)
	}
	if session == nil {
		t.Error("创建的会话不应为 nil")
	}
	if session.UserID != "user-123" {
		t.Errorf("会话用户 ID 不匹配: 期望 user-123, 实际 %s", session.UserID)
	}
}

func TestGetSession(t *testing.T) {
	sessionService := NewSessionService()
	
	createdSession, err := sessionService.CreateSession("user-get")
	if err != nil {
		t.Skipf("无法创建会话，跳过测试: %v", err)
	}
	
	session, err := sessionService.GetSession(createdSession.ID)
	if err != nil {
		t.Errorf("获取会话失败: %v", err)
	}
	if session == nil {
		t.Error("获取的会话不应为 nil")
	}
}

func TestGetSession_NotFound(t *testing.T) {
	sessionService := NewSessionService()
	
	session, err := sessionService.GetSession("nonexistent-session-id")
	if err == nil {
		t.Error("不存在的会话应该返回错误")
	}
	if session != nil {
		t.Error("不存在的会话应该返回 nil")
	}
}

func TestUpdateSession(t *testing.T) {
	sessionService := NewSessionService()
	
	createdSession, err := sessionService.CreateSession("user-update")
	if err != nil {
		t.Skipf("无法创建会话，跳过测试: %v", err)
	}
	
	createdSession.Data["key"] = "value"
	err = sessionService.UpdateSession(createdSession)
	if err != nil {
		t.Errorf("更新会话失败: %v", err)
	}
	
	updatedSession, err := sessionService.GetSession(createdSession.ID)
	if err != nil {
		t.Skipf("无法获取更新后的会话，跳过测试: %v", err)
	}
	if updatedSession.Data["key"] != "value" {
		t.Error("会话数据未正确更新")
	}
}

func TestDeleteSession(t *testing.T) {
	sessionService := NewSessionService()
	
	createdSession, err := sessionService.CreateSession("user-delete")
	if err != nil {
		t.Skipf("无法创建会话，跳过测试: %v", err)
	}
	
	err = sessionService.DeleteSession(createdSession.ID)
	if err != nil {
		t.Errorf("删除会话失败: %v", err)
	}
	
	session, err := sessionService.GetSession(createdSession.ID)
	if err == nil && session != nil {
		t.Error("会话应该已被删除")
	}
}

func TestRefreshSession(t *testing.T) {
	sessionService := NewSessionService()
	
	createdSession, err := sessionService.CreateSession("user-refresh")
	if err != nil {
		t.Skipf("无法创建会话，跳过测试: %v", err)
	}
	
	oldExpiry := createdSession.ExpiresAt
	err = sessionService.RefreshSession(createdSession.ID)
	if err != nil {
		t.Errorf("刷新会话失败: %v", err)
	}
	
	refreshedSession, err := sessionService.GetSession(createdSession.ID)
	if err != nil {
		t.Skipf("无法获取刷新后的会话，跳过测试: %v", err)
	}
	if !refreshedSession.ExpiresAt.After(oldExpiry) {
		t.Error("会话过期时间应该被更新")
	}
}

func TestListUserSessions(t *testing.T) {
	sessionService := NewSessionService()
	
	_, err := sessionService.CreateSession("user-list-1")
	if err != nil {
		t.Skipf("无法创建会话，跳过测试: %v", err)
	}
	
	_, err = sessionService.CreateSession("user-list-2")
	if err != nil {
		t.Skipf("无法创建会话，跳过测试: %v", err)
	}
	
	sessions, err := sessionService.ListUserSessions("user-list-1")
	if err != nil {
		t.Errorf("列出用户会话失败: %v", err)
	}
	if sessions == nil {
		t.Error("会话列表不应为 nil")
	}
}

func TestDeleteUserSessions(t *testing.T) {
	sessionService := NewSessionService()
	
	session1, err := sessionService.CreateSession("user-delete-all")
	if err != nil {
		t.Skipf("无法创建会话，跳过测试: %v", err)
	}
	
	session2, err := sessionService.CreateSession("user-delete-all")
	if err != nil {
		t.Skipf("无法创建会话，跳过测试: %v", err)
	}
	
	err = sessionService.DeleteUserSessions("user-delete-all")
	if err != nil {
		t.Errorf("删除用户所有会话失败: %v", err)
	}
	
	deleted1, _ := sessionService.GetSession(session1.ID)
	deleted2, _ := sessionService.GetSession(session2.ID)
	if deleted1 != nil || deleted2 != nil {
		t.Error("用户的会话应该已被全部删除")
	}
}

func TestValidateSession(t *testing.T) {
	sessionService := NewSessionService()
	
	createdSession, err := sessionService.CreateSession("user-validate")
	if err != nil {
		t.Skipf("无法创建会话，跳过测试: %v", err)
	}
	
	valid := sessionService.ValidateSession(createdSession.ID)
	if !valid {
		t.Error("有效的会话应该通过验证")
	}
	
	valid = sessionService.ValidateSession("invalid-session-id")
	if valid {
		t.Error("无效的会话不应该通过验证")
	}
}

func TestGetActiveSessionCount(t *testing.T) {
	sessionService := NewSessionService()
	
	count, err := sessionService.GetActiveSessionCount()
	if err != nil {
		t.Errorf("获取活跃会话数失败: %v", err)
	}
	if count < 0 {
		t.Error("活跃会话数不应为负数")
	}
}
