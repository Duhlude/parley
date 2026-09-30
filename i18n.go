package main

// Interface language. Parley's own text (menus, settings, notices) follows
// the Windows display language, or the language picked in Settings. Chat
// translation is separate: that's "Translate chat into".
//
// Strings are looked up by their English text, so untranslated text simply
// stays English. Translations live in i18n_data.go.

import (
	"fmt"
	"regexp"
	"strings"
)

// uiLanguages are the interface languages, matching WoW Classic's client
// languages. Code, native name.
var uiLanguages = []struct{ Code, Name string }{
	{"en", "English"},
	{"de", "Deutsch"},
	{"es", "Español"},
	{"fr", "Français"},
	{"it", "Italiano"},
	{"pt", "Português (Brasil)"},
	{"ru", "Русский"},
	{"ko", "한국어"},
	{"zh-Hans", "简体中文"},
	{"zh-Hant", "繁體中文"},
}

var uiLang = "en"

// setUILang switches the interface language ("" or "auto" = Windows).
func setUILang(code string) {
	if code == "" || code == "auto" {
		code = systemUILang()
	}
	if _, ok := uiText[code]; !ok && code != "en" {
		code = "en"
	}
	uiLang = code
}

// T translates an interface string.
func T(s string) string {
	if uiLang == "en" {
		return s
	}
	if t, ok := uiText[uiLang][s]; ok && t != "" {
		return t
	}
	return s
}

// Tf translates a format string, then formats it.
func Tf(format string, args ...any) string { return fmt.Sprintf(T(format), args...) }

// nonLatinUI reports whether the interface needs fonts beyond Latin
// (the WoW skins' Friz Quadrata only covers Latin letters).
func nonLatinUI() bool {
	switch uiLang {
	case "ru", "ko", "zh-Hans", "zh-Hant":
		return true
	}
	return false
}

// langLabel names a chat language (DeepL-style code) for menus: in English
// for an English interface, otherwise in that language's own name, which
// every player recognises.
func langLabel(code string) string {
	for _, l := range targetLangs {
		if l.Code == code {
			if uiLang == "en" {
				return l.Name
			}
			if n, ok := endonyms[code]; ok {
				return n
			}
			return l.Name
		}
	}
	return code
}

var endonyms = map[string]string{
	"EN-US": "English (US)", "EN-GB": "English (UK)", "ES": "Español", "PT-BR": "Português (Brasil)",
	"PT-PT": "Português (Portugal)", "FR": "Français", "DE": "Deutsch", "IT": "Italiano", "RU": "Русский",
	"UK": "Українська", "PL": "Polski", "NL": "Nederlands", "SV": "Svenska", "DA": "Dansk", "NB": "Norsk",
	"FI": "Suomi", "CS": "Čeština", "TR": "Türkçe", "EL": "Ελληνικά", "HU": "Magyar", "RO": "Română",
	"ZH-HANS": "简体中文", "ZH-HANT": "繁體中文", "JA": "日本語", "KO": "한국어", "AR": "العربية",
}

// uiLangForLocale maps a Windows locale name ("pt-BR", "zh-TW") to an
// interface language.
func uiLangForLocale(name string) string {
	n := strings.ToLower(name)
	switch {
	case strings.HasPrefix(n, "zh"):
		if strings.Contains(n, "hant") || strings.HasSuffix(n, "-tw") || strings.HasSuffix(n, "-hk") || strings.HasSuffix(n, "-mo") {
			return "zh-Hant"
		}
		return "zh-Hans"
	}
	base := n
	if i := strings.IndexAny(n, "-_"); i > 0 {
		base = n[:i]
	}
	for _, l := range uiLanguages {
		if l.Code == base {
			return base
		}
	}
	return "en"
}

