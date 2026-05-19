package service

import (
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"
)

type AvatarStyle string

const (
	AvatarStyleRealistic  AvatarStyle = "realistic"
	AvatarStyleCartoon    AvatarStyle = "cartoon"
	AvatarStyleAbstract   AvatarStyle = "abstract"
	AvatarStyleAnime      AvatarStyle = "anime"
)

type VirtualAvatar struct {
	ID          string        `json:"id"`
	UserID      string        `json:"user_id"`
	Style       AvatarStyle   `json:"style"`
	Name        string        `json:"name"`
	AvatarURL   string        `json:"avatar_url"`
	Emotion     string        `json:"emotion"`
	Expression  AvatarExpression `json:"expression"`
	Position    []float64     `json:"position"`
	Rotation    []float64     `json:"rotation"`
	Scale       float64       `json:"scale"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

type AvatarExpression struct {
	Eyes       string  `json:"eyes"`
	Mouth      string  `json:"mouth"`
	Emotion    string  `json:"emotion"`
	Intensity  float64 `json:"intensity"`
}

type SocialConnection struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	FriendID    string    `json:"friend_id"`
	ConnectionType string  `json:"connection_type"`
	Strength    float64   `json:"strength"`
	Verified    bool      `json:"verified"`
	CreatedAt   time.Time `json:"created_at"`
}

type SocialGraph struct {
	UserID           string             `json:"user_id"`
	Connections      []SocialConnection `json:"connections"`
	FriendsCount     int                `json:"friends_count"`
	VerifiedFriends  int                `json:"verified_friends"`
	TrustScore       float64            `json:"trust_score"`
	CommunityScore   float64            `json:"community_score"`
	RiskLevel        string             `json:"risk_level"`
}

type CommunityTrust struct {
	CommunityID   string    `json:"community_id"`
	CommunityName string    `json:"community_name"`
	MemberCount   int       `json:"member_count"`
	TrustLevel    float64   `json:"trust_level"`
	AvgTrustScore float64   `json:"avg_trust_score"`
	VerifiedMembers int     `json:"verified_members"`
	CreatedAt     time.Time `json:"created_at"`
}

type FriendVerificationRequest struct {
	ID            string   `json:"id"`
	UserID        string   `json:"user_id"`
	FriendIDs     []string `json:"friend_ids"`
	RequestType   string   `json:"request_type"`
	Message       string   `json:"message"`
	Status        string   `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	RespondedAt   *time.Time `json:"responded_at,omitempty"`
	Response      string   `json:"response,omitempty"`
}

type SocialVerificationRequest struct {
	SessionID     string                   `json:"session_id"`
	UserID        string                   `json:"user_id"`
	AvatarData    *VirtualAvatar           `json:"avatar_data"`
	SocialGraph   *SocialGraph             `json:"social_graph"`
	Communities   []CommunityTrust          `json:"communities"`
	FriendReq     *FriendVerificationRequest `json:"friend_request"`
	Timestamp     int64                    `json:"timestamp"`
}

type SocialVerificationResult struct {
	IsValid           bool                   `json:"is_valid"`
	Confidence        float64                `json:"confidence"`
	AvatarScore       float64                `json:"avatar_score"`
	SocialScore       float64                `json:"social_score"`
	TrustScore        float64                `json:"trust_score"`
	FriendSupportScore float64               `json:"friend_support_score"`
	CommunityBonus    float64                `json:"community_bonus"`
	OverallScore      float64                `json:"overall_score"`
	RiskLevel         string                 `json:"risk_level"`
	Details           string                 `json:"details"`
	Metrics           map[string]interface{} `json:"metrics"`
	ProcessingTime    int64                  `json:"processing_time"`
}

type SocialService struct {
	avatars      map[string]*VirtualAvatar
	socialGraphs map[string]*SocialGraph
	communities  map[string]*CommunityTrust
	friendReqs   map[string]*FriendVerificationRequest
	config       *SocialConfig
	mu           sync.RWMutex
}

