package main

import (
	"regexp"
	"testing"
)

func TestUILangForLocale(t *testing.T) {
	cases := map[string]string{"pt-BR": "pt", "pt-PT": "pt", "de-DE": "de", "es-MX": "es", "fr-CA": "fr", "ru-RU": "ru",
		"ko-KR": "ko", "zh-CN": "zh-Hans", "zh-TW": "zh-Hant", "zh-Hant-HK": "zh-Hant", "en-US": "en", "nl-NL": "en", "it-IT": "it"}
	for in, want := range cases {
		if got := uiLangForLocale(in); got != want {
			t.Errorf("%s -> %s, want %s", in, got, want)
		}
	}
}

func TestCatalogComplete(t *testing.T) {
	verb := regexp.MustCompile(`%[-+# 0]*\d*(?:\.\d+)?[a-zA-Z%]`)
	ref := uiText["de"]
	for lang, m := range uiText {
		if len(m) != len(ref) {
			t.Errorf("%s has %d strings, de has %d", lang, len(m), len(ref))
		}
		for k, v := range m {
			if k == "Overlay opacity % (40-100)" {
				continue
			}
			if a, b := verb.FindAllString(k, -1), verb.FindAllString(v, -1); len(a) != len(b) {
				t.Errorf("%s: verbs differ for %q", lang, k)
			}
		}
	}
	for _, m := range backendMessages {
		if _, ok := ref[m]; !ok {
			t.Errorf("backend message not translated: %q", m)
		}
	}
}

func TestTranslateAndLocalize(t *testing.T) {
	defer setUILang("en")
	setUILang("pt")
	if T("Save") != "Salvar" || Tf("Reply to %s", "Ana") != "Responder a Ana" {
		t.Error("T/Tf")
	}
	if T("not a key") != "not a key" {
		t.Error("fallback")
	}
	if got := localizeMsg("DeepL error 503"); got != "Erro do DeepL 503" {
		t.Errorf("pattern: %q", got)
	}
	if got := localizeMsg("DeepL rejected the API key"); got != "O DeepL recusou a chave de API" {
		t.Errorf("exact: %q", got)
	}
	if langLabel("RU") != "Русский" || langName("pt") != "Português" {
		t.Errorf("names %q %q", langLabel("RU"), langName("pt"))
	}
	setUILang("en")
	if langLabel("RU") != "Russian" || T("Save") != "Save" {
		t.Error("english")
	}
	setUILang("xx")
	if uiLang != "en" {
		t.Error("unknown falls back to en")
	}
}

func TestChatLangForLocale(t *testing.T) {
	cases := map[string]string{"pt-BR": "PT-BR", "pt-PT": "PT-PT", "de-AT": "DE", "en-GB": "EN-GB", "en-US": "EN-US",
		"zh-TW": "ZH-HANT", "zh-CN": "ZH-HANS", "nb-NO": "NB", "tr-TR": "TR", "xx-YY": "EN-US", "ru-RU": "RU"}
	for in, want := range cases {
		if got := chatLangForLocale(in); got != want {
			t.Errorf("%s -> %s, want %s", in, got, want)
		}
	}
}
