// Package icon provides icon-based captcha generation and verification.
// Users identify and select target icons from a grid of icons.
package icon

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"captchax/config"
	"captchax/pkg/cache"
	"context"

	"github.com/google/uuid"
)

// IconCaptcha handles icon-based captcha generation.
type IconCaptcha struct {
	cfg     *config.CaptchaConfig
	redis   *cache.RedisClient
	iconLib []IconInfo
}

// New creates a new IconCaptcha generator.
func New(cfg *config.CaptchaConfig, redisClient *cache.RedisClient) *IconCaptcha {
	return &IconCaptcha{
		cfg:     cfg,
		redis:   redisClient,
		iconLib: loadIconLibrary(),
	}
}

// loadIconLibrary loads a library of SVG icons.
func loadIconLibrary() []IconInfo {
	return []IconInfo{
		{ID: "icon_001", Name: "heart", SVG: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="#e91e63"><path d="M12 21.35l-1.45-1.32C5.4 15.36 2 12.28 2 8.5 2 5.42 4.42 3 7.5 3c1.74 0 3.41.81 4.5 2.09C13.09 3.81 14.76 3 16.5 3 19.58 3 22 5.42 22 8.5c0 3.78-3.4 6.86-8.55 11.54L12 21.35z"/></svg>`, Width: 64, Height: 64},
		{ID: "icon_002", Name: "star", SVG: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="#ff9800"><path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"/></svg>`, Width: 64, Height: 64},
		{ID: "icon_003", Name: "circle", SVG: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="#2196f3"><circle cx="12" cy="12" r="10"/></svg>`, Width: 64, Height: 64},
		{ID: "icon_004", Name: "square", SVG: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="#4caf50"><rect x="2" y="2" width="20" height="20" rx="2"/></svg>`, Width: 64, Height: 64},
		{ID: "icon_005", Name: "triangle", SVG: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="#9c27b0"><polygon points="12,2 22,22 2,22"/></svg>`, Width: 64, Height: 64},
		{ID: "icon_006", Name: "diamond", SVG: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="#00bcd4"><polygon points="12,2 22,12 12,22 2,12"/></svg>`, Width: 64, Height: 64},
		{ID: "icon_007", Name: "check", SVG: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="#8bc34a"><path d="M9 16.17L4.83 12l-1.41 1.41L9 19 21 7l-1.41-1.41L9 16.17z"/></svg>`, Width: 64, Height: 64},
		{ID: "icon_008", Name: "cross", SVG: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="#f44336"><path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12 19 6.41z"/></svg>`, Width: 64, Height: 64},
		{ID: "icon_009", Name: "home", SVG: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="#795548"><path d="M10 20v-6h4v6h5v-8h3L12 3 2 12h3v8z"/></svg>`, Width: 64, Height: 64},
		{ID: "icon_010", Name: "user", SVG: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="#607d8b"><path d="M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm0 2c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z"/></svg>`, Width: 64, Height: 64},
		{ID: "icon_011", Name: "folder", SVG: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="#ffeb3b"><path d="M10 4H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2h-8l-2-2z"/></svg>`, Width: 64, Height: 64},
		{ID: "icon_012", Name: "mail", SVG: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="#e53935"><path d="M20 4H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V6c0-1.1-.9-2-2-2zm0 4l-8 5-8-5V6l8 5 8-5v2z"/></svg>`, Width: 64, Height: 64},
		{ID: "icon_013", Name: "search", SVG: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="#1976d2"><path d="M15.5 14h-.79l-.28-.27C15.41 12.59 16 11.11 16 9.5 16 5.91 13.09 3 9.5 3S3 5.91 3 9.5 5.91 16 9.5 16c1.61 0 3.09-.59 4.23-1.57l.27.28v.79l5 4.99L20.49 19l-4.99-5zm-6 0C7.01 14 5 11.99 5 9.5S7.01 5 9.5 5 14 7.01 14 9.5 11.99 14 9.5 14z"/></svg>`, Width: 64, Height: 64},
		{ID: "icon_014", Name: "bell", SVG: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="#ff5722"><path d="M12 22c1.1 0 2-.9 2-2h-4c0 1.1.89 2 2 2zm6-6v-5c0-3.07-1.64-5.64-4.5-6.32V4c0-.83-.67-1.5-1.5-1.5s-1.5.67-1.5 1.5v.68C7.63 5.36 6 7.92 6 11v5l-2 2v1h16v-1l-2-2z"/></svg>`, Width: 64, Height: 64},
		{ID: "icon_015", Name: "clock", SVG: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="#78909c"><circle cx="12" cy="12" r="10" fill="none" stroke="currentColor" stroke-width="2"/><path d="M12 6v6l4 2" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"/></svg>`, Width: 64, Height: 64},
	}
}

// GenerateCaptcha generates a new icon-based captcha.
func (ic *IconCaptcha) GenerateCaptcha(ctx context.Context) (*CaptchaResult, error) {
	id := uuid.New().String()

	// Determine number of target icons (3-5)
	targetCount := MinTargetIcons + randInt(MaxTargetIcons-MinTargetIcons+1)
	
	// Determine grid size
	gridCols := DefaultGridCols
	gridRows := DefaultGridRows
	totalIcons := gridCols * gridRows
	
	// Select target icons
	targetIndices := selectRandomIcons(ic.iconLib, targetCount)
	
	// Select remaining icons (total - targetCount)
	remainingCount := totalIcons - targetCount
	remainingIndices := selectRandomIconsExcluding(ic.iconLib, remainingCount, getIconIDs(targetIndices))
	
	// Combine and shuffle
	allIcons := append(targetIndices, remainingIndices...)
	shuffleIcons(allIcons)
	
	// Collect target IDs for verification
	targetIDs := make([]string, len(targetIndices))
	for i, icon := range targetIndices {
		targetIDs[i] = icon.ID
	}
	
	// Store in cache if redis (CaptchaData not directly used but kept for documentation)
	
	if ic.redis != nil {
		cacheManager := NewCacheManager(ic.cfg, ic.redis)
		if err := cacheManager.Set(ctx, id, &CacheData{
			ID:             id,
			TargetIconIDs:  targetIDs,
			CreatedAt:     time.Now().Unix(),
			Verified:      false,
		}); err != nil {
			return nil, fmt.Errorf("failed to store captcha: %w", err)
		}
	}
	
	return &CaptchaResult{
		ID:          id,
		TargetIcons: targetIndices,
		AllIcons:    allIcons,
		GridCols:    gridCols,
		GridRows:    gridRows,
		IconSize:    DefaultIconSize,
	}, nil
}

// Helper functions
func randInt(max int) int {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(max)))
	return int(n.Int64())
}

func selectRandomIcons(icons []IconInfo, count int) []IconInfo {
	result := make([]IconInfo, 0, count)
	used := make(map[int]bool)
	
	for len(result) < count {
		idx := randInt(len(icons))
		if !used[idx] {
			used[idx] = true
			result = append(result, icons[idx])
		}
	}
	
	return result
}

func selectRandomIconsExcluding(icons []IconInfo, count int, excludeIDs []string) []IconInfo {
	excludeMap := make(map[string]bool)
	for _, id := range excludeIDs {
		excludeMap[id] = true
	}
	
	available := make([]IconInfo, 0)
	for _, icon := range icons {
		if !excludeMap[icon.ID] {
			available = append(available, icon)
		}
	}
	
	return selectRandomIcons(available, count)
}

func getIconIDs(icons []IconInfo) []string {
	ids := make([]string, len(icons))
	for i, icon := range icons {
		ids[i] = icon.ID
	}
	return ids
}

func shuffleIcons(icons []IconInfo) {
	for i := len(icons) - 1; i > 0; i-- {
		j := randInt(i + 1)
		icons[i], icons[j] = icons[j], icons[i]
	}
}
