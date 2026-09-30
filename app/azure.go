package main

// Microsoft Azure AI Translator (Translator Text API v3). Free tier: 2
// million characters a month. Needs a key and, for regional resources, the
// resource's region (e.g. "westeurope").

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Azure struct {
	mu    sync.Mutex
	cache map[string]Translation
	order []string
	http  *http.Client
}

func NewAzure() *Azure {
	return &Azure{cache: map[string]Translation{}, http: &http.Client{Timeout: 10 * time.Second}}
}

func azureEndpoint() string {
	if u := os.Getenv("PARLEY_AZURE_URL"); u != "" { // testing only
		return u
	}
	return "https://api.cognitive.microsofttranslator.com/translate"
}

// azCode turns Parley's DeepL-style codes ("PT-BR", "ZH-HANT", "EN-US")
// into Azure's ("pt", "zh-Hant", "en").
func azCode(code string) string {
	c := strings.ToUpper(strings.TrimSpace(code))
	switch c {
	case "":
		return ""
	case "PT-PT":
		return "pt-pt"
	case "ZH-HANT", "ZH-TW":
		return "zh-Hant"
	case "ZH", "ZH-HANS", "ZH-CN":
		return "zh-Hans"
	case "NO", "NB":
		return "nb"
	}
	return strings.ToLower(baseLang(c))
}

// Azure's HTML mode skips anything marked translate="no".
var azNoTagRe = regexp.MustCompile(`</?span[^>]*>`)

func toAzureHTML(s, target string) string {
	esc := html.EscapeString(keepRe.ReplaceAllString(s, "$1"))
	wrap := func(m string) string { return `<span translate="no">` + m + `</span>` }
	esc = lockRe.ReplaceAllString(esc, `<span translate="no">$1</span>`) // the player's own words
	esc = bracketRe.ReplaceAllStringFunc(esc, wrap)
	if baseLang(target) == "EN" {
		return esc
	}
	return jargonRe.ReplaceAllStringFunc(esc, wrap)
}

func fromAzureHTML(s string) string {
	s = regexp.MustCompile(`([\p{L}\p{N}])<span`).ReplaceAllString(s, "$1 <span")
	s = regexp.MustCompile(`</span>([\p{L}\p{N}])`).ReplaceAllString(s, "</span> $1")
	return strings.TrimSpace(html.UnescapeString(azNoTagRe.ReplaceAllString(s, "")))
}

// Translate sends one message to Azure. source may be "" to auto-detect.
func (a *Azure) Translate(key, region, text, source, target string) (Translation, error) {
	key, region = strings.TrimSpace(key), strings.TrimSpace(region)
	if key == "" {
		return Translation{}, errors.New("add your Azure key in Settings")
	}
	ck := source + "|" + target + "|" + text
	a.mu.Lock()
	if tr, ok := a.cache[ck]; ok {
		a.mu.Unlock()
		return tr, nil
	}
	a.mu.Unlock()

	q := url.Values{"api-version": {"3.0"}, "to": {azCode(target)}, "textType": {"html"}}
	if s := azCode(source); s != "" {
		q.Set("from", s)
	}
	body, _ := json.Marshal([]map[string]string{{"Text": toAzureHTML(text, target)}})
	req, _ := http.NewRequest("POST", azureEndpoint()+"?"+q.Encode(), bytes.NewReader(body))
	req.Header.Set("Ocp-Apim-Subscription-Key", key)
	if region != "" && !strings.EqualFold(region, "global") {
		req.Header.Set("Ocp-Apim-Subscription-Region", region)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.http.Do(req)
	if err != nil {
		return Translation{}, fmt.Errorf("Azure unreachable: %v", err)
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == 200:
	case resp.StatusCode == 401:
		return Translation{}, errors.New("Azure rejected the key or region")
	case resp.StatusCode == 403:
		return Translation{}, errors.New("Azure character allowance used up")
	case resp.StatusCode == 429:
		return Translation{}, errors.New("Azure rate limit, slowing down")
	default:
		return Translation{}, fmt.Errorf("Azure error %d", resp.StatusCode)
	}
	var out []struct {
		Detected struct {
			Language string `json:"language"`
		} `json:"detectedLanguage"`
		Translations []struct {
			Text string `json:"text"`
		} `json:"translations"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil || len(out) == 0 || len(out[0].Translations) == 0 {
		return Translation{}, errors.New("Azure sent an unreadable reply")
	}
	det := out[0].Detected.Language
	if det == "" {
		det = azCode(source)
	}
	tr := Translation{Text: fromAzureHTML(out[0].Translations[0].Text), Detected: uiCode(det)}
	a.mu.Lock()
	a.cache[ck] = tr
	a.order = append(a.order, ck)
	if len(a.order) > 2000 {
		delete(a.cache, a.order[0])
		a.order = a.order[1:]
	}
	a.mu.Unlock()
	return tr, nil
}
