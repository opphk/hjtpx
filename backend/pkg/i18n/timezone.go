package i18n

import (
	"fmt"
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
	IsDST        bool   `json:"is_dst"`
	DSTName      string `json:"dst_name,omitempty"`
	StandardName string `json:"standard_name,omitempty"`
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
		"America/New_York",
		"America/Denver",
		"America/Panama",
		"America/Bogota",
		"America/Lima",
		"Asia/Jerusalem",
		"Africa/Casablanca",
		"Africa/Cairo",
		"Asia/Riyadh",
		"Asia/Tehran",
		"Asia/Kabul",
		"Asia/Karachi",
		"Asia/Kolkata",
		"Asia/Kathmandu",
		"Asia/Dhaka",
		"Asia/Yangon",
		"Asia/Bangkok",
		"Asia/Jakarta",
		"Asia/Ho_Chi_Minh",
		"Asia/Manila",
		"Asia/Shanghai",
		"Asia/Hong_Kong",
		"Asia/Taipei",
		"Asia/Tokyo",
		"Asia/Seoul",
		"Australia/Perth",
		"Australia/Adelaide",
		"Australia/Brisbane",
		"Australia/Sydney",
		"Australia/Melbourne",
		"Pacific/Auckland",
		"Pacific/Fiji",
		"America/Anchorage",
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

func GetAllTimezoneInfos() []TimezoneInfo {
	result := make([]TimezoneInfo, 0, len(supportedTimezones))
	for _, tz := range supportedTimezones {
		info := GetTimezoneInfo(tz)
		result = append(result, info)
	}
	return result
}

func GetTimezonesByCountry(countryCode string) []string {
	result := make([]string, 0)
	for tz := range getCountryCodeToTimezones() {
		result = append(result, tz)
	}
	return result
}

func getCountryCodeToTimezones() map[string]string {
	return map[string]string{
		"CN": "Asia/Shanghai",
		"HK": "Asia/Hong_Kong",
		"JP": "Asia/Tokyo",
		"KR": "Asia/Seoul",
		"SG": "Asia/Singapore",
		"TH": "Asia/Bangkok",
		"ID": "Asia/Jakarta",
		"PH": "Asia/Manila",
		"MY": "Asia/Kuala_Lumpur",
		"VN": "Asia/Ho_Chi_Minh",
		"TW": "Asia/Taipei",
		"IN": "Asia/Kolkata",
		"AE": "Asia/Dubai",
		"SA": "Asia/Riyadh",
		"IL": "Asia/Jerusalem",
		"IR": "Asia/Tehran",
		"PK": "Asia/Karachi",
		"BD": "Asia/Dhaka",
		"NP": "Asia/Kathmandu",
		"LK": "Asia/Colombo",
		"AF": "Asia/Kabul",
		"UZ": "Asia/Tashkent",
		"KZ": "Asia/Almaty",
		"AZ": "Asia/Baku",
		"GE": "Asia/Tbilisi",
		"AM": "Asia/Yerevan",
		"GB": "Europe/London",
		"FR": "Europe/Paris",
		"DE": "Europe/Berlin",
		"ES": "Europe/Madrid",
		"IT": "Europe/Rome",
		"RU": "Europe/Moscow",
		"TR": "Europe/Istanbul",
		"NL": "Europe/Amsterdam",
		"BE": "Europe/Brussels",
		"AT": "Europe/Vienna",
		"SE": "Europe/Stockholm",
		"NO": "Europe/Oslo",
		"DK": "Europe/Copenhagen",
		"FI": "Europe/Helsinki",
		"PL": "Europe/Warsaw",
		"CZ": "Europe/Prague",
		"GR": "Europe/Athens",
		"UA": "Europe/Kiev",
		"BY": "Europe/Minsk",
		"RO": "Europe/Bucharest",
		"BG": "Europe/Sofia",
		"RS": "Europe/Belgrade",
		"HR": "Europe/Zagreb",
		"BA": "Europe/Sarajevo",
		"PT": "Europe/Lisbon",
		"IE": "Europe/Dublin",
		"SK": "Europe/Bratislava",
		"SI": "Europe/Ljubljana",
		"EE": "Europe/Tallinn",
		"LV": "Europe/Riga",
		"LT": "Europe/Vilnius",
		"US": "America/New_York",
		"CA": "America/Toronto",
		"MX": "America/Mexico_City",
		"BR": "America/Sao_Paulo",
		"CL": "America/Santiago",
		"AR": "America/Buenos_Aires",
		"AU": "Australia/Sydney",
		"NZ": "Pacific/Auckland",
		"EG": "Africa/Cairo",
		"ZA": "Africa/Johannesburg",
		"NG": "Africa/Lagos",
		"KE": "Africa/Nairobi",
	}
}

func FormatTimeRange(start, end time.Time, tz string, lang string) string {
	loc := GetLocation(tz)
	start = start.In(loc)
	end = end.In(loc)
	
	dateFormat := GetDateFormat(lang)
	timeFormat := "15:04"
	
	if start.Format("2006-01-02") == end.Format("2006-01-02") {
		return fmt.Sprintf("%s %s - %s",
			start.Format(dateFormat),
			start.Format(timeFormat),
			end.Format(timeFormat))
	}
	
	return fmt.Sprintf("%s %s - %s %s",
		start.Format(dateFormat),
		start.Format(timeFormat),
		end.Format(dateFormat),
		end.Format(timeFormat))
}

func IsValidTimezone(tz string) bool {
	_, err := time.LoadLocation(tz)
	return err == nil
}

func GetTimezoneAbbreviationFull(tz string) (standard, dst string) {
	loc := GetLocation(tz)
	jan := time.Date(1, 1, 1, 12, 0, 0, 0, loc)
	jul := time.Date(1, 7, 1, 12, 0, 0, 0, loc)
	
	standard, _ = jan.Zone()
	dst, _ = jul.Zone()
	
	return standard, dst
}

func GetCurrentOffset(tz string) int {
	loc := GetLocation(tz)
	_, offset := time.Now().In(loc).Zone()
	return offset
}

func GetTimezoneByCity(city string) []string {
	city = strings.ToLower(city)
	var result []string
	
	cityMap := map[string][]string{
		"shanghai":   {"Asia/Shanghai"},
		"beijing":    {"Asia/Shanghai"},
		"tokyo":      {"Asia/Tokyo"},
		"seoul":      {"Asia/Seoul"},
		"singapore":  {"Asia/Singapore"},
		"hong kong":  {"Asia/Hong_Kong"},
		"bangkok":    {"Asia/Bangkok"},
		"jakarta":    {"Asia/Jakarta"},
		"mumbai":     {"Asia/Kolkata"},
		"kolkata":    {"Asia/Kolkata"},
		"delhi":      {"Asia/Kolkata"},
		"dubai":      {"Asia/Dubai"},
		"riyadh":     {"Asia/Riyadh"},
		"tehran":     {"Asia/Tehran"},
		"jerusalem":   {"Asia/Jerusalem"},
		"moscow":     {"Europe/Moscow"},
		"paris":      {"Europe/Paris"},
		"berlin":     {"Europe/Berlin"},
		"london":     {"Europe/London"},
		"new york":   {"America/New_York"},
		"los angeles": {"America/Los_Angeles"},
		"san francisco": {"America/Los_Angeles"},
		"chicago":    {"America/Chicago"},
		"toronto":    {"America/Toronto"},
		"vancouver":  {"America/Vancouver"},
		"sydney":     {"Australia/Sydney"},
		"melbourne":  {"Australia/Melbourne"},
		"auckland":   {"Pacific/Auckland"},
	}
	
	for key, timezones := range cityMap {
		if strings.Contains(key, city) || strings.Contains(city, key) {
			result = append(result, timezones...)
		}
	}
	
	return result
}

func GetTimezoneByCountryCode(countryCode string) []string {
	countryCode = strings.ToUpper(countryCode)
	result := make([]string, 0)
	
	countryTimezones := map[string][]string{
		"CN": {"Asia/Shanghai", "Asia/Hong_Kong"},
		"JP": {"Asia/Tokyo"},
		"KR": {"Asia/Seoul"},
		"SG": {"Asia/Singapore"},
		"TH": {"Asia/Bangkok"},
		"ID": {"Asia/Jakarta"},
		"VN": {"Asia/Ho_Chi_Minh"},
		"TW": {"Asia/Taipei"},
		"IN": {"Asia/Kolkata"},
		"AE": {"Asia/Dubai"},
		"SA": {"Asia/Riyadh"},
		"IL": {"Asia/Jerusalem"},
		"IR": {"Asia/Tehran"},
		"PK": {"Asia/Karachi"},
		"BD": {"Asia/Dhaka"},
		"US": {"America/New_York", "America/Los_Angeles", "America/Chicago", "America/Denver", "America/Phoenix", "America/Anchorage", "Pacific/Honolulu"},
		"CA": {"America/Toronto", "America/Vancouver"},
		"MX": {"America/Mexico_City"},
		"BR": {"America/Sao_Paulo"},
		"GB": {"Europe/London"},
		"FR": {"Europe/Paris"},
		"DE": {"Europe/Berlin"},
		"ES": {"Europe/Madrid"},
		"IT": {"Europe/Rome"},
		"RU": {"Europe/Moscow"},
		"TR": {"Europe/Istanbul"},
		"AU": {"Australia/Sydney", "Australia/Melbourne", "Australia/Perth", "Australia/Adelaide", "Australia/Brisbane"},
		"NZ": {"Pacific/Auckland"},
		"EG": {"Africa/Cairo"},
		"ZA": {"Africa/Johannesburg"},
		"NG": {"Africa/Lagos"},
		"KE": {"Africa/Nairobi"},
	}
	
	if timezones, ok := countryTimezones[countryCode]; ok {
		result = append(result, timezones...)
	}
	
	return result
}

func FormatTimezoneOffset(offset int) string {
	hours := offset / 3600
	minutes := (offset % 3600) / 60
	
	if hours == 0 && minutes == 0 {
		return "UTC"
	}
	
	if minutes == 0 {
		return fmt.Sprintf("UTC%+d", hours)
	}
	
	return fmt.Sprintf("UTC%+d:%02d", hours, minutes)
}

func CompareTimezones(tz1, tz2 string) string {
	loc1 := GetLocation(tz1)
	loc2 := GetLocation(tz2)
	
	now := time.Now()
	t1 := now.In(loc1)
	t2 := now.In(loc2)
	
	_, offset1 := t1.Zone()
	_, offset2 := t2.Zone()
	
	diff := offset2 - offset1
	hours := diff / 3600
	minutes := (diff % 3600) / 60
	
	if diff == 0 {
		return "Same timezone"
	}
	
	sign := "+"
	if diff < 0 {
		sign = "-"
		hours = -hours
		minutes = -minutes
	}
	
	if minutes == 0 {
		return fmt.Sprintf("%s%s hours", sign, hours)
	}
	
	return fmt.Sprintf("%s%s:%02d hours", sign, hours, minutes)
}

func IsTimezoneAhead(tz1, tz2 string) bool {
	loc1 := GetLocation(tz1)
	loc2 := GetLocation(tz2)
	
	now := time.Now()
	t1 := now.In(loc1)
	t2 := now.In(loc2)
	
	_, offset1 := t1.Zone()
	_, offset2 := t2.Zone()
	
	return offset1 > offset2
}

func GetTimezoneDistance(tz1, tz2 string) (hours, minutes int) {
	loc1 := GetLocation(tz1)
	loc2 := GetLocation(tz2)
	
	now := time.Now()
	t1 := now.In(loc1)
	t2 := now.In(loc2)
	
	_, offset1 := t1.Zone()
	_, offset2 := t2.Zone()
	
	diff := offset2 - offset1
	if diff < 0 {
		diff = -diff
	}
	
	hours = diff / 3600
	minutes = (diff % 3600) / 60
	
	return hours, minutes
}

func GetNextWeekdayOccurrence(tz string, weekday time.Weekday, hour, minute int) time.Time {
	loc := GetLocation(tz)
	now := time.Now().In(loc)
	
	target := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc)
	
	for target.Before(now) || target.Weekday() != weekday {
		target = target.AddDate(0, 0, 1)
	}
	
	return target
}

