package i18n

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestInit(t *testing.T) {
	os.MkdirAll("test_translations", 0755)
	defer os.RemoveAll("test_translations")

	zhCN := `{"hello": "你好", "world": "世界"}`
	os.WriteFile("test_translations/zh-CN.json", []byte(zhCN), 0644)

	config := LocaleConfig{
		DefaultLang:     "zh-CN",
		TranslationsDir: "test_translations",
	}

	err := Init(config)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if defaultLang != "zh-CN" {
		t.Errorf("Expected defaultLang to be 'zh-CN', got '%s'", defaultLang)
	}
}

func setupTranslations() {
	wd, _ := os.Getwd()
	translationsPath := filepath.Join(wd, "..", "..", "translations")
	Init(LocaleConfig{
		DefaultLang:     "zh-CN",
		TranslationsDir: translationsPath,
	})
}

func TestIsSupported(t *testing.T) {
	setupTranslations()

	tests := []struct {
		lang     string
		expected bool
	}{
		{"zh-CN", true},
		{"en-US", true},
		{"ko-KR", true},
		{"pt-BR", true},
		{"ru-RU", true},
		{"ar-SA", true},
		{"fa-IR", true},
		{"he-IL", true},
		{"ur-PK", true},
		{"xx-XX", false},
		{"invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			result := IsSupported(tt.lang)
			if result != tt.expected {
				t.Errorf("IsSupported(%s) = %v, want %v", tt.lang, result, tt.expected)
			}
		})
	}
}

func TestTranslate(t *testing.T) {
	setupTranslations()

	tests := []struct {
		lang     string
		key      string
		expected string
	}{
		{"zh-CN", "title", "墨盾验证"},
		{"en-US", "title", "Modun Captcha"},
		{"ko-KR", "title", "모던 캡차"},
		{"pt-BR", "title", "Modun Captcha"},
		{"ru-RU", "title", "Modun Captcha"},
		{"ar-SA", "title", "مودان كابتشا"},
		{"ja-JP", "title", "Modun Captcha"},
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			result := Translate(tt.lang, tt.key)
			if result != tt.expected {
				t.Errorf("Translate(%s, %s) = %s, want %s", tt.lang, tt.key, result, tt.expected)
			}
		})
	}
}

func TestTranslateWithArgs(t *testing.T) {
	setupTranslations()

	result := Translate("zh-CN", "minutes_ago", 5)
	expected := "5分钟前"
	if result != expected {
		t.Errorf("Translate with args = %s, want %s", result, expected)
	}

	result = Translate("en-US", "minutes_ago", 10)
	if result != "10 minutes ago" {
		t.Errorf("Translate with args = %s, want '10 minutes ago'", result)
	}
}

func TestTranslateWithMultipleArgs(t *testing.T) {
	setupTranslations()

	result := Translate("zh-CN", "items_count", 3)
	expected := "3个项目"
	if result != expected {
		t.Errorf("Translate with multiple args = %s, want %s", result, expected)
	}

	result = Translate("en-US", "items_count", 7)
	if result != "7 items" {
		t.Errorf("Translate with multiple args = %s, want '7 items'", result)
	}
}

func TestIsRTL(t *testing.T) {
	setupTranslations()

	tests := []struct {
		lang     string
		expected bool
	}{
		{"zh-CN", false},
		{"en-US", false},
		{"ko-KR", false},
		{"ar-SA", true},
		{"fa-IR", true},
		{"he-IL", true},
		{"ur-PK", true},
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			result := IsRTL(tt.lang)
			if result != tt.expected {
				t.Errorf("IsRTL(%s) = %v, want %v", tt.lang, result, tt.expected)
			}
		})
	}
}

