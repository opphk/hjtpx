package i18n

import (
	"fmt"
	"time"
)

type DateTimeFormat struct {
	Full      string
	Long      string
	Medium    string
	Short     string
	TimeOnly  string
	DateOnly  string
	MonthDay  string
	YearMonth string
}

var dateFormats = map[string]DateTimeFormat{
	"zh-CN": {
		Full:      "2006年1月2日 15:04:05",
		Long:      "2006年1月2日 15:04",
		Medium:    "2006年1月2日",
		Short:     "2006/1/2",
		TimeOnly:  "15:04:05",
		DateOnly:  "2006-01-02",
		MonthDay:  "1月2日",
		YearMonth: "2006年1月",
	},
	"en-US": {
		Full:      "January 2, 2006 3:04:05 PM",
		Long:      "January 2, 2006 3:04 PM",
		Medium:    "January 2, 2006",
		Short:     "1/2/06",
		TimeOnly:  "3:04:05 PM",
		DateOnly:  "01/02/2006",
		MonthDay:  "January 2",
		YearMonth: "January 2006",
	},
	"ja-JP": {
		Full:      "2006年1月2日 15時04分05秒",
		Long:      "2006年1月2日 15時04分",
		Medium:    "2006年1月2日",
		Short:     "2006/1/2",
		TimeOnly:  "15:04:05",
		DateOnly:  "2006-01-02",
		MonthDay:  "1月2日",
		YearMonth: "2006年1月",
	},
	"ko-KR": {
		Full:      "2006년 1월 2일 15시 04분 05초",
		Long:      "2006년 1월 2일 15시 04분",
		Medium:    "2006년 1월 2일",
		Short:     "2006. 1. 2.",
		TimeOnly:  "15:04:05",
		DateOnly:  "2006-01-02",
		MonthDay:  "1월 2일",
		YearMonth: "2006년 1월",
	},
	"fr-FR": {
		Full:      "2 janvier 2006 15:04:05",
		Long:      "2 janvier 2006 15:04",
		Medium:    "2 janvier 2006",
		Short:     "02/01/2006",
		TimeOnly:  "15:04:05",
		DateOnly:  "02/01/2006",
		MonthDay:  "2 janvier",
		YearMonth: "janvier 2006",
	},
	"de-DE": {
		Full:      "2. Januar 2006 15:04:05",
		Long:      "2. Januar 2006 15:04",
		Medium:    "2. Januar 2006",
		Short:     "02.01.06",
		TimeOnly:  "15:04:05",
		DateOnly:  "02.01.2006",
		MonthDay:  "2. Januar",
		YearMonth: "Januar 2006",
	},
	"es-ES": {
		Full:      "2 de enero de 2006 15:04:05",
		Long:      "2 de enero de 2006 15:04",
		Medium:    "2 de enero de 2006",
		Short:     "02/01/06",
		TimeOnly:  "15:04:05",
		DateOnly:  "02/01/2006",
		MonthDay:  "2 de enero",
		YearMonth: "enero de 2006",
	},
	"pt-BR": {
		Full:      "2 de janeiro de 2006 15:04:05",
		Long:      "2 de janeiro de 2006 15:04",
		Medium:    "2 de janeiro de 2006",
		Short:     "02/01/06",
		TimeOnly:  "15:04:05",
		DateOnly:  "02/01/2006",
		MonthDay:  "2 de janeiro",
		YearMonth: "janeiro de 2006",
	},
	"it-IT": {
		Full:      "2 gennaio 2006 15:04:05",
		Long:      "2 gennaio 2006 15:04",
		Medium:    "2 gennaio 2006",
		Short:     "02/01/06",
		TimeOnly:  "15:04:05",
		DateOnly:  "02/01/2006",
		MonthDay:  "2 gennaio",
		YearMonth: "gennaio 2006",
	},
	"ru-RU": {
		Full:      "2 января 2006 г. 15:04:05",
		Long:      "2 января 2006 г. 15:04",
		Medium:    "2 января 2006 г.",
		Short:     "02.01.06",
		TimeOnly:  "15:04:05",
		DateOnly:  "02.01.2006",
		MonthDay:  "2 января",
		YearMonth: "январь 2006 г.",
	},
	"ar-SA": {
		Full:      "2 يناير 2006 15:04:05 م",
		Long:      "2 يناير 2006 15:04 م",
		Medium:    "2 يناير 2006",
		Short:     "02/01/06",
		TimeOnly:  "15:04:05 م",
		DateOnly:  "02/01/2006",
		MonthDay:  "2 يناير",
		YearMonth: "يناير 2006",
	},
	"fa-IR": {
		Full:      "2 ژانویه 2006 15:04:05",
		Long:      "2 ژانویه 2006 15:04",
		Medium:    "2 ژانویه 2006",
		Short:     "02/01/06",
		TimeOnly:  "15:04:05",
		DateOnly:  "02/01/2006",
		MonthDay:  "2 ژانویه",
		YearMonth: "ژانویه 2006",
	},
	"he-IL": {
		Full:      "2 בינואר 2006 15:04:05",
		Long:      "2 בינואר 2006 15:04",
		Medium:    "2 בינואר 2006",
		Short:     "02/01/06",
		TimeOnly:  "15:04:05",
		DateOnly:  "02/01/2006",
		MonthDay:  "2 בינואר",
		YearMonth: "ינואר 2006",
	},
	"ur-PK": {
		Full:      "2 جنوری 2006 15:04:05",
		Long:      "2 جنوری 2006 15:04",
		Medium:    "2 جنوری 2006",
		Short:     "02/01/06",
		TimeOnly:  "15:04:05",
		DateOnly:  "02/01/2006",
		MonthDay:  "2 جنوری",
		YearMonth: "جنوری 2006",
	},
	"hi-IN": {
		Full:      "2 जनवरी 2006 3:04:05 अपराह्न",
		Long:      "2 जनवरी 2006 3:04 अपराह्न",
		Medium:    "2 जनवरी 2006",
		Short:     "2/1/06",
		TimeOnly:  "3:04:05 अपराह्न",
		DateOnly:  "02/01/2006",
		MonthDay:  "2 जनवरी",
		YearMonth: "जनवरी 2006",
	},
	"vi-VN": {
		Full:      "2 tháng 1 năm 2006 15:04:05",
		Long:      "2 tháng 1 năm 2006 15:04",
		Medium:    "2 tháng 1 năm 2006",
		Short:     "2/1/06",
		TimeOnly:  "15:04:05",
		DateOnly:  "02/01/2006",
		MonthDay:  "2 tháng 1",
		YearMonth: "tháng 1 năm 2006",
	},
	"th-TH": {
		Full:      "2 มกราคม 2006 15:04:05",
		Long:      "2 มกราคม 2006 15:04",
		Medium:    "2 มกราคม 2006",
		Short:     "2/1/49",
		TimeOnly:  "15:04:05",
		DateOnly:  "02/01/2006",
		MonthDay:  "2 มกราคม",
		YearMonth: "มกราคม 2006",
	},
	"id-ID": {
		Full:      "2 Januari 2006 15.04.05",
		Long:      "2 Januari 2006 15.04",
		Medium:    "2 Januari 2006",
		Short:     "2/1/06",
		TimeOnly:  "15.04.05",
		DateOnly:  "02/01/2006",
		MonthDay:  "2 Januari",
		YearMonth: "Januari 2006",
	},
	"tr-TR": {
		Full:      "2 Ocak 2006 15:04:05",
		Long:      "2 Ocak 2006 15:04",
		Medium:    "2 Ocak 2006",
		Short:     "02.01.2006",
		TimeOnly:  "15:04:05",
		DateOnly:  "2006-01-02",
		MonthDay:  "2 Ocak",
		YearMonth: "Ocak 2006",
	},
}

