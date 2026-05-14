package fingerprint

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
)

type Generator struct {
	salt string
}

type DeviceInfo struct {
	UserAgent      string `json:"user_agent"`
	Language       string `json:"language"`
	ScreenWidth    int    `json:"screen_width"`
	ScreenHeight   int    `json:"screen_height"`
	ColorDepth     int    `json:"color_depth"`
	Timezone       string `json:"timezone"`
	Platform       string `json:"platform"`
	CookiesEnabled bool   `json:"cookies_enabled"`
	CanvasHash     string `json:"canvas_hash"`
	WebGLHash      string `json:"webgl_hash"`
	AudioHash      string `json:"audio_hash"`
	FontsHash      string `json:"fonts_hash"`
}

func NewGenerator(salt string) *Generator {
	return &Generator{salt: salt}
}

func (g *Generator) Generate(info *DeviceInfo) string {
	components := []string{
		fmt.Sprintf("ua=%s", info.UserAgent),
		fmt.Sprintf("lang=%s", info.Language),
		fmt.Sprintf("screen=%dx%d", info.ScreenWidth, info.ScreenHeight),
		fmt.Sprintf("color=%d", info.ColorDepth),
		fmt.Sprintf("tz=%s", info.Timezone),
		fmt.Sprintf("platform=%s", info.Platform),
		fmt.Sprintf("canvas=%s", info.CanvasHash),
		fmt.Sprintf("webgl=%s", info.WebGLHash),
		fmt.Sprintf("audio=%s", info.AudioHash),
		fmt.Sprintf("fonts=%s", info.FontsHash),
	}

	sort.Strings(components)

	combined := strings.Join(components, "&")

	hash := g.hashWithSalt(combined)

	return hash
}

func (g *Generator) hashWithSalt(data string) string {
	h := md5.New()
	h.Write([]byte(g.salt))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func (g *Generator) GenerateHash(data ...string) string {
	h := sha256.New()
	for _, d := range data {
		h.Write([]byte(d))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func (g *Generator) Similarity(fp1, fp2 string) float64 {
	if len(fp1) != len(fp2) {
		return 0
	}

	matches := 0
	for i := 0; i < len(fp1); i++ {
		if fp1[i] == fp2[i] {
			matches++
		}
	}

	return float64(matches) / float64(len(fp1))
}

func (g *Generator) IsDuplicate(fingerprints []string, newFP string, threshold float64) bool {
	for _, fp := range fingerprints {
		if g.Similarity(fp, newFP) >= threshold {
			return true
		}
	}
	return false
}

func (g *Generator) GenerateSessionToken() string {
	timestamp := fmt.Sprintf("%d", time.Now().UnixNano())
	random := g.GenerateHash(timestamp, fmt.Sprintf("%d", time.Now().Unix()))
	return g.hashWithSalt(timestamp + random)
}

func SimpleHash(data string) string {
	h := md5.New()
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func SHA256Hash(data string) string {
	h := sha256.New()
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func MultiHash(data ...string) string {
	h := sha256.New()
	for _, d := range data {
		h.Write([]byte(d))
	}
	return hex.EncodeToString(h.Sum(nil))
}
