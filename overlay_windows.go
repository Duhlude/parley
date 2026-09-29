package main

import (
	"strings"
	"syscall"
	"time"
	"unsafe"
)

// Colours
var (
	colBg      = rgb(18, 22, 28)
	colHeader  = rgb(24, 29, 37)
	colField   = rgb(32, 38, 48)
	colText    = rgb(228, 234, 240)
	colDim     = rgb(128, 139, 152)
	colAccent  = rgb(0x60, 0xCD, 0xFF)
	colLive    = rgb(90, 210, 140)
	colWarn    = rgb(255, 190, 90)
	colErr     = rgb(255, 110, 110)
	colSelect  = rgb(28, 40, 52)
	chanColors = map[string]uintptr{
		"W": rgb(255, 128, 255), "B": rgb(0, 255, 246), "S": rgb(240, 240, 240),
		"Y": rgb(255, 80, 80), "E": rgb(255, 128, 64), "P": rgb(170, 170, 255),
		"R": rgb(255, 127, 0), "G": rgb(64, 255, 64), "O": rgb(64, 192, 64),
		"C": rgb(255, 192, 192), "I": rgb(255, 127, 0),
	}
	chanNames = map[string]string{"W": "Whisper", "B": "Whisper", "S": "Say", "Y": "Yell",
		"E": "Emote", "P": "Party", "R": "Raid", "G": "Guild", "O": "Officer", "I": "Battleground"}
)

var (
	overlayEdit   uintptr
	editOldProc   uintptr
	editBrush     uintptr
	fBody, fSmall uintptr
	fBold, fTitle uintptr
	fIcon         uintptr
	curDPI        int
	scrollPx      int32
	maxScroll     int32
	hitEntries    []hitEntry
	btnSettings   RECT
	btnClear      RECT
	btnHide       RECT
	chipRect      RECT
	micRect       RECT
	toClearRect   RECT
	hoverBtn      int
	trackingMouse bool
)

type hitEntry struct {
	r RECT
	e *Entry
}

func sc(v int) int32 { return int32(v * curDPI / 96) }

func createOverlay() {
	cls := u16("ParleyOverlay")
	inst, _, _ := pGetModuleHandleW.Call(0)
	cur, _, _ := pLoadCursorW.Call(0, IDC_ARROW)
	wc := WNDCLASSEX{
		LpfnWndProc: syscall.NewCallback(overlayProc), HInstance: inst, HCursor: cur,
		LpszClassName: cls, HIcon: app.icon, HIconSm: app.iconSmall,
	}
	wc.CbSize = uint32(unsafe.Sizeof(wc))
	pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

	c := app.cfg
	h, _, _ := pCreateWindowExW.Call(WS_EX_TOPMOST|WS_EX_TOOLWINDOW|WS_EX_LAYERED,
		uintptr(unsafe.Pointer(cls)), u16p("Parley"), WS_POPUP|WS_CLIPCHILDREN,
		uintptr(c.X), uintptr(c.Y), uintptr(c.W), uintptr(c.H), 0, 0, inst, 0)
	app.overlay = h
	applyOpacity()
	pref := skin.Corner // DWM corner style (Windows 11)
	if pDwmSetWindowAttribute.Find() == nil {
		pDwmSetWindowAttribute.Call(h, 33, uintptr(unsafe.Pointer(&pref)), 4)
	}

	editBrush, _, _ = pCreateSolidBrush.Call(colField)
	overlayEdit, _, _ = pCreateWindowExW.Call(0, u16p("EDIT"), 0,
		WS_CHILD|WS_VISIBLE|WS_TABSTOP|ES_AUTOHSCROLL, 0, 0, 10, 10, h, 0, inst, 0)
	pSendMessageW.Call(overlayEdit, EM_LIMITTEXT, 400, 0)
	editOldProc, _, _ = pSetWindowLongPtrW.Call(overlayEdit, negIdx(GWLP_WNDPROC), syscall.NewCallback(editProc))

	updateDPI()
	showOverlay(true, false)
	pSetTimer.Call(h, 1, 1000, 0) // repaint for notice expiry / relative times
}

func applyOpacity() {
	a := app.cfg.Opacity * 255 / 100
	pSetLayeredWindowAttributes.Call(app.overlay, 0, uintptr(a), LWA_ALPHA)
}