func GetBusinessHoursInTimezone(tz string, startHour, startMin, endHour, endMin int) (start, end time.Time) {
	loc := GetLocation(tz)
	now := time.Now().In(loc)
	
	start = time.Date(now.Year(), now.Month(), now.Day(), startHour, startMin, 0, 0, loc)
	end = time.Date(now.Year(), now.Month(), now.Day(), endHour, endMin, 0, 0, loc)
	
	return start, end
}

func IsWithinBusinessHours(tz string, startHour, startMin, endHour, endMin int) bool {
	loc := GetLocation(tz)
	now := time.Now().In(loc)
	
	currentMinutes := now.Hour()*60 + now.Minute()
	startMinutes := startHour*60 + startMin
	endMinutes := endHour*60 + endMin
	
	return currentMinutes >= startMinutes && currentMinutes <= endMinutes
}

func GetTimeUntil(targetTime time.Time, tz string) string {
	loc := GetLocation(tz)
	now := time.Now().In(loc)
	target := targetTime.In(loc)
	
	diff := target.Sub(now)
	
	if diff < 0 {
		return "already passed"
	}
	
	days := int(diff.Hours() / 24)
	hours := int(diff.Hours()) % 24
	minutes := int(diff.Minutes()) % 60
	seconds := int(diff.Seconds()) % 60
	
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	} else {
		return fmt.Sprintf("%ds", seconds)
	}
}

