package service

import (
	"testing"
)

func TestNewUserService(t *testing.T) {
	userService := NewUserService()
	if userService == nil {
		t.Error("NewUserService 返回了 nil")
	}
}

func TestCreateUser(t *testing.T) {
	userService := NewUserService()
	
	user := &User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}
	
	createdUser, err := userService.CreateUser(user)
	if err != nil {
		t.Errorf("创建用户失败: %v", err)
	}
	if createdUser == nil {
		t.Error("创建的用户不应为 nil")
	}
	if createdUser.Username != user.Username {
		t.Errorf("用户名不匹配: 期望 %s, 实际 %s", user.Username, createdUser.Username)
	}
}

func TestGetUserByID(t *testing.T) {
	userService := NewUserService()
	
	user := &User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}
	
	createdUser, err := userService.CreateUser(user)
	if err != nil {
		t.Skipf("无法创建用户，跳过测试: %v", err)
	}
	
	fetchedUser, err := userService.GetUserByID(createdUser.ID)
	if err != nil {
		t.Errorf("获取用户失败: %v", err)
	}
	if fetchedUser == nil {
		t.Error("获取的用户不应为 nil")
	}
	if fetchedUser.ID != createdUser.ID {
		t.Errorf("用户 ID 不匹配: 期望 %s, 实际 %s", createdUser.ID, fetchedUser.ID)
	}
}

func TestGetUserByUsername(t *testing.T) {
	userService := NewUserService()
	
	user := &User{
		Username: "testuser_get",
		Email:    "test_get@example.com",
		Password: "password123",
	}
	
	createdUser, err := userService.CreateUser(user)
	if err != nil {
		t.Skipf("无法创建用户，跳过测试: %v", err)
	}
	
	fetchedUser, err := userService.GetUserByUsername(createdUser.Username)
	if err != nil {
		t.Errorf("获取用户失败: %v", err)
	}
	if fetchedUser == nil {
		t.Error("获取的用户不应为 nil")
	}
	if fetchedUser.Username != createdUser.Username {
		t.Errorf("用户名不匹配: 期望 %s, 实际 %s", createdUser.Username, fetchedUser.Username)
	}
}

func TestUpdateUser(t *testing.T) {
	userService := NewUserService()
	
	user := &User{
		Username: "testuser_update",
		Email:    "test_update@example.com",
		Password: "password123",
	}
	
	createdUser, err := userService.CreateUser(user)
	if err != nil {
		t.Skipf("无法创建用户，跳过测试: %v", err)
	}
	
	createdUser.Email = "newemail@example.com"
	updatedUser, err := userService.UpdateUser(createdUser)
	if err != nil {
		t.Errorf("更新用户失败: %v", err)
	}
	if updatedUser.Email != createdUser.Email {
		t.Errorf("邮箱未更新: 期望 %s, 实际 %s", createdUser.Email, updatedUser.Email)
	}
}

func TestDeleteUser(t *testing.T) {
	userService := NewUserService()
	
	user := &User{
		Username: "testuser_delete",
		Email:    "test_delete@example.com",
		Password: "password123",
	}
	
	createdUser, err := userService.CreateUser(user)
	if err != nil {
		t.Skipf("无法创建用户，跳过测试: %v", err)
	}
	
	err = userService.DeleteUser(createdUser.ID)
	if err != nil {
		t.Errorf("删除用户失败: %v", err)
	}
	
	fetchedUser, err := userService.GetUserByID(createdUser.ID)
	if err == nil && fetchedUser != nil {
		t.Error("用户应该已被删除")
	}
}

func TestListUsers(t *testing.T) {
	userService := NewUserService()
	
	users, err := userService.ListUsers(0, 10)
	if err != nil {
		t.Errorf("列出用户失败: %v", err)
	}
	if users == nil {
		t.Error("用户列表不应为 nil")
	}
}

func TestValidateUserData(t *testing.T) {
	userService := NewUserService()
	
	validUser := &User{
		Username: "validuser",
		Email:    "valid@example.com",
		Password: "password123",
	}
	
	err := userService.ValidateUserData(validUser)
	if err != nil {
		t.Errorf("有效用户数据验证失败: %v", err)
	}
	
	invalidUser := &User{
		Username: "",
		Email:    "invalid-email",
		Password: "123",
	}
	
	err = userService.ValidateUserData(invalidUser)
	if err == nil {
		t.Error("无效用户数据应该验证失败")
	}
}