func updateDPI() {
	curDPI = dpiOf(app.overlay)
	for _, f := range []uintptr{fBody, fSmall, fBold, fBoldAlt, fBodyAlt, fSmallAlt, fTitle, fIcon} {
		if f != 0 {
			pDeleteObject.Call(f)
		}
	}
	fs := app.cfg.FontSize
	body, small, title := pickFace(skin.BodyFaces), pickFace(skin.SmallFaces), pickFace(skin.TitleFaces)
	if nonLatinUI() { // Friz Quadrata has no Cyrillic/CJK; the brand title stays in the skin font
		body, small = "Segoe UI", "Segoe UI"
	}
	b := skin.BodyBoost
	fBody = newFont(int(sc(fs+b)), 400, body)
	fSmall = newFont(int(sc(fs-3+b)), 400, body)
	fBold = newFont(int(sc(fs-3+b)), skin.HeadWeight, small)
	// fallback for names in Cyrillic/CJK etc. that decorative fonts lack
	fBoldAlt = newFont(int(sc(fs-3+b)), 600, "Segoe UI")
	fBodyAlt = newFont(int(sc(fs)), 400, "Segoe UI")
	fSmallAlt = newFont(int(sc(fs-3)), 400, "Segoe UI")
	fTitle = newFont(int(sc(16)), skin.TitleWeight, title)
	fIcon = newFont(int(sc(13)), 400, "Segoe MDL2 Assets")
	pSendMessageW.Call(overlayEdit, WM_SETFONT, fBody, 1)
	layoutOverlay()
}

func layoutOverlay() {
	r := clientRect(app.overlay)
	barH := sc(40)
	pad := sc(10)
	chipW := chipWidth()
	x := pad + chipW + sc(8) + sc(8)
	editH := sc(app.cfg.FontSize + 8)
	y := r.Bottom - barH + (barH-editH)/2
	pMoveWindow.Call(overlayEdit, uintptr(x), uintptr(y), uintptr(r.Right-pad-sc(8)-micWidth()-x), uintptr(editH), 1)
}

func chipWidth() int32 { return sc(136) }

func micWidth() int32 { return sc(30) }

func showOverlay(v, focus bool) {
	app.visible = v
	if !v {
		pShowWindow.Call(app.overlay, SW_HIDE)
		return
	}
	pShowWindow.Call(app.overlay, SW_SHOWNOACTIVATE)
	pSetWindowPos.Call(app.overlay, ^uintptr(0) /*HWND_TOPMOST*/, 0, 0, 0, 0, SWP_NOMOVE|SWP_NOSIZE|SWP_NOACTIVATE)
	if focus {
		pSetForegroundWindow.Call(app.overlay)
		pSetFocus.Call(overlayEdit)
	}
}

func saveOverlayPos() {
	r := windowRect(app.overlay)
	app.mu.Lock()
	app.cfg.X, app.cfg.Y, app.cfg.W, app.cfg.H = int(r.Left), int(r.Top), int(r.Right-r.Left), int(r.Bottom-r.Top)
	cfg := app.cfg
	app.mu.Unlock()
	saveConfig(cfg)
}

func focusWow() {
	if w := app.wow.Load(); w != 0 {
		pSetForegroundWindow.Call(w)
	}
}

func editProc(h, msg, wp, lp uintptr) uintptr {
	switch msg {
	case WM_KEYDOWN:
		switch wp {
		case VK_RETURN:
			app.sendReply(getText(h))
			return 0
		case VK_ESCAPE:
			focusWow()
			return 0
		case VK_TAB:
			app.cycleLang()
			invalidate(app.overlay)
			return 0
		}
	case WM_CHAR:
		if wp == 13 || wp == 27 || wp == 9 {
			return 0
		}
	}
	r, _, _ := pCallWindowProcW.Call(editOldProc, h, msg, wp, lp)
	return r
}

func inRect(r RECT, x, y int32) bool {
	return x >= r.Left && x < r.Right && y >= r.Top && y < r.Bottom
}