func FormatDateTime(t time.Time, lang string, formatType string) string {
	targetLang := lang
	if !IsSupported(targetLang) {
		targetLang = defaultLang
	}

	formats, ok := dateFormats[targetLang]
	if !ok {
		formats = dateFormats[defaultLang]
	}

	loc := GetLocation(GetLangInfo(targetLang).Timezone)
	t = t.In(loc)

	switch formatType {
	case "full":
		return t.Format(formats.Full)
	case "long":
		return t.Format(formats.Long)
	case "medium":
		return t.Format(formats.Medium)
	case "short":
		return t.Format(formats.Short)
	case "time":
		return t.Format(formats.TimeOnly)
	case "date":
		return t.Format(formats.DateOnly)
	case "monthday":
		return t.Format(formats.MonthDay)
	case "yearmonth":
		return t.Format(formats.YearMonth)
	default:
		return t.Format(formats.Medium)
	}
}

func FormatDateTimeWithTimezone(t time.Time, lang string, timezone string, formatType string) string {
	targetLang := lang
	if !IsSupported(targetLang) {
		targetLang = defaultLang
	}

	formats, ok := dateFormats[targetLang]
	if !ok {
		formats = dateFormats[defaultLang]
	}

	loc := GetLocation(timezone)
	t = t.In(loc)

	switch formatType {
	case "full":
		return t.Format(formats.Full)
	case "long":
		return t.Format(formats.Long)
	case "medium":
		return t.Format(formats.Medium)
	case "short":
		return t.Format(formats.Short)
	case "time":
		return t.Format(formats.TimeOnly)
	case "date":
		return t.Format(formats.DateOnly)
	case "monthday":
		return t.Format(formats.MonthDay)
	case "yearmonth":
		return t.Format(formats.YearMonth)
	default:
		return t.Format(formats.Medium)
	}
}

