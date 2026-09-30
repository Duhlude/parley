package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

const (
	msgRefresh = WM_APP + 1
	msgTray    = WM_APP + 2
	msgReply   = WM_APP + 3
	hotkeyID   = 1
)

type entryState int

const (
	statePending entryState = iota
	stateDone
	stateError
	stateHidden
)

type Entry struct {
	Msg        ChatMessage
	Own        bool
	State      entryState
	Translated string
	Gloss      []string // explanations of WoW terms in the message
	Detected   string
	Err        string
	At         time.Time
	Alert      bool   // mentions your character or one of your alert words
	Back       string // own replies: the translation translated back, to check it
}

type ReplyTarget struct {
	Type, ChanNum, ChanName, Sender string
	Lang                            string // DeepL target code
}

type statusKind int

const (
	statusIdle statusKind = iota
	statusLive
	statusWarn
)

type App struct {
	mu         sync.Mutex
	cfg        Config
	key        string
	entries    []*Entry
	status     string
	kind       statusKind
	reply      ReplyTarget
	langs      []string // recently seen reply languages, newest first
	lastLang   string   // language of the latest foreign message
	langManual bool     // reply language chosen by hand
	notice     string   // one-off message shown in the reply bar
	noticeAt   time.Time

	replyResult struct {
		text, clip string
		sent       string // what was in the reply box, to clear only that
		err        error
	}

	tr        *Translator
	az        *Azure
	azKey     string
	update    release   // newer version on GitHub, if any
	lastChime time.Time // alert sound throttle
	off       *Offline
	eng       *procEngine
	overlay   uintptr
	tray      NOTIFYICONDATA
	icon      uintptr
	iconSmall uintptr
	wow       atomic.Uintptr // written by the capture goroutine
	visible   bool

	voiceText string
	voiceErr  error
}

var app = &App{tr: NewTranslator(), az: NewAzure()}

func init() { runtime.LockOSThread() }

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--benchmark" {
		runBenchmark()
		return
	}
	// single instance
	updated := len(os.Args) > 1 && os.Args[1] == "--updated" // started by the installer after an update
	for try := 0; ; try++ {
		h, _, err := pCreateMutexW.Call(0, 0, u16p("Local\\ParleyTranslatorApp"))
		if errno, ok := err.(syscall.Errno); !ok || errno != 183 {
			break
		}
		// Another Parley is running. Right after an update it's the old one
		// still shutting down, so wait for it a little.
		if !updated || try >= 40 {
			return
		}
		pCloseHandle.Call(h)
		time.Sleep(250 * time.Millisecond)
	}
	if pSetProcessDpiAwarenessContext.Find() == nil {
		pSetProcessDpiAwarenessContext.Call(^uintptr(3)) // PER_MONITOR_AWARE_V2 (-4)
	}

	app.cfg = loadConfig()
	liveConfig = func() Config {
		app.mu.Lock()
		defer app.mu.Unlock()
		return app.cfg
	}
	setKeepWords(app.cfg.KeepWords)
	setUILang(app.cfg.UILang)
	app.key = string(unprotect(app.cfg.DeepLKeyEnc))
	app.azKey = string(unprotect(app.cfg.AzureKeyEnc))
	app.status = T("Looking for WoW…")
	app.icon, app.iconSmall = appIcons()
	loadExtraFonts()
	applySkin(app.cfg.Skin)
	setupOffline()

	createOverlay()
	app.touch() // auto-hide counts quiet time from start-up
	setClickThrough(app.cfg.ClickThrough)
	addTray()
	pRegisterHotKey.Call(app.overlay, hotkeyID, MOD_CONTROL|MOD_SHIFT|MOD_NOREPEAT, 'T')
	pRegisterHotKey.Call(app.overlay, voiceHotkeyID, MOD_CONTROL|MOD_SHIFT|MOD_NOREPEAT, 'Y')

	if updated {
		app.setNotice(Tf("Parley was updated to version %s.", appVersion))
		go cleanUpdateLeftovers()
	}
	go autoInstall()
	go captureLoop()
	go voiceLoop()
	go updateLoop()

	if app.engineMode() == "deepl" && app.key != "" {
		go func() { // warn before the allowance runs out
			if used, limit, err := app.tr.Usage(app.key); err == nil && limit > 0 && used*10 >= limit*9 {
				app.setNotice(Tf("DeepL: %s. Parley falls back to offline when it runs out.", usageText(used, limit)))
			}
		}()
	}
	if app.engineMode() == "offline" {
		if pairs, _ := app.off.installedPairs(); len(pairs) == 0 {
			app.setNotice(T("Offline translation is on. Each language downloads once (about 50 MB) the first time someone uses it."))
		}
	}

	var m MSG
	for {
		r, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		if settingsHwnd != 0 {
			if r, _, _ := pIsDialogMessageW.Call(settingsHwnd, uintptr(unsafe.Pointer(&m))); r != 0 {
				continue
			}
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
	removeTray()
	if app.eng != nil {
		app.eng.Close()
	}
}

// setupOffline prepares the on-device translator (parley-mt.exe).
func setupOffline() {
	var eng mtEngine
	if p := engineExe(); p != "" {
		app.eng = &procEngine{Path: p}
		eng = app.eng
	}
	app.off = NewOffline(offlineModelsDir(), eng)
	app.off.Progress = func(m string) { app.setNotice(localizeMsg(m)) }
	app.prefetchMyLanguage()
}

// engineMode is "offline", "deepl" or "azure". Unset means the online
// service whose key is saved (DeepL first), otherwise offline.
func (a *App) engineMode() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.engineModeLocked()
}

