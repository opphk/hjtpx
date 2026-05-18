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
	translations   map[string]map[string]string
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

	translations = make(map[string]map[string]string)

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

			var langTrans map[string]string
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

	value, ok := trans[key]
	if !ok {
		if defaultTrans, ok := translations[defaultLang]; ok {
			if defaultValue, ok := defaultTrans[key]; ok {
				value = defaultValue
			} else {
				value = key
			}
		} else {
			value = key
		}
	}

	if len(args) > 0 {
		return fmt.Sprintf(value, args...)
	}
	return value
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
	for k, v := range trans {
		result[k] = v
	}
	return result
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