// Messages built by the translation back ends (English, with details
// filled in) are translated for display by matching their pattern.
var msgPatterns = []struct {
	re     *regexp.Regexp
	format string
}{
	{regexp.MustCompile(`^DeepL unreachable: (.*)$`), "DeepL unreachable: %s"},
	{regexp.MustCompile(`^DeepL error (\d+)$`), "DeepL error %s"},
	{regexp.MustCompile(`^Azure unreachable: (.*)$`), "Azure unreachable: %s"},
	{regexp.MustCompile(`^Azure error (\d+)$`), "Azure error %s"},
	{regexp.MustCompile(`^offline translation doesn't support (.+) → (.+) yet$`), "offline translation doesn't support %s → %s yet"},
	{regexp.MustCompile(`^offline translation failed: (.*)$`), "offline translation failed: %s"},
	{regexp.MustCompile(`^Offline translation ready: (.+)\.$`), "Offline translation ready: %s."},
	{regexp.MustCompile(`^couldn't download the language list: (.*)$`), "couldn't download the language list: %s"},
	{regexp.MustCompile(`^couldn't download the (.+) model: (.*)$`), "couldn't download the %s model: %s"},
	{regexp.MustCompile(`^couldn't download the (.+) model \(HTTP (\d+)\)$`), "couldn't download the %s model (HTTP %s)"},
	{regexp.MustCompile(`^Downloading the (.+) offline model \((\d+) MB\)… (\d+)%$`), "Downloading the %s offline model (%s MB)… %s%%"},
	{regexp.MustCompile(`^the (.+) model download was interrupted: (.*)$`), "the %s model download was interrupted: %s"},
	{regexp.MustCompile(`^the (.+) model download was damaged; it will retry next time$`), "the %s model download was damaged; it will retry next time"},
	{regexp.MustCompile(`^couldn't start the offline translator: (.*)$`), "couldn't start the offline translator: %s"},
	{regexp.MustCompile(`^the offline translator didn't start: (.*)$`), "the offline translator didn't start: %s"},
	{regexp.MustCompile(`^offline translator stopped: (.*)$`), "offline translator stopped: %s"},
	{regexp.MustCompile(`^WoW folder not found: (.*)$`), "WoW folder not found: %s"},
}

// Fixed back-end messages, shown through localizeMsg (listed here so the
// translation tools and tests know about them).
var backendMessages = []string{
	"Didn't hear anything. Click the mic and try again.",
	"add your DeepL API key in Settings",
	"DeepL rejected the API key",
	"DeepL character allowance used up",
	"DeepL rate limit, slowing down",
	"DeepL sent an unreadable reply",
	"add your Azure key in Settings",
	"Azure rejected the key or region",
	"Azure character allowance used up",
	"Azure rate limit, slowing down",
	"Azure sent an unreadable reply",
	"couldn't tell which language this is",
	"the offline translator isn't installed; reinstall Parley",
	"the offline translator keeps crashing; see Parley's voice/engine log",
	"set your WoW folder first",
}

// localizeMsg translates a message for display.
func localizeMsg(s string) string {
	if uiLang == "en" {
		return s
	}
	if t := T(s); t != s {
		return t
	}
	for _, p := range msgPatterns {
		if m := p.re.FindStringSubmatch(s); m != nil {
			args := make([]any, len(m)-1)
			for i, g := range m[1:] {
				args[i] = g
			}
			return Tf(p.format, args...)
		}
	}
	return s
}

// chatLangForLocale picks the default "Translate chat into" language for a
// Windows locale ("pt-BR" -> "PT-BR", "de-AT" -> "DE"), English if unknown.
func chatLangForLocale(name string) string {
	n := strings.ToUpper(strings.ReplaceAll(name, "_", "-"))
	base := n
	if i := strings.IndexByte(n, '-'); i > 0 {
		base = n[:i]
	}
	switch base {
	case "EN":
		if n == "EN-GB" || strings.HasPrefix(n, "EN-IE") || strings.HasPrefix(n, "EN-AU") || strings.HasPrefix(n, "EN-NZ") {
			return "EN-GB"
		}
		return "EN-US"
	case "PT":
		if n == "PT-PT" {
			return "PT-PT"
		}
		return "PT-BR"
	case "ZH":
		if uiLangForLocale(name) == "zh-Hant" {
			return "ZH-HANT"
		}
		return "ZH-HANS"
	case "NO", "NN":
		return "NB"
	}
	for _, l := range targetLangs {
		if l.Code == base {
			return base
		}
	}
	return "EN-US"
}