func overlayProc(h, msg, wp, lp uintptr) uintptr {
	switch msg {
	case WM_PAINT:
		paintOverlay(h)
		return 0
	case WM_ERASEBKGND:
		return 1
	case WM_SIZE:
		layoutOverlay()
		invalidate(h)
		return 0
	case WM_DPICHANGED:
		nr := (*RECT)(unsafe.Pointer(lp))
		pSetWindowPos.Call(h, 0, uintptr(nr.Left), uintptr(nr.Top), uintptr(nr.Right-nr.Left),
			uintptr(nr.Bottom-nr.Top), SWP_NOZORDER|SWP_NOACTIVATE)
		updateDPI()
		invalidate(h)
		return 0
	case 0x0232: // WM_EXITSIZEMOVE
		saveOverlayPos()
		return 0
	case WM_GETMINMAXINFO:
		mm := (*MINMAXINFO)(unsafe.Pointer(lp))
		mm.PtMinTrackSize = POINT{sc(300), sc(170)}
		return 0
	case WM_NCHITTEST:
		var pt POINT
		pt.X, pt.Y = loword(lp), hiword(lp)
		pScreenToClient.Call(h, uintptr(unsafe.Pointer(&pt)))
		r := clientRect(h)
		b := sc(6)
		left, right := pt.X < b, pt.X >= r.Right-b
		top, bottom := pt.Y < b, pt.Y >= r.Bottom-b
		switch {
		case top && left:
			return HTTOPLEFT
		case top && right:
			return HTTOPRIGHT
		case bottom && left:
			return HTBOTTOMLEFT
		case bottom && right:
			return HTBOTTOMRIGHT
		case left:
			return HTLEFT
		case right:
			return HTRIGHT
		case top:
			return HTTOP
		case bottom:
			return HTBOTTOM
		}
		if pt.Y < curHeadH && !inRect(btnSettings, pt.X, pt.Y) && !inRect(btnClear, pt.X, pt.Y) &&
			!inRect(btnHide, pt.X, pt.Y) {
			return HTCAPTION
		}
		return HTCLIENT
	case WM_MOUSEACTIVATE:
		var pt POINT
		pGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
		er := windowRect(overlayEdit)
		if inRect(er, pt.X, pt.Y) {
			return MA_ACTIVATE
		}
		return MA_NOACTIVATE
	case WM_SETCURSOR:
		var pt POINT
		pGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
		pScreenToClient.Call(h, uintptr(unsafe.Pointer(&pt)))
		if loword(lp) == HTCLIENT && (inRect(btnSettings, pt.X, pt.Y) || inRect(btnClear, pt.X, pt.Y) ||
			inRect(btnHide, pt.X, pt.Y) || inRect(chipRect, pt.X, pt.Y) || inRect(micRect, pt.X, pt.Y) || inRect(toClearRect, pt.X, pt.Y) || entryAt(pt.X, pt.Y) != nil) {
			c, _, _ := pLoadCursorW.Call(0, IDC_HAND)
			pSetCursorProc(c)
			return 1
		}
	case WM_MOUSEMOVE:
		x, y := loword(lp), hiword(lp)
		hb := 0
		switch {
		case inRect(btnSettings, x, y):
			hb = 1
		case inRect(btnClear, x, y):
			hb = 2
		case inRect(btnHide, x, y):
			hb = 3
		case inRect(chipRect, x, y):
			hb = 4
		case inRect(micRect, x, y):
			hb = 5
		case inRect(toClearRect, x, y):
			hb = 6
		}
		if hb != hoverBtn {
			hoverBtn = hb
			invalidate(h)
		}
		if !trackingMouse {
			tme := TRACKMOUSEEVENT{DwFlags: 2 /*TME_LEAVE*/, HwndTrack: h}
			tme.CbSize = uint32(unsafe.Sizeof(tme))
			pTrackMouseEvent.Call(uintptr(unsafe.Pointer(&tme)))
			trackingMouse = true
		}
		return 0
	case WM_MOUSELEAVE:
		trackingMouse = false
		if hoverBtn != 0 {
			hoverBtn = 0
			invalidate(h)
		}
		return 0
	case WM_LBUTTONUP:
		x, y := loword(lp), hiword(lp)
		switch {
		case inRect(btnSettings, x, y):
			openSettings()
		case inRect(btnClear, x, y):
			runCommand(cmdClear)
		case inRect(btnHide, x, y):
			showOverlay(false, false)
			focusWow()
		case inRect(chipRect, x, y):
			langMenu()
			invalidate(h)
		case inRect(toClearRect, x, y):
			app.clearReply()
			invalidate(h)
		case inRect(micRect, x, y):
			startVoice()
		default:
			if e := entryAt(x, y); e != nil && !e.Own {
				app.pickReply(e)
				invalidate(h)
			}
		}
		return 0
	case WM_RBUTTONUP:
		if e := entryAt(loword(lp), hiword(lp)); e != nil {
			messageMenu(e)
			invalidate(h)
		}
		return 0
	case WM_MOUSEWHEEL:
		delta := hiword(wp)
		scrollPx += delta / 120 * sc(48)
		if scrollPx < 0 {
			scrollPx = 0
		}
		if scrollPx > maxScroll {
			scrollPx = maxScroll
		}
		invalidate(h)
		return 0
	case WM_CTLCOLOREDIT:
		pSetTextColor.Call(wp, colText)
		pSetBkColor.Call(wp, colField)
		return editBrush
	case WM_HOTKEY:
		if wp == voiceHotkeyID {
			startVoice()
			return 0
		}
		if fg, _, _ := pGetForegroundWindow.Call(); fg == h {
			focusWow()
		} else {
			setClickThrough(false)
			showOverlay(true, true)
		}
		return 0
	case WM_ACTIVATE:
		// Click-through overlays take the mouse only while they're in use.
		setClickThrough(loword(wp) == 0 && app.cfg.ClickThrough)
	case WM_TIMER:
		if wp == voiceTimerID {
			pKillTimer.Call(h, voiceTimerID)
			sendWinH()
			return 0
		}
		invalidate(h)
		return 0
	case WM_APP + 9: // sent by the installer to close Parley before updating
		runCommand(cmdQuit)
		return 0
	case msgVoice:
		finishVoice()
		return 0
	case msgRefresh:
		invalidate(h)
		return 0
	case msgReply:
		app.finishReply()
		invalidate(h)
		return 0
	case msgTray:
		switch loword(lp) {
		case WM_LBUTTONUP:
			showOverlay(!app.visible, false)
		case WM_RBUTTONUP:
			trayMenu()
		}
		return 0
	case WM_CLOSE:
		showOverlay(false, false)
		return 0
	case WM_DESTROY:
		pPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(h, msg, wp, lp)
	return r
}

var pSetCursor = user32.NewProc("SetCursor")

func pSetCursorProc(c uintptr) { pSetCursor.Call(c) }

func entryAt(x, y int32) *Entry {
	for _, he := range hitEntries {
		if inRect(he.r, x, y) {
			return he.e
		}
	}
	return nil
}

func entryTag(e *Entry) string {
	m := e.Msg
	if m.Type == "C" {
		if m.ChanNum != "" {
			return m.ChanNum + ". " + m.ChanName
		}
		return m.ChanName
	}
	if n, ok := chanNames[m.Type]; ok {
		return T(n)
	}
	return m.Type
}

func shortName(s string) string {
	if i := strings.IndexByte(s, '-'); i > 0 {
		return s[:i]
	}
	return s
}

func paintOverlay(h uintptr) {
	var ps PAINTSTRUCT
	hdc, _, _ := pBeginPaint.Call(h, uintptr(unsafe.Pointer(&ps)))
	defer pEndPaint.Call(h, uintptr(unsafe.Pointer(&ps)))
	r := clientRect(h)
	w, ht := r.Right, r.Bottom
	mem, _, _ := pCreateCompatibleDC.Call(hdc)
	bmp, _, _ := pCreateCompatibleBitmap.Call(hdc, uintptr(w), uintptr(ht))
	old, _, _ := pSelectObject.Call(mem, bmp)
	defer func() {
		pBitBlt.Call(hdc, 0, 0, uintptr(w), uintptr(ht), mem, 0, 0, SRCCOPY)
		pSelectObject.Call(mem, old)
		pDeleteObject.Call(bmp)
		pDeleteDC.Call(mem)
	}()
	pSetBkMode.Call(mem, TRANSPARENT)

	app.mu.Lock()
	status, kind := app.status, app.kind
	reply := app.reply
	notice := app.notice
	if time.Since(app.noticeAt) > 15*time.Second {
		notice = ""
	}
	entries := make([]*Entry, 0, len(app.entries))
	for _, e := range app.entries {
		if e.State != stateHidden {
			cp := *e // snapshot: translation goroutines update the originals
			entries = append(entries, &cp)
		}
	}
	showOrig := app.cfg.ShowOriginal
	hasKey := app.key != "" || app.engineModeLocked() == "offline"
	app.mu.Unlock()

	fillRect(mem, r, colBg)

	// ---- header ----
	headH := sc(skin.HeaderH)
	curHeadH = headH
	dot := colDim
	switch kind {
	case statusLive:
		dot = colLive
	case statusWarn:
		dot = colWarn
	}
	if !hasKey && kind != statusWarn {
		status, dot = T("No DeepL key · open Settings"), colWarn
	}
	drawHeaderBg(mem, w, headH)
	pad := sc(10)
	if wowChrome() {
		drawWowHeader(mem, w, headH, status, kind, dot)
	} else {
		left := headerLeft()
		btnW := sc(30)
		if skin.Chrome != "plain" {
			btnW = sc(26)
		}
		btnTop, btnBot := (headH-sc(26))/2, (headH+sc(26))/2
		if skin.Chrome == "plain" {
			btnTop, btnBot = sc(3), headH-sc(3)
		}
		btnHide = RECT{w - btnW - sc(6), btnTop, w - sc(6), btnBot}
		btnClear = RECT{btnHide.Left - btnW, btnTop, btnHide.Left, btnBot}
		btnSettings = RECT{btnClear.Left - btnW, btnTop, btnClear.Left, btnBot}

		if !skin.Portrait {
			drawIcon(mem, 1, headerLogoRect())
		}
		pSelectObject.Call(mem, fTitle)
		pSetTextColor.Call(mem, skin.Title)
		mr0 := RECT{0, 0, w, headH}
		drawText(mem, "Parley", &mr0, DT_SINGLELINE|DT_CALCRECT|DT_NOPREFIX)
		tw := mr0.Right
		titleY := RECT{0, 0, 0, headH}
		if skin.Chrome == "dragon" {
			titleY = RECT{0, sc(8), 0, sc(28)} // on the name plate
		}
		statusLeft := left + tw + sc(10)
		statusRight := btnSettings.Left - sc(6)
		if skin.TitleCenter {
			tx := (w - tw) / 2
			if tx < left {
				tx = left
			}
			tr := RECT{tx, titleY.Top, tx + tw, titleY.Bottom}
			drawText(mem, "Parley", &tr, DT_SINGLELINE|DT_VCENTER|DT_NOPREFIX)
			statusLeft = left
			statusRight = min32(tx-sc(8), btnSettings.Left-sc(6))
			if statusRight-statusLeft < sc(40) { // too narrow: status goes after the title
				statusLeft, statusRight = tx+tw+sc(8), btnSettings.Left-sc(6)
			}
		} else {
			tr := RECT{left, titleY.Top, left + tw, titleY.Bottom}
			drawText(mem, "Parley", &tr, DT_SINGLELINE|DT_VCENTER|DT_NOPREFIX)
		}

		d := sc(4)
		cy := (titleY.Top + titleY.Bottom) / 2
		if statusRight > statusLeft+sc(12) {
			roundRect(mem, RECT{statusLeft, cy - d, statusLeft + 2*d, cy + d}, 2*d, dot)
			pSelectObject.Call(mem, fSmall)
			pSetTextColor.Call(mem, colDim)
			sr := RECT{statusLeft + 2*d + sc(6), titleY.Top, statusRight, titleY.Bottom}
			drawText(mem, status, &sr, DT_SINGLELINE|DT_VCENTER|DT_NOPREFIX|DT_END_ELLIPSIS)
		}
		for i, b := range []struct {
			r     RECT
			glyph string
		}{{btnSettings, "\uE713"}, {btnClear, "\uE74D"}, {btnHide, "\uE921"}} {
			glyph := b.glyph
			isClose := i == 2 && skin.Chrome != "plain"
			drawHeaderButton(mem, b.r, glyph, hoverBtn == i+1, isClose)
		}
	}

	// ---- reply bar ----
	barH := sc(40)
	barTop := ht - barH
	fillRect(mem, RECT{0, barTop, w, ht}, colHeader)
	fieldR := RECT{pad + chipWidth() + sc(8), barTop + sc(6), w - pad, ht - sc(6)}
	drawField(mem, fieldR)
	micRect = RECT{fieldR.Right - micWidth() - sc(2), fieldR.Top + sc(2), fieldR.Right - sc(2), fieldR.Bottom - sc(2)}
	if listening {
		roundRect(mem, micRect, sc(6), colErr)
	} else if hoverBtn == 5 {
		roundRect(mem, micRect, sc(6), colSelect)
	}
	pSelectObject.Call(mem, fIcon)
	if listening {
		pSetTextColor.Call(mem, rgb(20, 10, 10))
	} else if hoverBtn == 5 {
		pSetTextColor.Call(mem, colAccent)
	} else {
		pSetTextColor.Call(mem, colDim)
	}
	mr := micRect
	drawText(mem, "\uE720", &mr, DT_SINGLELINE|DT_VCENTER|DT_CENTER)
	chipRect = RECT{pad, barTop + sc(6), pad + chipWidth(), ht - sc(6)}
	var chipText uintptr
	if wowChrome() {
		chipText = drawDropdown(mem, chipRect, hoverBtn == 4)
	} else {
		chipText = drawButton(mem, chipRect, hoverBtn == 4)
	}
	chip := T("Reply language")
	if reply.Lang != "" {
		chip = Tf("Reply in %s", reply.Lang)
	}
	if !wowChrome() {
		pSelectObject.Call(mem, fIcon)
		pSetTextColor.Call(mem, chipText)
		chev := RECT{chipRect.Right - sc(24), chipRect.Top, chipRect.Right - sc(6), chipRect.Bottom}
		drawText(mem, "\uE70D", &chev, DT_SINGLELINE|DT_VCENTER|DT_CENTER)
	}
	pSelectObject.Call(mem, pickFont(fBold, fBoldAlt, chip))
	pSetTextColor.Call(mem, chipText)
	cr := RECT{chipRect.Left + sc(8), chipRect.Top, chipRect.Right - sc(26), chipRect.Bottom}
	drawText(mem, chip, &cr, DT_SINGLELINE|DT_VCENTER|DT_NOPREFIX|DT_END_ELLIPSIS)

	// ---- who the reply goes to ----
	toH := sc(24)
	toTop := barTop - toH
	fillRect(mem, RECT{0, toTop, w, barTop}, colHeader)
	pSelectObject.Call(mem, fSmall)
	toClearRect = RECT{}
	tr2 := RECT{pad, toTop, w - pad - sc(24), barTop}
	if reply.set() {
		to := Tf("To %s", replyWhere(reply))
		pSelectObject.Call(mem, pickFont(fSmall, fSmallAlt, to))
		pSetTextColor.Call(mem, colText)
		drawText(mem, to, &tr2, DT_SINGLELINE|DT_VCENTER|DT_NOPREFIX|DT_END_ELLIPSIS)
		toClearRect = RECT{w - pad - sc(22), toTop + sc(2), w - pad, barTop - sc(2)}
		if hoverBtn == 6 {
			roundRect(mem, toClearRect, sc(6), colField)
		}
		pSelectObject.Call(mem, fIcon)
		pSetTextColor.Call(mem, colDim)
		xr := toClearRect
		drawText(mem, "\uE711", &xr, DT_SINGLELINE|DT_VCENTER|DT_CENTER)
	} else {
		pSetTextColor.Call(mem, colDim)
		hint := T("To whatever chat is open in WoW · click a message to reply to someone")
		pSelectObject.Call(mem, pickFont(fSmall, fSmallAlt, hint))
		drawText(mem, hint, &tr2, DT_SINGLELINE|DT_VCENTER|DT_NOPREFIX|DT_END_ELLIPSIS)
	}

	// ---- messages ----
	listTop := headH + sc(4)
	listBottom := toTop - sc(4)
	if notice != "" {
		pSelectObject.Call(mem, fSmall)
		nr := RECT{pad, 0, w - pad, 0}
		drawText(mem, notice, &nr, DT_CALCRECT|DT_WORDBREAK|DT_NOPREFIX)
		nh := nr.Bottom - nr.Top
		nr = RECT{pad, listBottom - nh - sc(4), w - pad, listBottom - sc(4)}
		fillRect(mem, RECT{0, nr.Top - sc(6), w, listBottom + sc(4)}, colBg)
		pSetTextColor.Call(mem, colAccent)
		drawText(mem, notice, &nr, DT_WORDBREAK|DT_NOPREFIX)
		listBottom = nr.Top - sc(10)
	}

	hitEntries = hitEntries[:0]
	textW := w - 2*pad - sc(6)
	measure := func(font uintptr, s string) int32 {
		pSelectObject.Call(mem, font)
		mr := RECT{0, 0, textW, 0}
		drawText(mem, s, &mr, DT_CALCRECT|DT_WORDBREAK|DT_NOPREFIX|DT_EDITCONTROL)
		return mr.Bottom
	}
	// clip drawing to the list area
	pIntersectClipRect.Call(mem, 0, uintptr(listTop), uintptr(w), uintptr(listBottom))
	y := listBottom + scrollPx
	total := int32(0)
	gap := sc(9)
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		head := ""
		headCol := chanColors[e.Msg.Type]
		if headCol == 0 {
			headCol = colText
		}
		if e.Own {
			head = Tf("You → %s", e.Detected)
			if e.Msg.Sender != "" {
				head += " · " + shortName(e.Msg.Sender)
			}
			headCol = colAccent
		} else {
			head = "[" + entryTag(e) + "] " + shortName(e.Msg.Sender)
			if e.Detected != "" {
				head += "  ·  " + e.Detected
			}
		}
		body, bodyCol := e.Translated, colText
		sub, subCol := "", colDim
		switch e.State {
		case statePending:
			body, bodyCol = e.Msg.Text+"  …", colDim
		case stateError:
			body, bodyCol = e.Msg.Text, colText
			sub, subCol = e.Err, colErr
		case stateDone:
			if showOrig || e.Own {
				sub = e.Msg.Text
			}
			if len(e.Gloss) > 0 {
				if sub != "" {
					sub += "\n"
				}
				sub += T("Terms:") + " " + strings.Join(e.Gloss, " · ")
			}
		}
		hf := fBold
		if !latinOnly(head) {
			hf = fBoldAlt
		}
		hh := measure(hf, head)
		bf, sf := pickFont(fBody, fBodyAlt, body), pickFont(fSmall, fSmallAlt, sub)
		bh := measure(bf, body)
		sh := int32(0)
		if sub != "" {
			sh = measure(sf, sub) + sc(2)
		}
		blockH := hh + sc(2) + bh + sh
		top := y - blockH
		total += blockH + gap
		if top < listBottom && y > listTop {
			left := pad + sc(6)
			sel := !e.Own && reply.set() && reply.Sender == e.Msg.Sender && reply.Type == e.Msg.Type && reply.ChanNum == e.Msg.ChanNum
			rowR := RECT{pad - sc(4), top - sc(3), w - pad + sc(4), y + sc(3)}
			if sel {
				drawSelection(mem, rowR)
				if wowChrome() {
					headCol = gold
				}
			} else if wowChrome() {
				drawRow(mem, rowR)
			}
			pSelectObject.Call(mem, hf)
			pSetTextColor.Call(mem, headCol)
			hr := RECT{left, top, left + textW, top + hh}
			drawText(mem, head, &hr, DT_WORDBREAK|DT_NOPREFIX|DT_EDITCONTROL)
			pSelectObject.Call(mem, bf)
			pSetTextColor.Call(mem, bodyCol)
			brc := RECT{left, top + hh + sc(2), left + textW, top + hh + sc(2) + bh}
			drawText(mem, body, &brc, DT_WORDBREAK|DT_NOPREFIX|DT_EDITCONTROL)
			if sub != "" {
				pSelectObject.Call(mem, sf)
				pSetTextColor.Call(mem, subCol)
				src := RECT{left, brc.Bottom + sc(2), left + textW, brc.Bottom + sc(2) + sh}
				drawText(mem, sub, &src, DT_WORDBREAK|DT_NOPREFIX|DT_EDITCONTROL)
			}
			hitEntries = append(hitEntries, hitEntry{RECT{0, max32(top, listTop), w, min32(y, listBottom)}, e})
		}
		y = top - gap
	}
	if len(entries) == 0 {
		help := T("Foreign-language chat shows up here, translated.") + "\n\n" +
			T("Click a message to reply to that person. Ctrl+Shift+T jumps to the reply box; Tab switches the reply language.") + "\n\n" +
			T("In WoW, /parley test sends sample messages (/parley test all for eight languages). Right-click a message to copy it or mute the sender.")
		pSelectObject.Call(mem, pickFont(fSmall, fSmallAlt, help))
		pSetTextColor.Call(mem, colDim)
		er := RECT{pad, listTop + sc(8), w - pad, listBottom}
		drawText(mem, help, &er, DT_WORDBREAK|DT_NOPREFIX)
	}
	maxScroll = total - (listBottom - listTop)
	if maxScroll < 0 {
		maxScroll = 0
	}
	pSelectClipRgn.Call(mem, 0)
	drawFrameBorder(mem, w, ht)
}

