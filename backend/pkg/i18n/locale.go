package i18n

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type LocaleFormat struct {
	DateFormat   string
	TimeFormat   string
	DateTimeFormat string
	DecimalSeparator string
	ThousandSeparator string
	CurrencySymbol string
	CurrencyCode string
	NumberFormat string
	FirstWeekday time.Weekday
	ListSeparator string
}

type LocaleData struct {
	Name      string
	NativeName string
	Format    LocaleFormat
	PluralRules func(n int) string
}

var (
	localeData     map[string]*LocaleData
	localeDataOnce sync.Once
)

func initLocaleData() {
	localeData = map[string]*LocaleData{
		"zh-CN": {
			Name:      "Chinese (Simplified)",
			NativeName: "简体中文",
			Format: LocaleFormat{
				DateFormat:        "2006年1月2日",
				TimeFormat:        "15:04:05",
				DateTimeFormat:    "2006年1月2日 15:04:05",
				DecimalSeparator:  ".",
				ThousandSeparator: ",",
				CurrencySymbol:    "¥",
				CurrencyCode:      "CNY",
				NumberFormat:      "#,##0.##",
				FirstWeekday:      time.Monday,
				ListSeparator:     "、",
			},
			PluralRules: pluralNone,
		},
		"en-US": {
			Name:      "English (US)",
			NativeName: "English",
			Format: LocaleFormat{
				DateFormat:        "January 2, 2006",
				TimeFormat:        "3:04:05 PM",
				DateTimeFormat:    "January 2, 2006 3:04:05 PM",
				DecimalSeparator:  ".",
				ThousandSeparator: ",",
				CurrencySymbol:    "$",
				CurrencyCode:      "USD",
				NumberFormat:      "#,##0.##",
				FirstWeekday:      time.Sunday,
				ListSeparator:     ", ",
			},
			PluralRules: pluralEN,
		},
		"ja-JP": {
			Name:      "Japanese",
			NativeName: "日本語",
			Format: LocaleFormat{
				DateFormat:        "2006年1月2日",
				TimeFormat:        "15:04:05",
				DateTimeFormat:    "2006年1月2日 15:04:05",
				DecimalSeparator:  ".",
				ThousandSeparator: ",",
				CurrencySymbol:    "¥",
				CurrencyCode:      "JPY",
				NumberFormat:      "#,##0",
				FirstWeekday:      time.Sunday,
				ListSeparator:     "、",
			},
			PluralRules: pluralNone,
		},
		"ko-KR": {
			Name:      "Korean",
			NativeName: "한국어",
			Format: LocaleFormat{
				DateFormat:        "2006년 1월 2일",
				TimeFormat:        "15:04:05",
				DateTimeFormat:    "2006년 1월 2일 15:04:05",
				DecimalSeparator:  ".",
				ThousandSeparator: ",",
				CurrencySymbol:    "₩",
				CurrencyCode:      "KRW",
				NumberFormat:      "#,##0",
				FirstWeekday:      time.Sunday,
				ListSeparator:     ", ",
			},
			PluralRules: pluralNone,
		},
		"fr-FR": {
			Name:      "French",
			NativeName: "Français",
			Format: LocaleFormat{
				DateFormat:        "2 janvier 2006",
				TimeFormat:        "15:04:05",
				DateTimeFormat:    "2 janvier 2006 à 15:04:05",
				DecimalSeparator:  ",",
				ThousandSeparator: " ",
				CurrencySymbol:    "€",
				CurrencyCode:      "EUR",
				NumberFormat:      "# ##0,##",
				FirstWeekday:      time.Monday,
				ListSeparator:     ", ",
			},
			PluralRules: pluralFR,
		},
		"de-DE": {
			Name:      "German",
			NativeName: "Deutsch",
			Format: LocaleFormat{
				DateFormat:        "2. Januar 2006",
				TimeFormat:        "15:04:05",
				DateTimeFormat:    "2. Januar 2006, 15:04:05",
				DecimalSeparator:  ",",
				ThousandSeparator: ".",
				CurrencySymbol:    "€",
				CurrencyCode:      "EUR",
				NumberFormat:      "#.##0,##",
				FirstWeekday:      time.Monday,
				ListSeparator:     ", ",
			},
			PluralRules: pluralDE,
		},
		"es-ES": {
			Name:      "Spanish",
			NativeName: "Español",
			Format: LocaleFormat{
				DateFormat:        "2 de enero de 2006",
				TimeFormat:        "15:04:05",
				DateTimeFormat:    "2 de enero de 2006, 15:04:05",
				DecimalSeparator:  ",",
				ThousandSeparator: ".",
				CurrencySymbol:    "€",
				CurrencyCode:      "EUR",
				NumberFormat:      "#.##0,##",
				FirstWeekday:      time.Monday,
				ListSeparator:     ", ",
			},
			PluralRules: pluralES,
		},
		"pt-BR": {
			Name:      "Portuguese (Brazil)",
			NativeName: "Português (Brasil)",
			Format: LocaleFormat{
				DateFormat:        "2 de janeiro de 2006",
				TimeFormat:        "15:04:05",
				DateTimeFormat:    "2 de janeiro de 2006 às 15:04:05",
				DecimalSeparator:  ",",
				ThousandSeparator: ".",
				CurrencySymbol:    "R$",
				CurrencyCode:      "BRL",
				NumberFormat:      "#.##0,##",
				FirstWeekday:      time.Sunday,
				ListSeparator:     ", ",
			},
			PluralRules: pluralPT,
		},
		"it-IT": {
			Name:      "Italian",
			NativeName: "Italiano",
			Format: LocaleFormat{
				DateFormat:        "2 gennaio 2006",
				TimeFormat:        "15:04:05",
				DateTimeFormat:    "2 gennaio 2006, ore 15:04:05",
				DecimalSeparator:  ",",
				ThousandSeparator: ".",
				CurrencySymbol:    "€",
				CurrencyCode:      "EUR",
				NumberFormat:      "#.##0,##",
				FirstWeekday:      time.Monday,
				ListSeparator:     ", ",
			},
			PluralRules: pluralIT,
		},
		"ru-RU": {
			Name:      "Russian",
			NativeName: "Русский",
			Format: LocaleFormat{
				DateFormat:        "2 января 2006 г.",
				TimeFormat:        "15:04:05",
				DateTimeFormat:    "2 января 2006 г., 15:04:05",
				DecimalSeparator:  ",",
				ThousandSeparator: " ",
				CurrencySymbol:    "₽",
				CurrencyCode:      "RUB",
				NumberFormat:      "# ##0,##",
				FirstWeekday:      time.Monday,
				ListSeparator:     ", ",
			},
			PluralRules: pluralRU,
		},
		"ar-SA": {
			Name:      "Arabic",
			NativeName: "العربية",
			Format: LocaleFormat{
				DateFormat:        "2 يناير 2006",
				TimeFormat:        "15:04:05",
				DateTimeFormat:    "2 يناير 2006، 15:04:05",
				DecimalSeparator:  "٫",
				ThousandSeparator: "٬",
				CurrencySymbol:    "ر.س",
				CurrencyCode:      "SAR",
				NumberFormat:      "#,##0.##",
				FirstWeekday:      time.Saturday,
				ListSeparator:     "، ",
			},
			PluralRules: pluralAR,
		},
		"th-TH": {
			Name:      "Thai",
			NativeName: "ไทย",
			Format: LocaleFormat{
				DateFormat:        "2 มกราคม 2549",
				TimeFormat:        "15:04:05",
				DateTimeFormat:    "2 มกราคม 2549 เวลา 15:04:05 น.",
				DecimalSeparator:  ".",
				ThousandSeparator: ",",
				CurrencySymbol:    "฿",
				CurrencyCode:      "THB",
				NumberFormat:      "#,##0.##",
				FirstWeekday:      time.Sunday,
				ListSeparator:     ", ",
			},
			PluralRules: pluralTH,
		},
		"vi-VN": {
			Name:      "Vietnamese",
			NativeName: "Tiếng Việt",
			Format: LocaleFormat{
				DateFormat:        "2 tháng 1 năm 2006",
				TimeFormat:        "15:04:05",
				DateTimeFormat:    "2 tháng 1 năm 2006, 15:04:05",
				DecimalSeparator:  ",",
				ThousandSeparator: ".",
				CurrencySymbol:    "₫",
				CurrencyCode:      "VND",
				NumberFormat:      "#.##0,##",
				FirstWeekday:      time.Sunday,
				ListSeparator:     ", ",
			},
			PluralRules: pluralVI,
		},
		"id-ID": {
			Name:      "Indonesian",
			NativeName: "Bahasa Indonesia",
			Format: LocaleFormat{
				DateFormat:        "2 Januari 2006",
				TimeFormat:        "15.04.05",
				DateTimeFormat:    "2 Januari 2006 pukul 15.04.05",
				DecimalSeparator:  ",",
				ThousandSeparator: ".",
				CurrencySymbol:    "Rp",
				CurrencyCode:      "IDR",
				NumberFormat:      "#.##0,##",
				FirstWeekday:      time.Sunday,
				ListSeparator:     ", ",
			},
			PluralRules: pluralID,
		},
		"ms-MY": {
			Name:      "Malay",
			NativeName: "Bahasa Melayu",
			Format: LocaleFormat{
				DateFormat:        "2 Januari 2006",
				TimeFormat:        "15:04:05",
				DateTimeFormat:    "2 Januari 2006, 15:04:05",
				DecimalSeparator:  ".",
				ThousandSeparator: ",",
				CurrencySymbol:    "RM",
				CurrencyCode:      "MYR",
				NumberFormat:      "#,##0.##",
				FirstWeekday:      time.Sunday,
				ListSeparator:     ", ",
			},
			PluralRules: pluralMS,
		},
		"tl-PH": {
			Name:      "Filipino",
			NativeName: "Filipino",
			Format: LocaleFormat{
				DateFormat:        "Enero 2, 2006",
				TimeFormat:        "3:04:05 PM",
				DateTimeFormat:    "Enero 2, 2006 3:04:05 PM",
				DecimalSeparator:  ".",
				ThousandSeparator: ",",
				CurrencySymbol:    "₱",
				CurrencyCode:      "PHP",
				NumberFormat:      "#,##0.##",
				FirstWeekday:      time.Sunday,
				ListSeparator:     ", ",
			},
			PluralRules: pluralTL,
		},
		"fa-IR": {
			Name:      "Persian (Iran)",
			NativeName: "فارسی",
			Format: LocaleFormat{
				DateFormat:        "۲ ژانویه ۲۰۰۶",
				TimeFormat:        "15:04:05",
				DateTimeFormat:    "۲ ژانویه ۲۰۰۶، ساعت 15:04:05",
				DecimalSeparator:  "٫",
				ThousandSeparator: "٬",
				CurrencySymbol:    "ریال",
				CurrencyCode:      "IRR",
				NumberFormat:      "#,##0.##",
				FirstWeekday:      time.Saturday,
				ListSeparator:     "، ",
			},
			PluralRules: pluralFA,
		},
		"he-IL": {
			Name:      "Hebrew",
			NativeName: "עברית",
			Format: LocaleFormat{
				DateFormat:        "2 בינואר 2006",
				TimeFormat:        "15:04:05",
				DateTimeFormat:    "2 בינואר 2006, 15:04:05",
				DecimalSeparator:  ".",
				ThousandSeparator: ",",
				CurrencySymbol:    "₪",
				CurrencyCode:      "ILS",
				NumberFormat:      "#,##0.##",
				FirstWeekday:      time.Sunday,
				ListSeparator:     ", ",
			},
			PluralRules: pluralHE,
		},
		"tr-TR": {
			Name:      "Turkish",
			NativeName: "Türkçe",
			Format: LocaleFormat{
				DateFormat:        "2 Ocak 2006",
				TimeFormat:        "15:04:05",
				DateTimeFormat:    "2 Ocak 2006, 15:04:05",
				DecimalSeparator:  ",",
				ThousandSeparator: ".",
				CurrencySymbol:    "₺",
				CurrencyCode:      "TRY",
				NumberFormat:      "#.##0,##",
				FirstWeekday:      time.Monday,
				ListSeparator:     ", ",
			},
			PluralRules: pluralTR,
		},
		"pl-PL": {
			Name:      "Polish",
			NativeName: "Polski",
			Format: LocaleFormat{
				DateFormat:        "2 stycznia 2006",
				TimeFormat:        "15:04:05",
				DateTimeFormat:    "2 stycznia 2006, godz. 15:04:05",
				DecimalSeparator:  ",",
				ThousandSeparator: " ",
				CurrencySymbol:    "zł",
				CurrencyCode:      "PLN",
				NumberFormat:      "# ##0,##",
				FirstWeekday:      time.Monday,
				ListSeparator:     ", ",
			},
			PluralRules: pluralPL,
		},
		"nl-NL": {
			Name:      "Dutch",
			NativeName: "Nederlands",
			Format: LocaleFormat{
				DateFormat:        "2 januari 2006",
				TimeFormat:        "15:04:05",
				DateTimeFormat:    "2 januari 2006 om 15:04:05",
				DecimalSeparator:  ",",
				ThousandSeparator: ".",
				CurrencySymbol:    "€",
				CurrencyCode:      "EUR",
				NumberFormat:      "#.##0,##",
				FirstWeekday:      time.Monday,
				ListSeparator:     ", ",
			},
			PluralRules: pluralNL,
		},
		"el-GR": {
			Name:      "Greek",
			NativeName: "Ελληνικά",
			Format: LocaleFormat{
				DateFormat:        "2 Ιανουαρίου 2006",
				TimeFormat:        "15:04:05",
				DateTimeFormat:    "2 Ιανουαρίου 2006, 15:04:05",
				DecimalSeparator:  ",",
				ThousandSeparator: ".",
				CurrencySymbol:    "€",
				CurrencyCode:      "EUR",
				NumberFormat:      "#.##0,##",
				FirstWeekday:      time.Monday,
				ListSeparator:     ", ",
			},
			PluralRules: pluralEL,
		},
		"cs-CZ": {
			Name:      "Czech",
			NativeName: "Čeština",
			Format: LocaleFormat{
				DateFormat:        "2. ledna 2006",
				TimeFormat:        "15:04:05",
				DateTimeFormat:    "2. ledna 2006 15:04:05",
				DecimalSeparator:  ",",
				ThousandSeparator: " ",
				CurrencySymbol:    "Kč",
				CurrencyCode:      "CZK",
				NumberFormat:      "# ##0,##",
				FirstWeekday:      time.Monday,
				ListSeparator:     ", ",
			},
			PluralRules: pluralCS,
		},
		"sv-SE": {
			Name:      "Swedish",
			NativeName: "Svenska",
			Format: LocaleFormat{
				DateFormat:        "2 januari 2006",
				TimeFormat:        "15:04:05",
				DateTimeFormat:    "2 januari 2006 kl. 15:04:05",
				DecimalSeparator:  ",",
				ThousandSeparator: " ",
				CurrencySymbol:    "kr",
				CurrencyCode:      "SEK",
				NumberFormat:      "# ##0,##",
				FirstWeekday:      time.Monday,
				ListSeparator:     ", ",
			},
			PluralRules: pluralSV,
		},
		"da-DK": {
			Name:      "Danish",
			NativeName: "Dansk",
			Format: LocaleFormat{
				DateFormat:        "2. januar 2006",
				TimeFormat:        "15:04:05",
				DateTimeFormat:    "2. januar 2006 kl. 15:04:05",
				DecimalSeparator:  ",",
				ThousandSeparator: ".",
				CurrencySymbol:    "kr",
				CurrencyCode:      "DKK",
				NumberFormat:      "#.##0,##",
				FirstWeekday:      time.Monday,
				ListSeparator:     ", ",
			},
			PluralRules: pluralDA,
		},
		"fi-FI": {
			Name:      "Finnish",
			NativeName: "Suomi",
			Format: LocaleFormat{
				DateFormat:        "2. tammikuuta 2006",
				TimeFormat:        "15:04:05",
				DateTimeFormat:    "2. tammikuuta 2006 klo 15:04:05",
				DecimalSeparator:  ",",
				ThousandSeparator: " ",
				CurrencySymbol:    "€",
				CurrencyCode:      "EUR",
				NumberFormat:      "# ##0,##",
				FirstWeekday:      time.Monday,
				ListSeparator:     ", ",
			},
			PluralRules: pluralFI,
		},
		"no-NO": {
			Name:      "Norwegian",
			NativeName: "Norsk",
			Format: LocaleFormat{
				DateFormat:        "2. januar 2006",
				TimeFormat:        "15:04:05",
				DateTimeFormat:    "2. januar 2006 kl. 15:04:05",
				DecimalSeparator:  ",",
				ThousandSeparator: " ",
				CurrencySymbol:    "kr",
				CurrencyCode:      "NOK",
				NumberFormat:      "# ##0,##",
				FirstWeekday:      time.Monday,
				ListSeparator:     ", ",
			},
			PluralRules: pluralNO,
		},
		"hu-HU": {
			Name:      "Hungarian",
			NativeName: "Magyar",
			Format: LocaleFormat{
				DateFormat:        "2006. január 2.",
				TimeFormat:        "15:04:05",
				DateTimeFormat:    "2006. január 2. 15:04:05",
				DecimalSeparator:  ",",
				ThousandSeparator: " ",
				CurrencySymbol:    "Ft",
				CurrencyCode:      "HUF",
				NumberFormat:      "# ##0,##",
				FirstWeekday:      time.Monday,
				ListSeparator:     ", ",
			},
			PluralRules: pluralHU,
		},
		"ro-RO": {
			Name:      "Romanian",
			NativeName: "Română",
			Format: LocaleFormat{
				DateFormat:        "2 ianuarie 2006",
				TimeFormat:        "15:04:05",
				DateTimeFormat:    "2 ianuarie 2006, ora 15:04:05",
				DecimalSeparator:  ",",
				ThousandSeparator: ".",
				CurrencySymbol:    "lei",
				CurrencyCode:      "RON",
				NumberFormat:      "#.##0,##",
				FirstWeekday:      time.Monday,
				ListSeparator:     ", ",
			},
			PluralRules: pluralRO,
		},
		"uk-UA": {
			Name:      "Ukrainian",
			NativeName: "Українська",
			Format: LocaleFormat{
				DateFormat:        "2 січня 2006 р.",
				TimeFormat:        "15:04:05",
				DateTimeFormat:    "2 січня 2006 р., 15:04:05",
				DecimalSeparator:  ",",
				ThousandSeparator: " ",
				CurrencySymbol:    "₴",
				CurrencyCode:      "UAH",
				NumberFormat:      "# ##0,##",
				FirstWeekday:      time.Monday,
				ListSeparator:     ", ",
			},
			PluralRules: pluralUK,
		},
		"bg-BG": {
			Name:      "Bulgarian",
			NativeName: "Български",
			Format: LocaleFormat{
				DateFormat:        "2 януари 2006 г.",
				TimeFormat:        "15:04:05",
				DateTimeFormat:    "2 януари 2006 г., 15:04:05",
				DecimalSeparator:  ",",
				ThousandSeparator: " ",
				CurrencySymbol:    "лв",
				CurrencyCode:      "BGN",
				NumberFormat:      "# ##0,##",
				FirstWeekday:      time.Monday,
				ListSeparator:     ", ",
			},
			PluralRules: pluralBG,
		},
		"hr-HR": {
			Name:      "Croatian",
			NativeName: "Hrvatski",
			Format: LocaleFormat{
				DateFormat:        "2. siječnja 2006.",
				TimeFormat:        "15:04:05",
				DateTimeFormat:    "2. siječnja 2006. 15:04:05",
				DecimalSeparator:  ",",
				ThousandSeparator: ".",
				CurrencySymbol:    "€",
				CurrencyCode:      "EUR",
				NumberFormat:      "#.##0,##",
				FirstWeekday:      time.Monday,
				ListSeparator:     ", ",
			},
			PluralRules: pluralHR,
		},
		"sk-SK": {
			Name:      "Slovak",
			NativeName: "Slovenčina",
			Format: LocaleFormat{
				DateFormat:        "2. januára 2006",
				TimeFormat:        "15:04:05",
				DateTimeFormat:    "2. januára 2006 15:04:05",
				DecimalSeparator:  ",",
				ThousandSeparator: " ",
				CurrencySymbol:    "€",
				CurrencyCode:      "EUR",
				NumberFormat:      "# ##0,##",
				FirstWeekday:      time.Monday,
				ListSeparator:     ", ",
			},
			PluralRules: pluralSK,
		},
		"sl-SI": {
			Name:      "Slovenian",
			NativeName: "Slovenščina",
			Format: LocaleFormat{
				DateFormat:        "2. januar 2006",
				TimeFormat:        "15:04:05",
				DateTimeFormat:    "2. januar 2006 ob 15:04:05",
				DecimalSeparator:  ",",
				ThousandSeparator: ".",
				CurrencySymbol:    "€",
				CurrencyCode:      "EUR",
				NumberFormat:      "#.##0,##",
				FirstWeekday:      time.Monday,
				ListSeparator:     ", ",
			},
			PluralRules: pluralSL,
		},
	}
}

