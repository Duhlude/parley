package main

// Offline translation: Firefox Translations models (Mozilla, MPL-2.0) run on
// this PC by parley-mt.exe (bergamot-translator + CLD2). No account, no key,
// nothing sent anywhere except the one-time model downloads.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"parley/internal/zstd"
)

const (
	defaultSettingsURL = "https://firefox.settings.services.mozilla.com/v1"
	defaultAttachURL   = "https://firefox-settings-attachments.cdn.mozilla.net/"
	modelCollection    = "/buckets/main/collections/translations-models-v2/records"
)

type unsupportedError struct{ msg string }

func (e unsupportedError) Error() string { return e.msg }

// mtEngine is the translation process (parley-mt.exe in production).
type mtEngine interface {
	Detect(text string) (code string, percent int, reliable bool, err error)
	Translate(text string, html bool, first, second string) (string, error)
}

// modelRecord is one file in Mozilla's Remote Settings model catalog.
type modelRecord struct {
	Name             string `json:"name"`
	Version          string `json:"version"`
	FileType         string `json:"fileType"`
	Architecture     string `json:"architecture"`
	SourceLanguage   string `json:"sourceLanguage"`
	TargetLanguage   string `json:"targetLanguage"`
	FilterExpression string `json:"filter_expression"`
	DecompressedHash string `json:"decompressedHash"`
	DecompressedSize int64  `json:"decompressedSize"`
	Attachment       struct {
		Location string `json:"location"`
		Size     int64  `json:"size"`
	} `json:"attachment"`
}

type catalogFile struct {
	Fetched time.Time     `json:"fetched"`
	Attach  string        `json:"attach"`
	Records []modelRecord `json:"records"`
}

// Offline translates with locally downloaded models.
type Offline struct {
	Dir      string // where models live, e.g. %APPDATA%\Parley\models\translate
	Settings string // Remote Settings base URL
	Engine   mtEngine
	HTTP     *http.Client
	Progress func(msg string) // download progress for the UI (may be nil)

	mu       sync.Mutex
	catalog  *catalogFile
	pairLock map[string]*sync.Mutex
	cache    map[string]Translation
	order    []string
}

func NewOffline(dir string, engine mtEngine) *Offline {
	settings := defaultSettingsURL
	if u := os.Getenv("PARLEY_MODELS_URL"); u != "" { // testing only
		settings = strings.TrimRight(u, "/")
	}
	return &Offline{
		Dir: dir, Settings: settings, Engine: engine,
		HTTP:     &http.Client{Timeout: 5 * time.Minute},
		pairLock: map[string]*sync.Mutex{}, cache: map[string]Translation{},
	}
}

// ---------------------------------------------------------------------------
// Language codes
// ---------------------------------------------------------------------------

// ffCode turns a UI/DeepL code ("PT-BR", "ZH-HANT", "EN-US") into a Firefox
// model code ("pt", "zh-Hant", "en").
func ffCode(code string) string {
	c := strings.ToUpper(strings.TrimSpace(code))
	switch c {
	case "ZH", "ZH-HANS", "ZH-CN":
		return "zh-Hans"
	case "ZH-HANT", "ZH-TW":
		return "zh-Hant"
	case "NO", "NB":
		return "nb"
	}
	return strings.ToLower(baseLang(c))
}

// uiCode turns a Firefox code back into the DeepL-style code Parley shows.
func uiCode(ff string) string {
	switch ff {
	case "zh-Hans":
		return "ZH"
	case "zh-Hant":
		return "ZH-HANT"
	}
	return strings.ToUpper(ff)
}

// cld2Code maps CLD2 language codes to Firefox model codes.
func cld2Code(c string) string {
	switch c {
	case "zh":
		return "zh-Hans"
	case "no", "nn":
		return "nb"
	case "iw":
		return "he"
	case "un", "xx", "":
		return ""
	}
	return c
}

// ---------------------------------------------------------------------------
// Translate
// ---------------------------------------------------------------------------

// Translate mirrors Translator.Translate. source is a DeepL-style hint
// ("" = detect), target a DeepL-style code.
// scriptLang guesses from the alphabet alone: mostly-Cyrillic text is
// Ukrainian if it uses Ukrainian-only letters, otherwise Russian.
func scriptLang(text string) string {
	cyr, letters := 0, 0
	uk := false
	for _, r := range text {
		if !unicode.IsLetter(r) {
			continue
		}
		letters++
		if unicode.Is(unicode.Cyrillic, r) {
			cyr++
			switch unicode.ToLower(r) {
			case 'і', 'ї', 'є', 'ґ':
				uk = true
			}
		}
	}
	if letters == 0 || cyr*2 < letters {
		return ""
	}
	if uk {
		return "uk"
	}
	return "ru"
}