var (
	pIntersectClipRect = gdi32.NewProc("IntersectClipRect")
	pSelectClipRgn     = gdi32.NewProc("SelectClipRgn")
)

func negIdx(i int) uintptr { return uintptr(i) }

func max32(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}
func min32(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}

// ---------------------------------------------------------------------------
// Voice input (see speech_windows.go). Click the mic or press Ctrl+Shift+Y;
// click again to stop early.
// ---------------------------------------------------------------------------

const (
	voiceHotkeyID = 2
	msgVoice      = WM_APP + 4
)

var listening bool

func startVoice() {
	if listening {
		select {
		case voice.reqs <- false:
		default:
		}
		return
	}
	if app.cfg.VoiceMode == "typing" {
		showOverlay(true, true)
		app.setNotice(T("Speak your reply, then press Enter. (Windows voice typing)"))
		pSetTimer.Call(app.overlay, voiceTimerID, 150, 0)
		return
	}
	listening = true
	showOverlay(true, true)
	app.setNotice(T("Listening… speak your reply."))
	select {
	case voice.reqs <- true:
	default:
	}
	invalidate(app.overlay)
}

func (a *App) voiceDone(text string, err error) {
	a.mu.Lock()
	a.voiceText, a.voiceErr = text, err
	a.mu.Unlock()
	pPostMessageW.Call(a.overlay, msgVoice, 0, 0)
}

