package i18n

import (
	"fmt"
	"net"
	"strings"
	"time"
)

type TimezoneInfo struct {
	Name         string `json:"name"`
	Offset       string `json:"offset"`
	OffsetHours  int    `json:"offset_hours"`
	Zone         string `json:"zone"`
	Region       string `json:"region"`
	CountryCode  string `json:"country_code"`
}

var (
	defaultTimezone    = "Asia/Shanghai"
	supportedTimezones = []string{
		"Asia/Shanghai",
		"America/New_York",
		"America/Los_Angeles",
		"Europe/London",
		"Europe/Paris",
		"Europe/Berlin",
		"Asia/Tokyo",
		"Asia/Seoul",
		"Australia/Sydney",
		"Pacific/Auckland",
		"Asia/Dubai",
		"Asia/Jerusalem",
		"America/Sao_Paulo",
		"America/Mexico_City",
		"Asia/Kolkata",
		"Asia/Singapore",
		"Asia/Hong_Kong",
		"Asia/Bangkok",
		"Asia/Jakarta",
		"Asia/Manila",
		"Asia/Kuala_Lumpur",
		"Asia/Ho_Chi_Minh",
		"Asia/Taipei",
		"Europe/Madrid",
		"Europe/Rome",
		"Europe/Moscow",
		"Europe/Istanbul",
		"Europe/Amsterdam",
		"Europe/Brussels",
		"Europe/Vienna",
		"Europe/Stockholm",
		"Europe/Oslo",
		"Europe/Copenhagen",
		"Europe/Helsinki",
		"Europe/Warsaw",
		"Europe/Prague",
		"Europe/Athens",
		"America/Toronto",
		"America/Vancouver",
		"America/Chicago",
		"America/Denver",
		"America/Phoenix",
		"America/Anchorage",
		"America/Honolulu",
		"America/Santiago",
		"America/Buenos_Aires",
		"Africa/Cairo",
		"Africa/Johannesburg",
		"Africa/Lagos",
		"Africa/Nairobi",
		"Asia/Riyadh",
		"Asia/Tehran",
		"Asia/Karachi",
		"Asia/Dhaka",
		"Asia/Kathmandu",
		"Asia/Colombo",
		"Asia/Kabul",
		"Asia/Tashkent",
		"Asia/Almaty",
		"Asia/Baku",
		"Asia/Tbilisi",
		"Asia/Yerevan",
		"Europe/Kiev",
		"Europe/Minsk",
		"Europe/Bucharest",
		"Europe/Sofia",
		"Europe/Belgrade",
		"Europe/Zagreb",
		"Europe/Sarajevo",
		"Europe/Lisbon",
		"Europe/Dublin",
		"Europe/Bratislava",
		"Europe/Ljubljana",
		"Europe/Tallinn",
		"Europe/Riga",
		"Europe/Vilnius",
	}
)

func SetDefaultTimezone(tz string) error {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return err
	}
	defaultTimezone = tz
	time.Local = loc
	return nil
}

func GetDefaultTimezone() string {
	return defaultTimezone
}

func GetSupportedTimezones() []string {
	return append([]string{}, supportedTimezones...)
}

func IsSupportedTimezone(tz string) bool {
	for _, st := range supportedTimezones {
		if st == tz {
			return true
		}
	}
	return false
}

func GetLocation(tz string) *time.Location {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc, _ = time.LoadLocation(defaultTimezone)
	}
	return loc
}

func ConvertTime(t time.Time, fromTz, toTz string) time.Time {
	fromLoc := GetLocation(fromTz)
	toLoc := GetLocation(toTz)
	return t.In(fromLoc).In(toLoc)
}

func FormatTime(t time.Time, format, tz string) string {
	loc := GetLocation(tz)
	return t.In(loc).Format(format)
}

func FormatTimeLocal(t time.Time, format string) string {
	return t.Format(format)
}

func GetTimezoneInfo(tz string) TimezoneInfo {
	loc := GetLocation(tz)
	now := time.Now().In(loc)
	_, offset := now.Zone()

	hours := offset / 3600
	minutes := (offset % 3600) / 60

	var offsetStr string
	if minutes == 0 {
		offsetStr = fmt.Sprintf("UTC+%d", hours)
	} else {
		offsetStr = fmt.Sprintf("UTC+%d:%02d", hours, minutes)
	}

	parts := splitTimezone(tz)
	region := ""
	if len(parts) > 0 {
		region = parts[0]
	}

	return TimezoneInfo{
		Name:        tz,
		Offset:      offsetStr,
		OffsetHours: hours,
		Zone:        tz,
		Region:      region,
		CountryCode: getCountryCode(tz),
	}
}

