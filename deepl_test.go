package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestToXMLKeepsJargon(t *testing.T) {
	got := toXML("LF1M tank for SM, can pay gold for [Thunderfury] & heals", "PT-BR")
	want := "<x>LF1M</x> <x>tank</x> for <x>SM</x>, can pay <x>gold</x> for <x>[Thunderfury]</x> &amp; <x>heals</x>"
	if got != want {
		t.Fatalf("\n got %s\nwant %s", got, want)
	}
	if fromXML(got) != "LF1M tank for SM, can pay gold for [Thunderfury] & heals" {
		t.Fatal("round trip")
	}
	if toXML("standard stash", "PT-BR") != "standard stash" {
		t.Fatal("matched inside words")
	}
}

func TestExtrasFallback(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var b map[string]any
		json.NewDecoder(r.Body).Decode(&b)
		if _, ok := b["model_type"]; ok {
			w.WriteHeader(400)
			return
		}
		w.Write([]byte(`{"translations":[{"detected_source_language":"PT","text":"hi <x>tank</x>"}]}`))
	}))
	defer srv.Close()
	t.Setenv("PARLEY_DEEPL_URL", srv.URL)
	tr, err := NewTranslator().Translate("k", "oi tank", "", "EN-US")
	if err != nil || tr.Text != "hi tank" || calls != 2 {
		t.Fatalf("tr=%+v err=%v calls=%d", tr, err, calls)
	}
}

func TestUsage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/usage" || r.Header.Get("Authorization") != "DeepL-Auth-Key k:fx" {
			w.WriteHeader(403)
			return
		}
		w.Write([]byte(`{"character_count":412345,"character_limit":1000000}`))
	}))
	defer srv.Close()
	t.Setenv("PARLEY_DEEPL_URL", srv.URL+"/v2/translate")
	used, limit, err := NewTranslator().Usage("k:fx")
	if err != nil || used != 412345 || limit != 1000000 {
		t.Fatalf("%d %d %v", used, limit, err)
	}
	if got := usageText(used, limit); got != "412,345 of 1,000,000 characters used (41%)" {
		t.Fatal(got)
	}
	if commas(999) != "999" || commas(1000) != "1,000" {
		t.Fatal("commas")
	}
}
