package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	translations   map[string]map[string]interface{}
	mu             sync.RWMutex
	defaultLang    = "zh-CN"
	supportedLangs = []string{
		"zh-CN",
		"en-US",
		"ja-JP",
		"ko-KR",
		"fr-FR",
		"de-DE",
		"es-ES",
		"pt-BR",
		"it-IT",
		"ru-RU",
		"ar-SA",
		"fa-IR",
		"he-IL",
		"ur-PK",
		"hi-IN",
		"vi-VN",
		"th-TH",
		"id-ID",
		"tr-TR",
		"pl-PL",
		"nl-NL",
		"sv-SE",
		"da-DK",
		"nb-NO",
		"fi-FI",
		"cs-CZ",
		"hu-HU",
		"ro-RO",
		"bg-BG",
	}
)

type LocaleConfig struct {
	DefaultLang     string   `json:"default_lang"`
	SupportedLangs  []string `json:"supported_langs"`
	TranslationsDir string   `json:"translations_dir"`
}

type LangInfo struct {
	Code         string `json:"code"`
	Name         string `json:"name"`
	NativeName   string `json:"native_name"`
	Direction    string `json:"direction"`
	IsRTL        bool   `json:"is_rtl"`
	DateFormat   string `json:"date_format"`
	NumberFormat string `json:"number_format"`
	Currency     string `json:"currency"`
	Timezone     string `json:"timezone"`
}

