package i18n

import (
	"testing"
	"time"
)

func TestInit(t *testing.T) {
	err := Init(LocaleConfig{
		DefaultLang:     "zh-CN",
		SupportedLangs:  []string{"zh-CN", "en-US", "ja-JP", "ko-KR", "fr-FR", "de-DE", "es-ES", "pt-BR", "it-IT", "ru-RU", "ar-SA", "fa-IR", "he-IL", "ur-PK", "hi-IN", "vi-VN", "th-TH", "id-ID", "tr-TR", "zh-TW", "ms-MY", "bn-BD", "ta-IN"},
		TranslationsDir: "../../translations",
	})
	if err != nil {
		t.Fatalf("Failed to initialize i18n: %v", err)
	}
}

func TestTranslate(t *testing.T) {
	tests := []struct {
		name     string
		lang     string
		key      string
		args     []interface{}
		expected string
	}{
		{"Chinese hello", "zh-CN", "title", nil, "墨盾验证"},
		{"English hello", "en-US", "title", nil, "Modun Captcha"},
		{"Japanese hello", "ja-JP", "title", nil, "墨盾検証"},
		{"Fallback to default", "unknown", "title", nil, "墨盾验证"},
		{"Missing key", "zh-CN", "missing_key", nil, "missing_key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Translate(tt.lang, tt.key, tt.args...)
			if result != tt.expected {
				t.Errorf("Translate(%q, %q) = %q, want %q", tt.lang, tt.key, result, tt.expected)
			}
		})
	}
}

func TestIsSupported(t *testing.T) {
	tests := []struct {
		lang     string
		expected bool
	}{
		{"zh-CN", true},
		{"en-US", true},
		{"ja-JP", true},
		{"zh-TW", true},
		{"ms-MY", true},
		{"bn-BD", true},
		{"ta-IN", true},
		{"xx-XX", false},
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			result := IsSupported(tt.lang)
			if result != tt.expected {
				t.Errorf("IsSupported(%q) = %v, want %v", tt.lang, result, tt.expected)
			}
		})
	}
}

func TestGetLangInfo(t *testing.T) {
	info := GetLangInfo("zh-CN")
	if info.Code != "zh-CN" {
		t.Errorf("Expected Code 'zh-CN', got %q", info.Code)
	}
	if info.IsRTL != false {
		t.Errorf("Expected IsRTL false, got %v", info.IsRTL)
	}

	rtlInfo := GetLangInfo("ar-SA")
	if rtlInfo.IsRTL != true {
		t.Errorf("Expected IsRTL true for Arabic, got %v", rtlInfo.IsRTL)
	}
}

func TestIsRTL(t *testing.T) {
	tests := []struct {
		lang     string
		expected bool
	}{
		{"ar-SA", true},
		{"fa-IR", true},
		{"he-IL", true},
		{"ur-PK", true},
		{"zh-CN", false},
		{"en-US", false},
		{"ja-JP", false},
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			result := IsRTL(tt.lang)
			if result != tt.expected {
				t.Errorf("IsRTL(%q) = %v, want %v", tt.lang, result, tt.expected)
			}
		})
	}
}

func TestFormatDateTime(t *testing.T) {
	tm := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name       string
		lang       string
		formatType string
	}{
		{"Chinese date", "zh-CN", "medium"},
		{"English date", "en-US", "medium"},
		{"Japanese date", "ja-JP", "medium"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDateTime(tm, tt.lang, tt.formatType)
			if result == "" {
				t.Errorf("FormatDateTime returned empty string for %s", tt.lang)
			}
		})
	}
}

func TestFormatRelativeTime(t *testing.T) {
	tests := []struct {
		name string
		lang string
	}{
		{"Chinese relative time", "zh-CN"},
		{"English relative time", "en-US"},
		{"Arabic relative time", "ar-SA"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now()
			result := FormatRelativeTime(now, tt.lang)
			if result == "" {
				t.Errorf("FormatRelativeTime returned empty string for %s", tt.lang)
			}
		})
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		lang     string
	}{
		{"Chinese number", 1234567.89, "zh-CN"},
		{"English number", 1234567.89, "en-US"},
		{"German number", 1234567.89, "de-DE"},
		{"French number", 1234567.89, "fr-FR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatNumber(tt.value, tt.lang)
			if result == "" {
				t.Errorf("FormatNumber returned empty string for %s", tt.lang)
			}
		})
	}
}

