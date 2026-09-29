package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Translator wraps the DeepL API with a small cache.
type Translator struct {
	mu    sync.Mutex
	cache map[string]Translation
	order []string
	http  *http.Client
}

type Translation struct {
	Text     string
	Detected string // DeepL source code, e.g. "ES", "RU", "EN"
}

func NewTranslator() *Translator {
	return &Translator{cache: map[string]Translation{}, http: &http.Client{Timeout: 10 * time.Second}}
}

var bracketRe = regexp.MustCompile(`\[[^\[\]]{1,80}\]`)

// Protect [Item Links] (and WoW jargon when translating out of English)
// using DeepL's XML mode. Names Parley already localized are sent as plain
// text: DeepL keeps proper names on its own, and tagging them makes it add
// quotes ("Deadmines") or drop the space before them.
func toXML(s, target string) string {
	esc := html.EscapeString(stripKeep(s))
	esc = bracketRe.ReplaceAllStringFunc(esc, func(m string) string { return "<x>" + m + "</x>" })
	if baseLang(target) == "EN" {
		return esc // the jargon is English already; tagging it only adds quotes
	}
	// Keep WoW jargon in English: players everywhere say "tank", "dps", "lfg".
	return jargonRe.ReplaceAllStringFunc(esc, func(m string) string { return "<x>" + m + "</x>" })
}

// Gaming terms that players use untranslated in every language.
var jargonRe = regexp.MustCompile(`(?i)\b(` + strings.Join([]string{
	"tanks?", "healers?", "heals?", "dps", "lf[0-9]?m?", "lfg", "lfm", "wts", "wtb", "wtt", "pst",
	"inv", "buffs?", "debuffs?", "pulls?", "pulling", "aggro", "loot", "farm(ing)?", "boss(es)?",
	"mobs?", "raid", "dg", "dung", "quests?", "gg", "afk", "brb", "oom", "rez", "ress", "xp",
	"lvl", "gold", "mana", "ah", "gy", "cc", "sap", "sheep", "kite", "melee", "caster",
	"dm", "sm", "rfc", "wc", "sfk", "bfd", "stocks", "gnomer", "rfk", "rfd", "uld", "zf", "mara",
	"st", "brd", "lbrs", "ubrs", "strat", "scholo", "dme", "dmw", "dmn", "zg", "mc", "bwl",
	"ony", "aq20", "aq40", "naxx", "wsg", "ab", "av",
}, "|") + `)\b`)

var tagRe = regexp.MustCompile(`</?x>`)

var (
	glueBeforeXRe = regexp.MustCompile(`([\p{L}\p{N}])<x>`)
	glueAfterXRe  = regexp.MustCompile(`</x>([\p{L}\p{N}])`)
)

func fromXML(s string) string {
	s = glueBeforeXRe.ReplaceAllString(s, "$1 <x>")
	s = glueAfterXRe.ReplaceAllString(s, "</x> $1")
	return html.UnescapeString(tagRe.ReplaceAllString(s, ""))
}

func endpoint(key string) string {
	if u := os.Getenv("PARLEY_DEEPL_URL"); u != "" { // testing only
		return u
	}
	if strings.HasSuffix(strings.TrimSpace(key), ":fx") {
		return "https://api-free.deepl.com/v2/translate"
	}
	return "https://api.deepl.com/v2/translate"
}