// clearCache forgets earlier translations (benchmark only).
func (o *Offline) clearCache() {
	o.mu.Lock()
	o.cache, o.order = map[string]Translation{}, nil
	o.mu.Unlock()
}

func (o *Offline) Translate(text, source, target string) (Translation, error) {
	if o.Engine == nil {
		return Translation{}, errors.New("the offline translator isn't installed; reinstall Parley")
	}
	tgt := ffCode(target)
	ck := source + "|" + tgt + "|" + text
	o.mu.Lock()
	if tr, ok := o.cache[ck]; ok {
		o.mu.Unlock()
		return tr, nil
	}
	o.mu.Unlock()

	src := ""
	if source != "" && ffCode(source) != "zh-Hans" { // Chinese: let CLD2 tell simplified from traditional
		src = ffCode(source)
	}
	if src == "" {
		src = o.detect(stripKeep(text))
		if src == "" && source != "" {
			src = ffCode(source) // very short Chinese: go with the script
		}
		if src == "" {
			src = scriptLang(stripKeep(text)) // short Cyrillic chat CLD2 can't place
		}
		if src == "" {
			if isASCII(stripKeep(text)) { // unknown short Latin text: let the caller hide it
				return Translation{Text: stripKeep(text)}, nil
			}
			return Translation{}, errors.New("couldn't tell which language this is")
		}
	}
	if src == tgt {
		return Translation{Text: text, Detected: uiCode(src)}, nil
	}
	if src == "pt" {
		text = expandPortuguese(text)
	}

	var first, second string
	var err error
	switch {
	case src == "en":
		first, err = o.ensurePair("en", tgt)
	case tgt == "en":
		first, err = o.ensurePair(src, "en")
	default: // pivot through English
		if first, err = o.ensurePair(src, "en"); err == nil {
			second, err = o.ensurePair("en", tgt)
		}
	}
	if err != nil {
		var u unsupportedError
		if errors.As(err, &u) && source == "" && isASCII(text) {
			// Probably English slang CLD2 mistook for something exotic:
			// hand it back unchanged so the caller hides it.
			return Translation{Text: text, Detected: uiCode(src)}, nil
		}
		return Translation{}, err
	}
	out, err := o.Engine.Translate(toHTML(text), true, first, second)
	if err != nil {
		return Translation{}, fmt.Errorf("offline translation failed: %v", err)
	}
	tr := Translation{Text: dropEchoes(fromHTML(decapNearCode(out, text)), text), Detected: uiCode(src)}
	o.mu.Lock()
	o.cache[ck] = tr
	o.order = append(o.order, ck)
	if len(o.order) > 2000 {
		delete(o.cache, o.order[0])
		o.order = o.order[1:]
	}
	o.mu.Unlock()
	return tr, nil
}

// detect returns a Firefox language code, or "" if unsure.
func (o *Offline) detect(text string) string {
	code, pct, reliable, err := o.Engine.Detect(text)
	if err == nil {
		if c := cld2Code(code); c != "" && (reliable || pct >= 60) {
			return c
		}
	}
	return guessLatinLang(text)
}

// guessLatinLang votes with common chat words when CLD2 can't decide
// (very short lines like "vc vai?" or "hola").
func guessLatinLang(text string) string {
	votes := map[string]int{}
	for _, w := range strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && r != '\''
	}) {
		if isLaugh(w) {
			votes["pt"]++
			continue
		}
		for lang, words := range langWords {
			if words[w] {
				votes[lang]++
			}
		}
	}
	best, bestN := "", 0
	for _, lang := range []string{"pt", "es", "de", "fr", "it", "nl", "pl"} {
		if votes[lang] > bestN {
			best, bestN = lang, votes[lang]
		}
	}
	return best
}