func finishVoice() {
	listening = false
	app.mu.Lock()
	text, err := app.voiceText, app.voiceErr
	app.mu.Unlock()
	switch {
	case err != nil && err.Error() == "stuck":
		setVoiceMode("typing")
		app.setNotice(T("Silent voice input didn't respond on this PC, so Parley switched to Windows voice typing. Click the mic again. (Tray icon → Voice input to switch back.)"))
	case err != nil:
		app.setNotice(localizeMsg(err.Error()))
	case strings.TrimSpace(text) == "":
		app.setNotice("")
	default:
		cur := strings.TrimSpace(getText(overlayEdit))
		if cur != "" {
			text = cur + " " + text
		}
		setText(overlayEdit, text)
		pSendMessageW.Call(overlayEdit, EM_SETSEL, 0xFFFF, 0xFFFF)
		pSetFocus.Call(overlayEdit)
		if app.cfg.VoiceManual {
			app.setNotice(T("Press Enter to translate and copy, or click the mic to add more."))
		} else {
			app.sendReply(text) // auto-send: translate, copy, back to WoW
		}
	}
	invalidate(app.overlay)
}

// replyWhere describes where a reply will be sent, e.g. "Snekitysnek (whisper)".
func replyWhere(r ReplyTarget) string {
	name := shortName(r.Sender)
	switch r.Type {
	case "W", "B":
		return Tf("%s (whisper)", name)
	case "C":
		ch := r.ChanName
		if r.ChanNum != "" {
			ch = r.ChanNum + ". " + ch
		}
		return Tf("%s channel, answering %s", ch, name)
	}
	if n, ok := chanNames[r.Type]; ok {
		return Tf("%s chat, answering %s", T(n), name)
	}
	return name
}