func getLocaleData() map[string]*LocaleData {
	localeDataOnce.Do(initLocaleData)
	return localeData
}

func GetLocaleInfo(lang string) *LocaleData {
	localeDataOnce.Do(initLocaleData)
	if data, ok := localeData[lang]; ok {
		return data
	}
	if data, ok := localeData["en-US"]; ok {
		return data
	}
	return nil
}

func FormatNumber(num float64, lang string) string {
	localeDataOnce.Do(initLocaleData)
	locale := GetLocaleInfo(lang)
	if locale == nil {
		return fmt.Sprintf("%.2f", num)
	}

	format := locale.Format
	absNum := num
	if num < 0 {
		absNum = -num
	}

	intPart := int(absNum)
	decPart := absNum - float64(intPart)

	intStr := fmt.Sprintf("%d", intPart)
	var result strings.Builder

	for i, r := range intStr {
		if i > 0 && (len(intStr)-i)%3 == 0 {
			result.WriteString(format.ThousandSeparator)
		}
		result.WriteRune(r)
	}

	if decPart > 0 {
		result.WriteString(format.DecimalSeparator)
		decStr := fmt.Sprintf("%.2f", decPart)[2:]
		result.WriteString(decStr)
	}

	if num < 0 {
		return "-" + result.String()
	}
	return result.String()
}

