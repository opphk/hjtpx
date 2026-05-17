package utils

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
)

func GenerateUUID() string {
	return uuid.New().String()
}

func GenerateShortUUID() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")[:16]
}

func GenerateUUIDWithPrefix(prefix string) string {
	return fmt.Sprintf("%s_%s", prefix, uuid.New().String())
}

func ParseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

func IsValidUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

func MD5(data string) string {
	h := md5.New()
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func MD5Bytes(data []byte) string {
	h := md5.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func MD5Salt(data, salt string) string {
	return MD5(data + salt)
}

func MD5File(data io.Reader) (string, error) {
	h := md5.New()
	if _, err := io.Copy(h, data); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

const (
	FormatDateTime     = "2006-01-02 15:04:05"
	FormatDate         = "2006-01-02"
	FormatTime         = "15:04:05"
	FormatDateTimeShort = "20060102150405"
	FormatDateShort    = "20060102"
	FormatTimeShort    = "150405"
	FormatISO8601      = "2006-01-02T15:04:05Z07:00"
	FormatRFC3339      = time.RFC3339
)

func FormatTimeToString(t time.Time, format string) string {
	if format == "" {
		format = FormatDateTime
	}
	return t.Format(format)
}

func FormatTimeToUnix(t time.Time) int64 {
	return t.Unix()
}

func FormatTimeToUnixMilli(t time.Time) int64 {
	return t.UnixMilli()
}

func FormatTimeToUnixMicro(t time.Time) int64 {
	return t.UnixMicro()
}

func ParseStringToTime(s string, format string) (time.Time, error) {
	if format == "" {
		formats := []string{
			FormatDateTime,
			FormatDate,
			FormatISO8601,
			FormatRFC3339,
			time.RFC3339Nano,
		}
		for _, f := range formats {
			if t, err := time.Parse(f, s); err == nil {
				return t, nil
			}
		}
		return time.Time{}, fmt.Errorf("无法解析时间字符串: %s", s)
	}
	return time.Parse(format, s)
}

func ParseUnixToTime(unix int64) time.Time {
	return time.Unix(unix, 0)
}

func ParseUnixMilliToTime(milli int64) time.Time {
	return time.UnixMilli(milli)
}

func GetCurrentTime() time.Time {
	return time.Now()
}

func GetCurrentUnix() int64 {
	return time.Now().Unix()
}

func GetCurrentUnixMilli() int64 {
	return time.Now().UnixMilli()
}

func GetCurrentTimeString(format string) string {
	return FormatTimeToString(time.Now(), format)
}

func AddTime(t time.Time, duration time.Duration) time.Time {
	return t.Add(duration)
}

func SubTime(t1, t2 time.Time) time.Duration {
	return t1.Sub(t2)
}

func IsExpired(expireTime time.Time) bool {
	return time.Now().After(expireTime)
}

func IsValidTimeRange(start, end time.Time) bool {
	return start.Before(end) || start.Equal(end)
}

func GetTimezoneOffset(timezone string) (int, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return 0, err
	}
	_, offset := time.Now().In(loc).Zone()
	return offset / 3600, nil
}

func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func StringToLower(s string) string {
	return strings.ToLower(s)
}

func StringToUpper(s string) string {
	return strings.ToUpper(s)
}

func StringTrim(s string) string {
	return strings.TrimSpace(s)
}

func StringTrimLeft(s string, cutset string) string {
	return strings.TrimLeft(s, cutset)
}

func StringTrimRight(s string, cutset string) string {
	return strings.TrimRight(s, cutset)
}

func StringTrimPrefix(s, prefix string) string {
	return strings.TrimPrefix(s, prefix)
}

func StringTrimSuffix(s, suffix string) string {
	return strings.TrimSuffix(s, suffix)
}

func StringContains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func StringContainsAny(s, chars string) bool {
	return strings.ContainsAny(s, chars)
}

func StringHasPrefix(s, prefix string) bool {
	return strings.HasPrefix(s, prefix)
}

func StringHasSuffix(s, suffix string) bool {
	return strings.HasSuffix(s, suffix)
}

func StringReplace(s, old, new string) string {
	return strings.ReplaceAll(s, old, new)
}

func StringReplaceN(s, old, new string, n int) string {
	return strings.Replace(s, old, new, n)
}

func StringSplit(s, sep string) []string {
	if s == "" {
		return []string{}
	}
	return strings.Split(s, sep)
}

func StringSplitN(s, sep string, n int) []string {
	if s == "" {
		return []string{}
	}
	return strings.SplitN(s, sep, n)
}

func StringJoin(elems []string, sep string) string {
	return strings.Join(elems, sep)
}

func StringRepeat(s string, count int) string {
	return strings.Repeat(s, count)
}

func StringEqualFold(s, t string) bool {
	return strings.EqualFold(s, t)
}

func StringCount(s, substr string) int {
	return strings.Count(s, substr)
}

func StringIndex(s, substr string) int {
	return strings.Index(s, substr)
}

func StringLastIndex(s, substr string) int {
	return strings.LastIndex(s, substr)
}

func StringFields(s string) []string {
	return strings.Fields(s)
}

func StringToTitle(s string) string {
	return strings.ToTitle(s)
}

func StringToLowerSpecial(case mapping rune, s string) string {
	return strings.ToLowerSpecial(unicode.SpecialCase{mapping}, s)
}

func StringToUpperSpecial(case mapping rune, s string) string {
	return strings.ToUpperSpecial(unicode.SpecialCase{mapping}, s)
}

func StringMap(mapper func(rune) rune, s string) string {
	return strings.Map(mapper, s)
}

func StringIndexFunc(s string, f func(rune) bool) int {
	return strings.IndexFunc(s, f)
}

func StringLastIndexFunc(s string, f func(rune) bool) int {
	return strings.LastIndexFunc(s, f)
}

func StringContainsFunc(s string, f func(rune) bool) bool {
	return strings.ContainsFunc(s, f)
}

func StringTitle(s string) string {
	return strings.Title(s)
}

func StringEqual(s1, s2 string) bool {
	return s1 == s2
}

func IsEmptyString(s string) bool {
	return strings.TrimSpace(s) == ""
}

func IsNotEmptyString(s string) bool {
	return !IsEmptyString(s)
}

func IsBlank(s string) bool {
	for _, r := range s {
		if !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

func IsAlpha(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return s != ""
}

func IsDigit(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return s != ""
}

func IsAlphanumeric(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return s != ""
}

func IsNumeric(s string) bool {
	_, err := fmt.Sscanf(s, "%f", new(float64))
	return err == nil
}

func StringReverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func StringMask(s string, start, end int, maskChar rune) string {
	if start < 0 {
		start = 0
	}
	if end > len(s) {
		end = len(s)
	}
	if start >= end {
		return s
	}

	runes := []rune(s)
	mask := strings.Repeat(string(maskChar), end-start)
	before := string(runes[:start])
	after := string(runes[end:])

	return before + mask + after
}

func StringMaskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return StringMask(email, 0, len(email)/2, '*')
	}

	username := parts[0]
	if len(username) <= 2 {
		return StringMask(username, 0, len(username), '*') + "@" + parts[1]
	}

	return StringMask(username, 1, len(username)-1, '*') + "@" + parts[1]
}

func StringMaskPhone(phone string) string {
	if len(phone) < 7 {
		return phone
	}
	return phone[:3] + "****" + phone[len(phone)-4:]
}

func StringMaskIDCard(id string) string {
	if len(id) < 10 {
		return id
	}
	return id[:4] + "**********" + id[len(id)-4:]
}

func StringCapitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func StringUncapitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func StringCamelCase(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	for i, part := range parts {
		if i == 0 {
			parts[i] = strings.ToLower(part)
		} else {
			parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}
	return strings.Join(parts, "")
}

func StringSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result.WriteRune('_')
			}
			result.WriteRune(unicode.ToLower(r))
		} else if r == '-' || r == ' ' {
			result.WriteRune('_')
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func StringPascalCase(s string) string {
	parts := strings.Split(strings.ToLower(s), "_")
	for i, part := range parts {
		if part != "" {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

func StringKebabCase(s string) string {
	return strings.ReplaceAll(StringSnakeCase(s), "_", "-")
}

func StringRandom(length int, charset string) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("长度必须大于0")
	}
	if charset == "" {
		charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	}

	result := make([]byte, length)
	charsetLen := big.NewInt(int64(len(charset)))

	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", err
		}
		result[i] = charset[n.Int64()]
	}

	return string(result), nil
}

func StringRandomAlphanumeric(length int) (string, error) {
	return StringRandom(length, "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")
}

func StringRandomAlpha(length int) (string, error) {
	return StringRandom(length, "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")
}

func StringRandomNumeric(length int) (string, error) {
	return StringRandom(length, "0123456789")
}

func Base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func Base64Decode(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

func Base64URLEncode(data []byte) string {
	return base64.URLEncoding.EncodeToString(data)
}

func Base64URLDecode(s string) ([]byte, error) {
	return base64.URLEncoding.DecodeString(s)
}

func Base64RawEncode(data []byte) string {
	return base64.RawStdEncoding.EncodeToString(data)
}

func Base64RawDecode(s string) ([]byte, error) {
	return base64.RawStdEncoding.DecodeString(s)
}

func IntToString(i int) string {
	return fmt.Sprintf("%d", i)
}

func Int64ToString(i int64) string {
	return fmt.Sprintf("%d", i)
}

func StringToInt(s string) (int, error) {
	return fmt.Sscanf(s, "%d", new(int))
}

func StringToInt64(s string) (int64, error) {
	var result int64
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

func StringToIntDefault(s string, defaultVal int) int {
	if val, err := StringToInt(s); err == nil {
		return val
	}
	return defaultVal
}

func StringToInt64Default(s string, defaultVal int64) int64 {
	if val, err := StringToInt64(s); err == nil {
		return val
	}
	return defaultVal
}

func Float64ToString(f float64) string {
	return fmt.Sprintf("%f", f)
}

func Float64ToStringPrec(f float64, prec int) string {
	return fmt.Sprintf(fmt.Sprintf("%%.%df", prec), f)
}

func StringToFloat64(s string) (float64, error) {
	var result float64
	_, err := fmt.Sscanf(s, "%f", &result)
	return result, err
}

func StringToFloat64Default(s string, defaultVal float64) float64 {
	if val, err := StringToFloat64(s); err == nil {
		return val
	}
	return defaultVal
}

func BoolToString(b bool) string {
	return fmt.Sprintf("%t", b)
}

func StringToBool(s string) (bool, error) {
	lower := strings.ToLower(s)
	switch lower {
	case "true", "1", "yes", "on":
		return true, nil
	case "false", "0", "no", "off", "":
		return false, nil
	default:
		return false, fmt.Errorf("无法将 %s 转换为布尔值", s)
	}
}

func StringToBoolDefault(s string, defaultVal bool) bool {
	if val, err := StringToBool(s); err == nil {
		return val
	}
	return defaultVal
}

func BytesToString(b []byte) string {
	return string(b)
}

func StringToBytes(s string) []byte {
	return []byte(s)
}

func IntToBytes(i int) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(i))
	return buf
}

func BytesToInt(b []byte) int {
	return int(binary.BigEndian.Uint32(b))
}

func Int64ToBytes(i int64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(i))
	return buf
}

