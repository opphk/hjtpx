package i18n

import (
	"fmt"
	"strconv"
	"strings"
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

type RelativeTimeFormat struct {
	JustNow  string
	MinutesAgo  string
	HoursAgo  string
	DaysAgo  string
	WeeksAgo  string
	MonthsAgo  string
	YearsAgo  string
}

var relativeTimeFormats = map[string]RelativeTimeFormat{
	"zh-CN": {
		JustNow:   "刚刚",
		MinutesAgo: "{0}分钟前",
		HoursAgo:   "{0}小时前",
		DaysAgo:    "{0}天前",
		WeeksAgo:   "{0}周前",
		MonthsAgo:  "{0}个月前",
		YearsAgo:   "{0}年前",
	},
	"en-US": {
		JustNow:   "just now",
		MinutesAgo: "{0}m ago",
		HoursAgo:   "{0}h ago",
		DaysAgo:    "{0}d ago",
		WeeksAgo:   "{0}w ago",
		MonthsAgo:  "{0}mo ago",
		YearsAgo:   "{0}y ago",
	},
	"ja-JP": {
		JustNow:   "たった今",
		MinutesAgo: "{0}分前",
		HoursAgo:   "{0}時間前",
		DaysAgo:    "{0}日前",
		WeeksAgo:   "{0}週間前",
		MonthsAgo:  "{0}ヶ月前",
		YearsAgo:   "{0}年前",
	},
	"ko-KR": {
		JustNow:   "방금",
		MinutesAgo: "{0}분 전",
		HoursAgo:   "{0}시간 전",
		DaysAgo:    "{0}일 전",
		WeeksAgo:   "{0}주 전",
		MonthsAgo:  "{0}개월 전",
		YearsAgo:   "{0}년 전",
	},
	"fr-FR": {
		JustNow:   "à l'instant",
		MinutesAgo: "il y a {0} minute",
		HoursAgo:   "il y a {0} heure",
		DaysAgo:    "il y a {0} jour",
		WeeksAgo:   "il y a {0} semaine",
		MonthsAgo:  "il y a {0} mois",
		YearsAgo:   "il y a {0} an",
	},
	"de-DE": {
		JustNow:   "gerade eben",
		MinutesAgo: "vor {0} Minute",
		HoursAgo:   "vor {0} Stunde",
		DaysAgo:    "vor {0} Tag",
		WeeksAgo:   "vor {0} Woche",
		MonthsAgo:  "vor {0} Monat",
		YearsAgo:   "vor {0} Jahr",
	},
	"es-ES": {
		JustNow:   "ahora mismo",
		MinutesAgo: "hace {0} minuto",
		HoursAgo:   "hace {0} hora",
		DaysAgo:    "hace {0} día",
		WeeksAgo:   "hace {0} semana",
		MonthsAgo:  "hace {0} mes",
		YearsAgo:   "hace {0} año",
	},
	"pt-BR": {
		JustNow:   "agora mesmo",
		MinutesAgo: "há {0} minuto",
		HoursAgo:   "há {0} hora",
		DaysAgo:    "há {0} dia",
		WeeksAgo:   "há {0} semana",
		MonthsAgo:  "há {0} mês",
		YearsAgo:   "há {0} ano",
	},
	"it-IT": {
		JustNow:   "adesso",
		MinutesAgo: "{0} minuto fa",
		HoursAgo:   "{0} ora fa",
		DaysAgo:    "{0} giorno fa",
		WeeksAgo:   "{0} settimana fa",
		MonthsAgo:  "{0} mese fa",
		YearsAgo:   "{0} anno fa",
	},
	"ru-RU": {
		JustNow:   "только что",
		MinutesAgo: "{0} минуту назад",
		HoursAgo:   "{0} час назад",
		DaysAgo:    "{0} день назад",
		WeeksAgo:   "{0} неделю назад",
		MonthsAgo:  "{0} месяц назад",
		YearsAgo:   "{0} год назад",
	},
	"ar-SA": {
		JustNow:   "الآن",
		MinutesAgo: "منذ {0} دقيقة",
		HoursAgo:   "منذ {0} ساعة",
		DaysAgo:    "منذ {0} يوم",
		WeeksAgo:   "منذ {0} أسبوع",
		MonthsAgo:  "منذ {0} شهر",
		YearsAgo:   "منذ {0} سنة",
	},
	"fa-IR": {
		JustNow:   "الان",
		MinutesAgo: "{0} دقیقه پیش",
		HoursAgo:   "{0} ساعت پیش",
		DaysAgo:    "{0} روز پیش",
		WeeksAgo:   "{0} هفته پیش",
		MonthsAgo:  "{0} ماه پیش",
		YearsAgo:   "{0} سال پیش",
	},
	"he-IL": {
		JustNow:   "עכשיו",
		MinutesAgo: "לפני {0} דקות",
		HoursAgo:   "לפני {0} שעות",
		DaysAgo:    "לפני {0} ימים",
		WeeksAgo:   "לפני {0} שבועות",
		MonthsAgo:  "לפני {0} חודשים",
		YearsAgo:   "לפני {0} שנים",
	},
	"ur-PK": {
		JustNow:   "ابھی",
		MinutesAgo: "{0} منٹ پہلے",
		HoursAgo:   "{0} گھنٹے پہلے",
		DaysAgo:    "{0} دن پہلے",
		WeeksAgo:   "{0} ہفتے پہلے",
		MonthsAgo:  "{0} مہینے پہلے",
		YearsAgo:   "{0} سال پہلے",
	},
	"hi-IN": {
		JustNow:   "अभी",
		MinutesAgo: "{0} मिनट पहले",
		HoursAgo:   "{0} घंटे पहले",
		DaysAgo:    "{0} दिन पहले",
		WeeksAgo:   "{0} सप्ताह पहले",
		MonthsAgo:  "{0} महीने पहले",
		YearsAgo:   "{0} साल पहले",
	},
	"vi-VN": {
		JustNow:   "vừa xong",
		MinutesAgo: "{0} phút trước",
		HoursAgo:   "{0} giờ trước",
		DaysAgo:    "{0} ngày trước",
		WeeksAgo:   "{0} tuần trước",
		MonthsAgo:  "{0} tháng trước",
		YearsAgo:   "{0} năm trước",
	},
	"th-TH": {
		JustNow:   "เพิ่งจะ",
		MinutesAgo: "{0} นาทีที่แล้ว",
		HoursAgo:   "{0} ชั่วโมงที่แล้ว",
		DaysAgo:    "{0} วันที่แล้ว",
		WeeksAgo:   "{0} สัปดาห์ที่แล้ว",
		MonthsAgo:  "{0} เดือนที่แล้ว",
		YearsAgo:   "{0} ปีที่แล้ว",
	},
	"id-ID": {
		JustNow:   "baru saja",
		MinutesAgo: "{0} menit yang lalu",
		HoursAgo:   "{0} jam yang lalu",
		DaysAgo:    "{0} hari yang lalu",
		WeeksAgo:   "{0} minggu yang lalu",
		MonthsAgo:  "{0} bulan yang lalu",
		YearsAgo:   "{0} tahun yang lalu",
	},
	"tr-TR": {
		JustNow:   "az önce",
		MinutesAgo: "{0} dakika önce",
		HoursAgo:   "{0} saat önce",
		DaysAgo:    "{0} gün önce",
		WeeksAgo:   "{0} hafta önce",
		MonthsAgo:  "{0} ay önce",
		YearsAgo:   "{0} yıl önce",
	},
}

func FormatRelativeTimeFriendly(t time.Time, lang string) string {
	targetLang := lang
	if !IsSupported(targetLang) {
		targetLang = defaultLang
	}

	format, ok := relativeTimeFormats[targetLang]
	if !ok {
		format = relativeTimeFormats[defaultLang]
	}

	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Minute {
		return format.JustNow
	} else if diff < time.Hour {
		minutes := int(diff.Minutes())
		return strings.Replace(format.MinutesAgo, "{0}", fmt.Sprintf("%d", minutes), 1)
	} else if diff < 24*time.Hour {
		hours := int(diff.Hours())
		return strings.Replace(format.HoursAgo, "{0}", fmt.Sprintf("%d", hours), 1)
	} else if diff < 7*24*time.Hour {
		days := int(diff.Hours() / 24)
		return strings.Replace(format.DaysAgo, "{0}", fmt.Sprintf("%d", days), 1)
	} else if diff < 30*24*time.Hour {
		weeks := int(diff.Hours() / (24 * 7))
		return strings.Replace(format.WeeksAgo, "{0}", fmt.Sprintf("%d", weeks), 1)
	} else if diff < 365*24*time.Hour {
		months := int(diff.Hours() / (24 * 30))
		return strings.Replace(format.MonthsAgo, "{0}", fmt.Sprintf("%d", months), 1)
	} else {
		years := int(diff.Hours() / (24 * 365))
		return strings.Replace(format.YearsAgo, "{0}", fmt.Sprintf("%d", years), 1)
	}
}

func FormatCompactNumber(value float64, lang string) string {
	targetLang := lang
	if !IsSupported(targetLang) {
		targetLang = defaultLang
	}

	if value >= 1000000000 {
		return fmt.Sprintf("%.1fB", value/1000000000)
	} else if value >= 1000000 {
		return fmt.Sprintf("%.1fM", value/1000000)
	} else if value >= 1000 {
		return fmt.Sprintf("%.1fK", value/1000)
	}
	return fmt.Sprintf("%.0f", value)
}

func FormatDuration(d time.Duration, lang string) string {
	targetLang := lang
	if !IsSupported(targetLang) {
		targetLang = defaultLang
	}

	totalSeconds := int(d.Seconds())

	if totalSeconds < 60 {
		return fmt.Sprintf("%ds", totalSeconds)
	} else if totalSeconds < 3600 {
		minutes := totalSeconds / 60
		seconds := totalSeconds % 60
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	} else {
		hours := totalSeconds / 3600
		minutes := (totalSeconds % 3600) / 60
		seconds := totalSeconds % 60
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	}
}

type CultureInfo struct {
	Country         string
	Language        string
	CalendarType    string
	WeekStartDay   time.Weekday
	DateSeparator   string
	TimeSeparator   string
	DecimalSeparator string
	ThousandSeparator string
	CurrencySymbol string
	NumberFormat   string
	PaperSize      string
	MeasurementSystem string
}

var cultureInfos = map[string]CultureInfo{
	"zh-CN": {
		Country:           "China",
		Language:          "Chinese (Simplified)",
		CalendarType:     "Gregorian",
		WeekStartDay:     time.Monday,
		DateSeparator:    "-",
		TimeSeparator:    ":",
		DecimalSeparator: ".",
		ThousandSeparator: ",",
		CurrencySymbol:   "¥",
		NumberFormat:     "1,234.56",
		PaperSize:        "A4",
		MeasurementSystem: "Metric",
	},
	"en-US": {
		Country:           "United States",
		Language:          "English",
		CalendarType:     "Gregorian",
		WeekStartDay:     time.Sunday,
		DateSeparator:    "/",
		TimeSeparator:    ":",
		DecimalSeparator: ".",
		ThousandSeparator: ",",
		CurrencySymbol:   "$",
		NumberFormat:     "1,234.56",
		PaperSize:        "Letter",
		MeasurementSystem: "Imperial",
	},
	"ja-JP": {
		Country:           "Japan",
		Language:          "Japanese",
		CalendarType:     "Gregorian",
		WeekStartDay:     time.Sunday,
		DateSeparator:    "/",
		TimeSeparator:    ":",
		DecimalSeparator: ".",
		ThousandSeparator: ",",
		CurrencySymbol:   "¥",
		NumberFormat:     "1,234.56",
		PaperSize:        "A4",
		MeasurementSystem: "Metric",
	},
	"ar-SA": {
		Country:           "Saudi Arabia",
		Language:          "Arabic",
		CalendarType:     "Gregorian",
		WeekStartDay:     time.Saturday,
		DateSeparator:    "/",
		TimeSeparator:    ":",
		DecimalSeparator: "٫",
		ThousandSeparator: "٬",
		CurrencySymbol:   "ر.س",
		NumberFormat:     "١٬٢٣٤٫٥٦",
		PaperSize:        "A4",
		MeasurementSystem: "Metric",
	},
	"hi-IN": {
		Country:           "India",
		Language:          "Hindi",
		CalendarType:     "Gregorian",
		WeekStartDay:     time.Sunday,
		DateSeparator:    "/",
		TimeSeparator:    ":",
		DecimalSeparator: ".",
		ThousandSeparator: ",",
		CurrencySymbol:   "₹",
		NumberFormat:     "1,23,456.78",
		PaperSize:        "A4",
		MeasurementSystem: "Metric",
	},
}

func GetCultureInfo(lang string) CultureInfo {
	if info, ok := cultureInfos[lang]; ok {
		return info
	}
	return CultureInfo{
		Country:           "Unknown",
		Language:          lang,
		CalendarType:     "Gregorian",
		WeekStartDay:     time.Sunday,
		DateSeparator:    "/",
		TimeSeparator:    ":",
		DecimalSeparator: ".",
		ThousandSeparator: ",",
		CurrencySymbol:   "$",
		NumberFormat:     "1,234.56",
		PaperSize:        "A4",
		MeasurementSystem: "Metric",
	}
}

func FormatPhoneNumber(phone string, countryCode string) string {
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	phone = strings.ReplaceAll(phone, "(", "")
	phone = strings.ReplaceAll(phone, ")", "")

	switch countryCode {
	case "CN":
		if len(phone) == 11 {
			return fmt.Sprintf("%s %s %s", phone[:3], phone[3:7], phone[7:])
		}
	case "US", "CA":
		if len(phone) == 10 {
			return fmt.Sprintf("(%s) %s-%s", phone[:3], phone[3:6], phone[6:])
		}
	case "JP":
		if len(phone) == 10 {
			return fmt.Sprintf("%s-%s-%s", phone[:3], phone[3:7], phone[7:])
		}
	case "IN":
		if len(phone) == 10 {
			return fmt.Sprintf("%s-%s-%s", phone[:4], phone[4:7], phone[7:])
		}
	case "GB":
		if len(phone) == 10 {
			return fmt.Sprintf("%s %s %s", phone[:4], phone[4:7], phone[7:])
		}
	}

	return phone
}

func FormatPostalCode(postalCode string, countryCode string) string {
	postalCode = strings.ReplaceAll(postalCode, " ", "")

	switch countryCode {
	case "US":
		if len(postalCode) == 5 || len(postalCode) == 9 {
			if len(postalCode) == 9 {
				return fmt.Sprintf("%s-%s", postalCode[:5], postalCode[5:])
			}
			return postalCode
		}
	case "GB":
		if len(postalCode) >= 5 && len(postalCode) <= 7 {
			if len(postalCode) == 6 {
				return fmt.Sprintf("%s %s", postalCode[:3], postalCode[3:])
			} else if len(postalCode) == 7 {
				return fmt.Sprintf("%s %s", postalCode[:4], postalCode[4:])
			}
			return postalCode
		}
	case "CA":
		if len(postalCode) == 6 {
			return fmt.Sprintf("%s %s", postalCode[:3], postalCode[3:])
		}
	case "JP":
		if len(postalCode) == 7 {
			return fmt.Sprintf("%s-%s", postalCode[:3], postalCode[3:])
		}
	case "IN":
		if len(postalCode) == 6 {
			return postalCode
		}
	}

	return postalCode
}

func FormatAddress(address map[string]string, countryCode string) string {
	var parts []string

	if street, ok := address["street"]; ok && street != "" {
		parts = append(parts, street)
	}

	if city, ok := address["city"]; ok && city != "" {
		parts = append(parts, city)
	}

	if state, ok := address["state"]; ok && state != "" {
		parts = append(parts, state)
	}

	if postalCode, ok := address["postalCode"]; ok && postalCode != "" {
		formattedCode := FormatPostalCode(postalCode, countryCode)
		parts = append(parts, formattedCode)
	}

	if country, ok := address["country"]; ok && country != "" {
		parts = append(parts, country)
	}

	return strings.Join(parts, ", ")
}

func GetFirstDayOfWeek(lang string) time.Weekday {
	info := GetCultureInfo(lang)
	return info.WeekStartDay
}

func GetWeekendDays(lang string) []time.Weekday {
	switch lang {
	case "en-US":
		return []time.Weekday{time.Saturday, time.Sunday}
	case "ar-SA":
		return []time.Weekday{time.Friday, time.Saturday}
	case "zh-CN", "ja-JP", "ko-KR":
		return []time.Weekday{time.Saturday, time.Sunday}
	default:
		return []time.Weekday{time.Saturday, time.Sunday}
	}
}

func IsWeekend(t time.Time, lang string) bool {
	weekendDays := GetWeekendDays(lang)
	for _, day := range weekendDays {
		if t.Weekday() == day {
			return true
		}
	}
	return false
}

func GetBusinessDaysInRange(start, end time.Time, lang string) int {
	businessDays := 0
	current := start

	weekendDays := GetWeekendDays(lang)

	for !current.After(end) {
		isWeekend := false
		for _, day := range weekendDays {
			if current.Weekday() == day {
				isWeekend = true
				break
			}
		}

		if !isWeekend {
			businessDays++
		}

		current = current.AddDate(0, 0, 1)
	}

	return businessDays
}

func GetFiscalYear(date time.Time, fiscalStartMonth time.Month) int {
	if date.Month() >= fiscalStartMonth {
		return date.Year()
	}
	return date.Year() - 1
}

func FormatFiscalQuarter(date time.Time) string {
	quarter := (int(date.Month())-1)/3 + 1
	return fmt.Sprintf("Q%d FY%d", quarter, GetFiscalYear(date, time.January))
}

func GetOrdinalSuffix(n int) string {
	switch n {
	case 1, 21, 31:
		return "st"
	case 2, 22:
		return "nd"
	case 3, 23:
		return "rd"
	default:
		return "th"
	}
}

func FormatOrdinal(n int, lang string) string {
	suffix := GetOrdinalSuffix(n)

	switch lang {
	case "zh-CN":
		return fmt.Sprintf("第%d", n)
	case "ja-JP":
		return fmt.Sprintf("%d番目", n)
	case "ko-KR":
		return fmt.Sprintf("%d번째", n)
	case "ar-SA", "he-IL":
		return fmt.Sprintf("%d%s", n, suffix)
	default:
		return fmt.Sprintf("%d%s", n, suffix)
	}
}

func FormatTimeWithLocale(t time.Time, lang string, style string) string {
	targetLang := lang
	if !IsSupported(targetLang) {
		targetLang = defaultLang
	}

	info := GetCultureInfo(targetLang)
	timeFormat := fmt.Sprintf("15%s04%s05", info.TimeSeparator, info.TimeSeparator)

	switch style {
	case "short":
		return t.Format(fmt.Sprintf("15%s04", info.TimeSeparator))
	case "medium":
		return t.Format(timeFormat)
	case "long":
		return t.Format(fmt.Sprintf("3%s04%s05 PM", info.TimeSeparator, info.TimeSeparator))
	default:
		return t.Format(timeFormat)
	}
}

func FormatDateWithLocale(t time.Time, lang string, style string) string {
	targetLang := lang
	if !IsSupported(targetLang) {
		targetLang = defaultLang
	}

	info := GetCultureInfo(targetLang)

	switch style {
	case "short":
		return t.Format(fmt.Sprintf("01%s02%s06", info.DateSeparator, info.DateSeparator))
	case "medium":
		return t.Format(fmt.Sprintf("02%s01%s2006", info.DateSeparator, info.DateSeparator))
	case "long":
		return FormatDateTime(t, targetLang, "long")
	default:
		return t.Format(fmt.Sprintf("02%s01%s2006", info.DateSeparator, info.DateSeparator))
	}
}

func FormatNumberForLocale(value float64, lang string, showThousandSeparator bool) string {
	if !showThousandSeparator {
		return fmt.Sprintf("%.2f", value)
	}
	return FormatNumber(value, lang)
}

func ParseLocalizedNumber(value string, lang string) (float64, error) {
	info := GetCultureInfo(lang)

	value = strings.ReplaceAll(value, info.ThousandSeparator, "")
	value = strings.ReplaceAll(value, info.DecimalSeparator, ".")

	return strconv.ParseFloat(value, 64)
}

func ParseLocalizedDate(dateStr string, lang string) (time.Time, error) {
	targetLang := lang
	if !IsSupported(targetLang) {
		targetLang = defaultLang
	}

	info := GetCultureInfo(targetLang)

	dateStr = strings.ReplaceAll(dateStr, info.DateSeparator, "-")

	formats := []string{
		"2006-01-02",
		"01-02-2006",
		"02-01-2006",
		"2006/01/02",
		"01/02/2006",
		"02/01/2006",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

func FormatISODate(t time.Time) string {
	return t.Format("2006-01-02")
}

func FormatISOTime(t time.Time) string {
	return t.Format("15:04:05")
}

func FormatISODateTime(t time.Time) string {
	return t.Format("2006-01-02T15:04:05Z")
}

func GetLocalCalendarMonth(lang string, month time.Month) string {
	months := map[string][]string{
		"zh-CN": {"一月", "二月", "三月", "四月", "五月", "六月", "七月", "八月", "九月", "十月", "十一月", "十二月"},
		"en-US": {"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"},
		"ja-JP": {"1月", "2月", "3月", "4月", "5月", "6月", "7月", "8月", "9月", "10月", "11月", "12月"},
		"ar-SA": {"يناير", "فبراير", "مارس", "أبريل", "مايو", "يونيو", "يوليو", "أغسطس", "سبتمبر", "أكتوبر", "نوفمبر", "ديسمبر"},
	}

	if monthNames, ok := months[lang]; ok {
		return monthNames[month-1]
	}

	if enNames, ok := months["en-US"]; ok {
		return enNames[month-1]
	}

	return month.String()
}

func GetLocalCalendarWeekday(lang string, weekday time.Weekday) string {
	weekdays := map[string][]string{
		"zh-CN": {"周日", "周一", "周二", "周三", "周四", "周五", "周六"},
		"en-US": {"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"},
		"ja-JP": {"日", "月", "火", "水", "木", "金", "土"},
		"ar-SA": {"الأحد", "الإثنين", "الثلاثاء", "الأربعاء", "الخميس", "الجمعة", "السبت"},
	}

	if dayNames, ok := weekdays[lang]; ok {
		return dayNames[weekday]
	}

	if enNames, ok := weekdays["en-US"]; ok {
		return enNames[weekday]
	}

	return weekday.String()
}