func ParseTimeString(timeStr string) (time.Time, error) {
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04",
		"2006-01-02T15:04",
		"01/02/2006 15:04:05",
		"01/02/2006T15:04:05",
		"01/02/2006 15:04",
		"01/02/2006T15:04",
	}
	
	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t, nil
		}
	}
	
	return time.Time{}, fmt.Errorf("unable to parse time string: %s", timeStr)
}

func ConvertTimeString(timeStr string, fromTz, toTz string) (string, error) {
	t, err := ParseTimeString(timeStr)
	if err != nil {
		return "", err
	}
	
	loc := GetLocation(toTz)
	t = t.In(GetLocation(fromTz)).In(loc)
	
	return t.Format("2006-01-02 15:04:05"), nil
}

func GetTimezoneDisplayName(tz string) string {
	parts := splitTimezone(tz)
	if len(parts) >= 2 {
		return fmt.Sprintf("%s/%s", parts[0], parts[1])
	}
	return tz
}

func GetTimezoneCityName(tz string) string {
	parts := splitTimezone(tz)
	if len(parts) >= 2 {
		city := parts[1]
		city = strings.Replace(city, "_", " ", -1)
		city = strings.Title(strings.ToLower(city))
		return city
	}
	return tz
}

func FormatTimestamp(t time.Time, tz string) string {
	loc := GetLocation(tz)
	return t.In(loc).Format("2006-01-02 15:04:05 MST")
}

func FormatTimestampISO(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05Z")
}

func GetRelativeTimezoneAbbreviation(tz1, tz2 string) string {
	_, dst1 := GetTimezoneAbbreviationFull(tz1)
	_, dst2 := GetTimezoneAbbreviationFull(tz2)
	
	if dst1 == dst2 {
		return dst1
	}
	
	return fmt.Sprintf("%s/%s", dst1, dst2)
}