func TestGetTextDirection(t *testing.T) {
	setupTranslations()

	tests := []struct {
		lang     string
		expected string
	}{
		{"zh-CN", "ltr"},
		{"en-US", "ltr"},
		{"ar-SA", "rtl"},
		{"fa-IR", "rtl"},
		{"he-IL", "rtl"},
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			result := GetTextDirection(tt.lang)
			if result != tt.expected {
				t.Errorf("GetTextDirection(%s) = %s, want %s", tt.lang, result, tt.expected)
			}
		})
	}
}

func TestGetSupportedLangs(t *testing.T) {
	setupTranslations()

	langs := GetSupportedLangs()

	expectedLangs := []string{
		"zh-CN", "en-US", "ja-JP", "ko-KR", "fr-FR", "de-DE", "es-ES",
		"pt-BR", "it-IT", "ru-RU", "ar-SA", "fa-IR", "he-IL", "ur-PK",
		"hi-IN", "vi-VN", "th-TH", "id-ID", "tr-TR",
	}

	if len(langs) != len(expectedLangs) {
		t.Errorf("GetSupportedLangs() returned %d languages, want %d", len(langs), len(expectedLangs))
	}

	for _, expected := range expectedLangs {
		found := false
		for _, lang := range langs {
			if lang == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected language %s not found in supported languages", expected)
		}
	}
}

func TestGetLangInfo(t *testing.T) {
	setupTranslations()

	tests := []struct {
		lang        string
		expectedRTL bool
		currency    string
	}{
		{"zh-CN", false, "CNY"},
		{"en-US", false, "USD"},
		{"ko-KR", false, "KRW"},
		{"pt-BR", false, "BRL"},
		{"ru-RU", false, "RUB"},
		{"ar-SA", true, "SAR"},
		{"fa-IR", true, "IRR"},
		{"he-IL", true, "ILS"},
		{"ur-PK", true, "PKR"},
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			info := GetLangInfo(tt.lang)
			if info.IsRTL != tt.expectedRTL {
				t.Errorf("GetLangInfo(%s).IsRTL = %v, want %v", tt.lang, info.IsRTL, tt.expectedRTL)
			}
			if info.Currency != tt.currency {
				t.Errorf("GetLangInfo(%s).Currency = %s, want %s", tt.lang, info.Currency, tt.currency)
			}
		})
	}
}

func TestGetAllLangInfos(t *testing.T) {
	setupTranslations()

	infos := GetAllLangInfos()
	if len(infos) == 0 {
		t.Error("GetAllLangInfos() returned empty slice")
	}

	for _, info := range infos {
		if info.Code == "" {
			t.Error("LangInfo has empty Code")
		}
		if info.NativeName == "" {
			t.Error("LangInfo has empty NativeName")
		}
		if info.Direction != "ltr" && info.Direction != "rtl" {
			t.Errorf("LangInfo %s has invalid Direction: %s", info.Code, info.Direction)
		}
	}
}

func TestFormatDateTime(t *testing.T) {
	setupTranslations()

	testTime := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)

	tests := []struct {
		lang      string
		format    string
		expected  string
	}{
		{"zh-CN", "medium", "2024年1月15日"},
		{"en-US", "medium", "January 15, 2024"},
		{"ja-JP", "medium", "2024年1月15日"},
		{"ko-KR", "medium", "2024년 1월 15일"},
		{"pt-BR", "medium", "15 de janeiro de 2024"},
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			result := FormatDateTime(testTime, tt.lang, tt.format)
			if result != tt.expected {
				t.Errorf("FormatDateTime(%s, %s) = %s, want %s", tt.lang, tt.format, result, tt.expected)
			}
		})
	}
}

func TestFormatNumber(t *testing.T) {
	setupTranslations()

	tests := []struct {
		lang     string
		value    float64
		expected string
	}{
		{"zh-CN", 1234567.89, "1,234,567.89"},
		{"en-US", 1234567.89, "1,234,567.89"},
		{"fr-FR", 1234567.89, "1 234 567,89"},
		{"de-DE", 1234567.89, "1.234.567,89"},
		{"ru-RU", 1234567.89, "1 234 567,89"},
		{"ja-JP", 1234567.0, "1,234,567"},
		{"ko-KR", 1234567.0, "1,234,567"},
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			result := FormatNumber(tt.value, tt.lang)
			if result != tt.expected {
				t.Errorf("FormatNumber(%f, %s) = %s, want %s", tt.value, tt.lang, result, tt.expected)
			}
		})
	}
}