func FormatCurrency(amount float64, lang string) string {
	locale := GetLocaleInfo(lang)
	if locale == nil {
		return fmt.Sprintf("%.2f", amount)
	}

	formatted := FormatNumber(amount, lang)
	return fmt.Sprintf("%s%s", locale.Format.CurrencySymbol, formatted)
}

func FormatDate(t time.Time, lang string) string {
	locale := GetLocaleInfo(lang)
	if locale == nil {
		return t.Format("2006-01-02")
	}

	format := locale.Format.DateFormat
	format = strings.ReplaceAll(format, "2006", fmt.Sprintf("%d", t.Year()))
	format = strings.ReplaceAll(format, "06", fmt.Sprintf("%02d", t.Year()%100))
	format = strings.ReplaceAll(format, "January", t.Month().String())
	format = strings.ReplaceAll(format, "janvier", t.Month().String())
	format = strings.ReplaceAll(format, "Januar", t.Month().String())
	format = strings.ReplaceAll(format, "enero", t.Month().String())
	format = strings.ReplaceAll(format, "gennaio", t.Month().String())
	format = strings.ReplaceAll(format, "1", fmt.Sprintf("%d", t.Day()))

	return t.Format(format)
}

func FormatTime(t time.Time, lang string) string {
	locale := GetLocaleInfo(lang)
	if locale == nil {
		return t.Format("15:04:05")
	}

	format := locale.Format.TimeFormat
	isPM := t.Hour() >= 12
	hour := t.Hour()
	if strings.Contains(format, "PM") || strings.Contains(format, "pm") {
		if isPM && hour > 12 {
			hour -= 12
		}
		if !isPM && hour == 0 {
			hour = 12
		}
		ampm := "AM"
		if isPM {
			ampm = "PM"
		}
		format = strings.ReplaceAll(format, "PM", ampm)
		format = strings.ReplaceAll(format, "pm", strings.ToLower(ampm))
		format = strings.ReplaceAll(format, "3", fmt.Sprintf("%d", hour))
	} else {
		format = strings.ReplaceAll(format, "15", fmt.Sprintf("%02d", t.Hour()))
	}
	format = strings.ReplaceAll(format, "04", fmt.Sprintf("%02d", t.Minute()))
	format = strings.ReplaceAll(format, "05", fmt.Sprintf("%02d", t.Second()))

	return format
}

