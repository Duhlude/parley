package main

// Skinned tooltips for menu items, settings controls and overlay buttons.
// One tooltip window is shared; whoever shows it "owns" it, so the overlay's
// hover poller can't hide a menu's tooltip and vice versa.

import (
	"syscall"
	"time"
	"unsafe"
)

const (
	tipTimerID     = 9 // overlay and settings hover polling
	menuTipTimerID = 8
	tipDelay       = 550 * time.Millisecond
)

var (
	tipHwnd      uintptr
	tipText      string
	tipOwner     string
	tipFont      uintptr
	tipClassDone bool

	pWindowFromPoint = user32.NewProc("WindowFromPoint")
	pGetDlgCtrlID    = user32.NewProc("GetDlgCtrlID")
)

func tipPad() int32 { return sc(9) }

func ensureTipWindow() {
	if tipHwnd != 0 {
		return
	}
	inst, _, _ := pGetModuleHandleW.Call(0)
	if !tipClassDone {
		wc := WNDCLASSEX{LpfnWndProc: syscall.NewCallback(tipProc), HInstance: inst, LpszClassName: u16("ParleyTip")}
		wc.CbSize = uint32(unsafe.Sizeof(wc))
		pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
		tipClassDone = true
	}
	const exNoActivate, exTransparent = 0x08000000, 0x00000020
	tipHwnd, _, _ = pCreateWindowExW.Call(WS_EX_TOPMOST|WS_EX_TOOLWINDOW|exNoActivate|exTransparent,
		uintptr(unsafe.Pointer(u16("ParleyTip"))), 0, WS_POPUP, 0, 0, 10, 10, 0, 0, inst, 0)
}

// tipSize measures text wrapped at the tooltip's maximum width.
func tipSize(text string) (int32, int32) {
	if tipFont == 0 {
		tipFont = newFont(int(sc(13)), 400, "Segoe UI")
	}
	dc, _, _ := pGetDC.Call(0)
	old, _, _ := pSelectObject.Call(dc, tipFont)
	r := RECT{0, 0, sc(320), 0}
	drawText(dc, text, &r, DT_CALCRECT|DT_WORDBREAK|DT_NOPREFIX)
	pSelectObject.Call(dc, old)
	pReleaseDC.Call(0, dc)
	return r.Right + 2*tipPad(), r.Bottom + 2*tipPad()
}

// showTipAt shows text with its top-left at (x, y), kept on screen.
func showTipAt(owner, text string, x, y int32) {
	if text == "" {
		return
	}
	ensureTipWindow()
	tipText, tipOwner = text, owner
	w, h := tipSize(text)
	wa := workAreaAt(x, y)
	if x+w > wa.Right {
		x = wa.Right - w
	}
	if x < wa.Left {
		x = wa.Left
	}
	if y+h > wa.Bottom {
		y = wa.Bottom - h
	}
	const swpNoActivate, swpShow = 0x0010, 0x0040
	pSetWindowPos.Call(tipHwnd, ^uintptr(0) /*HWND_TOPMOST*/, uintptr(x), uintptr(y), uintptr(w), uintptr(h), swpNoActivate|swpShow)
	invalidate(tipHwnd)
}

// showTipBelow shows text under a control (screen rect), or above it when
// there's no room below.
func showTipBelow(owner, text string, r RECT) {
	if text == "" {
		return
	}
	_, h := tipSize(text)
	y := r.Bottom + sc(6)
	if wa := workAreaAt(r.Left, r.Bottom); y+h > wa.Bottom {
		y = r.Top - h - sc(6)
	}
	showTipAt(owner, text, r.Left, y)
}

func hideTip(owner string) {
	if tipHwnd != 0 && (owner == "" || owner == tipOwner) {
		pShowWindow.Call(tipHwnd, 0)
		tipOwner = ""
	}
}

func tipProc(h, msg, wp, lp uintptr) uintptr {
	switch msg {
	case WM_ERASEBKGND:
		return 1
	case WM_PAINT:
		var ps PAINTSTRUCT
		hdc, _, _ := pBeginPaint.Call(h, uintptr(unsafe.Pointer(&ps)))
		cr := clientRect(h)
		drawMenuBackground(hdc, cr.Right, cr.Bottom)
		drawMenuFrame(hdc, cr.Right, cr.Bottom)
		pSetBkMode.Call(hdc, TRANSPARENT)
		pSetTextColor.Call(hdc, colText)
		pSelectObject.Call(hdc, tipFont)
		r := RECT{tipPad(), tipPad(), cr.Right - tipPad(), cr.Bottom - tipPad()}
		drawText(hdc, tipText, &r, DT_WORDBREAK|DT_NOPREFIX)
		pEndPaint.Call(h, uintptr(unsafe.Pointer(&ps)))
		return 0
	case 0x0084: // WM_NCHITTEST: let the mouse through
		return ^uintptr(0) // HTTRANSPARENT
	}
	r, _, _ := pDefWindowProcW.Call(h, msg, wp, lp)
	return r
}

// hoverTip shows a tooltip once the pointer has rested on the same thing.
type hoverTip struct {
	owner string
	key   string
	since time.Time
	shown bool
}