func TestFormatCurrency(t *testing.T) {
	setupTranslations()

	tests := []struct {
		lang     string
		value    float64
		expected string
	}{
		{"zh-CN", 1234.56, "¥1,234.56"},
		{"en-US", 1234.56, "$1,234.56"},
		{"ja-JP", 1234.0, "¥1,234"},
		{"ko-KR", 1234.0, "₩1,234"},
		{"fr-FR", 1234.56, "1 234,56 €"},
		{"de-DE", 1234.56, "1.234,56 €"},
		{"ru-RU", 1234.56, "1 234,56 ₽"},
		{"pt-BR", 1234.56, "R$1.234,56"},
		{"ar-SA", 1234.56, "1,234.56 ر.س"},
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			result := FormatCurrency(tt.value, tt.lang)
			if result != tt.expected {
				t.Errorf("FormatCurrency(%f, %s) = %s, want %s", tt.value, tt.lang, result, tt.expected)
			}
		})
	}
}

func TestFormatPercent(t *testing.T) {
	setupTranslations()

	tests := []struct {
		lang     string
		value    float64
		expected string
	}{
		{"zh-CN", 0.1234, "12,34%"},
		{"en-US", 0.1234, "12.34%"},
		{"fr-FR", 0.1234, "12,34%"},
		{"de-DE", 0.1234, "12,34%"},
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			result := FormatPercent(tt.value, tt.lang)
			if result != tt.expected {
				t.Errorf("FormatPercent(%f, %s) = %s, want %s", tt.value, tt.lang, result, tt.expected)
			}
		})
	}
}

func TestFormatRelativeTime(t *testing.T) {
	setupTranslations()

	now := time.Now()

	tests := []struct {
		lang     string
		duration time.Duration
	}{
		{"zh-CN", 30 * time.Second},
		{"en-US", 30 * time.Second},
		{"ko-KR", 30 * time.Second},
		{"pt-BR", 30 * time.Second},
		{"ru-RU", 30 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			target := now.Add(-tt.duration)
			result := FormatRelativeTime(target, tt.lang)
			if len(result) == 0 {
				t.Errorf("FormatRelativeTime returned empty string for %s", tt.lang)
			}
		})
	}
}

func TestT(t *testing.T) {
	setupTranslations()

	result := T("zh-CN", "title")
	if result == "" {
		t.Error("T function returned empty string")
	}

	result = T("xx-XX", "title")
	if result == "title" {
		t.Error("T function should return key for unsupported language")
	}
}

func TestSetDefaultLang(t *testing.T) {
	setupTranslations()

	original := GetDefaultLang()

	SetDefaultLang("en-US")
	if GetDefaultLang() != "en-US" {
		t.Error("SetDefaultLang did not update default language")
	}

	SetDefaultLang("xx-XX")
	if GetDefaultLang() != "en-US" {
		t.Error("SetDefaultLang should not accept unsupported languages")
	}

	SetDefaultLang(original)
}

func TestTranslateFallback(t *testing.T) {
	setupTranslations()

	result := Translate("unsupported-lang", "title")
	if result == "" || result == "title" {
		t.Error("Translate should fallback to default language")
	}
}

func TestRTLLanguageSupport(t *testing.T) {
	setupTranslations()

	rtlLangs := []string{"ar-SA", "fa-IR", "he-IL", "ur-PK"}

	for _, lang := range rtlLangs {
		info := GetLangInfo(lang)
		if !info.IsRTL {
			t.Errorf("Language %s should be RTL", lang)
		}
		if info.Direction != "rtl" {
			t.Errorf("Language %s direction should be 'rtl'", lang)
		}
	}
}