type SocialConfig struct {
	AvatarRequired        bool    `json:"avatar_required"`
	MinConnections        int     `json:"min_connections"`
	MinTrustScore         float64 `json:"min_trust_score"`
	AvatarWeight          float64 `json:"avatar_weight"`
	SocialWeight          float64 `json:"social_weight"`
	TrustWeight           float64 `json:"trust_weight"`
	FriendSupportWeight   float64 `json:"friend_support_weight"`
	CommunityBonusWeight  float64 `json:"community_bonus_weight"`
	FriendVerificationEnabled bool `json:"friend_verification_enabled"`
	CrossVerificationRequired bool `json:"cross_verification_required"`
	MinVerifiedFriends    int     `json:"min_verified_friends"`
}

func NewSocialService() *SocialService {
	return &SocialService{
		avatars:      make(map[string]*VirtualAvatar),
		socialGraphs: make(map[string]*SocialGraph),
		communities:  make(map[string]*CommunityTrust),
		friendReqs:   make(map[string]*FriendVerificationRequest),
		config: &SocialConfig{
			AvatarRequired:        true,
			MinConnections:        3,
			MinTrustScore:         0.6,
			AvatarWeight:          0.25,
			SocialWeight:          0.30,
			TrustWeight:           0.25,
			FriendSupportWeight:   0.10,
			CommunityBonusWeight:  0.10,
			FriendVerificationEnabled: true,
			CrossVerificationRequired: true,
			MinVerifiedFriends:    2,
		},
	}
}

func (s *SocialService) GetConfig() *SocialConfig {
	return s.config
}

func (s *SocialService) UpdateConfig(config *SocialConfig) {
	s.config = config
}