var langWords = map[string]map[string]bool{
	"pt": set(`bora pras pros para vamos chama chamar chamem pv privado vendo vendendo compro prata cada mensagem manda mande preço preco quanto custa você voce vc vcs vocês obrigado obrigada alguém alguem preciso tenho onde ajuda estou não nao também tambem já pra aqui agora cara galera pq tbm tb blz beleza mano mto muito td tudo hj hoje obg vlw valeu tmj flw nois bora vamo quero sabe tava tá tô aí oi eae salve kd cadê agr dps depois então msm mesmo ngm ninguém gnt fazer quem quando ainda nada coisa mais bom noite masmorra upar ajudar precisa procurando mlk eu meu minha sim`),
	"es": set(`vendiendo plata mensaje precio cuanto cuánto cuesta hola que qué por para con los las una pero muy bien gracias alguien quiere quieres necesito tengo tienes vamos donde dónde como cómo esta está estoy soy eres amigo gente ayuda mazmorra busco buscando falta faltan quien bueno tambien también ahora aquí puedo puedes quiero hacer ayudar tmb porfa grax sí`),
	"de": set(`der das und ich nicht ist ein eine mit auf für fur bitte danke suche gruppe noch wer hast sind auch aber oder habe mir mich dich kann gibt jemand hier jetzt kommt gut schon mal doch ja`),
	"fr": set(`le la les des une est pas je tu il nous vous avec pour merci bonjour salut cherche groupe qui oui très tres besoin quelqu'un aide suis veux avoir faire allez ici maintenant aussi`),
	"it": set(`ciao grazie sono cerco gruppo andiamo aiuto`),
	"nl": set(`ik het een niet wat maar jij mijn ook zijn hebben goed dank bedankt hallo zoek iemand wil kan groep`),
	"pl": set(`jest nie tak dziekuje szukam grupy kto`),
}

// Portuguese words players type without accents, or short forms that are
// ambiguous in other languages, fixed only once we know the text is Portuguese.
var ptOnly = map[string]string{
	"to": "estou", "tou": "estou", "alguem": "alguém", "voce": "você", "nao": "não", "entao": "então",
	"tambem": "também", "ninguem": "ninguém", "ate": "até", "so": "só", "vo": "vou", "ce": "você",
	"mt": "muito", "pra": "para", "pro": "para o", "tava": "estava", "tamo": "estamos",
}

func expandPortuguese(text string) string {
	return wordRe.ReplaceAllStringFunc(text, func(w string) string {
		if full, ok := ptOnly[strings.ToLower(w)]; ok {
			if w[0] >= 'A' && w[0] <= 'Z' {
				return strings.ToUpper(full[:1]) + full[1:]
			}
			return full
		}
		return w
	})
}

var htmlBracketRe = regexp.MustCompile(`\[[^\[\]]{1,80}\]`)
var codeTagRe = regexp.MustCompile(`</?code>`)
var anyTagRe = regexp.MustCompile(`<[^>]{0,40}>`)

// toHTML protects [Item Links] and WoW jargon: bergamot copies <code>
// elements through untranslated.
func toHTML(s string) string {
	esc := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s) // keep ' and " as-is: the model reads them better
	esc = htmlBracketRe.ReplaceAllStringFunc(esc, func(m string) string { return "<code>" + m + "</code>" })
	esc = keepRe.ReplaceAllString(esc, "<code>$1</code>")
	esc = lockRe.ReplaceAllString(esc, "<code>$1</code>")
	return jargonRe.ReplaceAllStringFunc(esc, func(m string) string {
		if strings.HasPrefix(m, "code") { // never inside a tag we just added
			return m
		}
		return "<code>" + m + "</code>"
	})
}

// bergamot tends to move the space before an inline element to after it
// ("um  <code>tank</code>para"); put spaces back where the words meet.
var (
	spaceBeforeCodeRe = regexp.MustCompile(`[ \t]{2,}<code>`)
	glueBeforeCodeRe  = regexp.MustCompile(`([\p{L}\p{N}])<code>`)
	glueAfterCodeRe   = regexp.MustCompile(`</code>([\p{L}\p{N}])`)
)

// bergamot treats a word next to an inline element like the start of a
// sentence ("healer For <code>Deadmines</code>", "<code>raid</code> Tomorrow").
// Lower-case a Titlecase word touching a <code> span unless it really starts
// a sentence or the source wrote it capitalized.
var (
	capBeforeCodeRe = regexp.MustCompile(`(^|[^\p{L}\p{N}])(\p{Lu}\p{Ll}+)(\s*<code>)`)
	capAfterCodeRe  = regexp.MustCompile(`(</code>\s*)(\p{Lu}\p{Ll}+)`)
	sentenceEndRe   = regexp.MustCompile(`(^|[.!?:;]["')\]]?)\s*$`)
)