func (a *App) engineModeLocked() string {
	switch a.cfg.Engine {
	case "offline", "deepl", "azure":
		return a.cfg.Engine
	}
	if a.key != "" {
		return "deepl"
	}
	if a.azKey != "" {
		return "azure"
	}
	return "offline"
}

// translate picks the engine. Online failures that offline can cover
// (no internet, allowance used up, bad key) fall back to offline.
func (a *App) translate(key, text, source, target string) (Translation, error) {
	tr, err := a.translateRaw(key, text, source, target)
	tr.Text = stripKeep(tr.Text)
	return tr, err
}

func (a *App) translateRaw(key, text, source, target string) (Translation, error) {
	a.mu.Lock()
	mode, azKey, azRegion := a.engineModeLocked(), a.azKey, a.cfg.AzureRegion
	a.mu.Unlock()
	var tr Translation
	var err error
	switch mode {
	case "offline":
		return a.off.Translate(text, source, target)
	case "azure":
		tr, err = a.az.Translate(azKey, azRegion, text, source, target)
	default:
		tr, err = a.tr.Translate(key, text, source, target)
	}
	if err == nil || a.off.Engine == nil {
		return tr, err
	}
	msg := err.Error()
	if strings.Contains(msg, "unreachable") || strings.Contains(msg, "allowance") ||
		strings.Contains(msg, "rejected") || strings.Contains(msg, "add your") {
		if tr2, err2 := a.off.Translate(text, source, target); err2 == nil {
			return tr2, nil
		}
	}
	return tr, err
}

func (a *App) refresh() { pPostMessageW.Call(a.overlay, msgRefresh, 0, 0) }

func (a *App) setStatus(s string, k statusKind) {
	a.mu.Lock()
	changed := a.status != s || a.kind != k
	a.status, a.kind = s, k
	a.mu.Unlock()
	if changed {
		a.refresh()
	}
}

func (a *App) setNotice(s string) {
	a.mu.Lock()
	a.notice, a.noticeAt = s, time.Now()
	a.mu.Unlock()
	a.refresh()
	a.wake()
}

// ---------------------------------------------------------------------------
// Addon install
// ---------------------------------------------------------------------------

func autoInstall() {
	app.mu.Lock()
	cfg := app.cfg
	app.mu.Unlock()
	if _, err := os.Stat(cfg.WowPath); err != nil {
		found := ""
		for _, w := range wowFolders() {
			if st, err := os.Stat(w); err == nil && st.IsDir() {
				found = w
				break
			}
		}
		if found == "" {
			app.setNotice(T("WoW folder not found. Set it in Settings, then press Install addon."))
			return
		}
		app.mu.Lock()
		app.cfg.WowPath = found
		cfg = app.cfg
		app.mu.Unlock()
		saveConfig(cfg)
	}
	outdated := false
	for _, w := range wowInstalls(cfg.WowPath) {
		if addonInstalledVersion(w) != embeddedAddonVersion() {
			outdated = true
		}
	}
	if !outdated {
		return
	}
	if _, err := installAddonAll(cfg.WowPath); err != nil {
		app.setNotice(Tf("Couldn't install the addon automatically (%s). Copy the addon\\Parley folder into Interface\\AddOns yourself.", localizeMsg(err.Error())))
		return
	}
	app.setNotice(T("Parley addon installed. Type /reload in WoW if it's already running."))
}

// ---------------------------------------------------------------------------
// Finding WoW and reading the strip
// ---------------------------------------------------------------------------

var wowExes = map[string]bool{"wowclassic.exe": true, "wow.exe": true, "wowclassict.exe": true,
	"wowclassicb.exe": true, "wowt.exe": true, "wowb.exe": true}

// wowPIDs lists the running WoW processes by executable name. It reads the
// system's process list (a Toolhelp snapshot), so Parley never opens a
// handle to the game process itself.
func wowPIDs() map[uint32]bool {
	out := map[uint32]bool{}
	snap, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return out
	}
	defer syscall.CloseHandle(snap)
	var pe syscall.ProcessEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))
	for err = syscall.Process32First(snap, &pe); err == nil; err = syscall.Process32Next(snap, &pe) {
		if wowExes[strings.ToLower(syscall.UTF16ToString(pe.ExeFile[:]))] {
			out[pe.ProcessID] = true
		}
	}
	return out
}

var (
	enumBest     uintptr
	enumBestArea int32
	enumPIDs     map[uint32]bool
	enumCB       = syscall.NewCallback(func(h, _ uintptr) uintptr {
		if v, _, _ := pIsWindowVisible.Call(h); v == 0 {
			return 1
		}
		var pid uint32
		pGetWindowThreadProcessId.Call(h, uintptr(unsafe.Pointer(&pid)))
		if !enumPIDs[pid] {
			return 1
		}
		r := clientRect(h)
		if area := (r.Right - r.Left) * (r.Bottom - r.Top); area > enumBestArea {
			enumBest, enumBestArea = h, area
		}
		return 1
	})
)

