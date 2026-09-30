package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestNewerVersion(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want bool
	}{{"1.6.0", "1.5.0", true}, {"1.10.0", "1.9.9", true}, {"v1.6.0", "1.6.0", false}, {"1.5.0", "1.6.0", false}, {"2.0", "1.9.9", true}} {
		if got := newerVersion(c.a, c.b); got != c.want {
			t.Errorf("newerVersion(%s,%s)=%v", c.a, c.b, got)
		}
	}
}

func TestLatestRelease(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("User-Agent"), "Parley/") {
			t.Error("no user agent")
		}
		w.Write([]byte(`{"tag_name":"v1.7.0","html_url":"https://github.com/Duhlude/parley/releases/tag/v1.7.0","draft":false,"prerelease":false}`))
	}))
	defer srv.Close()
	t.Setenv("PARLEY_UPDATE_URL", srv.URL)
	r, err := latestRelease()
	if err != nil || r.Version != "1.7.0" || !strings.Contains(r.URL, "v1.7.0") {
		t.Fatalf("%+v %v", r, err)
	}
}

func TestAlerts(t *testing.T) {
	terms := alertTerms("Deadmines, healer ; WTB,", "Palshall-Whitemane")
	if strings.Join(terms, "|") != "Deadmines|healer|WTB|Palshall" {
		t.Fatalf("terms %q", terms)
	}
	for _, c := range []struct {
		texts []string
		want  bool
	}{
		{[]string{"alguém pra Minas Mortas?", "anyone for Deadmines?"}, true},
		{[]string{"palshall vem aqui"}, true},
		{[]string{"we need healers"}, false}, // whole words only
		{[]string{"wtb linen"}, true},
		{[]string{"nothing"}, false},
	} {
		if got := mentions(terms, c.texts...); got != c.want {
			t.Errorf("mentions(%q) = %v", c.texts, got)
		}
	}
	if !mentions([]string{"死亡矿井"}, "有人去死亡矿井吗") {
		t.Error("CJK term should match inside text")
	}
}

func TestPhrasebook(t *testing.T) {
	p := phrasebook[0]
	for code, want := range map[string]string{"PT-BR": "Me convida, por favor", "ZH-HANT": "請邀請我一下", "ZH-HANS": "请邀请我一下", "JA": "招待お願いします", "EN-GB": "Invite me, please", "PL": ""} {
		if got := p.ready(code); got != want {
			t.Errorf("ready(%s) = %q, want %q", code, got, want)
		}
	}
	for _, p := range phrasebook {
		if len(p.in) != 10 {
			t.Errorf("%q has %d languages", p.EN, len(p.in))
		}
	}
}

func TestAzure(t *testing.T) {
	var got struct {
		key, region, query string
		body               []map[string]string
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.key, got.region, got.query = r.Header.Get("Ocp-Apim-Subscription-Key"), r.Header.Get("Ocp-Apim-Subscription-Region"), r.URL.RawQuery
		json.NewDecoder(r.Body).Decode(&got.body)
		switch r.Header.Get("Ocp-Apim-Subscription-Key") {
		case "bad":
			w.WriteHeader(401)
			return
		case "quota":
			w.WriteHeader(403)
			return
		}
		w.Write([]byte(`[{"detectedLanguage":{"language":"pt","score":1},"translations":[{"text":"Anyone for<span translate=\"no\">[Linen Bag]</span>? &amp; thanks","to":"en"}]}]`))
	}))
	defer srv.Close()
	t.Setenv("PARLEY_AZURE_URL", srv.URL)
	a := NewAzure()
	tr, err := a.Translate("k1", "westeurope", "alguém pra [Linen Bag]? & vlw", "", "EN-US")
	if err != nil {
		t.Fatal(err)
	}
	if tr.Text != "Anyone for [Linen Bag]? & thanks" || tr.Detected != "PT" {
		t.Errorf("got %+v", tr)
	}
	if got.key != "k1" || got.region != "westeurope" || !strings.Contains(got.query, "to=en") || strings.Contains(got.query, "from=") {
		t.Errorf("request %+v", got)
	}
	if !strings.Contains(got.body[0]["Text"], `<span translate="no">[Linen Bag]</span>`) {
		t.Errorf("item link not protected: %q", got.body[0]["Text"])
	}
	a.Translate("k1", "global", "tank pls", "EN", "PT-BR")
	if got.region != "" || !strings.Contains(got.query, "from=en") || !strings.Contains(got.query, "to=pt") ||
		!strings.Contains(got.body[0]["Text"], `<span translate="no">tank</span>`) {
		t.Errorf("reply request %+v", got)
	}
	if _, err := a.Translate("bad", "x", "hola", "", "EN-US"); err == nil || !strings.Contains(err.Error(), "rejected") {
		t.Errorf("bad key: %v", err)
	}
	if _, err := a.Translate("quota", "x", "hola", "", "EN-US"); err == nil || !strings.Contains(err.Error(), "allowance") {
		t.Errorf("quota: %v", err)
	}
	if azCode("ZH-HANS") != "zh-Hans" || azCode("PT-BR") != "pt" || azCode("EN-US") != "en" || azCode("ZH-HANT") != "zh-Hant" {
		t.Error("azCode")
	}
}