func splitTimezone(tz string) []string {
	var parts []string
	current := ""
	for _, c := range tz {
		if c == '/' || c == '_' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func getCountryCode(tz string) string {
	countryMap := map[string]string{
		"Asia/Shanghai":       "CN",
		"Asia/Hong_Kong":      "HK",
		"Asia/Tokyo":          "JP",
		"Asia/Seoul":          "KR",
		"Asia/Singapore":      "SG",
		"Asia/Bangkok":        "TH",
		"Asia/Jakarta":        "ID",
		"Asia/Manila":         "PH",
		"Asia/Kuala_Lumpur":   "MY",
		"Asia/Ho_Chi_Minh":    "VN",
		"Asia/Taipei":         "TW",
		"Asia/Kolkata":        "IN",
		"Asia/Dubai":          "AE",
		"Asia/Riyadh":         "SA",
		"Asia/Jerusalem":      "IL",
		"Asia/Tehran":         "IR",
		"Asia/Karachi":        "PK",
		"Asia/Dhaka":          "BD",
		"Asia/Kathmandu":      "NP",
		"Asia/Colombo":        "LK",
		"Asia/Kabul":          "AF",
		"Asia/Tashkent":       "UZ",
		"Asia/Almaty":         "KZ",
		"Asia/Baku":           "AZ",
		"Asia/Tbilisi":        "GE",
		"Asia/Yerevan":        "AM",
		"Europe/London":        "GB",
		"Europe/Paris":         "FR",
		"Europe/Berlin":        "DE",
		"Europe/Madrid":        "ES",
		"Europe/Rome":          "IT",
		"Europe/Moscow":        "RU",
		"Europe/Istanbul":      "TR",
		"Europe/Amsterdam":      "NL",
		"Europe/Brussels":      "BE",
		"Europe/Vienna":        "AT",
		"Europe/Stockholm":     "SE",
		"Europe/Oslo":          "NO",
		"Europe/Copenhagen":    "DK",
		"Europe/Helsinki":      "FI",
		"Europe/Warsaw":        "PL",
		"Europe/Prague":        "CZ",
		"Europe/Athens":        "GR",
		"Europe/Kiev":          "UA",
		"Europe/Minsk":         "BY",
		"Europe/Bucharest":     "RO",
		"Europe/Sofia":         "BG",
		"Europe/Belgrade":      "RS",
		"Europe/Zagreb":        "HR",
		"Europe/Sarajevo":      "BA",
		"Europe/Lisbon":        "PT",
		"Europe/Dublin":        "IE",
		"Europe/Bratislava":     "SK",
		"Europe/Ljubljana":     "SI",
		"Europe/Tallinn":       "EE",
		"Europe/Riga":          "LV",
		"Europe/Vilnius":       "LT",
		"America/New_York":     "US",
		"America/Los_Angeles":  "US",
		"America/Chicago":      "US",
		"America/Denver":       "US",
		"America/Phoenix":      "US",
		"America/Anchorage":    "US",
		"America/Honolulu":     "US",
		"America/Toronto":      "CA",
		"America/Vancouver":    "CA",
		"America/Mexico_City":  "MX",
		"America/Sao_Paulo":    "BR",
		"America/Santiago":     "CL",
		"America/Buenos_Aires": "AR",
		"Australia/Sydney":     "AU",
		"Pacific/Auckland":     "NZ",
		"Africa/Cairo":          "EG",
		"Africa/Johannesburg":  "ZA",
		"Africa/Lagos":         "NG",
		"Africa/Nairobi":       "KE",
	}

	if code, ok := countryMap[tz]; ok {
		return code
	}
	return ""
}

func DetectTimezoneFromOffset(offset int) string {
	offsetMap := map[int]string{
		-12: "Pacific/Baker_Island",
		-11: "Pacific/Samoa",
		-10: "Pacific/Honolulu",
		-9:  "America/Anchorage",
		-8:  "America/Los_Angeles",
		-7:  "America/Denver",
		-6:  "America/Chicago",
		-5:  "America/New_York",
		-4:  "America/Santiago",
		-3:  "America/Sao_Paulo",
		-2:  "Atlantic/South_Georgia",
		-1:  "Atlantic/Azores",
		0:   "Europe/London",
		1:   "Europe/Paris",
		2:   "Europe/Berlin",
		3:   "Europe/Moscow",
		4:   "Asia/Dubai",
		5:   "Asia/Karachi",
		6:   "Asia/Kolkata",
		7:   "Asia/Bangkok",
		8:   "Asia/Shanghai",
		9:   "Asia/Tokyo",
		10:  "Australia/Sydney",
		11:  "Pacific/Noumea",
		12:  "Pacific/Auckland",
	}

	if tz, ok := offsetMap[offset]; ok {
		return tz
	}
	return defaultTimezone
}

func GetTimezonesByRegion() map[string][]string {
	regions := make(map[string][]string)

	for _, tz := range supportedTimezones {
		parts := splitTimezone(tz)
		if len(parts) >= 1 {
			region := parts[0]
			regions[region] = append(regions[region], tz)
		}
	}

	return regions
}

func ConvertTimeToUTC(t time.Time, fromTz string) time.Time {
	loc := GetLocation(fromTz)
	return t.In(loc).UTC()
}

func ConvertTimeFromUTC(t time.Time, toTz string) time.Time {
	return t.UTC().In(GetLocation(toTz))
}

func GetCurrentTimeInTimezone(tz string) time.Time {
	return time.Now().In(GetLocation(tz))
}

func GetTimezoneOffset(tz string) (int, error) {
	loc := GetLocation(tz)
	now := time.Now().In(loc)
	_, offset := now.Zone()
	return offset, nil
}

func GetTimezoneAbbreviation(tz string) string {
	loc := GetLocation(tz)
	now := time.Now().In(loc)
	name, _ := now.Zone()
	return name
}

func FormatTimeWithTimezone(t time.Time, format string, tz string) string {
	loc := GetLocation(tz)
	return t.In(loc).Format(format)
}

func GetRelativeTimezone(tz1, tz2 string) string {
	loc1 := GetLocation(tz1)
	loc2 := GetLocation(tz2)

	now := time.Now()
	t1 := now.In(loc1)
	t2 := now.In(loc2)

	_, offset1 := t1.Zone()
	_, offset2 := t2.Zone()

	diff := offset2 - offset1
	hours := diff / 3600

	if hours == 0 {
		return "Same timezone"
	} else if hours > 0 {
		return fmt.Sprintf("+%d hours", hours)
	} else {
		return fmt.Sprintf("%d hours", hours)
	}
}

func IsDST(tz string) bool {
	loc := GetLocation(tz)
	now := time.Now().In(loc)
	_, offset := now.Zone()

	t1 := time.Date(1, 7, 1, 12, 0, 0, 0, loc)
	t2 := time.Date(1, 1, 1, 12, 0, 0, 0, loc)

	_, offset1 := t1.Zone()
	_, offset2 := t2.Zone()

	return offset != offset1 && offset != offset2 && (offset == offset1 || offset == offset2)
}

func GetDSTInfo(tz string) map[string]interface{} {
	loc := GetLocation(tz)

	t1 := time.Date(1, 7, 1, 12, 0, 0, 0, loc)
	t2 := time.Date(1, 1, 1, 12, 0, 0, 0, loc)

	name1, offset1 := t1.Zone()
	name2, offset2 := t2.Zone()

	return map[string]interface{}{
		"is_dst":         IsDST(tz),
		"standard_name":  name2,
		"standard_offset": offset2,
		"dst_name":       name1,
		"dst_offset":     offset1,
	}
}

func GetTimezonesByOffset() map[int][]string {
	result := make(map[int][]string)

	for _, tz := range supportedTimezones {
		offset, err := GetTimezoneOffset(tz)
		if err != nil {
			continue
		}

		hours := offset / 3600
		result[hours] = append(result[hours], tz)
	}

	return result
}

func GetTimezoneOffsetMinutes(tz string) (int, error) {
	loc := GetLocation(tz)
	now := time.Now().In(loc)
	_, offset := now.Zone()
	return offset / 60, nil
}

func IsTimezoneInDSTRange(tz string, t time.Time) bool {
	loc := GetLocation(tz)

	jan := time.Date(t.Year(), 1, 1, 12, 0, 0, 0, loc)
	jul := time.Date(t.Year(), 7, 1, 12, 0, 0, 0, loc)

	_, janOffset := jan.Zone()
	_, julOffset := jul.Zone()

	standardOffset := janOffset
	if janOffset != julOffset {
		if janOffset < julOffset {
			standardOffset = janOffset
		} else {
			standardOffset = julOffset
		}
	}

	_, currentOffset := t.Zone()

	return currentOffset != standardOffset
}

func ConvertTimeWithDST(t time.Time, fromTz, toTz string) time.Time {
	fromLoc := GetLocation(fromTz)
	toLoc := GetLocation(toTz)

	tInFrom := t.In(fromLoc)

	_, fromOffset := tInFrom.Zone()
	_, toOffset := time.Now().In(toLoc).Zone()

	diff := toOffset - fromOffset

	return tInFrom.Add(time.Duration(diff) * time.Second).In(toLoc)
}

func GetNextDSTTransition(tz string, after time.Time) (time.Time, error) {
	for i := 0; i < 365; i++ {
		check := after.AddDate(0, 0, i)
		if IsTimezoneInDSTRange(tz, check) != IsTimezoneInDSTRange(tz, check.Add(time.Hour)) {
			return check, nil
		}
	}

	return time.Time{}, fmt.Errorf("no DST transition found within a year")
}

func GetPreviousDSTTransition(tz string, before time.Time) (time.Time, error) {
	for i := 0; i < 365; i++ {
		check := before.AddDate(0, 0, -i)
		if i > 0 && IsTimezoneInDSTRange(tz, check) != IsTimezoneInDSTRange(tz, check.Add(-time.Hour)) {
			return check, nil
		}
	}

	return time.Time{}, fmt.Errorf("no DST transition found within a year")
}

var timezoneAliases = map[string]string{
	"UTC":             "UTC",
	"Z":               "UTC",
	"GMT":             "GMT",
	"EST":             "America/New_York",
	"EDT":             "America/New_York",
	"CST":             "America/Chicago",
	"CDT":             "America/Chicago",
	"MST":             "America/Denver",
	"MDT":             "America/Denver",
	"PST":             "America/Los_Angeles",
	"PDT":             "America/Los_Angeles",
	"AKST":            "America/Anchorage",
	"AKDT":            "America/Anchorage",
	"HST":             "America/Honolulu",
	"AST":             "America/Santiago",
	"ADT":             "America/Santiago",
	"BST":             "Europe/London",
	"CET":             "Europe/Berlin",
	"CEST":            "Europe/Berlin",
	"EET":             "Europe/Istanbul",
	"EEST":            "Europe/Istanbul",
	"MSK":             "Europe/Moscow",
	"WIB":             "Asia/Jakarta",
	"WITA":            "Asia/Makassar",
	"WIT":             "Asia/Jayapura",
	"AWST":            "Australia/Perth",
	"ACST":            "Australia/Adelaide",
	"AEST":            "Australia/Sydney",
	"AEDT":            "Australia/Sydney",
	"NZST":            "Pacific/Auckland",
	"NZDST":           "Pacific/Auckland",
	"JST":             "Asia/Tokyo",
	"KST":             "Asia/Seoul",
	"HKT":             "Asia/Hong_Kong",
	"CST8":            "Asia/Shanghai",
	"CT":              "Asia/Shanghai",
	"ICT":             "Asia/Bangkok",
	"IST":             "Asia/Kolkata",
	"PKT":             "Asia/Karachi",
	"ART":             "America/Buenos_Aires",
	"BRT":             "America/Sao_Paulo",
	"MSD":             "Europe/Moscow",
	"CAT":             "Africa/Johannesburg",
	"EAT":             "Africa/Nairobi",
	"SAST":            "Africa/Johannesburg",
}

func ResolveTimezoneAlias(alias string) string {
	alias = strings.ToUpper(strings.TrimSpace(alias))
	
	if resolved, ok := timezoneAliases[alias]; ok {
		return resolved
	}
	
	return alias
}

func NormalizeTimezone(tz string) string {
	tz = strings.TrimSpace(tz)
	
	if tz == "" {
		return defaultTimezone
	}
	
	resolved := ResolveTimezoneAlias(tz)
	
	if IsSupportedTimezone(resolved) {
		return resolved
	}
	
	_, err := time.LoadLocation(resolved)
	if err == nil {
		return resolved
	}
	
	return defaultTimezone
}

func ValidateTimezone(tz string) error {
	_, err := time.LoadLocation(tz)
	if err != nil {
		return fmt.Errorf("invalid timezone: %s", tz)
	}
	return nil
}

func DetectTimezoneFromIP(ip string) (string, error) {
	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return "", fmt.Errorf("invalid IP address")
	}
	
	if ipAddr.IsPrivate() {
		return defaultTimezone, nil
	}
	
	return defaultTimezone, nil
}

func DetectTimezoneFromLocale(lang string) string {
	lang = strings.ToLower(lang)
	
	localeTzMap := map[string]string{
		"zh-cn":     "Asia/Shanghai",
		"zh-tw":     "Asia/Taipei",
		"zh-hk":     "Asia/Hong_Kong",
		"en-us":     "America/New_York",
		"en-gb":     "Europe/London",
		"en-au":     "Australia/Sydney",
		"en-ca":     "America/Toronto",
		"ja-jp":     "Asia/Tokyo",
		"ko-kr":     "Asia/Seoul",
		"fr-fr":     "Europe/Paris",
		"de-de":     "Europe/Berlin",
		"es-es":     "Europe/Madrid",
		"pt-br":     "America/Sao_Paulo",
		"pt-pt":     "Europe/Lisbon",
		"it-it":     "Europe/Rome",
		"ru-ru":     "Europe/Moscow",
		"ar-sa":     "Asia/Riyadh",
		"fa-ir":     "Asia/Tehran",
		"he-il":     "Asia/Jerusalem",
		"ur-pk":     "Asia/Karachi",
		"hi-in":     "Asia/Kolkata",
		"vi-vn":     "Asia/Ho_Chi_Minh",
		"th-th":     "Asia/Bangkok",
		"id-id":     "Asia/Jakarta",
		"tr-tr":     "Europe/Istanbul",
		"ms-my":     "Asia/Kuala_Lumpur",
		"bn-bd":     "Asia/Dhaka",
		"ta-in":     "Asia/Kolkata",
	}
	
	if tz, ok := localeTzMap[lang]; ok {
		return tz
	}
	
	parts := strings.Split(lang, "-")
	if len(parts) > 0 {
		for l := range localeTzMap {
			if strings.HasPrefix(l, parts[0]) {
				return localeTzMap[l]
			}
		}
	}
	
	return defaultTimezone
}

func GetTimezoneDisplayName(tz string, lang string) string {
	tzNames := map[string]map[string]string{
		"Asia/Shanghai": {
			"zh-CN": "中国标准时间",
			"en-US": "China Standard Time",
			"ja-JP": "中国標準時",
			"ko-KR": "중국 표준 시간",
		},
		"America/New_York": {
			"zh-CN": "美国东部时间",
			"en-US": "Eastern Time",
			"ja-JP": "アメリカ東部時間",
		},
		"Europe/London": {
			"zh-CN": "英国时间",
			"en-US": "Greenwich Mean Time",
			"ja-JP": "英国時間",
		},
		"Asia/Tokyo": {
			"zh-CN": "日本标准时间",
			"en-US": "Japan Standard Time",
			"ja-JP": "日本標準時",
			"ko-KR": "일본 표준 시간",
		},
		"Asia/Seoul": {
			"zh-CN": "韩国标准时间",
			"en-US": "Korea Standard Time",
			"ja-JP": "韓国標準時",
			"ko-KR": "한국 표준 시간",
		},
	}
	
	if names, ok := tzNames[tz]; ok {
		if name, ok := names[lang]; ok {
			return name
		}
		if name, ok := names["en-US"]; ok {
			return name
		}
	}
	
	return tz
}

func GetCommonTimezones() []TimezoneInfo {
	common := []string{
		"UTC",
		"Asia/Shanghai",
		"America/New_York",
		"America/Los_Angeles",
		"Europe/London",
		"Europe/Paris",
		"Europe/Berlin",
		"Asia/Tokyo",
		"Asia/Seoul",
		"Australia/Sydney",
		"Asia/Singapore",
		"Asia/Dubai",
		"Asia/Kolkata",
	}
	
	result := make([]TimezoneInfo, 0, len(common))
	for _, tz := range common {
		result = append(result, GetTimezoneInfo(tz))
	}
	return result
}

func GetAllTimezoneInfos() []TimezoneInfo {
	result := make([]TimezoneInfo, 0, len(supportedTimezones))
	for _, tz := range supportedTimezones {
		result = append(result, GetTimezoneInfo(tz))
	}
	return result
}