func findWow() uintptr {
	enumBest, enumBestArea = 0, 0
	if enumPIDs = wowPIDs(); len(enumPIDs) == 0 {
		return 0
	}
	pEnumWindows.Call(enumCB, 0)
	return enumBest
}

type bgraPixels struct {
	buf  []byte
	w, h int
}

func (p *bgraPixels) At(x, y int) (uint8, uint8, uint8) {
	i := (y*p.w + x) * 4
	return p.buf[i+2], p.buf[i+1], p.buf[i]
}
func (p *bgraPixels) Size() (int, int) { return p.w, p.h }

func captureLoop() {
	runtime.LockOSThread()
	const W, H = stripCols * 4, stripRows * 4
	screen, _, _ := pGetDC.Call(0)
	mem, _, _ := pCreateCompatibleDC.Call(screen)
	bi := BITMAPINFOHEADER{BiSize: 40, BiWidth: W, BiHeight: -H, BiPlanes: 1, BiBitCount: 32}
	var bits uintptr
	dib, _, _ := pCreateDIBSection.Call(screen, uintptr(unsafe.Pointer(&bi)), 0,
		uintptr(unsafe.Pointer(&bits)), 0, 0)
	pSelectObject.Call(mem, dib)
	pReleaseDC.Call(0, screen)
	px := &bgraPixels{buf: unsafe.Slice((*byte)(unsafe.Pointer(bits)), W*H*4), w: W, h: H}

	var (
		lastFind  time.Time
		cs        = app.cfg.CellSize
		lastSeq   = -1
		seenAny   bool
		badFrames int
		lastGood  time.Time
		lastMsg   ChatMessage
	)
	nap := 35 * time.Millisecond
	for {
		time.Sleep(nap)
		nap = 35 * time.Millisecond // about 28 looks a second while WoW is up
		wow := app.wow.Load()
		if wow == 0 || !isWin(wow) {
			app.wow.Store(0)
			nap = 500 * time.Millisecond // no WoW: just look for it now and then
			if time.Since(lastFind) < 2*time.Second {
				continue
			}
			lastFind = time.Now()
			if wow = findWow(); wow == 0 {
				app.setStatus(T("Waiting for WoW"), statusIdle)
				continue
			}
			app.wow.Store(wow)
			nap = 35 * time.Millisecond
		}
		if ic, _, _ := pIsIconic.Call(wow); ic != 0 {
			app.setStatus(T("WoW minimized"), statusIdle)
			nap = 250 * time.Millisecond
			continue
		}
		var o POINT
		pClientToScreen.Call(wow, uintptr(unsafe.Pointer(&o)))
		screen, _, _ := pGetDC.Call(0)
		pBitBlt.Call(mem, 0, 0, W, H, screen, uintptr(o.X), uintptr(o.Y), SRCCOPY)
		pReleaseDC.Call(0, screen)

		if cs == 0 || !hasMarker(px, cs) {
			cs = FindCellSize(px, app.cfg.CellSize)
		}
		if cs == 0 {
			switch {
			case !seenAny:
				app.setStatus(T("WoW found · type /parley test in game"), statusIdle)
			case time.Since(lastGood) > 3*time.Second:
				app.setStatus(T("Live"), statusLive)
			}
			continue
		}
		m, err := Decode(px, cs)
		if err != nil {
			badFrames++
			if err == errCalibration && badFrames > 30 {
				app.setStatus(T("Colours distorted: reset WoW gamma/brightness, turn off HDR"), statusWarn)
			}
			continue
		}
		badFrames = 0
		paused := time.Since(lastGood) > 2*time.Second
		seenAny, lastGood = true, time.Now()
		app.setStatus(T("Live"), statusLive)
		// The addon keeps showing a message for a while, so the same frame is
		// read many times. After a pause (a /reload in WoW restarts the count)
		// a repeated number only counts as new if the message itself differs.
		if m.Seq == lastSeq && (!paused || sameChat(m, lastMsg)) {
			continue
		}
		lastSeq, lastMsg = m.Seq, m
		app.incoming(m)
	}
}

func isWin(h uintptr) bool { r, _, _ := pIsWindow.Call(h); return r != 0 }

// ---------------------------------------------------------------------------
// Translation pipeline
// ---------------------------------------------------------------------------