func decapNearCode(out, src string) string {
	fix := func(word, before string) string {
		if strings.Contains(src, word) || sentenceEndRe.MatchString(anyTagRe.ReplaceAllString(before, "")) {
			return word
		}
		r := []rune(word)
		return strings.ToLower(string(r[:1])) + string(r[1:])
	}
	out = replaceSubmatchIndex(capBeforeCodeRe, out, 2, fix)
	return replaceSubmatchIndex(capAfterCodeRe, out, 2, fix)
}

// replaceSubmatchIndex rewrites group g of every match with f(group, textBeforeGroup).
func replaceSubmatchIndex(re *regexp.Regexp, s string, g int, f func(word, before string) string) string {
	var b strings.Builder
	last := 0
	for _, m := range re.FindAllStringSubmatchIndex(s, -1) {
		ws, we := m[2*g], m[2*g+1]
		b.WriteString(s[last:ws])
		b.WriteString(f(s[ws:we], s[:ws]))
		last = we
	}
	b.WriteString(s[last:])
	return b.String()
}

// Some models echo a short exclamation ("Yes! I have 5" -> "Sim! Sim! Tenho 5").
// Drop a sentence that repeats the one right before it unless the source did too.
var sentenceRe = regexp.MustCompile(`[^.!?]+[.!?]+\s*|[^.!?]+$`)

func dropEchoes(out, src string) string {
	norm := func(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
	srcParts := sentenceRe.FindAllString(src, -1)
	for i := 1; i < len(srcParts); i++ {
		if norm(srcParts[i]) == norm(srcParts[i-1]) {
			return out // the player repeated themselves; keep it
		}
	}
	parts := sentenceRe.FindAllString(out, -1)
	if len(parts) < 2 {
		return out
	}
	var b strings.Builder
	for i, p := range parts {
		if i > 0 && norm(p) == norm(parts[i-1]) {
			continue
		}
		b.WriteString(p)
	}
	return strings.TrimSpace(b.String())
}

func fromHTML(s string) string {
	s = spaceBeforeCodeRe.ReplaceAllString(s, " <code>")
	s = strings.ReplaceAll(s, "</code><code>", "</code> <code>")
	s = glueBeforeCodeRe.ReplaceAllString(s, "$1 <code>")
	s = glueAfterCodeRe.ReplaceAllString(s, "</code> $1")
	s = codeTagRe.ReplaceAllString(s, "")
	s = anyTagRe.ReplaceAllString(s, "")
	return strings.TrimSpace(html.UnescapeString(s))
}

// ---------------------------------------------------------------------------
// Models: catalog, selection, download
// ---------------------------------------------------------------------------

func (o *Offline) progress(msg string) {
	if o.Progress != nil {
		o.Progress(msg)
	}
}

func (o *Offline) pairDir(src, dst string) string { return filepath.Join(o.Dir, src+"-"+dst) }

type pairInfo struct {
	Version, Architecture string
	Files                 []string
}

// ensurePair returns the folder of a ready-to-use model, downloading it on
// first use.
func (o *Offline) ensurePair(src, dst string) (string, error) {
	key := src + "-" + dst
	o.mu.Lock()
	l := o.pairLock[key]
	if l == nil {
		l = &sync.Mutex{}
		o.pairLock[key] = l
	}
	o.mu.Unlock()
	l.Lock()
	defer l.Unlock()

	dir := o.pairDir(src, dst)
	if pairReady(dir) {
		return dir, nil
	}
	cat, err := o.getCatalog(false)
	if err != nil {
		return "", err
	}
	recs := pickRecords(cat.Records, src, dst)
	if recs == nil {
		if cat2, err2 := o.getCatalog(true); err2 == nil { // maybe the language is new
			recs = pickRecords(cat2.Records, src, dst)
			cat = cat2
		}
	}
	if recs == nil {
		return "", unsupportedError{fmt.Sprintf("offline translation doesn't support %s → %s yet", langName(src), langName(dst))}
	}
	var total int64
	for _, r := range recs {
		total += r.Attachment.Size
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	label := fmt.Sprintf("%s → %s", langName(src), langName(dst))
	var done int64
	for _, r := range recs {
		if err := o.download(cat.Attach, r, dir, label, &done, total); err != nil {
			return "", err
		}
	}
	info := pairInfo{Version: recs[0].Version, Architecture: recs[0].Architecture}
	for _, r := range recs {
		info.Files = append(info.Files, r.Name)
	}
	b, _ := json.MarshalIndent(info, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "ready.json"), b, 0o644); err != nil {
		return "", err
	}
	o.progress(fmt.Sprintf("Offline translation ready: %s.", label))
	return dir, nil
}

func pairReady(dir string) bool {
	b, err := os.ReadFile(filepath.Join(dir, "ready.json"))
	if err != nil {
		return false
	}
	var info pairInfo
	if json.Unmarshal(b, &info) != nil || len(info.Files) == 0 {
		return false
	}
	for _, f := range info.Files {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			return false
		}
	}
	return true
}