// langMenu lets you pick the reply language: automatic, recent ones, or
// any language from the "All languages" submenu.
func langMenu() {
	app.mu.Lock()
	cur, manual, recent := app.reply.Lang, app.langManual, append([]string{}, app.langs...)
	app.mu.Unlock()
	items := []MenuItem{
		{Label: T("Reply language"), Title: true},
		{Label: T("Match their language automatically"), ID: 1000, Checked: !manual, Radio: true},
	}
	if len(recent) > 0 {
		items = append(items, MenuItem{Sep: true}, MenuItem{Label: T("Recent"), Title: true})
		for i, c := range recent {
			items = append(items, MenuItem{Label: langLabel(c), Hint: c, ID: 1100 + i, Checked: manual && c == cur, Radio: true})
		}
	}
	var all []MenuItem
	for i, l := range targetLangs {
		all = append(all, MenuItem{Label: langLabel(l.Code), Hint: l.Code, ID: 1200 + i, Checked: manual && l.Code == cur, Radio: true})
	}
	items = append(items, MenuItem{Sep: true}, MenuItem{Label: T("All languages"), Sub: all})

	pt := POINT{chipRect.Left, chipRect.Top - sc(2)}
	pClientToScreen.Call(app.overlay, uintptr(unsafe.Pointer(&pt)))
	cmd := showMenu(items, pt.X, pt.Y, true)
	switch {
	case cmd == 1000:
		app.setLang("")
	case cmd >= 1100 && cmd < 1100+len(recent):
		app.setLang(recent[cmd-1100])
	case cmd >= 1200 && cmd < 1200+len(targetLangs):
		app.setLang(targetLangs[cmd-1200].Code)
	}
}