var langInfos = map[string]LangInfo{
	"zh-CN": {
		Code:         "zh-CN",
		Name:         "Chinese (Simplified)",
		NativeName:   "简体中文",
		Direction:    "ltr",
		IsRTL:        false,
		DateFormat:   "2006-01-02",
		NumberFormat: "en-US",
		Currency:     "CNY",
		Timezone:     "Asia/Shanghai",
	},
	"en-US": {
		Code:         "en-US",
		Name:         "English",
		NativeName:   "English",
		Direction:    "ltr",
		IsRTL:        false,
		DateFormat:   "01/02/2006",
		NumberFormat: "en-US",
		Currency:     "USD",
		Timezone:     "America/New_York",
	},
	"ja-JP": {
		Code:         "ja-JP",
		Name:         "Japanese",
		NativeName:   "日本語",
		Direction:    "ltr",
		IsRTL:        false,
		DateFormat:   "2006/01/02",
		NumberFormat: "ja-JP",
		Currency:     "JPY",
		Timezone:     "Asia/Tokyo",
	},
	"ko-KR": {
		Code:         "ko-KR",
		Name:         "Korean",
		NativeName:   "한국어",
		Direction:    "ltr",
		IsRTL:        false,
		DateFormat:   "2006. 01. 02",
		NumberFormat: "ko-KR",
		Currency:     "KRW",
		Timezone:     "Asia/Seoul",
	},
	"fr-FR": {
		Code:         "fr-FR",
		Name:         "French",
		NativeName:   "Français",
		Direction:    "ltr",
		IsRTL:        false,
		DateFormat:   "02/01/2006",
		NumberFormat: "fr-FR",
		Currency:     "EUR",
		Timezone:     "Europe/Paris",
	},
	"de-DE": {
		Code:         "de-DE",
		Name:         "German",
		NativeName:   "Deutsch",
		Direction:    "ltr",
		IsRTL:        false,
		DateFormat:   "02.01.2006",
		NumberFormat: "de-DE",
		Currency:     "EUR",
		Timezone:     "Europe/Berlin",
	},
	"es-ES": {
		Code:         "es-ES",
		Name:         "Spanish",
		NativeName:   "Español",
		Direction:    "ltr",
		IsRTL:        false,
		DateFormat:   "02/01/2006",
		NumberFormat: "es-ES",
		Currency:     "EUR",
		Timezone:     "Europe/Madrid",
	},
	"pt-BR": {
		Code:         "pt-BR",
		Name:         "Portuguese (Brazil)",
		NativeName:   "Português",
		Direction:    "ltr",
		IsRTL:        false,
		DateFormat:   "02/01/2006",
		NumberFormat: "pt-BR",
		Currency:     "BRL",
		Timezone:     "America/Sao_Paulo",
	},
	"it-IT": {
		Code:         "it-IT",
		Name:         "Italian",
		NativeName:   "Italiano",
		Direction:    "ltr",
		IsRTL:        false,
		DateFormat:   "02/01/2006",
		NumberFormat: "it-IT",
		Currency:     "EUR",
		Timezone:     "Europe/Rome",
	},
	"ru-RU": {
		Code:         "ru-RU",
		Name:         "Russian",
		NativeName:   "Русский",
		Direction:    "ltr",
		IsRTL:        false,
		DateFormat:   "02.01.2006",
		NumberFormat: "ru-RU",
		Currency:     "RUB",
		Timezone:     "Europe/Moscow",
	},
	"ar-SA": {
		Code:         "ar-SA",
		Name:         "Arabic",
		NativeName:   "العربية",
		Direction:    "rtl",
		IsRTL:        true,
		DateFormat:   "02/01/2006",
		NumberFormat: "ar-SA",
		Currency:     "SAR",
		Timezone:     "Asia/Riyadh",
	},
	"fa-IR": {
		Code:         "fa-IR",
		Name:         "Persian",
		NativeName:   "فارسی",
		Direction:    "rtl",
		IsRTL:        true,
		DateFormat:   "02/01/2006",
		NumberFormat: "fa-IR",
		Currency:     "IRR",
		Timezone:     "Asia/Tehran",
	},
	"he-IL": {
		Code:         "he-IL",
		Name:         "Hebrew",
		NativeName:   "עברית",
		Direction:    "rtl",
		IsRTL:        true,
		DateFormat:   "02/01/2006",
		NumberFormat: "he-IL",
		Currency:     "ILS",
		Timezone:     "Asia/Jerusalem",
	},
	"ur-PK": {
		Code:         "ur-PK",
		Name:         "Urdu",
		NativeName:   "اردو",
		Direction:    "rtl",
		IsRTL:        true,
		DateFormat:   "02/01/2006",
		NumberFormat: "ur-PK",
		Currency:     "PKR",
		Timezone:     "Asia/Karachi",
	},
	"hi-IN": {
		Code:         "hi-IN",
		Name:         "Hindi",
		NativeName:   "हिन्दी",
		Direction:    "ltr",
		IsRTL:        false,
		DateFormat:   "02/01/2006",
		NumberFormat: "hi-IN",
		Currency:     "INR",
		Timezone:     "Asia/Kolkata",
	},
	"vi-VN": {
		Code:         "vi-VN",
		Name:         "Vietnamese",
		NativeName:   "Tiếng Việt",
		Direction:    "ltr",
		IsRTL:        false,
		DateFormat:   "02/01/2006",
		NumberFormat: "vi-VN",
		Currency:     "VND",
		Timezone:     "Asia/Ho_Chi_Minh",
	},
	"th-TH": {
		Code:         "th-TH",
		Name:         "Thai",
		NativeName:   "ไทย",
		Direction:    "ltr",
		IsRTL:        false,
		DateFormat:   "02/01/2006",
		NumberFormat: "th-TH",
		Currency:     "THB",
		Timezone:     "Asia/Bangkok",
	},
	"id-ID": {
		Code:         "id-ID",
		Name:         "Indonesian",
		NativeName:   "Bahasa Indonesia",
		Direction:    "ltr",
		IsRTL:        false,
		DateFormat:   "02/01/2006",
		NumberFormat: "id-ID",
		Currency:     "IDR",
		Timezone:     "Asia/Jakarta",
	},
	"tr-TR": {
		Code:         "tr-TR",
		Name:         "Turkish",
		NativeName:   "Türkçe",
		Direction:    "ltr",
		IsRTL:        false,
		DateFormat:   "02.01.2006",
		NumberFormat: "tr-TR",
		Currency:     "TRY",
		Timezone:     "Europe/Istanbul",
	},
	"pl-PL": {
		Code:         "pl-PL",
		Name:         "Polish",
		NativeName:   "Polski",
		Direction:    "ltr",
		IsRTL:        false,
		DateFormat:   "02.01.2006",
		NumberFormat: "pl-PL",
		Currency:     "PLN",
		Timezone:     "Europe/Warsaw",
	},
	"nl-NL": {
		Code:         "nl-NL",
		Name:         "Dutch",
		NativeName:   "Nederlands",
		Direction:    "ltr",
		IsRTL:        false,
		DateFormat:   "02-01-2006",
		NumberFormat: "nl-NL",
		Currency:     "EUR",
		Timezone:     "Europe/Amsterdam",
	},
	"sv-SE": {
		Code:         "sv-SE",
		Name:         "Swedish",
		NativeName:   "Svenska",
		Direction:    "ltr",
		IsRTL:        false,
		DateFormat:   "2006-01-02",
		NumberFormat: "sv-SE",
		Currency:     "SEK",
		Timezone:     "Europe/Stockholm",
	},
	"da-DK": {
		Code:         "da-DK",
		Name:         "Danish",
		NativeName:   "Dansk",
		Direction:    "ltr",
		IsRTL:        false,
		DateFormat:   "02-01-2006",
		NumberFormat: "da-DK",
		Currency:     "DKK",
		Timezone:     "Europe/Copenhagen",
	},
	"nb-NO": {
		Code:         "nb-NO",
		Name:         "Norwegian",
		NativeName:   "Norsk",
		Direction:    "ltr",
		IsRTL:        false,
		DateFormat:   "02.01.2006",
		NumberFormat: "nb-NO",
		Currency:     "NOK",
		Timezone:     "Europe/Oslo",
	},
	"fi-FI": {
		Code:         "fi-FI",
		Name:         "Finnish",
		NativeName:   "Suomi",
		Direction:    "ltr",
		IsRTL:        false,
		DateFormat:   "02.01.2006",
		NumberFormat: "fi-FI",
		Currency:     "EUR",
		Timezone:     "Europe/Helsinki",
	},
	"cs-CZ": {
		Code:         "cs-CZ",
		Name:         "Czech",
		NativeName:   "Čeština",
		Direction:    "ltr",
		IsRTL:        false,
		DateFormat:   "02.01.2006",
		NumberFormat: "cs-CZ",
		Currency:     "CZK",
		Timezone:     "Europe/Prague",
	},
	"hu-HU": {
		Code:         "hu-HU",
		Name:         "Hungarian",
		NativeName:   "Magyar",
		Direction:    "ltr",
		IsRTL:        false,
		DateFormat:   "2006. 01. 02.",
		NumberFormat: "hu-HU",
		Currency:     "HUF",
		Timezone:     "Europe/Budapest",
	},
	"ro-RO": {
		Code:         "ro-RO",
		Name:         "Romanian",
		NativeName:   "Română",
		Direction:    "ltr",
		IsRTL:        false,
		DateFormat:   "02.01.2006",
		NumberFormat: "ro-RO",
		Currency:     "RON",
		Timezone:     "Europe/Bucharest",
	},
	"bg-BG": {
		Code:         "bg-BG",
		Name:         "Bulgarian",
		NativeName:   "Български",
		Direction:    "ltr",
		IsRTL:        false,
		DateFormat:   "02.01.2006",
		NumberFormat: "bg-BG",
		Currency:     "BGN",
		Timezone:     "Europe/Sofia",
	},
}