func TestFormatCurrency(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		lang  string
	}{
		{"Chinese currency", 1234.56, "zh-CN"},
		{"English currency", 1234.56, "en-US"},
		{"Japanese currency", 1234, "ja-JP"},
		{"Euro currency", 1234.56, "de-DE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatCurrency(tt.value, tt.lang)
			if result == "" {
				t.Errorf("FormatCurrency returned empty string for %s", tt.lang)
			}
		})
	}
}

func TestFormatCurrencyWithCode(t *testing.T) {
	result := FormatCurrencyWithCode(1234.56, "en-US", "USD")
	if result == "" {
		t.Error("FormatCurrencyWithCode returned empty string")
	}
}

func TestFormatPercent(t *testing.T) {
	result := FormatPercent(0.4567, "en-US")
	if result == "" {
		t.Error("FormatPercent returned empty string")
	}
}

func TestFormatInteger(t *testing.T) {
	result := FormatInteger(1234567, "de-DE")
	if result == "" {
		t.Error("FormatInteger returned empty string")
	}
}

func TestGetCSSDirection(t *testing.T) {
	tests := []struct {
		lang     string
		expected string
	}{
		{"ar-SA", "rtl"},
		{"fa-IR", "rtl"},
		{"zh-CN", "ltr"},
		{"en-US", "ltr"},
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			result := GetCSSDirection(tt.lang)
			if result != tt.expected {
				t.Errorf("GetCSSDirection(%q) = %q, want %q", tt.lang, result, tt.expected)
			}
		})
	}
}

func TestGetDirectionClass(t *testing.T) {
	tests := []struct {
		lang     string
		expected string
	}{
		{"ar-SA", "direction-rtl"},
		{"zh-CN", "direction-ltr"},
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			result := GetDirectionClass(tt.lang)
			if result != tt.expected {
				t.Errorf("GetDirectionClass(%q) = %q, want %q", tt.lang, result, tt.expected)
			}
		})
	}
}

func TestMirrorCSSProperty(t *testing.T) {
	tests := []struct {
		lang     string
		property string
		expected string
	}{
		{"ar-SA", "left", "right"},
		{"ar-SA", "right", "left"},
		{"zh-CN", "left", "left"},
		{"zh-CN", "right", "right"},
	}

	for _, tt := range tests {
		t.Run(tt.lang+":"+tt.property, func(t *testing.T) {
			result := MirrorCSSProperty(tt.lang, tt.property)
			if result != tt.expected {
				t.Errorf("MirrorCSSProperty(%q, %q) = %q, want %q", tt.lang, tt.property, result, tt.expected)
			}
		})
	}
}

func TestHasRTLScript(t *testing.T) {
	tests := []struct {
		text     string
		expected bool
	}{
		{"العربية", true},
		{"فارسی", true},
		{"עברית", true},
		{"English", false},
		{"中文", false},
		{"日本語", false},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			result := HasRTLScript(tt.text)
			if result != tt.expected {
				t.Errorf("HasRTLScript(%q) = %v, want %v", tt.text, result, tt.expected)
			}
		})
	}
}

func TestGetTimezoneInfo(t *testing.T) {
	info := GetTimezoneInfo("Asia/Shanghai")
	if info.Name != "Asia/Shanghai" {
		t.Errorf("Expected Name 'Asia/Shanghai', got %q", info.Name)
	}
	if info.CountryCode != "CN" {
		t.Errorf("Expected CountryCode 'CN', got %q", info.CountryCode)
	}
}

func TestConvertTime(t *testing.T) {
	tm := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	result := ConvertTime(tm, "UTC", "Asia/Shanghai")
	if result.IsZero() {
		t.Error("ConvertTime returned zero time")
	}
}

func TestGetSupportedLangs(t *testing.T) {
	langs := GetSupportedLangs()
	if len(langs) == 0 {
		t.Error("GetSupportedLangs returned empty list")
	}
}

func TestGetAllLangInfos(t *testing.T) {
	infos := GetAllLangInfos()
	if len(infos) == 0 {
		t.Error("GetAllLangInfos returned empty list")
	}
}