// poll is called every ~100 ms with what's under the pointer ("" = nothing).
func (t *hoverTip) poll(key, text string, anchor RECT) {
	if key != t.key {
		t.key, t.since = key, time.Now()
		if t.shown {
			hideTip(t.owner)
			t.shown = false
		}
		return
	}
	if key == "" || text == "" || t.shown || time.Since(t.since) < tipDelay {
		return
	}
	showTipBelow(t.owner, text, anchor)
	t.shown = true
}

func (t *hoverTip) reset() {
	if t.shown {
		hideTip(t.owner)
	}
	t.key, t.shown = "", false
}

func cursorPos() POINT {
	var pt POINT
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	return pt
}

func windowAt(pt POINT) uintptr {
	w, _, _ := pWindowFromPoint.Call(uintptr(uint32(pt.X)) | uintptr(uint32(pt.Y))<<32)
	return w
}

// ---- overlay buttons ----

var overlayTip = &hoverTip{owner: "overlay"}

func pollOverlayTip() {
	pt := cursorPos()
	if windowAt(pt) != app.overlay || len(menuStack) > 0 {
		overlayTip.poll("", "", RECT{})
		return
	}
	c := pt
	pScreenToClient.Call(app.overlay, uintptr(unsafe.Pointer(&c)))
	for _, b := range []struct {
		key  string
		r    RECT
		text string
	}{
		{"settings", btnSettings, T("Settings: translation engine, languages, skin, alerts and more.")},
		{"clear", btnClear, T("Clear all messages from the overlay. Chat history files are kept.")},
		{"hide", btnHide, T("Hide the overlay. Ctrl+Shift+T or the tray icon brings it back.")},
		{"chip", chipRect, T("Reply language. It follows the person you reply to; click to pick one yourself. Tab cycles recent languages.")},
		{"quick", qrRect, T("Quick replies: common phrases, already translated. One click copies it, ready to paste in WoW.")},
		{"mic", micRect, T("Voice reply: speak in your language and Parley types, translates and copies it (Ctrl+Shift+Y).")},
		{"toclear", toClearRect, T("Stop replying to this person.")},
	} {
		if b.r.Right > b.r.Left && inRect(b.r, c.X, c.Y) {
			a := b.r
			p1, p2 := POINT{a.Left, a.Top}, POINT{a.Right, a.Bottom}
			pClientToScreen.Call(app.overlay, uintptr(unsafe.Pointer(&p1)))
			pClientToScreen.Call(app.overlay, uintptr(unsafe.Pointer(&p2)))
			overlayTip.poll(b.key, b.text, RECT{p1.X, p1.Y, p2.X, p2.Y})
			return
		}
	}
	overlayTip.poll("", "", RECT{})
}

// ---- settings controls ----

var settingsTip = &hoverTip{owner: "settings"}

func settingsTipText(id int) string {
	switch id {
	case idEngine:
		return T("Offline: free and private, translates on your PC; good, a bit rougher with slang.\nDeepL: the best quality, needs a DeepL API key.\nAzure: very good, 2 million free characters a month with a Microsoft Azure key.\nIf an online service fails, Parley translates offline.")
	case idKey:
		if setKeyShows == 2 {
			return T("Your Azure Translator key, from the \"Keys and Endpoint\" page of your Translator resource. Stored encrypted on this PC.")
		}
		return T("Your DeepL API key, from your DeepL account page. Only needed for the DeepL engine. Stored encrypted on this PC.")
	case idRegion:
		return T("The region of your Azure Translator resource, e.g. eastus or westeurope. Use global if your resource is global.")
	case idGetKey:
		return T("Opens the page that explains how to get a key.")
	case idLang:
		return T("The language chat is translated into. You type your replies in this language too.")
	case idSkin:
		return T("How the overlay, its menus and this window look.")
	case idUILang:
		return T("The language of Parley's own menus and windows. Automatic follows Windows.")
	case idWow:
		return T("Your _classic_era_ or _retail_ folder. Parley installs the addon into every WoW version it finds next to it.")
	case idInstall:
		return T("Copies the Parley addon into WoW. Type /reload in game afterwards.")
	case idFont:
		return T("Size of the chat text in the overlay.")
	case idOpacity:
		return T("How see-through the overlay is. 100 is solid.")
	case idKeepWords:
		return T("Names and words Parley leaves exactly as written, in messages and in your replies: your guild's name, nicknames, inside jokes. Separate them with commas.")
	case idAlert:
		return T("Messages that mention these words, or your character, get an orange bar and a soft sound. Separate words with commas.")
	case idOrig:
		return T("Shows the untranslated message in small text under each translation.")
	}
	return ""
}

func pollSettingsTip() {
	if settingsHwnd == 0 {
		return
	}
	pt := cursorPos()
	w := windowAt(pt)
	if w == 0 || len(menuStack) > 0 {
		settingsTip.poll("", "", RECT{})
		return
	}
	for id, c := range setCtl {
		if c == w {
			settingsTip.poll(string(rune('A'+id)), settingsTipText(id), windowRect(c))
			return
		}
	}
	settingsTip.poll("", "", RECT{})
}
