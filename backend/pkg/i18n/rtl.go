package i18n

import (
	"regexp"
	"strings"
)

var rtlScriptRegex = regexp.MustCompile(`[\x{0590}-\x{05FF}\x{0600}-\x{06FF}\x{0750}-\x{077F}\x{08A0}-\x{08FF}\x{FB50}-\x{FDFF}\x{FE70}-\x{FEFF}]`)

func HasRTLScript(text string) bool {
	return rtlScriptRegex.MatchString(text)
}

func GetBaseDirection(text string) string {
	if HasRTLScript(text) {
		return "rtl"
	}
	return "ltr"
}

func ResolveBidirectionalText(text string) string {
	if !HasRTLScript(text) {
		return text
	}
	
	hasLTR := regexp.MustCompile(`[A-Za-z0-9]`).MatchString(text)
	
	if hasLTR {
		return "\u202B" + text + "\u202C"
	}
	
	return "\u202B" + text
}

func WrapLTRText(text string) string {
	if !HasRTLScript(text) {
		return text
	}
	
	return "\u202A" + text + "\u202C"
}

func GetCSSDirection(lang string) string {
	if IsRTL(lang) {
		return "rtl"
	}
	return "ltr"
}

func GetCSSTextAlign(lang string) string {
	if IsRTL(lang) {
		return "right"
	}
	return "left"
}

func GetCSSFloat(lang string) string {
	if IsRTL(lang) {
		return "right"
	}
	return "left"
}

func GetOppositeCSSFloat(lang string) string {
	if IsRTL(lang) {
		return "left"
	}
	return "right"
}

func GetDirectionClass(lang string) string {
	if IsRTL(lang) {
		return "direction-rtl"
	}
	return "direction-ltr"
}

func GetLangClass(lang string) string {
	return "lang-" + strings.ReplaceAll(lang, "-", "_")
}

func GetLayoutClasses(lang string) string {
	return GetDirectionClass(lang) + " " + GetLangClass(lang)
}

func MirrorCSSProperty(lang string, property string) string {
	if !IsRTL(lang) {
		return property
	}
	
	mirrored := map[string]string{
		"left":        "right",
		"right":       "left",
		"margin-left": "margin-right",
		"margin-right": "margin-left",
		"padding-left": "padding-right",
		"padding-right": "padding-left",
		"border-left": "border-right",
		"border-right": "border-left",
		"left-border": "right-border",
		"right-border": "left-border",
		"text-align-left": "text-align-right",
		"text-align-right": "text-align-left",
		"float-left": "float-right",
		"float-right": "float-left",
		"clear-left": "clear-right",
		"clear-right": "clear-left",
		"start": "end",
		"end": "start",
	}
	
	if val, ok := mirrored[strings.ToLower(property)]; ok {
		return val
	}
	
	return property
}

func GenerateRTLCSS(lang string) string {
	if !IsRTL(lang) {
		return ""
	}
	
	className := "." + GetDirectionClass(lang)
	
	return className + ` {
	direction: rtl;
	text-align: right;
}
` + className + ` .float-left {
	float: right;
}
` + className + ` .float-right {
	float: left;
}
` + className + ` .text-left {
	text-align: right;
}
` + className + ` .text-right {
	text-align: left;
}
` + className + ` .ml-auto {
	margin-left: initial;
	margin-right: auto;
}
` + className + ` .mr-auto {
	margin-right: initial;
	margin-left: auto;
}
`
}

func AdjustTextForRTL(text string, lang string) string {
	if !IsRTL(lang) {
		return text
	}
	
	text = ResolveBidirectionalText(text)
	
	text = strings.ReplaceAll(text, "left", "\u202Aleft\u202C")
	text = strings.ReplaceAll(text, "right", "\u202Aright\u202C")
	
	return text
}

type RTLConfig struct {
	Direction           string
	TextAlign           string
	Float               string
	OppositeFloat       string
	DirectionClass      string
	LangClass           string
	LayoutClasses       string
	IsRTL               bool
}

func GetRTLConfig(lang string) RTLConfig {
	isRTL := IsRTL(lang)
	
	var direction, textAlign, floatDir, oppositeFloat string
	if isRTL {
		direction = "rtl"
		textAlign = "right"
		floatDir = "right"
		oppositeFloat = "left"
	} else {
		direction = "ltr"
		textAlign = "left"
		floatDir = "left"
		oppositeFloat = "right"
	}
	
	return RTLConfig{
		Direction:      direction,
		TextAlign:      textAlign,
		Float:          floatDir,
		OppositeFloat:  oppositeFloat,
		DirectionClass: GetDirectionClass(lang),
		LangClass:      GetLangClass(lang),
		LayoutClasses:  GetLayoutClasses(lang),
		IsRTL:          isRTL,
	}
}