func (o *Offline) getCatalog(refresh bool) (*catalogFile, error) {
	o.mu.Lock()
	cat := o.catalog
	o.mu.Unlock()
	path := filepath.Join(o.Dir, "catalog.json")
	if cat == nil {
		if b, err := os.ReadFile(path); err == nil {
			var c catalogFile
			if json.Unmarshal(b, &c) == nil && len(c.Records) > 0 {
				cat = &c
			}
		}
	}
	if cat != nil && !refresh && time.Since(cat.Fetched) < 7*24*time.Hour {
		o.mu.Lock()
		o.catalog = cat
		o.mu.Unlock()
		return cat, nil
	}
	fresh, err := o.fetchCatalog()
	if err != nil {
		if cat != nil { // offline: the old list is fine
			return cat, nil
		}
		return nil, fmt.Errorf("couldn't download the language list: %v", err)
	}
	os.MkdirAll(o.Dir, 0o755)
	if b, err := json.Marshal(fresh); err == nil {
		os.WriteFile(path, b, 0o644)
	}
	o.mu.Lock()
	o.catalog = fresh
	o.mu.Unlock()
	return fresh, nil
}

func (o *Offline) fetchCatalog() (*catalogFile, error) {
	get := func(url string, v any) error {
		resp, err := o.HTTP.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return fmt.Errorf("HTTP %d", resp.StatusCode)
		}
		return json.NewDecoder(resp.Body).Decode(v)
	}
	var root struct {
		Capabilities struct {
			Attachments struct {
				BaseURL string `json:"base_url"`
			} `json:"attachments"`
		} `json:"capabilities"`
	}
	attach := defaultAttachURL
	if get(o.Settings+"/", &root) == nil && root.Capabilities.Attachments.BaseURL != "" {
		attach = root.Capabilities.Attachments.BaseURL
	}
	var list struct {
		Data []modelRecord `json:"data"`
	}
	if err := get(o.Settings+modelCollection, &list); err != nil {
		return nil, err
	}
	if len(list.Data) == 0 {
		return nil, errors.New("empty model list")
	}
	return &catalogFile{Fetched: time.Now(), Attach: attach, Records: list.Data}, nil
}

// filterOK keeps records meant for desktop release builds.
func filterOK(expr string) bool {
	e := strings.ReplaceAll(expr, " ", "")
	switch {
	case e == "":
		return true
	case strings.Contains(e, "env.channel"):
		return false // nightly-only experiments
	case strings.Contains(e, "OS=='Android'"):
		return false
	case strings.Contains(e, "OS!='Android'"):
		return true
	}
	return false
}