func (a *App) incoming(m ChatMessage) {
	if m.Type == "X" { // the addon saw chat it isn't allowed to read (Retail)
		a.setNotice(T("Blizzard hides chat from addons during Mythic+ keys, PvP matches and boss fights. Parley picks it back up afterwards."))
		return
	}
	if m.Type == "N" { // the addon tells us which character you're playing
		a.mu.Lock()
		changed := a.cfg.CharName != m.Sender && m.Sender != ""
		a.cfg.CharName = m.Sender
		cfg := a.cfg
		a.mu.Unlock()
		if changed {
			saveConfig(cfg)
		}
		return
	}
	a.mu.Lock()
	myLang := a.cfg.MyLang
	key := a.key
	terms := alertTerms(a.cfg.AlertWords, a.cfg.CharName)
	a.mu.Unlock()

	if a.isMuted(m.Sender) {
		return
	}
	english, hint := Guess(m.Text)
	myBase := baseLang(myLang)
	if english {
		if myBase == "EN" {
			return
		}
		hint = "EN"
	}
	e := &Entry{Msg: m, State: statePending, At: time.Now()}
	a.mu.Lock()
	a.entries = append(a.entries, e)
	if len(a.entries) > 300 {
		a.entries = a.entries[len(a.entries)-300:]
	}
	a.mu.Unlock()
	a.refresh()

	go func() {
		translateSlots <- struct{}{} // a few at a time, roughly in arrival order
		defer func() { <-translateSlots }()
		if !a.listed(e) { // scrolled out of the list while waiting: don't spend characters on it
			return
		}
		src := ExpandSlang(m.Text)
		if myBase != "EN" { // spell out English chat shortcuts for non-English readers
			src = expandShortcuts(src)
		}
		src = prepareIncoming(src, myLang)
		tr, err := a.translate(key, src, hint, myLang)
		if err != nil && strings.Contains(err.Error(), "rate limit") {
			time.Sleep(1200 * time.Millisecond)
			tr, err = a.translate(key, src, hint, myLang)
		}
		a.mu.Lock()
		switch {
		case err != nil:
			e.State, e.Err = stateError, localizeMsg(err.Error())
		case baseLang(tr.Detected) == myBase, sameText(tr.Text, m.Text),
			isASCII(m.Text) && !plausibleForASCII(tr.Detected):
			e.State = stateHidden
		default:
			e.State, e.Translated, e.Detected = stateDone, tr.Text, tr.Detected
			if a.explainTerms() {
				e.Gloss = glossFor(m.Text, myLang, 4)
			}
			a.noteLanguage(e)
			e.Alert = mentions(terms, m.Text, tr.Text)
		}
		done := e.State == stateDone
		alert := done && e.Alert
		a.mu.Unlock()
		a.refresh()
		if alert {
			a.chime()
		}
		if done {
			a.wake()
		}
		if done {
			a.logHistory(e)
		}
	}()
}

// translateSlots limits how many incoming messages translate at once, so a
// busy Trade chat queues up instead of piling onto the engines.
var translateSlots = make(chan struct{}, 3)

// listed reports whether e is still in the message list.
func (a *App) listed(e *Entry) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i := len(a.entries) - 1; i >= 0; i-- {
		if a.entries[i] == e {
			return true
		}
	}
	return false
}

// Reply model: the person/channel you reply to is only ever chosen by
// clicking a message. The reply language follows that person's language
// (or the latest foreign message when nobody is picked) unless you choose a
// language yourself from the language button.

// noteLanguage records a detected language (caller holds mu).
func (a *App) noteLanguage(e *Entry) {
	code := replyTarget(e.Detected)
	if code == "" {
		return
	}
	out := []string{code}
	for _, l := range a.langs {
		if l != code && len(out) < 6 {
			out = append(out, l)
		}
	}
	a.langs = out
	a.lastLang = code
	if a.langManual {
		return
	}
	if !a.reply.set() {
		a.reply.Lang = code
	} else if a.reply.Sender == e.Msg.Sender && a.reply.Type == e.Msg.Type {
		a.reply.Lang = code
	}
}

func (r ReplyTarget) set() bool { return r.Type != "" }

// pickReply makes a clicked message the reply target; clicking the
// selected message again clears it.
func (a *App) pickReply(e *Entry) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.reply.set() && a.reply.Sender == e.Msg.Sender && a.reply.Type == e.Msg.Type &&
		a.reply.ChanNum == e.Msg.ChanNum {
		a.clearReplyLocked()
		return
	}
	lang := a.reply.Lang
	if l := replyTarget(e.Detected); l != "" {
		lang = l
	}
	a.reply = ReplyTarget{Type: e.Msg.Type, ChanNum: e.Msg.ChanNum, ChanName: e.Msg.ChanName,
		Sender: e.Msg.Sender, Lang: lang}
	a.langManual = false
}

func (a *App) clearReply() {
	a.mu.Lock()
	a.clearReplyLocked()
	a.mu.Unlock()
}

func (a *App) clearReplyLocked() {
	lang := a.reply.Lang
	if !a.langManual && a.lastLang != "" {
		lang = a.lastLang
	}
	a.reply = ReplyTarget{Lang: lang}
}

// setLang picks the reply language by hand ("" = match automatically).
func (a *App) setLang(code string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if code == "" {
		a.langManual = false
		if !a.reply.set() && a.lastLang != "" {
			a.reply.Lang = a.lastLang
		}
		return
	}
	a.reply.Lang = code
	a.langManual = true
}

// cycleLang (Tab) steps through recent and common languages.
func (a *App) cycleLang() {
	a.mu.Lock()
	opts := append([]string{}, a.langs...)
	cur := a.reply.Lang
	a.mu.Unlock()
	for _, l := range []string{"ES", "PT-BR", "PT-PT", "RU", "DE", "FR", "ZH-HANS"} {
		found := false
		for _, o := range opts {
			if o == l {
				found = true
			}
		}
		if !found {
			opts = append(opts, l)
		}
	}
	next := opts[0]
	for i, o := range opts {
		if o == cur {
			next = opts[(i+1)%len(opts)]
		}
	}
	a.setLang(next)
}

