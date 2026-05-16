package i18n

import (
	"time"
)

type TimezoneInfo struct {
	Name      string
	Offset    int
	Zone      string
}

var (
	defaultTimezone = "Asia/Shanghai"
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