func FormatDateTime(t time.Time, lang string) string {
	locale := GetLocaleInfo(lang)
	if locale == nil {
		return t.Format("2006-01-02 15:04:05")
	}

	dateStr := FormatDate(t, lang)
	timeStr := FormatTime(t, lang)

	result := locale.Format.DateTimeFormat
	result = strings.ReplaceAll(result, "2006年1月2日", dateStr)
	result = strings.ReplaceAll(result, "January 2, 2006", dateStr)
	result = strings.ReplaceAll(result, "2 janvier 2006", dateStr)
	result = strings.ReplaceAll(result, "2. Januar 2006", dateStr)
	result = strings.ReplaceAll(result, "2 de enero de 2006", dateStr)
	result = strings.ReplaceAll(result, "2 gennaio 2006", dateStr)
	result = strings.ReplaceAll(result, "2 января 2006 г.", dateStr)
	result = strings.ReplaceAll(result, "2 يناير 2006", dateStr)
	result = strings.ReplaceAll(result, "2 มกราคม 2549", dateStr)
	result = strings.ReplaceAll(result, "2 tháng 1 năm 2006", dateStr)
	result = strings.ReplaceAll(result, "2 Januari 2006", dateStr)
	result = strings.ReplaceAll(result, "2 Ocak 2006", dateStr)
	result = strings.ReplaceAll(result, "2 stycznia 2006", dateStr)
	result = strings.ReplaceAll(result, "2. ledna 2006", dateStr)
	result = strings.ReplaceAll(result, "2. januar 2006", dateStr)
	result = strings.ReplaceAll(result, "2. tammikuuta 2006", dateStr)
	result = strings.ReplaceAll(result, "2. január 2006", dateStr)
	result = strings.ReplaceAll(result, "2006. január 2.", dateStr)
	result = strings.ReplaceAll(result, "2 ianuarie 2006", dateStr)
	result = strings.ReplaceAll(result, "2 січня 2006 р.", dateStr)
	result = strings.ReplaceAll(result, "2 януари 2006 г.", dateStr)
	result = strings.ReplaceAll(result, "2. siječnja 2006.", dateStr)
	result = strings.ReplaceAll(result, "2. januára 2006", dateStr)
	result = strings.ReplaceAll(result, "2. januar 2006", dateStr)
	result = strings.ReplaceAll(result, "2006년 1월 2일", dateStr)

	result = strings.ReplaceAll(result, "15:04:05", timeStr)
	result = strings.ReplaceAll(result, "3:04:05 PM", timeStr)
	result = strings.ReplaceAll(result, "15.04.05", timeStr)

	return result
}