func chatPrefix(r ReplyTarget) string {
	switch r.Type {
	case "W":
		if r.Sender != "" {
			return "/w " + r.Sender + " "
		}
		return "/r "
	case "B":
		return "/r "
	case "S", "E":
		return "/s "
	case "Y":
		return "/y "
	case "P":
		return "/p "
	case "R":
		return "/ra "
	case "G":
		return "/g "
	case "O":
		return "/o "
	case "I":
		return "/bg "
	case "C":
		if r.ChanNum != "" {
			return "/" + r.ChanNum + " "
		}
	}
	return ""
}

// sendReply translates text for the current reply target (called on GUI thread).
func (a *App) sendReply(text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	a.mu.Lock()
	r := a.reply
	key, myLang := a.key, a.cfg.MyLang
	a.mu.Unlock()
	if r.Lang == "" {
		a.setNotice(T("Pick a reply language first: click a message, or the language button."))
		return
	}
	a.setNotice(T("Translating…"))
	go func() {
		src := text
		if baseLang(myLang) == "EN" { // "sry w8" -> "sorry wait" so DeepL gets it
			src = expandShortcuts(src)
			src = localizeRoleVerbs(src, r.Lang)
		}
		src = localizePlaces(lockUserWords(src), r.Lang)
		tr, err := a.translate(key, src, baseLang(myLang), r.Lang)
		if own := a.deliverReply(r, text, tr.Text, err); own != nil {
			a.backCheck(own, key, r.Lang, myLang)
		}
	}()
}

// backCheck translates a finished reply back into your language, so you can
// see what it will say before you paste it. Runs on the reply's goroutine.
func (a *App) backCheck(own *Entry, key, from, to string) {
	a.mu.Lock()
	on := a.cfg.BackCheck != "off"
	text := own.Translated
	a.mu.Unlock()
	if !on || text == "" {
		return
	}
	tr, err := a.translate(key, prepareIncoming(text, to), baseLang(from), to)
	if err != nil || strings.TrimSpace(tr.Text) == "" {
		return
	}
	a.mu.Lock()
	own.Back = strings.TrimSpace(tr.Text)
	a.mu.Unlock()
	a.refresh()
}

// deliverReply stores a finished reply (or its error) and lets the GUI
// thread copy it (finishReply). Safe from any goroutine.
func (a *App) deliverReply(r ReplyTarget, original, translated string, err error) *Entry {
	a.mu.Lock()
	a.replyResult.err = err
	a.replyResult.sent = original
	var own *Entry
	if err == nil {
		a.replyResult.text = translated
		a.replyResult.clip = chatPrefix(r) + translated
		own = &Entry{Own: true, State: stateDone, At: time.Now(), Translated: translated, Detected: r.Lang,
			Msg: ChatMessage{Type: r.Type, ChanNum: r.ChanNum, ChanName: r.ChanName, Sender: r.Sender, Text: original}}
		a.entries = append(a.entries, own)
	}
	a.mu.Unlock()
	if own != nil {
		a.logHistory(own)
	}
	pPostMessageW.Call(a.overlay, msgReply, 0, 0)
	return own
}

// sendPhrase sends a quick reply: the built-in translation when the
// phrasebook has the reply language, otherwise through the engine.
func (a *App) sendPhrase(p phrase) {
	a.mu.Lock()
	r, key, myLang := a.reply, a.key, a.cfg.MyLang
	a.mu.Unlock()
	if r.Lang == "" {
		a.setNotice(T("Pick a reply language first: click a message, or the language button."))
		return
	}
	if t := p.ready(r.Lang); t != "" {
		a.deliverReply(r, p.EN, t, nil)
		return
	}
	a.setNotice(T("Translating…"))
	go func() {
		tr, err := a.translate(key, p.EN, "EN", r.Lang)
		if own := a.deliverReply(r, p.EN, tr.Text, err); own != nil {
			a.backCheck(own, key, r.Lang, myLang)
		}
	}()
}

// finishReply runs on the GUI thread once the reply translation is back.
func (a *App) finishReply() {
	a.mu.Lock()
	res := a.replyResult
	wow := a.wow.Load()
	a.mu.Unlock()
	if res.err != nil {
		a.setNotice(Tf("Reply not translated: %s", localizeMsg(res.err.Error())))
		return
	}
	if strings.TrimSpace(getText(overlayEdit)) == strings.TrimSpace(res.sent) {
		setText(overlayEdit, "") // keep anything typed since, or a draft under a quick reply
	}
	if !setClipboard(a.overlay, res.clip) {
		a.setNotice(T("Couldn't reach the clipboard, try again."))
		return
	}
	msg := T("Copied. In WoW press Enter, Ctrl+V, Enter.")
	if len(res.clip) > 255 {
		msg = Tf("Copied, but it's %d bytes and WoW allows 255, so shorten it.", len(res.clip))
	}
	a.setNotice(msg)
	if wow != 0 {
		pSetForegroundWindow.Call(wow)
	}
}

// ---------------------------------------------------------------------------
// Tray icon
// ---------------------------------------------------------------------------

func addTray() {
	t := &app.tray
	t.CbSize = uint32(unsafe.Sizeof(*t))
	t.HWnd = app.overlay
	t.UID = 1
	t.UFlags = NIF_MESSAGE | NIF_ICON | NIF_TIP
	t.UCallbackMessage = msgTray
	t.HIcon = app.iconSmall
	copy(t.SzTip[:], syscall.StringToUTF16(T("Parley: WoW chat translator")))
	pShellNotifyIconW.Call(NIM_ADD, uintptr(unsafe.Pointer(t)))
}

