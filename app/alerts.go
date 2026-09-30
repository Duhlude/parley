package main

// Alerts: highlight (and optionally chime for) messages that mention your
// character or words you care about ("Deadmines", "healer", "WTB").

import (
	"strings"
	"unicode"
)

// alertTerms splits the settings field ("Deadmines, healer; WTB") and adds
// the character name the addon reported (without the realm).
func alertTerms(words, char string) []string {
	var out []string
	seen := map[string]bool{}
	add := func(s string) {
		s = strings.TrimSpace(s)
		if len([]rune(s)) < 2 || seen[strings.ToLower(s)] {
			return
		}
		seen[strings.ToLower(s)] = true
		out = append(out, s)
	}
	for _, w := range strings.FieldsFunc(words, func(r rune) bool { return r == ',' || r == ';' || r == '\n' }) {
		add(w)
	}
	if i := strings.IndexByte(char, '-'); i > 0 {
		char = char[:i]
	}
	add(char)
	return out
}

// mentions reports whether any term appears in any of the texts as a whole
// word (case-insensitive). Terms in scripts without spaces (Chinese,
// Japanese) match anywhere.
func mentions(terms []string, texts ...string) bool {
	for _, t := range terms {
		lt := strings.ToLower(t)
		for _, s := range texts {
			ls := strings.ToLower(s)
			for from := 0; ; {
				i := strings.Index(ls[from:], lt)
				if i < 0 {
					break
				}
				i += from
				if wordEdge(ls, i, i+len(lt)) {
					return true
				}
				from = i + 1
			}
		}
	}
	return false
}

func wordEdge(s string, start, end int) bool {
	isWord := func(r rune) bool {
		return (unicode.IsLetter(r) || unicode.IsDigit(r)) && !unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana)
	}
	if start > 0 {
		r := []rune(s[:start])
		if isWord(r[len(r)-1]) {
			return false
		}
	}
	if end < len(s) {
		r := []rune(s[end:])
		if isWord(r[0]) {
			return false
		}
	}
	return true
}