// Translate text into target (DeepL code like "EN-US", "ES", "RU").
// source may be "" to auto-detect.
func (t *Translator) Translate(key, text, source, target string) (Translation, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return Translation{}, errors.New("add your DeepL API key in Settings")
	}
	ck := source + "|" + target + "|" + text
	t.mu.Lock()
	if tr, ok := t.cache[ck]; ok {
		t.mu.Unlock()
		return tr, nil
	}
	t.mu.Unlock()

	body := map[string]any{
		"text":         []string{toXML(text, target)},
		"target_lang":  target,
		"tag_handling": "xml",
		"ignore_tags":  []string{"x"},
	}
	if source != "" {
		body["source_lang"] = source
	}
	// Quality extras: casual register, better model, and a hint that this is
	// game chat (context text is free and isn't translated). If DeepL rejects
	// any of them for this plan, we retry without.
	extras := map[string]any{
		"formality":  "prefer_less",
		"model_type": "prefer_quality_optimized",
		"context": "Casual in-game chat between players in World of Warcraft Classic. " +
			"Players write informally, use gaming slang, and keep terms like tank, heal, dps, LFG in English.",
	}
	for k, v := range extras {
		body[k] = v
	}
	buf, _ := json.Marshal(body)
	url := endpoint(key)
	post := func(u string) (*http.Response, error) {
		req, _ := http.NewRequest("POST", u, bytes.NewReader(buf))
		req.Header.Set("Authorization", "DeepL-Auth-Key "+key)
		req.Header.Set("Content-Type", "application/json")
		return t.http.Do(req)
	}
	resp, err := post(url)
	if err == nil && resp.StatusCode == 403 && os.Getenv("PARLEY_DEEPL_URL") == "" {
		// Key may belong to the other API host (free vs paid); try it once.
		other := "https://api.deepl.com/v2/translate"
		if url == other {
			other = "https://api-free.deepl.com/v2/translate"
		}
		if r2, err2 := post(other); err2 == nil {
			resp.Body.Close()
			resp, url = r2, other
		}
	}
	if err == nil && resp.StatusCode == 400 {
		resp.Body.Close()
		for k := range extras {
			delete(body, k)
		}
		buf, _ = json.Marshal(body)
		resp, err = post(url)
	}
	if err != nil {
		return Translation{}, fmt.Errorf("DeepL unreachable: %v", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case 200:
	case 403:
		return Translation{}, errors.New("DeepL rejected the API key")
	case 456:
		return Translation{}, errors.New("DeepL character allowance used up")
	case 429:
		return Translation{}, errors.New("DeepL rate limit, slowing down")
	default:
		return Translation{}, fmt.Errorf("DeepL error %d", resp.StatusCode)
	}
	var out struct {
		Translations []struct {
			Text     string `json:"text"`
			Detected string `json:"detected_source_language"`
		} `json:"translations"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil || len(out.Translations) == 0 {
		return Translation{}, errors.New("DeepL sent an unreadable reply")
	}
	tr := Translation{Text: fromXML(out.Translations[0].Text), Detected: out.Translations[0].Detected}
	t.mu.Lock()
	t.cache[ck] = tr
	t.order = append(t.order, ck)
	if len(t.order) > 2000 {
		delete(t.cache, t.order[0])
		t.order = t.order[1:]
	}
	t.mu.Unlock()
	return tr, nil
}

// Usage returns characters used and the limit for the key's billing period.
func (t *Translator) Usage(key string) (used, limit int64, err error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return 0, 0, errors.New("no key")
	}
	url := strings.Replace(endpoint(key), "/v2/translate", "/v2/usage", 1)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "DeepL-Auth-Key "+key)
	resp, err := t.http.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return 0, 0, fmt.Errorf("DeepL error %d", resp.StatusCode)
	}
	var u struct {
		Count int64 `json:"character_count"`
		Limit int64 `json:"character_limit"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return 0, 0, err
	}
	return u.Count, u.Limit, nil
}

// usageText is "412,000 of 1,000,000 characters used (41%)".
func usageText(used, limit int64) string {
	if limit <= 0 {
		return Tf("%s characters used", commas(used))
	}
	return Tf("%s of %s characters used (%d%%)", commas(used), commas(limit), used*100/limit)
}

func commas(n int64) string {
	s := fmt.Sprint(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

// Base language of a DeepL code: "EN-US" -> "EN".
func baseLang(code string) string {
	code = strings.ToUpper(code)
	if i := strings.IndexByte(code, '-'); i > 0 {
		return code[:i]
	}
	return code
}

// Languages offered in the UI (DeepL target codes).
var targetLangs = []struct{ Code, Name string }{
	{"EN-US", "English (US)"}, {"EN-GB", "English (UK)"}, {"ES", "Spanish"},
	{"PT-BR", "Portuguese (BR)"}, {"PT-PT", "Portuguese (PT)"}, {"FR", "French"},
	{"DE", "German"}, {"IT", "Italian"}, {"RU", "Russian"}, {"UK", "Ukrainian"},
	{"PL", "Polish"}, {"NL", "Dutch"}, {"SV", "Swedish"}, {"DA", "Danish"},
	{"NB", "Norwegian"}, {"FI", "Finnish"}, {"CS", "Czech"}, {"TR", "Turkish"},
	{"EL", "Greek"}, {"HU", "Hungarian"}, {"RO", "Romanian"}, {"ZH-HANS", "Chinese (Simpl.)"},
	{"ZH-HANT", "Chinese (Trad.)"}, {"JA", "Japanese"}, {"KO", "Korean"}, {"AR", "Arabic"},
}

// Reply target for a detected source language.
func replyTarget(detected string) string {
	switch strings.ToUpper(detected) {
	case "EN":
		return "EN-US"
	case "PT":
		return "PT-BR"
	case "ZH":
		return "ZH-HANS"
	case "":
		return ""
	}
	return strings.ToUpper(detected)
}
