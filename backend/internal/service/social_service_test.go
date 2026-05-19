package service

import (
	"testing"
)

func TestSocialServiceGetConfig(t *testing.T) {
	service := NewSocialService()

	config := service.GetConfig()
	if config == nil {
		t.Error("GetConfig() returned nil")
	}

	if config.AvatarWeight != 0.25 {
		t.Errorf("GetConfig() AvatarWeight = %f, want 0.25", config.AvatarWeight)
	}
}

func TestSocialServiceCreateAvatar(t *testing.T) {
	service := NewSocialService()

	tests := []struct {
		name  string
		style AvatarStyle
	}{
		{"Realistic avatar", AvatarStyleRealistic},
		{"Cartoon avatar", AvatarStyleCartoon},
		{"Abstract avatar", AvatarStyleAbstract},
		{"Anime avatar", AvatarStyleAnime},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			avatar, err := service.CreateAvatar("user123", tt.style, "Test User")
			if err != nil {
				t.Errorf("CreateAvatar() error = %v", err)
				return
			}

			if avatar == nil {
				t.Error("CreateAvatar() returned nil")
				return
			}

			if avatar.UserID != "user123" {
				t.Errorf("CreateAvatar() UserID = %s, want user123", avatar.UserID)
			}

			if avatar.Style != tt.style {
				t.Errorf("CreateAvatar() Style = %s, want %s", avatar.Style, tt.style)
			}

			if avatar.Name != "Test User" {
				t.Errorf("CreateAvatar() Name = %s, want Test User", avatar.Name)
			}
		})
	}
}

func TestSocialServiceUpdateAvatarExpression(t *testing.T) {
	service := NewSocialService()

	_, _ = service.CreateAvatar("user123", AvatarStyleRealistic, "Test User")

	err := service.UpdateAvatarExpression("user123", "happy", 0.8)
	if err != nil {
		t.Errorf("UpdateAvatarExpression() error = %v", err)
		return
	}

	avatar, _ := service.GetAvatar("user123")
	if avatar.Emotion != "happy" {
		t.Errorf("UpdateAvatarExpression() Emotion = %s, want happy", avatar.Emotion)
	}
}

func TestSocialServiceUpdateAvatarPose(t *testing.T) {
	service := NewSocialService()

	_, _ = service.CreateAvatar("user123", AvatarStyleRealistic, "Test User")

	position := []float64{1.0, 2.0, 3.0}
	rotation := []float64{0.0, 90.0, 0.0}
	scale := 1.5

	err := service.UpdateAvatarPose("user123", position, rotation, scale)
	if err != nil {
		t.Errorf("UpdateAvatarPose() error = %v", err)
		return
	}

	avatar, _ := service.GetAvatar("user123")
	if avatar.Scale != 1.5 {
		t.Errorf("UpdateAvatarPose() Scale = %f, want 1.5", avatar.Scale)
	}
}

func TestSocialServiceBuildSocialGraph(t *testing.T) {
	service := NewSocialService()

	friendIDs := []string{"friend1", "friend2", "friend3"}

	graph, err := service.BuildSocialGraph("user123", friendIDs)
	if err != nil {
		t.Errorf("BuildSocialGraph() error = %v", err)
		return
	}

	if graph == nil {
		t.Error("BuildSocialGraph() returned nil")
		return
	}

	if graph.UserID != "user123" {
		t.Errorf("BuildSocialGraph() UserID = %s, want user123", graph.UserID)
	}

	if graph.FriendsCount != len(friendIDs) {
		t.Errorf("BuildSocialGraph() FriendsCount = %d, want %d", graph.FriendsCount, len(friendIDs))
	}
}

func TestSocialServiceCreateCommunity(t *testing.T) {
	service := NewSocialService()

	community, err := service.CreateCommunity("Test Community")
	if err != nil {
		t.Errorf("CreateCommunity() error = %v", err)
		return
	}

	if community == nil {
		t.Error("CreateCommunity() returned nil")
		return
	}

	if community.CommunityName != "Test Community" {
		t.Errorf("CreateCommunity() CommunityName = %s, want Test Community", community.CommunityName)
	}

	if community.MemberCount != 1 {
		t.Errorf("CreateCommunity() MemberCount = %d, want 1", community.MemberCount)
	}
}

func TestSocialServiceAddUserToCommunity(t *testing.T) {
	service := NewSocialService()

	community, _ := service.CreateCommunity("Test Community")

	err := service.AddUserToCommunity(community.CommunityID, 0.85)
	if err != nil {
		t.Errorf("AddUserToCommunity() error = %v", err)
		return
	}

	updated, _ := service.GetCommunity(community.CommunityID)
	if updated.MemberCount != 2 {
		t.Errorf("AddUserToCommunity() MemberCount = %d, want 2", updated.MemberCount)
	}
}