func Init(config LocaleConfig) error {
	mu.Lock()
	defer mu.Unlock()

	if config.DefaultLang != "" {
		defaultLang = config.DefaultLang
	}
	if len(config.SupportedLangs) > 0 {
		supportedLangs = config.SupportedLangs
	}

	translations = make(map[string]map[string]interface{})

	dir := config.TranslationsDir
	if dir == "" {
		dir = "translations"
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read translations dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			lang := strings.TrimSuffix(entry.Name(), ".json")
			data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
			if err != nil {
				return fmt.Errorf("failed to read translation file %s: %w", entry.Name(), err)
			}

			var langTrans map[string]interface{}
			if err := json.Unmarshal(data, &langTrans); err != nil {
				return fmt.Errorf("failed to parse translation file %s: %w", entry.Name(), err)
			}

			translations[lang] = langTrans
		}
	}

	return nil
}

func SetDefaultLang(lang string) {
	mu.Lock()
	defer mu.Unlock()
	if IsSupported(lang) {
		defaultLang = lang
	}
}

func GetDefaultLang() string {
	mu.RLock()
	defer mu.RUnlock()
	return defaultLang
}

func IsSupported(lang string) bool {
	mu.RLock()
	defer mu.RUnlock()
	for _, l := range supportedLangs {
		if l == lang {
			return true
		}
	}
	return false
}