func removeTray() { pShellNotifyIconW.Call(NIM_DELETE, uintptr(unsafe.Pointer(&app.tray))) }

const (
	cmdToggle = 100 + iota
	cmdSettings
	cmdInstall
	cmdClear
	cmdQuit
	cmdSkin0         = 300
	cmdVoiceSilent   = 400
	cmdVoiceTyping   = 401
	cmdVoiceLog      = 402
	cmdVoiceAuto     = 403
	cmdModelAccurate = 404
	cmdModelFast     = 405
	cmdExplain       = 406
	cmdEngOffline    = 500
	cmdEngDeepL      = 501
	cmdModelsFolder  = 502
	cmdHistoryOpen   = 503
	cmdHistoryToggle = 504
	cmdUnmuteAll     = 505
	cmdClickThrough  = 506
	cmdEngAzure      = 507
	cmdAlertSound    = 508
	cmdBackCheck     = 511
	cmdUpdateCheck   = 509
	cmdUpdateGet     = 510
	cmdAutoHide0     = 520 // + index in autoHideChoices
	cmdLang0         = 600 // + index in targetLangs: download ahead
	cmdMsgCopy       = 800
	cmdMsgCopyOrig   = 801
	cmdMsgMute       = 802
	cmdMsgReply      = 803
	cmdMsgThread     = 804
	cmdMsgAll        = 805
)

func trayMenu() {
	label := T("Show overlay")
	if app.visible && !autoHide.hidden {
		label = T("Hide overlay")
	}
	var skinItems []MenuItem
	for i, sk := range skins {
		skinItems = append(skinItems, MenuItem{Label: T(sk.Label), ID: cmdSkin0 + i, Checked: sk.ID == skin.ID, Radio: true})
	}
	silent := app.cfg.VoiceMode != "typing"
	voice := []MenuItem{
		{Label: T("Offline voice (silent, on this PC)"), ID: cmdVoiceSilent, Checked: silent, Radio: true,
			Tip: T("Your speech is recognised on this PC. No popup, and no audio is sent anywhere.")},
		{Label: T("Windows voice typing (Win+H popup)"), ID: cmdVoiceTyping, Checked: !silent, Radio: true,
			Tip: T("Uses Windows' own voice typing instead. It shows a small popup and needs Windows speech settings turned on.")},
		{Sep: true},
		{Label: T("Send automatically after speaking"), ID: cmdVoiceAuto, Checked: !app.cfg.VoiceManual,
			Tip: T("Translate and copy your reply as soon as you stop talking. Turn off to check the text first.")},
		{Sep: true},
		{Label: T("Speech model: Accurate (466 MB)"), ID: cmdModelAccurate, Checked: app.cfg.VoiceModel != "fast", Radio: true,
			Tip: T("Better recognition, bigger one-time download, a little slower.")},
		{Label: T("Speech model: Fast (148 MB)"), ID: cmdModelFast, Checked: app.cfg.VoiceModel == "fast", Radio: true,
			Tip: T("Smaller and quicker, a bit less accurate.")},
		{Sep: true},
		{Label: T("Open voice log"), ID: cmdVoiceLog, Tip: T("Troubleshooting details for voice input.")},
	}
	history := []MenuItem{
		{Label: T("Save translated chat to a daily file"), ID: cmdHistoryToggle, Checked: app.historyOn(),
			Tip: T("Keeps one text file per day with translated chat and your replies, so you can look things up later.")},
		{Label: T("Open chat history folder"), ID: cmdHistoryOpen},
	}
	items := []MenuItem{
		{Label: "Parley", Title: true},
		{Label: label, Hint: "Ctrl+Shift+T", ID: cmdToggle},
		{Label: T("Settings…"), ID: cmdSettings},
		{Sep: true},
		{Label: T("Translation"), Sub: translationMenu()},
		{Label: T("Voice input"), Sub: voice},
		{Label: T("Skin"), Sub: skinItems},
		{Label: T("Chat history"), Sub: history},
		{Label: T("Auto-hide overlay"), Sub: autoHideMenu()},
		{Sep: true},
		{Label: T("Click-through overlay"), ID: cmdClickThrough, Checked: app.cfg.ClickThrough,
			Tip: T("Mouse clicks go through the overlay to WoW, so you can't hit it by accident. Ctrl+Shift+T to use it, Esc to go back.")},
		{Label: T("Check replies by translating them back"), ID: cmdBackCheck, Checked: app.cfg.BackCheck != "off",
			Tip: T("After Parley translates your reply, it translates it back into your language and shows that under the reply, so you can see what the other player will read. With DeepL or Azure this uses a few extra characters.")},
		{Label: T("Alert sound"), ID: cmdAlertSound, Checked: app.cfg.AlertSound != "off",
			Tip: T("Play a soft sound when a message mentions your character or one of your alert words (Settings).")},
		{Label: T("Check for updates"), ID: cmdUpdateCheck, Checked: app.cfg.UpdateCheck != "off",
			Tip: T("Check GitHub for a new version of Parley when it opens, and every few hours while it runs.")},
		{Label: T("Explain gaming terms"), ID: cmdExplain, Checked: app.explainTerms(),
			Tip: T("Adds a \"Terms\" line under messages that explains WoW jargon (LFG, dungeon names, class terms) in your language.")},
		{Label: T("Install / update WoW addon"), ID: cmdInstall, Tip: T("Copies the latest Parley addon into your WoW folders. Type /reload in game afterwards.")},
		{Label: T("Clear messages"), ID: cmdClear, Tip: T("Empties the overlay. Chat history files are kept.")},
	}
	if n := len(app.cfg.Muted); n > 0 {
		items = append(items, MenuItem{Label: Tf("Unmute everyone (%d muted)", n), ID: cmdUnmuteAll, Tip: T("Show messages from everyone you muted again.")})
	}
	app.mu.Lock()
	upd := app.update
	app.mu.Unlock()
	if upd.Version != "" {
		up := MenuItem{Label: Tf("Download Parley %s", upd.Version), ID: cmdUpdateGet, Tip: T("Opens the download page for the new version.")}
		if canSelfUpdate(upd) {
			up = MenuItem{Label: Tf("Update to Parley %s", upd.Version), ID: cmdUpdateGet,
				Tip: T("Downloads and installs the new version, then restarts Parley. Your settings are kept.")}
		}
		if updating.Load() {
			up = MenuItem{Label: Tf("Updating to Parley %s…", upd.Version), Disabled: true}
		}
		items = append([]MenuItem{items[0], up, {Sep: true}}, items[1:]...)
	}
	items = append(items, MenuItem{Sep: true}, MenuItem{Label: T("Quit Parley"), ID: cmdQuit})
	pSetForegroundWindow.Call(app.overlay)
	runCommand(showMenuAtCursor(items))
}