func FormatRelativeTime(t time.Time, lang string) string {
	targetLang := lang
	if !IsSupported(targetLang) {
		targetLang = defaultLang
	}

	now := time.Now()
	diff := now.Sub(t)

	translations := GetAllTranslations(targetLang)

	if diff < time.Minute {
		return translations["just_now"]
	} else if diff < time.Hour {
		minutes := int(diff.Minutes())
		return fmt.Sprintf(translations["minutes_ago"], minutes)
	} else if diff < 24*time.Hour {
		hours := int(diff.Hours())
		return fmt.Sprintf(translations["hours_ago"], hours)
	} else if diff < 7*24*time.Hour {
		days := int(diff.Hours() / 24)
		return fmt.Sprintf(translations["days_ago"], days)
	} else if diff < 30*24*time.Hour {
		weeks := int(diff.Hours() / (24 * 7))
		return fmt.Sprintf(translations["weeks_ago"], weeks)
	} else if diff < 365*24*time.Hour {
		months := int(diff.Hours() / (24 * 30))
		return fmt.Sprintf(translations["months_ago"], months)
	} else {
		years := int(diff.Hours() / (24 * 365))
		return fmt.Sprintf(translations["years_ago"], years)
	}
}

type NumberFormat struct {
	DecimalSep   string
	ThousandSep  string
	DecimalDigits int
}

var numberFormats = map[string]NumberFormat{
	"zh-CN": {
		DecimalSep:   ".",
		ThousandSep:  ",",
		DecimalDigits: 2,
	},
	"en-US": {
		DecimalSep:   ".",
		ThousandSep:  ",",
		DecimalDigits: 2,
	},
	"ja-JP": {
		DecimalSep:   ".",
		ThousandSep:  ",",
		DecimalDigits: 0,
	},
	"ko-KR": {
		DecimalSep:   ".",
		ThousandSep:  ",",
		DecimalDigits: 0,
	},
	"fr-FR": {
		DecimalSep:   ",",
		ThousandSep:  " ",
		DecimalDigits: 2,
	},
	"de-DE": {
		DecimalSep:   ",",
		ThousandSep:  ".",
		DecimalDigits: 2,
	},
	"es-ES": {
		DecimalSep:   ",",
		ThousandSep:  ".",
		DecimalDigits: 2,
	},
	"pt-BR": {
		DecimalSep:   ",",
		ThousandSep:  ".",
		DecimalDigits: 2,
	},
	"it-IT": {
		DecimalSep:   ",",
		ThousandSep:  ".",
		DecimalDigits: 2,
	},
	"ru-RU": {
		DecimalSep:   ",",
		ThousandSep:  " ",
		DecimalDigits: 2,
	},
	"ar-SA": {
		DecimalSep:   "٫",
		ThousandSep:  "٬",
		DecimalDigits: 3,
	},
	"fa-IR": {
		DecimalSep:   "٫",
		ThousandSep:  "٬",
		DecimalDigits: 0,
	},
	"he-IL": {
		DecimalSep:   ".",
		ThousandSep:  ",",
		DecimalDigits: 2,
	},
	"ur-PK": {
		DecimalSep:   ".",
		ThousandSep:  ",",
		DecimalDigits: 0,
	},
	"hi-IN": {
		DecimalSep:   ".",
		ThousandSep:  ",",
		DecimalDigits: 2,
	},
	"vi-VN": {
		DecimalSep:   ",",
		ThousandSep:  ".",
		DecimalDigits: 2,
	},
	"th-TH": {
		DecimalSep:   ".",
		ThousandSep:  ",",
		DecimalDigits: 2,
	},
	"id-ID": {
		DecimalSep:   ",",
		ThousandSep:  ".",
		DecimalDigits: 2,
	},
	"tr-TR": {
		DecimalSep:   ",",
		ThousandSep:  ".",
		DecimalDigits: 2,
	},
}

