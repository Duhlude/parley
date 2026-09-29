package main

// Gaming slang: chat shortcuts expanded before translation, and short
// explanations of WoW terms shown under a translation (data: slangdata.go).

import (
	"regexp"
	"sort"
	"strings"
)

// DeepL language -> dictionary locale.
var glossLocale = map[string]string{
	"EN": "enUS", "ES": "esES", "PT": "ptBR", "DE": "deDE", "FR": "frFR", "IT": "itIT",
	"RU": "ruRU", "KO": "koKR", "PL": "plPL", "SV": "svSE", "NB": "noNO", "TR": "trTR",
	"JA": "jaJP", "NL": "nlNL", "CS": "csCZ", "UK": "ukUA",
}

func localeFor(lang string) string {
	switch strings.ToUpper(lang) {
	case "ZH-HANT":
		return "zhTW"
	case "ZH", "ZH-HANS":
		return "zhCN"
	case "PT-PT":
		return "ptBR"
	}
	return glossLocale[baseLang(lang)]
}

var (
	shortcutRe *regexp.Regexp
	glossRe    *regexp.Regexp
)

func init() {
	shortcutRe = wordsRe(keys(shortcuts))
	glossRe = wordsRe(keys(glossary))
	// both lists are English gamer talk: count them as English words
	for k := range shortcuts {
		englishWords[k] = true
	}
	for k := range glossary {
		if !strings.Contains(k, " ") {
			englishWords[k] = true
		}
	}
}

func keys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	// longest first so "lf1m" wins over "lf" and "sm cath" over "sm"
	sort.Slice(out, func(i, j int) bool { return len(out[i]) > len(out[j]) })
	return out
}

func wordsRe(words []string) *regexp.Regexp {
	q := make([]string, len(words))
	for i, w := range words {
		q[i] = strings.ReplaceAll(regexp.QuoteMeta(w), `\ `, `\s+`)
	}
	return regexp.MustCompile(`(?i)(^|[^\p{L}\p{N}\[])(` + strings.Join(q, "|") + `)($|[^\p{L}\p{N}\]])`)
}

// expandShortcuts turns chat shortcuts into plain English ("sry, w8" ->
// "Sorry, Wait") so a translator understands them. [Item links] are left alone.
func expandShortcuts(text string) string {
	// run twice: neighbouring matches share a boundary character
	for i := 0; i < 2; i++ {
		text = shortcutRe.ReplaceAllStringFunc(text, func(m string) string {
			sm := shortcutRe.FindStringSubmatch(m)
			full := shortcuts[strings.ToLower(sm[2])]
			if sm[2] == strings.ToLower(sm[2]) {
				full = strings.ToLower(full[:1]) + full[1:]
			}
			return sm[1] + full + sm[3]
		})
	}
	return text
}

// glossFor lists WoW terms in text with an explanation in lang, e.g.
// "w2w: Wall-to-wall pull (everything)". At most max entries.
func glossFor(text, lang string, max int) []string {
	loc := localeFor(lang)
	if loc == "" {
		loc = "enUS"
	}
	var out []string
	seen := map[string]bool{}
	for _, sm := range glossRe.FindAllStringSubmatch(text, -1) {
		k := strings.ToLower(strings.Join(strings.Fields(sm[2]), " "))
		if seen[k] {
			continue
		}
		seen[k] = true
		e := glossary[k]
		v := e[loc]
		if v == "" {
			v = e["enUS"]
		}
		if v == "" || strings.EqualFold(v, k) {
			continue
		}
		out = append(out, sm[2]+": "+v)
		if len(out) >= max {
			break
		}
	}
	return out
}