func FormatDuration(d time.Duration, lang string) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

func FormatRelativeTime(t time.Time, lang string) string {
	locale := GetLocaleInfo(lang)
	if locale == nil {
		lang = "en-US"
		locale = GetLocaleInfo(lang)
	}

	now := time.Now()
	diff := now.Sub(t)

	if diff < 0 {
		diff = -diff
	}

	seconds := int(diff.Seconds())
	minutes := seconds / 60
	hours := minutes / 60
	days := hours / 24

	switch {
	case seconds < 60:
		return Translate(lang, "just_now")
	case minutes < 60:
		return Translate(lang, "minutes_ago", minutes)
	case hours < 24:
		return Translate(lang, "hours_ago", hours)
	case days < 7:
		return Translate(lang, "days_ago", days)
	default:
		return FormatDate(t, lang)
	}
}

func pluralNone(n int) string {
	return "other"
}

func pluralEN(n int) string {
	if n == 1 {
		return "one"
	}
	return "other"
}

func pluralFR(n int) string {
	if n > 1 {
		return "other"
	}
	return "one"
}

func pluralDE(n int) string {
	if n != 1 {
		return "other"
	}
	return "one"
}

func pluralES(n int) string {
	if n != 1 {
		return "other"
	}
	return "one"
}

func pluralPT(n int) string {
	if n > 1 {
		return "other"
	}
	return "one"
}