func FormatNumber(value float64, lang string) string {
	targetLang := lang
	if !IsSupported(targetLang) {
		targetLang = defaultLang
	}

	format, ok := numberFormats[targetLang]
	if !ok {
		format = numberFormats[defaultLang]
	}

	return formatFloat(value, format.DecimalSep, format.ThousandSep, format.DecimalDigits)
}

func FormatNumberWithDigits(value float64, lang string, decimalDigits int) string {
	targetLang := lang
	if !IsSupported(targetLang) {
		targetLang = defaultLang
	}

	format, ok := numberFormats[targetLang]
	if !ok {
		format = numberFormats[defaultLang]
	}

	return formatFloat(value, format.DecimalSep, format.ThousandSep, decimalDigits)
}

func formatFloat(value float64, decimalSep, thousandSep string, decimalDigits int) string {
	if decimalDigits < 0 {
		decimalDigits = 0
	}

	multiplier := 1.0
	for i := 0; i < decimalDigits; i++ {
		multiplier *= 10
	}

	intPart := int64(value * multiplier)
	decPart := intPart % int64(multiplier)

	intValue := intPart / int64(multiplier)

	intStr := fmt.Sprintf("%d", intValue)
	var result []byte
	length := len(intStr)

	for i := 0; i < length; i++ {
		if i > 0 && (length-i)%3 == 0 {
			result = append(result, thousandSep[0])
		}
		result = append(result, intStr[i])
	}

	if decimalDigits > 0 {
		decStr := fmt.Sprintf("%0*d", decimalDigits, decPart)
		result = append(result, decimalSep[0])
		result = append(result, decStr...)
	}

	return string(result)
}

func FormatInteger(value int64, lang string) string {
	targetLang := lang
	if !IsSupported(targetLang) {
		targetLang = defaultLang
	}

	format, ok := numberFormats[targetLang]
	if !ok {
		format = numberFormats[defaultLang]
	}

	return formatInt(value, format.ThousandSep)
}

func formatInt(value int64, thousandSep string) string {
	str := fmt.Sprintf("%d", value)
	length := len(str)
	var result []byte

	for i := 0; i < length; i++ {
		if i > 0 && (length-i)%3 == 0 {
			result = append(result, thousandSep[0])
		}
		result = append(result, str[i])
	}

	return string(result)
}

func FormatPercent(value float64, lang string) string {
	targetLang := lang
	if !IsSupported(targetLang) {
		targetLang = defaultLang
	}

	percentValue := FormatNumberWithDigits(value*100, lang, 2)
	return percentValue + "%"
}

type CurrencyFormat struct {
	Symbol      string
	SymbolPos   string
	DecimalSep  string
	ThousandSep string
	DecimalDigits int
}

