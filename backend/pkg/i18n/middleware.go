package i18n

import (
	"github.com/gin-gonic/gin"
)

const (
	LangKey     = "i18n_lang"
	TimezoneKey = "i18n_timezone"
)

func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := GetLangFromRequest(c)
		timezone := GetTimezoneFromRequest(c)

		c.Set(LangKey, lang)
		c.Set(TimezoneKey, timezone)

		c.Next()
	}
}

func GetLangFromRequest(c *gin.Context) string {
	lang := c.Query("lang")
	if lang != "" && IsSupported(lang) {
		return lang
	}

	lang = c.GetHeader("Accept-Language")
	if lang != "" {
		if len(lang) > 5 {
			lang = lang[:5]
		}
		if IsSupported(lang) {
			return lang
		}
	}

	cookie, err := c.Cookie("lang")
	if err == nil && cookie != "" && IsSupported(cookie) {
		return cookie
	}

	return GetDefaultLang()
}

func GetTimezoneFromRequest(c *gin.Context) string {
	tz := c.Query("timezone")
	if tz != "" && IsSupportedTimezone(tz) {
		return tz
	}

	cookie, err := c.Cookie("timezone")
	if err == nil && cookie != "" && IsSupportedTimezone(cookie) {
		return cookie
	}

	return GetDefaultTimezone()
}

func GetLang(c *gin.Context) string {
	if v, exists := c.Get(LangKey); exists {
		if lang, ok := v.(string); ok {
			return lang
		}
	}
	return GetDefaultLang()
}

func GetTimezone(c *gin.Context) string {
	if v, exists := c.Get(TimezoneKey); exists {
		if tz, ok := v.(string); ok {
			return tz
		}
	}
	return GetDefaultTimezone()
}

func SetLangCookie(c *gin.Context, lang string) {
	if IsSupported(lang) {
		c.SetCookie("lang", lang, 86400*365, "/", "", false, false)
	}
}

func SetTimezoneCookie(c *gin.Context, timezone string) {
	if IsSupportedTimezone(timezone) {
		c.SetCookie("timezone", timezone, 86400*365, "/", "", false, false)
	}
}