func GetSupportedLangs() []string {
	mu.RLock()
	defer mu.RUnlock()
	return append([]string{}, supportedLangs...)
}

func Translate(lang, key string, args ...interface{}) string {
	mu.RLock()
	defer mu.RUnlock()

	targetLang := lang
	if !IsSupported(targetLang) {
		targetLang = defaultLang
	}

	trans, ok := translations[targetLang]
	if !ok {
		return key
	}

	value := getNestedValue(trans, key)
	if value == key {
		if defaultTrans, ok := translations[defaultLang]; ok {
			if defaultValue := getNestedValue(defaultTrans, key); defaultValue != key {
				value = defaultValue
			}
		}
	}

	if strValue, ok := value.(string); ok {
		if len(args) > 0 {
			return fmt.Sprintf(strValue, args...)
		}
		return strValue
	}
	return key
}

func getNestedValue(data map[string]interface{}, key string) interface{} {
	parts := strings.Split(key, ".")
	current := data
	for i, part := range parts {
		if i == len(parts)-1 {
			if val, ok := current[part]; ok {
				return val
			}
			return key
		}
		if val, ok := current[part]; ok {
			if nextMap, ok := val.(map[string]interface{}); ok {
				current = nextMap
			} else {
				return key
			}
		} else {
			return key
		}
	}
	return key
}

func T(lang, key string, args ...interface{}) string {
	return Translate(lang, key, args...)
}

func GetAllTranslations(lang string) map[string]string {
	mu.RLock()
	defer mu.RUnlock()

	targetLang := lang
	if !IsSupported(targetLang) {
		targetLang = defaultLang
	}

	trans, ok := translations[targetLang]
	if !ok {
		return map[string]string{}
	}

	result := make(map[string]string)
	flattenMap(trans, "", result)
	return result
}

func flattenMap(data map[string]interface{}, prefix string, result map[string]string) {
	for k, v := range data {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		if strVal, ok := v.(string); ok {
			result[key] = strVal
		} else if nestedMap, ok := v.(map[string]interface{}); ok {
			flattenMap(nestedMap, key, result)
		}
	}
}

func GetLangInfo(lang string) LangInfo {
	targetLang := lang
	if !IsSupported(targetLang) {
		targetLang = defaultLang
	}

	if info, ok := langInfos[targetLang]; ok {
		return info
	}
	return langInfos[defaultLang]
}

func GetAllLangInfos() []LangInfo {
	mu.RLock()
	defer mu.RUnlock()

	result := make([]LangInfo, 0, len(supportedLangs))
	for _, lang := range supportedLangs {
		if info, ok := langInfos[lang]; ok {
			result = append(result, info)
		}
	}
	return result
}

func IsRTL(lang string) bool {
	info := GetLangInfo(lang)
	return info.IsRTL
}