func (s *SocialService) CreateAvatar(userID string, style AvatarStyle, name string) (*VirtualAvatar, error) {
	avatar := &VirtualAvatar{
		ID:        fmt.Sprintf("avatar_%s_%d", userID, time.Now().UnixNano()),
		UserID:    userID,
		Style:     style,
		Name:      name,
		AvatarURL: fmt.Sprintf("/avatars/%s/%s.png", style, userID),
		Emotion:   "neutral",
		Expression: AvatarExpression{
			Eyes:      "open",
			Mouth:     "neutral",
			Emotion:   "neutral",
			Intensity: 0.5,
		},
		Position: []float64{0, 0, 0},
		Rotation: []float64{0, 0, 0},
		Scale:    1.0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	s.avatars[userID] = avatar
	return avatar, nil
}

func (s *SocialService) GetAvatar(userID string) (*VirtualAvatar, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	avatar, exists := s.avatars[userID]
	if !exists {
		return nil, fmt.Errorf("avatar not found for user: %s", userID)
	}

	return avatar, nil
}

func (s *SocialService) UpdateAvatarExpression(userID string, emotion string, intensity float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	avatar, exists := s.avatars[userID]
	if !exists {
		return fmt.Errorf("avatar not found for user: %s", userID)
	}

	avatar.Emotion = emotion
	avatar.Expression = AvatarExpression{
		Eyes:      s.getEyesFromEmotion(emotion),
		Mouth:     s.getMouthFromEmotion(emotion),
		Emotion:   emotion,
		Intensity: intensity,
	}
	avatar.UpdatedAt = time.Now()

	return nil
}

func (s *SocialService) getEyesFromEmotion(emotion string) string {
	eyes := map[string]string{
		"happy":     "curved",
		"sad":       "down",
		"angry":     "narrowed",
		"surprised": "wide",
		"neutral":   "open",
		"confused":  "asymmetric",
	}

	if e, ok := eyes[emotion]; ok {
		return e
	}
	return "open"
}

func (s *SocialService) getMouthFromEmotion(emotion string) string {
	mouth := map[string]string{
		"happy":     "smile",
		"sad":       "frown",
		"angry":     "grimace",
		"surprised": "open",
		"neutral":   "neutral",
		"confused":  "wavy",
	}

	if m, ok := mouth[emotion]; ok {
		return m
	}
	return "neutral"
}

func (s *SocialService) UpdateAvatarPose(userID string, position, rotation []float64, scale float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	avatar, exists := s.avatars[userID]
	if !exists {
		return fmt.Errorf("avatar not found for user: %s", userID)
	}

	if len(position) >= 3 {
		avatar.Position = position[:3]
	}
	if len(rotation) >= 3 {
		avatar.Rotation = rotation[:3]
	}
	avatar.Scale = scale
	avatar.UpdatedAt = time.Now()

	return nil
}

func (s *SocialService) BuildSocialGraph(userID string, connectionIDs []string) (*SocialGraph, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	connections := make([]SocialConnection, 0)

	for i, friendID := range connectionIDs {
		connection := SocialConnection{
			ID:             fmt.Sprintf("conn_%d", i),
			UserID:         userID,
			FriendID:       friendID,
			ConnectionType: "friend",
			Strength:       s.calculateConnectionStrength(friendID),
			Verified:       i%2 == 0,
			CreatedAt:      time.Now(),
		}
		connections = append(connections, connection)
	}

	verifiedCount := 0
	var totalTrust float64
	for _, conn := range connections {
		if conn.Verified {
			verifiedCount++
			totalTrust += conn.Strength
		}
	}

	avgTrust := 0.0
	if len(connections) > 0 {
		avgTrust = totalTrust / float64(len(connections))
	}

	trustScore := avgTrust * (1.0 + float64(verifiedCount)*0.1)

	communityScore := s.calculateCommunityScore(connections)

	riskLevel := "low"
	if trustScore < 0.5 || len(connections) < 3 {
		riskLevel = "medium"
	}
	if trustScore < 0.3 || len(connections) < 1 {
		riskLevel = "high"
	}

	graph := &SocialGraph{
		UserID:          userID,
		Connections:     connections,
		FriendsCount:    len(connections),
		VerifiedFriends: verifiedCount,
		TrustScore:     math.Min(1.0, trustScore),
		CommunityScore:  communityScore,
		RiskLevel:      riskLevel,
	}

	s.socialGraphs[userID] = graph
	return graph, nil
}

func (s *SocialService) calculateConnectionStrength(friendID string) float64 {
	baseStrength := 0.5 + math.Sin(float64(time.Now().UnixNano()%100)/10.0)*0.3

	interactionBonus := 0.1
	commonFriendsBonus := 0.15

	return math.Min(1.0, baseStrength+interactionBonus+commonFriendsBonus)
}

func (s *SocialService) calculateCommunityScore(connections []SocialConnection) float64 {
	if len(connections) == 0 {
		return 0.0
	}

	var totalStrength float64
	for _, conn := range connections {
		totalStrength += conn.Strength
	}

	avgStrength := totalStrength / float64(len(connections))

	verifiedBonus := 0.1 * float64(countVerified(connections))

	return math.Min(1.0, avgStrength+verifiedBonus)
}

func countVerified(connections []SocialConnection) int {
	count := 0
	for _, conn := range connections {
		if conn.Verified {
			count++
		}
	}
	return count
}

func (s *SocialService) GetSocialGraph(userID string) (*SocialGraph, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	graph, exists := s.socialGraphs[userID]
	if !exists {
		return nil, fmt.Errorf("social graph not found for user: %s", userID)
	}

	return graph, nil
}

func (s *SocialService) CreateCommunity(name string) (*CommunityTrust, error) {
	community := &CommunityTrust{
		CommunityID:   fmt.Sprintf("community_%d", time.Now().UnixNano()),
		CommunityName: name,
		MemberCount:   1,
		TrustLevel:    0.8,
		AvgTrustScore: 0.75,
		VerifiedMembers: 1,
		CreatedAt:     time.Now(),
	}

	s.communities[community.CommunityID] = community
	return community, nil
}

func (s *SocialService) AddUserToCommunity(communityID string, trustScore float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	community, exists := s.communities[communityID]
	if !exists {
		return fmt.Errorf("community not found: %s", communityID)
	}

	community.MemberCount++
	totalTrust := community.AvgTrustScore * float64(community.VerifiedMembers)
	community.AvgTrustScore = (totalTrust + trustScore) / float64(community.MemberCount)

	if trustScore > 0.7 {
		community.VerifiedMembers++
	}

	return nil
}

func (s *SocialService) GetCommunity(communityID string) (*CommunityTrust, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	community, exists := s.communities[communityID]
	if !exists {
		return nil, fmt.Errorf("community not found: %s", communityID)
	}

	return community, nil
}

func (s *SocialService) CreateFriendVerificationRequest(userID string, friendIDs []string, requestType string, message string) (*FriendVerificationRequest, error) {
	request := &FriendVerificationRequest{
		ID:          fmt.Sprintf("freq_%d", time.Now().UnixNano()),
		UserID:      userID,
		FriendIDs:   friendIDs,
		RequestType: requestType,
		Message:     message,
		Status:      "pending",
		CreatedAt:   time.Now(),
	}

	s.friendReqs[request.ID] = request
	return request, nil
}

func (s *SocialService) GetFriendVerificationRequest(requestID string) (*FriendVerificationRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	request, exists := s.friendReqs[requestID]
	if !exists {
		return nil, fmt.Errorf("friend verification request not found: %s", requestID)
	}

	return request, nil
}

func (s *SocialService) RespondToFriendVerification(requestID string, friendID string, accept bool, response string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	request, exists := s.friendReqs[requestID]
	if !exists {
		return fmt.Errorf("friend verification request not found: %s", requestID)
	}

	now := time.Now()
	request.RespondedAt = &now

	if accept {
		request.Response = "accepted"
		request.Status = "accepted"

		for _, fid := range request.FriendIDs {
			if fid == friendID {
				s.updateConnectionVerification(request.UserID, fid)
				break
			}
		}
	} else {
		request.Response = "declined"
		request.Status = "declined"
	}

	if response != "" {
		request.Response = response
	}

	return nil
}

func (s *SocialService) updateConnectionVerification(userID, friendID string) error {
	graph, exists := s.socialGraphs[userID]
	if !exists {
		return nil
	}

	for i := range graph.Connections {
		if graph.Connections[i].FriendID == friendID {
			graph.Connections[i].Verified = true
			graph.VerifiedFriends++
			break
		}
	}

	return nil
}

func (s *SocialService) Verify(request *SocialVerificationRequest) (*SocialVerificationResult, error) {
	startTime := time.Now()

	result := &SocialVerificationResult{
		Metrics: make(map[string]interface{}),
	}

	if request.AvatarData == nil && s.config.AvatarRequired {
		result.Details = "avatar data is required"
		return result, nil
	}

	avatarScore := s.evaluateAvatar(request.AvatarData)
	result.AvatarScore = avatarScore

	socialScore := s.evaluateSocialGraph(request.SocialGraph)
	result.SocialScore = socialScore

	trustScore := s.evaluateTrustScore(request.SocialGraph, request.Communities)
	result.TrustScore = trustScore

	friendSupportScore := s.evaluateFriendSupport(request.FriendReq)
	result.FriendSupportScore = friendSupportScore

	communityBonus := s.calculateCommunityBonus(request.Communities)
	result.CommunityBonus = communityBonus

	result.Confidence = avatarScore*s.config.AvatarWeight +
		socialScore*s.config.SocialWeight +
		trustScore*s.config.TrustWeight +
		friendSupportScore*s.config.FriendSupportWeight +
		communityBonus*s.config.CommunityBonusWeight

	result.OverallScore = result.Confidence

	if s.config.FriendVerificationEnabled && friendSupportScore < 0.3 {
		result.OverallScore *= 0.8
	}

	result.RiskLevel = s.determineRiskLevel(result.OverallScore, socialScore, trustScore)

	result.IsValid = result.OverallScore >= s.config.MinTrustScore &&
		socialScore >= 0.5 &&
		result.RiskLevel != "high"

	result.Details = fmt.Sprintf(
		"avatar: %.2f, social: %.2f, trust: %.2f, friend: %.2f, community: %.2f, overall: %.2f, risk: %s",
		avatarScore, socialScore, trustScore, friendSupportScore, communityBonus, result.OverallScore, result.RiskLevel,
	)

	result.ProcessingTime = time.Since(startTime).Milliseconds()

	result.Metrics["friends_count"] = len(request.SocialGraph.Connections)
	result.Metrics["verified_friends"] = request.SocialGraph.VerifiedFriends
	result.Metrics["community_count"] = len(request.Communities)
	result.Metrics["avatar_style"] = request.AvatarData.Style

	return result, nil
}

func (s *SocialService) evaluateAvatar(avatar *VirtualAvatar) float64 {
	if avatar == nil {
		return 0.0
	}

	completenessScore := 0.0
	if avatar.Name != "" {
		completenessScore += 0.2
	}
	if avatar.AvatarURL != "" {
		completenessScore += 0.3
	}
	if avatar.Emotion != "" {
		completenessScore += 0.2
	}
	if avatar.Expression.Intensity > 0 {
		completenessScore += 0.1
	}
	if avatar.Position != nil && len(avatar.Position) >= 3 {
		completenessScore += 0.2
	}

	styleBonus := 0.0
	switch avatar.Style {
	case AvatarStyleRealistic:
		styleBonus = 0.1
	case AvatarStyleAnime:
		styleBonus = 0.05
	}

	recencyBonus := 0.0
	if time.Since(avatar.UpdatedAt) < 24*time.Hour {
		recencyBonus = 0.05
	}

	return math.Min(1.0, completenessScore+styleBonus+recencyBonus)
}

func (s *SocialService) evaluateSocialGraph(graph *SocialGraph) float64 {
	if graph == nil {
		return 0.0
	}

	connectionScore := 0.0
	if graph.FriendsCount >= s.config.MinConnections {
		connectionScore = 0.5 + 0.3*math.Min(1.0, float64(graph.FriendsCount-s.config.MinConnections)/10.0)
	}

	verifiedBonus := float64(graph.VerifiedFriends) / math.Max(1.0, float64(graph.FriendsCount)) * 0.3

	communityBonus := graph.CommunityScore * 0.2

	return math.Min(1.0, connectionScore+verifiedBonus+communityBonus)
}

func (s *SocialService) evaluateTrustScore(graph *SocialGraph, communities []CommunityTrust) float64 {
	if graph == nil {
		return 0.0
	}

	trustScore := graph.TrustScore

	verifiedBonus := float64(graph.VerifiedFriends) / math.Max(1.0, float64(graph.FriendsCount)) * 0.15

	var communityBonus float64
	for _, comm := range communities {
		communityBonus += comm.TrustLevel * comm.AvgTrustScore
	}
	if len(communities) > 0 {
		communityBonus /= float64(len(communities))
	}

	return math.Min(1.0, trustScore+verifiedBonus+communityBonus*0.2)
}

func (s *SocialService) evaluateFriendSupport(request *FriendVerificationRequest) float64 {
	if request == nil {
		return 0.5
	}

	statusScore := 0.0
	switch request.Status {
	case "accepted":
		statusScore = 1.0
	case "pending":
		statusScore = 0.6
	case "declined":
		statusScore = 0.2
	}

	participationRatio := float64(len(request.FriendIDs)) / math.Max(1.0, float64(len(request.FriendIDs)))

	return statusScore * 0.7 + participationRatio*0.3
}

func (s *SocialService) calculateCommunityBonus(communities []CommunityTrust) float64 {
	if len(communities) == 0 {
		return 0.0
	}

	var totalScore float64
	for _, comm := range communities {
		totalScore += comm.TrustLevel * comm.AvgTrustScore
	}

	avgScore := totalScore / float64(len(communities))

	participationBonus := math.Min(0.2, float64(len(communities))*0.05)

	return math.Min(1.0, avgScore+participationBonus)
}

func (s *SocialService) determineRiskLevel(overallScore, socialScore, trustScore float64) string {
	if overallScore < 0.4 || trustScore < 0.3 {
		return "high"
	}
	if overallScore < 0.6 || socialScore < 0.5 {
		return "medium"
	}
	return "low"
}

func (s *SocialService) GenerateAvatarScene(avatarID string, sceneType string) (string, error) {
	avatar, err := s.GetAvatar(avatarID)
	if err != nil {
		return "", err
	}

	sceneConfig := map[string]interface{}{
		"type":     sceneType,
		"avatar":   avatar,
		"lighting": "natural",
		"background": s.getSceneBackground(sceneType),
	}

	sceneJSON, _ := json.Marshal(sceneConfig)
	return string(sceneJSON), nil
}

func (s *SocialService) getSceneBackground(sceneType string) string {
	backgrounds := map[string]string{
		"social":   "gradual_blue",
		"gaming":   "dark_purple",
		"business": "office",
		"casual":   "park",
	}

	if bg, ok := backgrounds[sceneType]; ok {
		return bg
	}
	return "neutral"
}

func (s *SocialService) ExportSocialData(userID string) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data := make(map[string]interface{})

	if avatar, exists := s.avatars[userID]; exists {
		data["avatar"] = avatar
	}

	if graph, exists := s.socialGraphs[userID]; exists {
		data["social_graph"] = graph
	}

	communityList := make([]*CommunityTrust, 0)
	for _, comm := range s.communities {
		communityList = append(communityList, comm)
	}
	data["communities"] = communityList

	return data, nil
}

func (s *SocialService) ExportConfig() ([]byte, error) {
	return json.Marshal(s.config)
}