func TestSocialServiceCreateFriendVerificationRequest(t *testing.T) {
	service := NewSocialService()

	friendIDs := []string{"friend1", "friend2"}

	request, err := service.CreateFriendVerificationRequest("user123", friendIDs, "support", "Help me verify!")
	if err != nil {
		t.Errorf("CreateFriendVerificationRequest() error = %v", err)
		return
	}

	if request == nil {
		t.Error("CreateFriendVerificationRequest() returned nil")
		return
	}

	if request.UserID != "user123" {
		t.Errorf("CreateFriendVerificationRequest() UserID = %s, want user123", request.UserID)
	}

	if len(request.FriendIDs) != len(friendIDs) {
		t.Errorf("CreateFriendVerificationRequest() FriendIDs length = %d, want %d", len(request.FriendIDs), len(friendIDs))
	}
}

func TestSocialServiceRespondToFriendVerification(t *testing.T) {
	service := NewSocialService()

	request, _ := service.CreateFriendVerificationRequest("user123", []string{"friend1", "friend2"}, "support", "")

	err := service.RespondToFriendVerification(request.ID, "friend1", true, "Accepted")
	if err != nil {
		t.Errorf("RespondToFriendVerification() error = %v", err)
		return
	}

	updated, _ := service.GetFriendVerificationRequest(request.ID)
	if updated.Status != "accepted" {
		t.Errorf("RespondToFriendVerification() Status = %s, want accepted", updated.Status)
	}
}

func TestSocialServiceVerify(t *testing.T) {
	service := NewSocialService()

	avatar, _ := service.CreateAvatar("user123", AvatarStyleRealistic, "Test User")
	graph, _ := service.BuildSocialGraph("user123", []string{"friend1", "friend2", "friend3"})
	community, _ := service.CreateCommunity("Test Community")
	friendReq, _ := service.CreateFriendVerificationRequest("user123", []string{"friend1"}, "support", "")

	request := &SocialVerificationRequest{
		SessionID:   "session123",
		UserID:      "user123",
		AvatarData:  avatar,
		SocialGraph: graph,
		Communities: []CommunityTrust{*community},
		FriendReq:   friendReq,
	}

	result, err := service.Verify(request)
	if err != nil {
		t.Errorf("Verify() error = %v", err)
		return
	}

	if result == nil {
		t.Error("Verify() returned nil")
		return
	}

	if result.ProcessingTime < 0 {
		t.Error("Verify() ProcessingTime should not be negative")
	}
}

func TestSocialServiceGenerateAvatarScene(t *testing.T) {
	service := NewSocialService()

	avatar, _ := service.CreateAvatar("user123", AvatarStyleRealistic, "Test User")

	tests := []struct {
		name      string
		sceneType string
	}{
		{"Social scene", "social"},
		{"Gaming scene", "gaming"},
		{"Business scene", "business"},
		{"Casual scene", "casual"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scene, err := service.GenerateAvatarScene(avatar.UserID, tt.sceneType)
			if err != nil {
				t.Errorf("GenerateAvatarScene() error = %v", err)
				return
			}

			if scene == "" {
				t.Error("GenerateAvatarScene() returned empty scene")
			}
		})
	}
}

func TestSocialServiceExportSocialData(t *testing.T) {
	service := NewSocialService()

	_, _ = service.CreateAvatar("user123", AvatarStyleRealistic, "Test User")
	_, _ = service.BuildSocialGraph("user123", []string{"friend1", "friend2"})
	_, _ = service.CreateCommunity("Test Community")

	data, err := service.ExportSocialData("user123")
	if err != nil {
		t.Errorf("ExportSocialData() error = %v", err)
		return
	}

	if data == nil {
		t.Error("ExportSocialData() returned nil")
		return
	}

	if _, ok := data["avatar"]; !ok {
		t.Error("ExportSocialData() missing avatar data")
	}

	if _, ok := data["social_graph"]; !ok {
		t.Error("ExportSocialData() missing social_graph data")
	}
}

func TestSocialServiceExportConfig(t *testing.T) {
	service := NewSocialService()

	data, err := service.ExportConfig()
	if err != nil {
		t.Errorf("ExportConfig() error = %v", err)
		return
	}

	if len(data) == 0 {
		t.Error("ExportConfig() returned empty data")
	}
}