func GetTextDirection(lang string) string {
	info := GetLangInfo(lang)
	return info.Direction
}

func GetDateFormat(lang string) string {
	info := GetLangInfo(lang)
	return info.DateFormat
}

func GetCurrency(lang string) string {
	info := GetLangInfo(lang)
	return info.Currency
}

func GetNumberFormat(lang string) string {
	info := GetLangInfo(lang)
	return info.NumberFormat
}

type LayoutDirection int

const (
	LayoutLeftToRight LayoutDirection = iota
	LayoutRightToLeft
)

func GetLayoutDirection(lang string) LayoutDirection {
	if IsRTL(lang) {
		return LayoutRightToLeft
	}
	return LayoutLeftToRight
}

func GetTextAlignment(lang string) string {
	if IsRTL(lang) {
		return "right"
	}
	return "left"
}

func GetFlexDirection(lang string) string {
	if IsRTL(lang) {
		return "row-reverse"
	}
	return "row"
}

func GetRTLSupport(lang string) bool {
	return IsRTL(lang)
}

func GetLogicalProperty(property string, lang string) string {
	if !IsRTL(lang) {
		return property
	}

	logicalProps := map[string]map[string]string{
		"margin-left": {
			"margin-right": "margin-start",
		},
		"margin-right": {
			"margin-left": "margin-end",
		},
		"padding-left": {
			"padding-right": "padding-start",
		},
		"padding-right": {
			"padding-left": "padding-end",
		},
		"left": {
			"right": "inset-start",
		},
		"right": {
			"left": "inset-end",
		},
		"text-align-left": {
			"text-align-right": "text-align-start",
		},
		"text-align-right": {
			"text-align-left": "text-align-end",
		},
	}

	if props, ok := logicalProps[property]; ok {
		if newProp, ok := props[property]; ok {
			return newProp
		}
	}

	return property
}

func ShouldFlipIcon(iconType string) bool {
	flipIcons := map[string]bool{
		"arrow-left":      true,
		"arrow-right":     true,
		"chevron-left":    true,
		"chevron-right":   true,
		"caret-left":      true,
		"caret-right":     true,
		"angle-left":      true,
		"angle-right":     true,
		"long-arrow-left": true,
		"long-arrow-right": true,
		"back":            true,
		"forward":         true,
		"previous":        true,
		"next":            true,
	}

	return flipIcons[iconType]
}

func GetFlipTransform(lang string, iconType string) string {
	if !ShouldFlipIcon(iconType) {
		return ""
	}

	if IsRTL(lang) {
		return "scaleX(-1)"
	}

	return ""
}

func GetTextDirectionClass(lang string) string {
	if IsRTL(lang) {
		return "rtl"
	}
	return "ltr"
}

func GetDocumentDirection(lang string) string {
	if IsRTL(lang) {
		return "rtl"
	}
	return "ltr"
}

func GetTextAlignStyle(lang string) string {
	if IsRTL(lang) {
		return "text-align: right;"
	}
	return "text-align: left;"
}

func GetMarginStart(lang string, margin string) string {
	if IsRTL(lang) {
		return fmt.Sprintf("margin-right: %s;", margin)
	}
	return fmt.Sprintf("margin-left: %s;", margin)
}

func GetMarginEnd(lang string, margin string) string {
	if IsRTL(lang) {
		return fmt.Sprintf("margin-left: %s;", margin)
	}
	return fmt.Sprintf("margin-right: %s;", margin)
}

func GetPaddingStart(lang string, padding string) string {
	if IsRTL(lang) {
		return fmt.Sprintf("padding-right: %s;", padding)
	}
	return fmt.Sprintf("padding-left: %s;", padding)
}

func GetPaddingEnd(lang string, padding string) string {
	if IsRTL(lang) {
		return fmt.Sprintf("padding-left: %s;", padding)
	}
	return fmt.Sprintf("padding-right: %s;", padding)
}
