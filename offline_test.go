package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

type fakeEngine struct {
	detect    string
	calls     []string
	lastFirst string
	lastSec   string
}

func (f *fakeEngine) Detect(text string) (string, int, bool, error) {
	return f.detect, 95, f.detect != "un", nil
}

func (f *fakeEngine) Translate(text string, html bool, first, second string) (string, error) {
	f.calls = append(f.calls, text)
	f.lastFirst, f.lastSec = first, second
	return "<b>T:</b> " + text, nil
}

func raw(kind, pair string) string { return strings.Repeat(kind+"-"+pair+"-", 2000) }

func sha(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func rec(kind, src, dst, version, arch, filter string) modelRecord {
	r := modelRecord{Name: kind + "." + src + dst + ".bin", Version: version, FileType: kind, Architecture: arch,
		SourceLanguage: src, TargetLanguage: dst, FilterExpression: filter,
		DecompressedHash: sha(raw(kind, src+dst))}
	if kind == "vocab" {
		r.Name = "vocab." + src + dst + ".spm"
	}
	r.Attachment.Location = "files/" + kind + "." + src + dst + ".zst"
	r.Attachment.Size = 29
	return r
}

func testServer(t *testing.T, records []modelRecord, hits *int32) *httptest.Server {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/":
			json.NewEncoder(w).Encode(map[string]any{"capabilities": map[string]any{
				"attachments": map[string]any{"base_url": srv.URL + "/att/"}}})
		case strings.HasSuffix(r.URL.Path, "/records"):
			json.NewEncoder(w).Encode(map[string]any{"data": records})
		case strings.HasPrefix(r.URL.Path, "/att/files/"):
			if hits != nil {
				atomic.AddInt32(hits, 1)
			}
			http.ServeFile(w, r, filepath.Join("testdata/offline", strings.TrimPrefix(r.URL.Path, "/att/files/")))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func allRecords() []modelRecord {
	var rs []modelRecord
	for _, p := range [][2]string{{"pt", "en"}, {"en", "es"}, {"en", "pt"}} {
		for _, k := range []string{"model", "vocab", "lex"} {
			rs = append(rs, rec(k, p[0], p[1], "3.0", "base-memory", ""))
		}
	}
	// An Android-only newer pt->en and a nightly-only one must be ignored.
	rs = append(rs, rec("model", "pt", "en", "9.0", "base", "env.appinfo.OS == 'Android'"))
	rs = append(rs, rec("model", "pt", "en", "9.1", "base", "env.channel == 'nightly'"))
	return rs
}

func TestOfflineDownloadAndTranslate(t *testing.T) {
	var hits int32
	srv := testServer(t, allRecords(), &hits)
	eng := &fakeEngine{detect: "pt"}
	o := NewOffline(t.TempDir(), eng)
	o.Settings = srv.URL + "/v1"
	var msgs []string
	o.Progress = func(m string) { msgs = append(msgs, m) }

	tr, err := o.Translate("vc vai no [Ragefire Chasm] tank?", "", "EN-US")
	if err != nil {
		t.Fatal(err)
	}
	if tr.Detected != "PT" {
		t.Errorf("detected %q", tr.Detected)
	}
	if tr.Text != "T: vc vai no [Ragefire Chasm] tank?" {
		t.Errorf("text %q", tr.Text)
	}
	if got := eng.calls[0]; !strings.Contains(got, "<code>[Ragefire Chasm]</code>") || !strings.Contains(got, "<code>tank</code>") {
		t.Errorf("not protected: %q", got)
	}
	if filepath.Base(eng.lastFirst) != "pt-en" || eng.lastSec != "" {
		t.Errorf("models %q %q", eng.lastFirst, eng.lastSec)
	}
	for _, f := range []string{"model.pten.bin", "vocab.pten.spm", "lex.pten.bin", "ready.json"} {
		b, err := os.ReadFile(filepath.Join(o.Dir, "pt-en", f))
		if err != nil {
			t.Fatal(err)
		}
		if f == "model.pten.bin" && string(b) != raw("model", "pten") {
			t.Error("model not decompressed correctly")
		}
	}
	if hits != 3 {
		t.Errorf("downloads = %d", hits)
	}
	if len(msgs) == 0 || !strings.Contains(msgs[len(msgs)-1], "ready") {
		t.Errorf("progress %v", msgs)
	}

	// Pivot: Portuguese -> Spanish goes pt->en then en->es; pt-en is reused.
	tr, err = o.Translate("bora", "PT", "ES")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(eng.lastFirst) != "pt-en" || filepath.Base(eng.lastSec) != "en-es" {
		t.Errorf("pivot models %q %q", eng.lastFirst, eng.lastSec)
	}
	if hits != 6 {
		t.Errorf("downloads after pivot = %d", hits)
	}

	// Replies: English -> Portuguese.
	if _, err = o.Translate("sorry wait", "EN", "PT-BR"); err != nil {
		t.Fatal(err)
	}
	if filepath.Base(eng.lastFirst) != "en-pt" {
		t.Errorf("reply model %q", eng.lastFirst)
	}

	// Same language: nothing to do, reported so the caller hides it.
	tr, err = o.Translate("hello there", "EN", "EN-GB")
	if err != nil || tr.Detected != "EN" {
		t.Errorf("same-language %v %v", tr, err)
	}

	// Short Chinese that CLD2 can't place falls back to the script hint.
	eng.detect = "un"
	if _, err := o.Translate("坦克", "ZH", "EN-US"); err == nil || !strings.Contains(err.Error(), "Chinese") {
		t.Errorf("zh fallback: %v", err)
	}
	eng.detect = "pt"

	// Unsupported pair gives a clear error.
	if _, err = o.Translate("ciao ç", "IT", "EN-US"); err == nil || !strings.Contains(err.Error(), "Italian") {
		t.Errorf("unsupported: %v", err)
	}
	// ASCII in an unsupported language comes back unchanged (caller hides it).
	eng.detect = "it"
	if tr, err := o.Translate("ciao", "", "EN-US"); err != nil || tr.Text != "ciao" {
		t.Errorf("unsupported ascii: %v %v", tr, err)
	}
	// ...but a reply into an unsupported language must fail loudly.
	if _, err := o.Translate("hello", "EN", "IT"); err == nil {
		t.Error("reply into unsupported language should error")
	}
	eng.detect = "pt"

	// A new instance finds the models already on disk and the cached catalog.
	o2 := NewOffline(o.Dir, eng)
	o2.Settings = "http://127.0.0.1:1/v1" // unreachable
	if _, err := o2.Translate("obrigado", "PT", "EN-US"); err != nil {
		t.Fatalf("offline reuse: %v", err)
	}
}

func TestOfflineRejectsDamagedDownload(t *testing.T) {
	rs := allRecords()
	for i := range rs {
		if rs[i].FileType == "model" && rs[i].SourceLanguage == "pt" {
			rs[i].DecompressedHash = sha("something else")
		}
	}
	srv := testServer(t, rs, nil)
	o := NewOffline(t.TempDir(), &fakeEngine{detect: "pt"})
	o.Settings = srv.URL + "/v1"
	if _, err := o.Translate("bora", "PT", "EN-US"); err == nil || !strings.Contains(err.Error(), "damaged") {
		t.Fatalf("want damaged error, got %v", err)
	}
	if pairReady(filepath.Join(o.Dir, "pt-en")) {
		t.Error("damaged pair marked ready")
	}
}

func TestPickRecordsPrefersNewestDesktop(t *testing.T) {
	rs := []modelRecord{
		rec("model", "ru", "en", "3.0", "tiny", ""), rec("vocab", "ru", "en", "3.0", "tiny", ""),
		rec("model", "ru", "en", "3.1", "base-memory", ""), rec("vocab", "ru", "en", "3.1", "base-memory", ""),
		rec("lex", "ru", "en", "3.1", "base-memory", ""),
		rec("model", "ru", "en", "3.2", "base", "env.appinfo.OS == 'Android'"),
		rec("vocab", "ru", "en", "3.2", "base", "env.appinfo.OS == 'Android'"),
	}
	got := pickRecords(rs, "ru", "en")
	if len(got) != 3 || got[0].Version != "3.1" {
		t.Fatalf("got %+v", got)
	}
	rs = []modelRecord{
		rec("model", "en", "ru", "3.0", "base", "env.appinfo.OS != 'Android'"), rec("vocab", "en", "ru", "3.0", "base", "env.appinfo.OS != 'Android'"),
		rec("model", "en", "ru", "3.0", "base-memory", ""), rec("vocab", "en", "ru", "3.0", "base-memory", ""),
	}
	if got := pickRecords(rs, "en", "ru"); got[0].Architecture != "base" {
		t.Errorf("want base for desktop, got %s", got[0].Architecture)
	}
	if pickRecords(rs, "en", "xx") != nil {
		t.Error("expected nil for unknown pair")
	}
	if versionKey("3.0a1") == versionKey("3.0") || !(versionKey("3.0")[3] > versionKey("3.0a1")[3]) {
		t.Error("pre-release ordering")
	}
}

func TestCodes(t *testing.T) {
	cases := map[string]string{"PT-BR": "pt", "EN-US": "en", "ZH-HANS": "zh-Hans", "ZH-HANT": "zh-Hant", "NB": "nb", "RU": "ru"}
	for in, want := range cases {
		if got := ffCode(in); got != want {
			t.Errorf("ffCode(%s)=%s want %s", in, got, want)
		}
	}
	if uiCode("zh-Hant") != "ZH-HANT" || uiCode("pt") != "PT" || replyTarget(uiCode("pt")) != "PT-BR" {
		t.Error("uiCode")
	}
	if cld2Code("iw") != "he" || cld2Code("no") != "nb" || cld2Code("un") != "" {
		t.Error("cld2Code")
	}
	if got := expandPortuguese("To sem mana, alguem tem? so um pouco"); got != "Estou sem mana, alguém tem? só um pouco" {
		t.Errorf("expandPortuguese %q", got)
	}
	if guessLatinLang("vc vai?") != "pt" || guessLatinLang("hola amigo") != "es" || guessLatinLang("qwerty") != "" {
		t.Error("guessLatinLang")
	}
	if got := fromHTML("Olá <code>[Linen Cloth]</code> &amp; <code>tank</code>"); got != "Olá [Linen Cloth] & tank" {
		t.Errorf("fromHTML %q", got)
	}
	for in, want := range map[string]string{
		"Eu preciso de um  <code>healer</code>para Deadmines":                             "Eu preciso de um healer para Deadmines",
		"de <code>Deadmines</code>,  <code>LFG</code><code>tank</code>, <code>pst</code>": "de Deadmines, LFG tank, pst",
		"you go in the  <code>SM</code>today?":                                            "you go in the SM today?",
	} {
		if got := fromHTML(in); got != want {
			t.Errorf("fromHTML(%q) = %q, want %q", in, got, want)
		}
	}
	for _, c := range [][3]string{
		{"Sim! Sim! Tenho 5 restantes.", "Yes! I have 5 left.", "Sim! Tenho 5 restantes."},
		{"Obrigado! Obrigado! Convida-me", "Thanks! Invite me", "Obrigado! Convida-me"},
		{"Não! Não! Pare!", "No! No! Stop!", "Não! Não! Pare!"},
		{"Olá.", "Hi.", "Olá."},
	} {
		if got := dropEchoes(c[0], c[1]); got != c[2] {
			t.Errorf("dropEchoes(%q) = %q, want %q", c[0], got, c[2])
		}
	}
	for in, want := range map[string]string{
		"Searching tank and healer For <code>Deadmines</code>, there are only two": "Searching tank and healer for <code>Deadmines</code>, there are only two",
		"Who goes in the <code>raid</code> Tomorrow?":                              "Who goes in the <code>raid</code> tomorrow?",
		"Hello! <code>Tank</code> Here":                                            "Hello! <code>Tank</code> here",
		"Welcome To <code>Stormwind</code>":                                        "Welcome to <code>Stormwind</code>",
		"<code>LFG</code>. Deadmines now":                                          "<code>LFG</code>. Deadmines now",
		"Need Mage for <code>ZF</code>":                                            "Need Mage for <code>ZF</code>",
	} {
		src := "Mage"
		if got := decapNearCode(in, src); got != want {
			t.Errorf("decapNearCode(%q) = %q, want %q", in, got, want)
		}
	}
	if got := toHTML("I'll <b> & tank"); got != "I'll &lt;b&gt; &amp; <code>tank</code>" {
		t.Errorf("toHTML %q", got)
	}
	if langName("pt") != "Portuguese" || langName("zh-Hant") != "Chinese (Trad.)" {
		t.Errorf("langName %s %s", langName("pt"), langName("zh-Hant"))
	}
}

func TestScriptLang(t *testing.T) {
	for in, want := range map[string]string{"не агрите патруль, ждём хила": "ru", "дякую, їдемо": "uk", "hello": "", "ok спс": "ru", "LFM ZF need tank спс": ""} {
		if got := scriptLang(in); got != want {
			t.Errorf("scriptLang(%q) = %q, want %q", in, got, want)
		}
	}
}
