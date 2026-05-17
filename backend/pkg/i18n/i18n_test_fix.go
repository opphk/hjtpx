package i18n

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatDuration(t *testing.T) {
	testCases := []struct {
		duration time.Duration
		lang     string
	}{
		{time.Second * 30, "en-US"},
		{time.Minute * 5, "en-US"},
		{time.Hour * 2, "en-US"},
		{time.Hour*24 + time.Minute*5, "zh-CN"},
	}

	for _, tc := range testCases {
		t.Run(tc.lang, func(t *testing.T) {
			result := FormatDuration(tc.duration, tc.lang)
			if result == "" {
				t.Error("FormatDuration returned empty string")
			}
		})
	}
}

func TestLocaleConfig_Structure(t *testing.T) {
	config := LocaleConfig{
		DefaultLang:     "zh-CN",
		SupportedLangs:  []string{"zh-CN", "en-US"},
		TranslationsDir: "/tmp/translations",
	}

	assert.Equal(t, "zh-CN", config.DefaultLang)
	assert.Len(t, config.SupportedLangs, 2)
	assert.Equal(t, "/tmp/translations", config.TranslationsDir)
}