func runCommand(cmd int) {
	if cmd >= cmdSkin0 && cmd < cmdSkin0+len(skins) {
		setSkin(skins[cmd-cmdSkin0].ID)
		return
	}
	if cmd >= cmdAutoHide0 && cmd < cmdAutoHide0+len(autoHideChoices) {
		setAutoHide(autoHideChoices[cmd-cmdAutoHide0])
		return
	}
	if cmd >= cmdLang0 && cmd < cmdLang0+len(targetLangs) {
		app.prefetch(ffCode(targetLangs[cmd-cmdLang0].Code))
		return
	}
	switch cmd {
	case cmdEngOffline, cmdEngDeepL, cmdEngAzure:
		app.mu.Lock()
		app.cfg.Engine = map[int]string{cmdEngOffline: "offline", cmdEngDeepL: "deepl", cmdEngAzure: "azure"}[cmd]
		noKey, noAz := app.key == "", app.azKey == ""
		cfg := app.cfg
		app.mu.Unlock()
		saveConfig(cfg)
		if cmd == cmdEngDeepL && noKey {
			app.setNotice(T("DeepL needs an API key (Settings). Until then Parley translates offline."))
		}
		if cmd == cmdEngAzure && noAz {
			app.setNotice(T("Azure needs a key (Settings). Until then Parley translates offline."))
		}
		app.refresh()
	case cmdAlertSound, cmdUpdateCheck, cmdBackCheck:
		app.mu.Lock()
		p := &app.cfg.AlertSound
		if cmd == cmdUpdateCheck {
			p = &app.cfg.UpdateCheck
		} else if cmd == cmdBackCheck {
			p = &app.cfg.BackCheck
		}
		if *p == "off" {
			*p = ""
		} else {
			*p = "off"
		}
		cfg := app.cfg
		app.mu.Unlock()
		saveConfig(cfg)
	case cmdUpdateGet:
		app.mu.Lock()
		r := app.update
		app.mu.Unlock()
		if canSelfUpdate(r) {
			startUpdate(r)
		} else if r.URL != "" {
			openURL(r.URL)
		}
	case cmdModelsFolder:
		os.MkdirAll(app.off.Dir, 0o755)
		openURL(app.off.Dir)
	case cmdHistoryOpen:
		dir := filepath.Join(configDir(), "history")
		os.MkdirAll(dir, 0o755)
		openURL(dir)
	case cmdHistoryToggle:
		app.mu.Lock()
		if app.historyOn() {
			app.cfg.History = "off"
		} else {
			app.cfg.History = "on"
		}
		cfg := app.cfg
		app.mu.Unlock()
		saveConfig(cfg)
	case cmdClickThrough:
		toggleClickThrough()
	case cmdUnmuteAll:
		app.mu.Lock()
		app.cfg.Muted = nil
		cfg := app.cfg
		app.mu.Unlock()
		saveConfig(cfg)
		app.setNotice(T("Everyone is unmuted."))
	case cmdVoiceSilent:
		setVoiceMode("silent")
	case cmdVoiceTyping:
		setVoiceMode("typing")
	case cmdVoiceAuto:
		app.mu.Lock()
		app.cfg.VoiceManual = !app.cfg.VoiceManual
		cfg := app.cfg
		app.mu.Unlock()
		saveConfig(cfg)
	case cmdModelAccurate, cmdModelFast:
		app.mu.Lock()
		app.cfg.VoiceModel = map[int]string{cmdModelAccurate: "accurate", cmdModelFast: "fast"}[cmd]
		cfg := app.cfg
		app.mu.Unlock()
		saveConfig(cfg)
		app.setNotice(T("Speech model changed. The next mic click downloads it if needed."))
	case cmdExplain:
		app.mu.Lock()
		if app.explainTerms() {
			app.cfg.ExplainTerms = "off"
		} else {
			app.cfg.ExplainTerms = "on"
		}
		cfg := app.cfg
		app.mu.Unlock()
		saveConfig(cfg)
	case cmdVoiceLog:
		openURL(filepath.Join(configDir(), "voice.log"))
	case cmdToggle:
		showOverlay(!app.visible, false)
	case cmdSettings:
		openSettings()
	case cmdInstall:
		app.mu.Lock()
		wowPath := app.cfg.WowPath
		app.mu.Unlock()
		dests, err := installAddonAll(wowPath)
		if err != nil {
			skinAlert(app.overlay, T("Couldn't install the addon"), localizeMsg(err.Error()))
		} else {
			skinAlert(app.overlay, T("Addon installed"), alertLines(append(dests, "", T("Type /reload in WoW if it's running."))...))
		}
	case cmdClear:
		app.mu.Lock()
		app.entries = nil
		app.mu.Unlock()
		threadWith = ""
		app.refresh()
	case cmdQuit:
		saveOverlayPos()
		pDestroyWindow.Call(app.overlay)
	}
}

