package service

import (
	"net/http"
	"time"
)

type DDoSProtectionService struct {
	enhancedService *EnhancedDDoSProtectionService
}

func NewDDoSProtectionService() *DDoSProtectionService {
	return &DDoSProtectionService{
		enhancedService: NewEnhancedDDoSProtectionService(),
	}
}

func (s *DDoSProtectionService) CheckRequest(r *http.Request) *EnhancedDDoSCheckResult {
	return s.enhancedService.CheckRequest(r)
}

func (s *DDoSProtectionService) GetGlobalStats() map[string]interface{} {
	return s.enhancedService.GetAllStats()
}

func (s *DDoSProtectionService) SetAttackThreshold(threshold float64) {
	cfg := s.enhancedService.GetConfig()
	cfg.TrafficAnomalyThreshold = threshold
	s.enhancedService.UpdateConfig(cfg)
}

func (s *DDoSProtectionService) AddToBlacklist(ip string, reason string, duration time.Duration) {
	s.enhancedService.AddToBlacklist(ip, reason, duration)
}

func (s *DDoSProtectionService) RemoveFromBlacklist(ip string) {
	s.enhancedService.RemoveFromBlacklist(ip)
}

func (s *DDoSProtectionService) AddToWhitelist(ip string) {
	s.enhancedService.AddToWhitelist(ip)
}

func (s *DDoSProtectionService) RemoveFromWhitelist(ip string) {
	s.enhancedService.RemoveFromWhitelist(ip)
}