func TestNewlyAddedLanguages(t *testing.T) {
	setupTranslations()

	newLangs := []string{"ko-KR", "pt-BR", "ru-RU", "pl-PL", "nl-NL", "sv-SE", "da-DK", "nb-NO", "fi-FI", "cs-CZ", "hu-HU", "ro-RO", "bg-BG"}

	for _, lang := range newLangs {
		if !IsSupported(lang) {
			t.Errorf("Language %s should be supported", lang)
		}

		info := GetLangInfo(lang)
		if info.Code != lang {
			t.Errorf("LangInfo code mismatch for %s", lang)
		}

		trans := GetAllTranslations(lang)
		if len(trans) == 0 {
			t.Errorf("Language %s has no translations", lang)
		}
	}
}

func TestAllSupportedLanguages(t *testing.T) {
	setupTranslations()

	langs := GetSupportedLangs()

	expectedCount := 29
	if len(langs) != expectedCount {
		t.Errorf("Expected %d supported languages, got %d", expectedCount, len(langs))
	}

	essentialKeys := []string{"title", "home", "login", "dashboard", "verification_success"}

	for _, lang := range langs {
		t.Run(lang, func(t *testing.T) {
			trans := GetAllTranslations(lang)
			if len(trans) == 0 {
				t.Errorf("Language %s has no translations", lang)
			}

			for _, key := range essentialKeys {
				if _, ok := trans[key]; !ok {
					t.Errorf("Language %s missing essential key: %s", lang, key)
				}
			}
		})
	}
}

func TestTranslationCompleteness(t *testing.T) {
	setupTranslations()

	zhCNKeys := GetAllTranslations("zh-CN")
	enUSKeys := GetAllTranslations("en-US")

	requiredKeys := []string{
		"title", "admin_title", "home", "features", "demo", "pricing", "contact",
		"admin", "dashboard", "stats", "advanced_analytics", "applications", "logs",
		"risk_rules", "blacklist", "monitoring", "ab_testing", "real_time_screen",
		"logout", "login", "username", "password", "submit", "cancel", "save",
		"delete", "edit", "add", "search", "filter", "success", "error", "warning",
		"info", "loading", "confirm", "welcome", "slider_captcha", "click_captcha",
		"image_captcha", "drag_slider_to_complete", "click_in_order", "enter_characters",
		"verification_success", "verification_failed", "try_again", "open_source_free",
	}

	for _, key := range requiredKeys {
		if _, ok := zhCNKeys[key]; !ok {
			t.Errorf("zh-CN missing key: %s", key)
		}
		if _, ok := enUSKeys[key]; !ok {
			t.Errorf("en-US missing key: %s", key)
		}
	}
}

func TestClickCaptchaTranslations(t *testing.T) {
	setupTranslations()

	langs := []string{"zh-CN", "en-US", "ja-JP", "ko-KR", "fr-FR", "de-DE", "es-ES"}

	for _, lang := range langs {
		t.Run(lang, func(t *testing.T) {
			trans := GetAllTranslations(lang)

			hints := []string{
				"click_captcha_hints.number",
				"click_captcha_hints.letter",
				"click_captcha_hints.chinese",
				"click_captcha_hints.icon",
				"click_captcha_hints.mixed",
				"click_captcha_hints.order_hint",
			}

			messages := []string{
				"click_captcha_messages.success",
				"click_captcha_messages.failed",
				"click_captcha_messages.wrong_order",
				"click_captcha_messages.wrong_position",
				"click_captcha_messages.timeout",
				"click_captcha_messages.invalid_session",
				"click_captcha_messages.try_again",
				"click_captcha_messages.too_fast",
				"click_captcha_messages.too_slow",
				"click_captcha_messages.abnormal_behavior",
			}

			for _, key := range hints {
				if _, ok := trans[key]; !ok {
					t.Errorf("Language %s missing hint key: %s", lang, key)
				}
			}

			for _, key := range messages {
				if _, ok := trans[key]; !ok {
					t.Errorf("Language %s missing message key: %s", lang, key)
				}
			}
		})
	}
}