func TestDownloadSetup(t *testing.T) {
	payload := []byte("MZ" + strings.Repeat("parley", 50000))
	sum := sha256.Sum256(payload)
	good := hex.EncodeToString(sum[:])
	mux := http.NewServeMux()
	mux.HandleFunc("/Parley-Setup.exe", func(w http.ResponseWriter, r *http.Request) { w.Write(payload) })
	srv := httptest.NewServer(mux)
	defer srv.Close()
	mux.HandleFunc("/latest", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tag_name":"1.7.0","html_url":"x","assets":[` +
			`{"name":"Parley-addon-1.6.0.zip","browser_download_url":"` + srv.URL + `/a.zip","size":5,"digest":"sha256:` + good + `"},` +
			`{"name":"Parley-Setup.exe","browser_download_url":"` + srv.URL + `/Parley-Setup.exe","size":` + strconv.Itoa(len(payload)) + `,"digest":"sha256:` + good + `"}]}`))
	})
	t.Setenv("PARLEY_UPDATE_URL", srv.URL+"/latest")
	r, err := latestRelease()
	if err != nil || !r.canInstall() || r.SetupSHA256 != good || r.SetupSize != int64(len(payload)) {
		t.Fatalf("%+v %v", r, err)
	}
	dir := t.TempDir()
	var seen []int
	p, err := downloadSetup(r, dir, func(pct int) { seen = append(seen, pct) })
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(p); !bytes.Equal(b, payload) || filepath.Base(p) != "Parley-Setup-1.7.0.exe" {
		t.Fatalf("bad file %s", p)
	}
	if len(seen) == 0 || seen[len(seen)-1] != 100 {
		t.Errorf("progress %v", seen)
	}

	bad := r
	bad.SetupSHA256 = strings.Repeat("0", 64)
	if _, err := downloadSetup(bad, dir, nil); err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Errorf("tampered download accepted: %v", err)
	}
	short := r
	short.SetupSize++
	if _, err := downloadSetup(short, dir, nil); err == nil {
		t.Error("short download accepted")
	}
	long := r
	long.SetupSize--
	if _, err := downloadSetup(long, dir, nil); err == nil {
		t.Error("oversized download accepted")
	}
	if left, _ := filepath.Glob(filepath.Join(dir, "*.part")); len(left) != 0 {
		t.Errorf("left behind %v", left)
	}
	none := r
	none.SetupSHA256 = ""
	if none.canInstall() {
		t.Error("installable without a checksum")
	}
	if sanitizeVersion(`1.7.0\..\evil`) != "1.7.0.." {
		t.Error(sanitizeVersion(`1.7.0\..\evil`))
	}
}
