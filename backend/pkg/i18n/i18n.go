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
	translations     map[string]map[string]string
	mu               sync.RWMutex
	defaultLang      = "zh-CN"
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
	}
)

type LocaleConfig struct {
	DefaultLang   string   `json:"default_lang"`
	SupportedLangs []string `json:"supported_langs"`
	TranslationsDir string `json:"translations_dir"`
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
		if defaultTrans, ok := translations[defaultLang]
		if ok {
			if defaultValue, ok := defaultTrans[key]
			if ok {
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