func pluralIT(n int) string {
	if n != 1 {
		return "other"
	}
	return "one"
}

func pluralRU(n int) string {
	lastTwo := n % 100
	lastOne := n % 10
	if lastTwo >= 11 && lastTwo <= 14 {
		return "other"
	}
	if lastOne == 1 {
		return "one"
	}
	if lastOne >= 2 && lastOne <= 4 {
		return "few"
	}
	return "other"
}

func pluralAR(n int) string {
	if n == 1 {
		return "one"
	}
	if n == 2 {
		return "two"
	}
	return "other"
}

func pluralTH(n int) string {
	return "other"
}

func pluralVI(n int) string {
	return "other"
}

func pluralID(n int) string {
	return "other"
}

func pluralMS(n int) string {
	return "other"
}

func pluralTL(n int) string {
	if n == 1 {
		return "one"
	}
	return "other"
}

func pluralFA(n int) string {
	if n == 1 {
		return "one"
	}
	return "other"
}

func pluralHE(n int) string {
	if n == 1 {
		return "one"
	}
	return "other"
}

func pluralTR(n int) string {
	if n != 1 {
		return "other"
	}
	return "one"
}

func pluralPL(n int) string {
	lastTwo := n % 100
	lastOne := n % 10
	if lastTwo >= 12 && lastTwo <= 14 {
		return "other"
	}
	if lastOne == 1 {
		return "one"
	}
	if lastOne >= 2 && lastOne <= 4 {
		return "few"
	}
	return "other"
}