func BytesToInt64(b []byte) int64 {
	return int64(binary.BigEndian.Uint64(b))
}

func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func MaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func MinInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func MaxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func ClampInt(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

func ClampInt64(val, min, max int64) int64 {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

func AbsInt(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func AbsInt64(a int64) int64 {
	if a < 0 {
		return -a
	}
	return a
}

func InIntSlice(val int, slice []int) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}

func InStringSlice(val string, slice []string) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}

func RemoveDuplicatesInt(slice []int) []int {
	seen := make(map[int]bool)
	result := []int{}
	for _, v := range slice {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}

func RemoveDuplicatesString(slice []string) []string {
	seen := make(map[string]bool)
	result := []string{}
	for _, v := range slice {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}

func SliceContains(slice []string, val string) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}

func SliceContainsInt(slice []int, val int) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}

func StringPtr(s string) *string {
	return &s
}

func IntPtr(i int) *int {
	return &i
}

func Int64Ptr(i int64) *int64 {
	return &i
}

func BoolPtr(b bool) *bool {
	return &b
}

func StringPtrVal(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func IntPtrVal(ptr *int) int {
	if ptr == nil {
		return 0
	}
	return *ptr
}

func Int64PtrVal(ptr *int64) int64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

func BoolPtrVal(ptr *bool) bool {
	if ptr == nil {
		return false
	}
	return *ptr
}