// versionKey orders "3.1" > "3.0" > "3.0a1".
func versionKey(v string) [4]int {
	var k [4]int
	pre := 1 // release sorts after pre-release
	if i := strings.IndexAny(v, "ab"); i >= 0 {
		pre = 0
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	for i := 0; i < len(parts) && i < 3; i++ {
		k[i], _ = strconv.Atoi(parts[i])
	}
	k[3] = pre
	return k
}

var archRank = map[string]int{"base": 3, "base-memory": 2, "tiny": 1}

// pickRecords chooses the newest complete set of files for a language pair.
func pickRecords(all []modelRecord, src, dst string) []modelRecord {
	groups := map[string][]modelRecord{}
	for _, r := range all {
		if r.SourceLanguage != src || r.TargetLanguage != dst || !filterOK(r.FilterExpression) {
			continue
		}
		if r.FileType == "qualityModel" || r.Attachment.Location == "" {
			continue
		}
		k := r.Version + "|" + r.Architecture
		groups[k] = append(groups[k], r)
	}
	var keys []string
	for k, g := range groups {
		has := map[string]bool{}
		for _, r := range g {
			has[r.FileType] = true
		}
		if has["model"] && (has["vocab"] || (has["srcvocab"] && has["trgvocab"])) {
			keys = append(keys, k)
		}
	}
	if len(keys) == 0 {
		return nil
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := groups[keys[i]][0], groups[keys[j]][0]
		va, vb := versionKey(a.Version), versionKey(b.Version)
		if va != vb {
			for n := 0; n < 4; n++ {
				if va[n] != vb[n] {
					return va[n] > vb[n]
				}
			}
		}
		return archRank[a.Architecture] > archRank[b.Architecture]
	})
	g := groups[keys[0]]
	// one file per type (the catalog can list duplicates)
	seen := map[string]bool{}
	var out []modelRecord
	for _, r := range g {
		if !seen[r.FileType] {
			seen[r.FileType] = true
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].FileType < out[j].FileType })
	return out
}

func (o *Offline) download(attachBase string, r modelRecord, dir, label string, done *int64, total int64) error {
	name := filepath.Base(r.Name)
	if name == "." || name == "" || strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("bad file name in model list: %q", r.Name)
	}
	dest := filepath.Join(dir, name)
	if r.DecompressedHash != "" && fileSHA256(dest) == r.DecompressedHash {
		*done += r.Attachment.Size
		return nil
	}
	url := strings.TrimRight(attachBase, "/") + "/" + strings.TrimLeft(r.Attachment.Location, "/")
	resp, err := o.HTTP.Get(url)
	if err != nil {
		return fmt.Errorf("couldn't download the %s model: %v", label, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("couldn't download the %s model (HTTP %d)", label, resp.StatusCode)
	}
	counter := &progressReader{r: resp.Body, done: done, total: total, report: func(pct int) {
		o.progress(fmt.Sprintf("Downloading the %s offline model (%d MB)… %d%%", label, total>>20, pct))
	}}
	var src io.Reader = counter
	if strings.HasSuffix(r.Attachment.Location, ".zst") {
		src = zstd.NewReader(counter)
	}
	tmp := dest + ".part"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	h := sha256.New()
	_, err = io.Copy(io.MultiWriter(out, h), src)
	out.Close()
	if err != nil {
		os.Remove(tmp)
		return fmt.Errorf("the %s model download was interrupted: %v", label, err)
	}
	if r.DecompressedHash != "" && hex.EncodeToString(h.Sum(nil)) != r.DecompressedHash {
		os.Remove(tmp)
		return fmt.Errorf("the %s model download was damaged; it will retry next time", label)
	}
	return os.Rename(tmp, dest)
}

type progressReader struct {
	r       io.Reader
	done    *int64
	total   int64
	lastPct int
	report  func(int)
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	*p.done += int64(n)
	if p.total > 0 {
		pct := int(*p.done * 100 / p.total)
		if pct/10 != p.lastPct/10 || p.lastPct == 0 {
			p.lastPct = pct
			if pct > 0 {
				p.report(pct)
			}
		}
	}
	return n, err
}

func fileSHA256(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return ""
	}
	return hex.EncodeToString(h.Sum(nil))
}

// langName gives an English name for a Firefox language code.
func langName(ff string) string {
	for _, l := range targetLangs {
		if ffCode(l.Code) == ff {
			name := langLabel(l.Code)
			if i := strings.Index(name, " ("); i > 0 && ff != "zh-Hans" && ff != "zh-Hant" {
				name = name[:i]
			}
			return name
		}
	}
	return strings.ToUpper(ff)
}

// installedPairs lists downloaded model folders with their size, for the UI.
func (o *Offline) installedPairs() (pairs []string, bytes int64) {
	entries, _ := os.ReadDir(o.Dir)
	for _, e := range entries {
		if !e.IsDir() || !pairReady(filepath.Join(o.Dir, e.Name())) {
			continue
		}
		pairs = append(pairs, e.Name())
		files, _ := os.ReadDir(filepath.Join(o.Dir, e.Name()))
		for _, f := range files {
			if fi, err := f.Info(); err == nil {
				bytes += fi.Size()
			}
		}
	}
	return pairs, bytes
}