func pluralNL(n int) string {
	if n != 1 {
		return "other"
	}
	return "one"
}

func pluralEL(n int) string {
	if n != 1 {
		return "other"
	}
	return "one"
}

func pluralCS(n int) string {
	if n == 1 {
		return "one"
	}
	if n >= 2 && n <= 4 {
		return "few"
	}
	return "other"
}

func pluralSV(n int) string {
	if n != 1 {
		return "other"
	}
	return "one"
}

func pluralDA(n int) string {
	if n != 1 {
		return "other"
	}
	return "one"
}

func pluralFI(n int) string {
	if n != 1 {
		return "other"
	}
	return "one"
}

func pluralNO(n int) string {
	if n != 1 {
		return "other"
	}
	return "one"
}

func pluralHU(n int) string {
	if n == 1 {
		return "one"
	}
	return "other"
}

func pluralRO(n int) string {
	lastTwo := n % 100
	if n == 1 {
		return "one"
	}
	if lastTwo >= 1 && lastTwo <= 19 {
		return "few"
	}
	return "other"
}

func pluralUK(n int) string {
	lastTwo := n % 100
	lastOne := n % 10
	if lastTwo >= 11 && lastTwo <= 14 {
		return "other"
	}
	if lastOne == 1 {
		return "one"
	}
	if lastOne >= 2 && lastOne <= 4 {
		return "few"
	}
	return "other"
}

func pluralBG(n int) string {
	return "other"
}

func pluralHR(n int) string {
	lastTwo := n % 100
	lastOne := n % 10
	if lastTwo >= 12 && lastTwo <= 14 {
		return "other"
	}
	if lastOne == 1 {
		return "one"
	}
	if lastOne >= 2 && lastOne <= 4 {
		return "few"
	}
	return "other"
}

func pluralSK(n int) string {
	if n == 1 {
		return "one"
	}
	if n >= 2 && n <= 4 {
		return "few"
	}
	return "other"
}

func pluralSL(n int) string {
	lastTwo := n % 100
	if lastTwo == 1 {
		return "one"
	}
	if lastTwo == 2 {
		return "two"
	}
	if lastTwo >= 3 && lastTwo <= 4 {
		return "few"
	}
	return "other"
}