var currencyFormats = map[string]CurrencyFormat{
	"zh-CN": {
		Symbol:       "¥",
		SymbolPos:    "before",
		DecimalSep:   ".",
		ThousandSep:  ",",
		DecimalDigits: 2,
	},
	"en-US": {
		Symbol:       "$",
		SymbolPos:    "before",
		DecimalSep:   ".",
		ThousandSep:  ",",
		DecimalDigits: 2,
	},
	"ja-JP": {
		Symbol:       "¥",
		SymbolPos:    "before",
		DecimalSep:   ".",
		ThousandSep:  ",",
		DecimalDigits: 0,
	},
	"ko-KR": {
		Symbol:       "₩",
		SymbolPos:    "before",
		DecimalSep:   ".",
		ThousandSep:  ",",
		DecimalDigits: 0,
	},
	"fr-FR": {
		Symbol:       "€",
		SymbolPos:    "after",
		DecimalSep:   ",",
		ThousandSep:  " ",
		DecimalDigits: 2,
	},
	"de-DE": {
		Symbol:       "€",
		SymbolPos:    "after",
		DecimalSep:   ",",
		ThousandSep:  ".",
		DecimalDigits: 2,
	},
	"es-ES": {
		Symbol:       "€",
		SymbolPos:    "after",
		DecimalSep:   ",",
		ThousandSep:  ".",
		DecimalDigits: 2,
	},
	"pt-BR": {
		Symbol:       "R$",
		SymbolPos:    "before",
		DecimalSep:   ",",
		ThousandSep:  ".",
		DecimalDigits: 2,
	},
	"it-IT": {
		Symbol:       "€",
		SymbolPos:    "after",
		DecimalSep:   ",",
		ThousandSep:  ".",
		DecimalDigits: 2,
	},
	"ru-RU": {
		Symbol:       "₽",
		SymbolPos:    "after",
		DecimalSep:   ",",
		ThousandSep:  " ",
		DecimalDigits: 2,
	},
	"ar-SA": {
		Symbol:       "ر.س",
		SymbolPos:    "after",
		DecimalSep:   "٫",
		ThousandSep:  "٬",
		DecimalDigits: 2,
	},
	"fa-IR": {
		Symbol:       "ریال",
		SymbolPos:    "after",
		DecimalSep:   "٫",
		ThousandSep:  "٬",
		DecimalDigits: 0,
	},
	"he-IL": {
		Symbol:       "₪",
		SymbolPos:    "before",
		DecimalSep:   ".",
		ThousandSep:  ",",
		DecimalDigits: 2,
	},
	"ur-PK": {
		Symbol:       "₨",
		SymbolPos:    "before",
		DecimalSep:   ".",
		ThousandSep:  ",",
		DecimalDigits: 0,
	},
	"hi-IN": {
		Symbol:       "₹",
		SymbolPos:    "before",
		DecimalSep:   ".",
		ThousandSep:  ",",
		DecimalDigits: 2,
	},
	"vi-VN": {
		Symbol:       "₫",
		SymbolPos:    "after",
		DecimalSep:   ",",
		ThousandSep:  ".",
		DecimalDigits: 0,
	},
	"th-TH": {
		Symbol:       "฿",
		SymbolPos:    "before",
		DecimalSep:   ".",
		ThousandSep:  ",",
		DecimalDigits: 2,
	},
	"id-ID": {
		Symbol:       "Rp",
		SymbolPos:    "before",
		DecimalSep:   ",",
		ThousandSep:  ".",
		DecimalDigits: 0,
	},
	"tr-TR": {
		Symbol:       "₺",
		SymbolPos:    "before",
		DecimalSep:   ",",
		ThousandSep:  ".",
		DecimalDigits: 2,
	},
}

func FormatCurrency(value float64, lang string) string {
	targetLang := lang
	if !IsSupported(targetLang) {
		targetLang = defaultLang
	}

	currency, ok := currencyFormats[targetLang]
	if !ok {
		currency = currencyFormats[defaultLang]
	}

	number := formatFloat(value, currency.DecimalSep, currency.ThousandSep, currency.DecimalDigits)

	if currency.SymbolPos == "before" {
		return currency.Symbol + number
	} else {
		return number + currency.Symbol
	}
}

func FormatCurrencyWithCode(value float64, lang string, currencyCode string) string {
	return FormatCurrency(value, lang) + " " + currencyCode
}