func TestCurrencyFormatting(t *testing.T) {
	setupTranslations()

	currencyTests := []struct {
		lang     string
		currency string
	}{
		{"zh-CN", "CNY"},
		{"en-US", "USD"},
		{"ja-JP", "JPY"},
		{"ko-KR", "KRW"},
		{"fr-FR", "EUR"},
		{"de-DE", "EUR"},
		{"es-ES", "EUR"},
		{"pt-BR", "BRL"},
		{"it-IT", "EUR"},
		{"ru-RU", "RUB"},
		{"ar-SA", "SAR"},
		{"fa-IR", "IRR"},
		{"he-IL", "ILS"},
		{"ur-PK", "PKR"},
		{"hi-IN", "INR"},
		{"vi-VN", "VND"},
		{"th-TH", "THB"},
		{"id-ID", "IDR"},
		{"tr-TR", "TRY"},
		{"pl-PL", "PLN"},
		{"nl-NL", "EUR"},
		{"sv-SE", "SEK"},
		{"da-DK", "DKK"},
		{"nb-NO", "NOK"},
		{"fi-FI", "EUR"},
		{"cs-CZ", "CZK"},
		{"hu-HU", "HUF"},
		{"ro-RO", "RON"},
		{"bg-BG", "BGN"},
	}

	for _, tt := range currencyTests {
		t.Run(tt.lang, func(t *testing.T) {
			info := GetLangInfo(tt.lang)
			if info.Currency != tt.currency {
				t.Errorf("Language %s expected currency %s, got %s", tt.lang, tt.currency, info.Currency)
			}
		})
	}
}

func TestTimezoneSupport(t *testing.T) {
	setupTranslations()

	timezoneTests := []struct {
		lang     string
		timezone string
	}{
		{"zh-CN", "Asia/Shanghai"},
		{"en-US", "America/New_York"},
		{"ja-JP", "Asia/Tokyo"},
		{"ko-KR", "Asia/Seoul"},
		{"fr-FR", "Europe/Paris"},
		{"de-DE", "Europe/Berlin"},
		{"es-ES", "Europe/Madrid"},
		{"pt-BR", "America/Sao_Paulo"},
		{"it-IT", "Europe/Rome"},
		{"ru-RU", "Europe/Moscow"},
		{"ar-SA", "Asia/Riyadh"},
		{"fa-IR", "Asia/Tehran"},
		{"he-IL", "Asia/Jerusalem"},
		{"ur-PK", "Asia/Karachi"},
		{"hi-IN", "Asia/Kolkata"},
		{"vi-VN", "Asia/Ho_Chi_Minh"},
		{"th-TH", "Asia/Bangkok"},
		{"id-ID", "Asia/Jakarta"},
		{"tr-TR", "Europe/Istanbul"},
		{"pl-PL", "Europe/Warsaw"},
		{"nl-NL", "Europe/Amsterdam"},
		{"sv-SE", "Europe/Stockholm"},
		{"da-DK", "Europe/Copenhagen"},
		{"nb-NO", "Europe/Oslo"},
		{"fi-FI", "Europe/Helsinki"},
		{"cs-CZ", "Europe/Prague"},
		{"hu-HU", "Europe/Budapest"},
		{"ro-RO", "Europe/Bucharest"},
		{"bg-BG", "Europe/Sofia"},
	}

	for _, tt := range timezoneTests {
		t.Run(tt.lang, func(t *testing.T) {
			info := GetLangInfo(tt.lang)
			if info.Timezone != tt.timezone {
				t.Errorf("Language %s expected timezone %s, got %s", tt.lang, tt.timezone, info.Timezone)
			}
		})
	}
}