// makeIcon draws a small speech-bubble icon in the app accent colour.
func makeIcon() uintptr {
	const S = 32
	bi := BITMAPINFOHEADER{BiSize: 40, BiWidth: S, BiHeight: -S, BiPlanes: 1, BiBitCount: 32}
	var bits uintptr
	hdc, _, _ := pGetDC.Call(0)
	color, _, _ := pCreateDIBSection.Call(hdc, uintptr(unsafe.Pointer(&bi)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	pReleaseDC.Call(0, hdc)
	px := unsafe.Slice((*uint32)(unsafe.Pointer(bits)), S*S)
	inBubble := func(x, y float64) bool {
		// rounded rectangle 3..29 x 4..22 plus a tail at bottom-left
		if x >= 3 && x <= 29 && y >= 4 && y <= 22 {
			cx, cy := x, y
			if cx < 8 {
				cx = 8
			} else if cx > 24 {
				cx = 24
			}
			if cy < 9 {
				cy = 9
			} else if cy > 17 {
				cy = 17
			}
			dx, dy := x-cx, y-cy
			return dx*dx+dy*dy <= 25
		}
		// tail
		return y > 21 && y < 29 && x > 7 && x < 15 && (x-7) < (29-y)*0.9
	}
	for y := 0; y < S; y++ {
		for x := 0; x < S; x++ {
			hits := 0
			for sy := 0; sy < 4; sy++ {
				for sx := 0; sx < 4; sx++ {
					if inBubble(float64(x)+float64(sx)/4+0.125, float64(y)+float64(sy)/4+0.125) {
						hits++
					}
				}
			}
			a := uint32(hits * 255 / 16)
			// accent #60CDFF, premultiplied
			r, g, b := uint32(0x60)*a/255, uint32(0xCD)*a/255, uint32(0xFF)*a/255
			// two dark "text" dots inside the bubble
			for _, dx := range []int{11, 16, 21} {
				if (x-dx)*(x-dx)+(y-13)*(y-13) <= 3 && hits == 16 {
					r, g, b = 0x14, 0x1a, 0x22
				}
			}
			px[y*S+x] = a<<24 | r<<16 | g<<8 | b
		}
	}
	mask, _, _ := pCreateBitmap.Call(S, S, 1, 1, 0)
	ii := ICONINFO{FIcon: 1, HbmMask: mask, HbmColor: color}
	icon, _, _ := pCreateIconIndirect.Call(uintptr(unsafe.Pointer(&ii)))
	return icon
}

// wowFolders lists likely game folders (Classic Era first, then Retail):
// Battle.net's registry entry first, then common install locations on each drive.
func wowFolders() []string {
	var out []string
	for _, key := range []string{`HKLM\SOFTWARE\WOW6432Node\Blizzard Entertainment\World of Warcraft`,
		`HKLM\SOFTWARE\Blizzard Entertainment\World of Warcraft`} {
		cmd := exec.Command("reg", "query", key, "/v", "InstallPath")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
		b, err := cmd.Output()
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(b), "\n") {
			if i := strings.Index(line, "REG_SZ"); i >= 0 {
				p := strings.TrimRight(strings.TrimSpace(line[i+6:]), `\`)
				for _, fl := range wowFlavors {
					out = append(out, filepath.Join(filepath.Dir(p), fl))
				}
			}
		}
	}
	for _, drive := range []string{"C", "D", "E", "F"} {
		for _, rel := range []string{`Program Files (x86)\World of Warcraft`, `Program Files\World of Warcraft`,
			`World of Warcraft`, `Games\World of Warcraft`, `Games\WoW\World of Warcraft`, `Battle.net\World of Warcraft`} {
			for _, fl := range wowFlavors {
				out = append(out, drive+`:\`+rel+`\`+fl)
			}
		}
	}
	return out
}

// explainTerms: show WoW-term explanations under translations. "auto"
// (default) = on when you translate into a language other than English.
func (a *App) explainTerms() bool {
	switch a.cfg.ExplainTerms {
	case "on":
		return true
	case "off":
		return false
	}
	return baseLang(a.cfg.MyLang) != "EN"
}