var fBoldAlt, fBodyAlt, fSmallAlt uintptr

// pickFont uses the skin font for Latin text and Segoe UI (which Windows
// links to Cyrillic, Chinese, Korean... fonts) for everything else.
func pickFont(skinFont, alt uintptr, s string) uintptr {
	if latinOnly(s) {
		return skinFont
	}
	return alt
}

// latinOnly reports whether s only uses Latin letters, which decorative
// WoW fonts cover; anything else is drawn with a fallback font.
func latinOnly(s string) bool {
	for _, r := range s {
		if r > 0x24F && r != '→' && r != '·' {
			return false
		}
	}
	return true
}

const voiceTimerID = 7

var (
	pSendInput = user32.NewProc("SendInput")
	pKillTimer = user32.NewProc("KillTimer")
)

type keyInput struct {
	Type  uint32
	_     uint32
	Vk    uint16
	Scan  uint16
	Flags uint32
	Time  uint32
	_     uint32
	Extra uintptr
	_     [8]byte
}

// sendWinH opens Windows voice typing in Parley's reply box (fallback mode).
func sendWinH() {
	if fg, _, _ := pGetForegroundWindow.Call(); fg != app.overlay {
		app.setNotice(T("Click the reply box, then the mic again."))
		return
	}
	const up, ext = 0x2, 0x1
	in := []keyInput{
		{Type: 1, Vk: 0x5B, Flags: ext},
		{Type: 1, Vk: 'H'},
		{Type: 1, Vk: 'H', Flags: up},
		{Type: 1, Vk: 0x5B, Flags: ext | up},
	}
	pSendInput.Call(uintptr(len(in)), uintptr(unsafe.Pointer(&in[0])), unsafe.Sizeof(in[0]))
}

func setVoiceMode(m string) {
	app.mu.Lock()
	app.cfg.VoiceMode = m
	cfg := app.cfg
	app.mu.Unlock()
	saveConfig(cfg)
}

var curHeadH int32 = 32
