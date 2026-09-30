package main

import (
	"os"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

const (
	idKey = 101 + iota
	idGetKey
	idLang
	idWow
	idInstall
	idFont
	idOpacity
	idOrig
	idSave
	idCancel
	idSkin
	idEngine
	idUILang
	idRegion
	idAlert
	idKeepWords
)

const msgUsage = WM_APP + 20

// The settings window draws its own title bar and frame in the skin.
var (
	setTitleFont uintptr
	setCloseHot  bool
	setTracking  bool
)

func setHeadH() int32 { return sc(40) }

func setCloseRect(w int32) RECT {
	s := sc(28)
	top := (setHeadH() - s) / 2
	return RECT{w - s - sc(8), top, w - sc(8), top + s}
}

// Settings' dropdowns and checkbox are drawn by Parley (skinned) and open
// skinned menus, instead of Windows' white combo boxes.
var (
	setEngineSel, setLangSel, setSkinSel int
	setUISel                             int // 0 = automatic, else uiLanguages[i-1]
	setOrigOn                            bool
	engineLabels                         = []string{"Offline: free, private, runs on this PC", "DeepL: best quality, needs an API key", "Azure: 2 million free characters a month, needs a key"}
	// The key row shows the DeepL or the Azure key depending on the engine;
	// both are kept while the window is open.
	setKeyDeepL, setKeyAzure, setRegion string
	setKeyShows                         int // engine whose key the row shows (1 DeepL, 2 Azure)
	keyLabel, keyField, regionField     = -1, -1, -1
	setKeyRight                         int32
)

func isDropdown(id int) bool { return id == idEngine || id == idLang || id == idSkin || id == idUILang }

// uiLangChoices: "Automatic (Windows: …)" then every interface language.
func uiLangChoices() []string {
	auto := systemUILang()
	autoName := auto
	for _, l := range uiLanguages {
		if l.Code == auto {
			autoName = l.Name
		}
	}
	out := []string{Tf("Automatic (Windows: %s)", autoName)}
	for _, l := range uiLanguages {
		out = append(out, l.Name)
	}
	return out
}

func dropdownText(id int) string {
	switch id {
	case idEngine:
		return T(engineLabels[setEngineSel])
	case idLang:
		return langLabel(targetLangs[setLangSel].Code)
	case idSkin:
		return T(skins[setSkinSel].Label)
	case idUILang:
		return uiLangChoices()[setUISel]
	}
	return ""
}

// openDropdown shows the choices for a settings dropdown as a skinned menu.
func openDropdown(id int) {
	var items []MenuItem
	cur := 0
	switch id {
	case idEngine:
		cur = setEngineSel
		tips := []string{
			T("Free and private: translates on your PC. Each language downloads once (about 50 MB)."),
			T("The best quality. Needs a DeepL API key in Settings. Falls back to offline if DeepL fails."),
			T("Very good quality, 2 million free characters a month. Needs an Azure key in Settings. Falls back to offline if Azure fails."),
		}
		for i, l := range engineLabels {
			items = append(items, MenuItem{Label: T(l), ID: i + 1, Tip: tips[i]})
		}
	case idLang:
		cur = setLangSel
		for i, l := range targetLangs {
			items = append(items, MenuItem{Label: langLabel(l.Code), Hint: l.Code, ID: i + 1})
		}
	case idSkin:
		cur = setSkinSel
		for i, s := range skins {
			items = append(items, MenuItem{Label: T(s.Label), ID: i + 1})
		}
	case idUILang:
		cur = setUISel
		for i, l := range uiLangChoices() {
			items = append(items, MenuItem{Label: l, ID: i + 1})
		}
	}
	items[cur].Checked, items[cur].Radio = true, true
	c := setCtl[id]
	r := windowRect(c)
	menuMinWidth = r.Right - r.Left
	menuStartHover = cur
	pick := showMenu(items, r.Left, r.Bottom+sc(2), false)
	if settingsHwnd != 0 {
		pSetForegroundWindow.Call(settingsHwnd)
		pSetFocus.Call(c)
	}
	if pick == 0 {
		return
	}
	switch id {
	case idEngine:
		setEngineSel = pick - 1
		showKeyFor(setEngineSel)
	case idLang:
		setLangSel = pick - 1
	case idUILang:
		setUISel = pick - 1
	case idSkin:
		setSkinSel = pick - 1
		applySkin(skins[setSkinSel].ID) // live preview; Cancel undoes it
		if setTitleFont != 0 {
			pDeleteObject.Call(setTitleFont)
		}
		setTitleFont = newFont(int(sc(17)), skin.TitleWeight, settingsTitleFace())
		if settingsHwnd != 0 {
			pref := skin.Corner
			if pDwmSetWindowAttribute.Find() == nil {
				pDwmSetWindowAttribute.Call(settingsHwnd, 33, uintptr(unsafe.Pointer(&pref)), 4)
			}
			invalidateAll(settingsHwnd)
		}
	}
	invalidate(c)
}

// drawCheckbox paints the skinned checkbox with its label.
func drawCheckbox(hdc uintptr, r RECT, on bool, text string) {
	box := sc(18)
	br := RECT{r.Left, (r.Top + r.Bottom - box) / 2, r.Left + box, (r.Top+r.Bottom-box)/2 + box}
	drawField(hdc, br)
	if on {
		g := newGfx(hdc)
		cx, cy := (br.Left+br.Right)/2, (br.Top+br.Bottom)/2
		a := sc(5)
		c := colAccent
		g.line(cx-a, cy, cx-a/3, cy+a*2/3, argb(c, 255), 2.4)
		g.line(cx-a/3, cy+a*2/3, cx+a, cy-a*2/3, argb(c, 255), 2.4)
		g.Close()
	}
	pSetBkMode.Call(hdc, TRANSPARENT)
	pSetTextColor.Call(hdc, colText)
	pSelectObject.Call(hdc, setFont)
	tr := RECT{br.Right + sc(10), r.Top, r.Right, r.Bottom}
	drawText(hdc, text, &tr, DT_SINGLELINE|DT_VCENTER|DT_NOPREFIX)
}

// drawDropdownControl paints a settings dropdown in the skin's style.
func drawDropdownControl(hdc uintptr, r RECT, text string, hot bool) {
	txt := colText
	if wowChrome() {
		txt = drawDropdown(hdc, r, hot)
		txt = colText
	} else {
		drawField(hdc, r)
		g := newGfx(hdc)
		cx, cy := r.Right-sc(16), (r.Top+r.Bottom)/2
		a := sc(4)
		g.line(cx-a, cy-a/2, cx, cy+a/2, argb(colAccent, 255), 2)
		g.line(cx, cy+a/2, cx+a, cy-a/2, argb(colAccent, 255), 2)
		g.Close()
	}
	pSetBkMode.Call(hdc, TRANSPARENT)
	pSetTextColor.Call(hdc, txt)
	pSelectObject.Call(hdc, setFont)
	tr := RECT{r.Left + sc(10), r.Top, r.Right - sc(34), r.Bottom}
	drawText(hdc, text, &tr, DT_SINGLELINE|DT_VCENTER|DT_NOPREFIX|DT_END_ELLIPSIS)
}

var (
	usageLabel    = -1
	usageResult   string
	settingsHwnd  uintptr
	setCtl        = map[int]uintptr{}
	setFont       uintptr
	setBrushBg    uintptr
	setBrushField uintptr
	setFields     []RECT
	setLabels     []setLabel
	setClassDone  bool
)

type setLabel struct {
	r    RECT
	text string
	dim  bool
}

func openSettings() {
	if settingsHwnd != 0 {
		pSetForegroundWindow.Call(settingsHwnd)
		return
	}
	inst, _, _ := pGetModuleHandleW.Call(0)
	cls := u16("ParleySettings")
	if !setClassDone {
		cur, _, _ := pLoadCursorW.Call(0, IDC_ARROW)
		wc := WNDCLASSEX{LpfnWndProc: syscall.NewCallback(settingsProc), HInstance: inst, HCursor: cur,
			LpszClassName: cls, HIcon: app.icon, HIconSm: app.iconSmall}
		wc.CbSize = uint32(unsafe.Sizeof(wc))
		pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
		setClassDone = true
		setBrushBg, _, _ = pCreateSolidBrush.Call(colBg)
		setBrushField, _, _ = pCreateSolidBrush.Call(colField)
	}
	W, H := sc(480), sc(720)
	sw, _, _ := pGetSystemMetrics.Call(0)
	shh, _, _ := pGetSystemMetrics.Call(1)
	h, _, _ := pCreateWindowExW.Call(WS_EX_TOPMOST|WS_EX_CONTROLPARENT, uintptr(unsafe.Pointer(cls)),
		u16p("Parley settings"), WS_POPUP|WS_CLIPCHILDREN,
		uintptr((int32(sw)-W)/2), uintptr((int32(shh)-H)/2), uintptr(W), uintptr(H), 0, 0, inst, 0)
	settingsHwnd = h
	darkTitle(h)
	setFont = newFont(int(sc(14)), 400, "Segoe UI")
	setTitleFont = newFont(int(sc(17)), skin.TitleWeight, settingsTitleFace())
	pref := skin.Corner
	if pDwmSetWindowAttribute.Find() == nil {
		pDwmSetWindowAttribute.Call(h, 33, uintptr(unsafe.Pointer(&pref)), 4)
	}
	buildSettings(h, inst)
	pSetTimer.Call(h, tipTimerID, 100, 0)
	pShowWindow.Call(h, SW_SHOW)
	pSetForegroundWindow.Call(h)
	pSetFocus.Call(setCtl[idKey])
	if key := app.key; key != "" && setKeyShows == 1 {
		go func() {
			used, limit, err := app.tr.Usage(key)
			app.mu.Lock()
			if err == nil {
				usageResult = Tf("DeepL: %s. Falls back to offline if it fails.", usageText(used, limit))
			} else {
				usageResult = ""
			}
			app.mu.Unlock()
			pPostMessageW.Call(h, msgUsage, 0, 0)
		}()
	}
}

// showKeyFor switches the key row between DeepL (engines 0-1) and Azure (2).
func showKeyFor(sel int) {
	want := 1
	if sel == 2 {
		want = 2
	}
	if setKeyShows == want || keyField < 0 {
		return
	}
	switch setKeyShows {
	case 1:
		setKeyDeepL = strings.TrimSpace(getText(setCtl[idKey]))
	case 2:
		setKeyAzure = strings.TrimSpace(getText(setCtl[idKey]))
		setRegion = strings.TrimSpace(getText(setCtl[idRegion]))
	}
	setKeyShows = want
	kr := setFields[keyField]
	keyR, regR := RECT{kr.Left, kr.Top, setKeyRight, kr.Bottom}, RECT{}
	if want == 2 {
		regW := sc(120)
		keyR.Right = setKeyRight - regW - sc(8)
		regR = RECT{keyR.Right + sc(8), kr.Top, setKeyRight, kr.Bottom}
	}
	setFields[keyField], setFields[regionField] = keyR, regR
	place := func(c uintptr, r RECT) {
		inset, eh := sc(8), sc(18)
		w := r.Right - r.Left - 2*inset
		if w < 0 {
			w = 0
		}
		pMoveWindow.Call(c, uintptr(r.Left+inset), uintptr(r.Top+(r.Bottom-r.Top-eh)/2), uintptr(w), uintptr(eh), 1)
	}
	place(setCtl[idKey], keyR)
	place(setCtl[idRegion], regR)
	if want == 2 {
		setText(setCtl[idKey], setKeyAzure)
		setText(setCtl[idRegion], setRegion)
		pShowWindow.Call(setCtl[idRegion], 5)
		setLabels[keyLabel].text = T("Azure Translator key and region")
		setLabels[usageLabel].text = T("Find the region next to your key in the Azure portal.")
	} else {
		setText(setCtl[idKey], setKeyDeepL)
		pShowWindow.Call(setCtl[idRegion], 0)
		setLabels[keyLabel].text = T("DeepL API key (optional)")
		setLabels[usageLabel].text = T("Only for DeepL. If DeepL fails, Parley falls back to offline.")
	}
	if settingsHwnd != 0 {
		invalidateAll(settingsHwnd)
	}
}

func buildSettings(h, inst uintptr) {
	setFields, setLabels = nil, nil
	pad := sc(20)
	r := clientRect(h)
	fullW := r.Right - 2*pad
	y := setHeadH() + sc(14)
	rowH := sc(30)

	label := func(text string, dim bool) {
		setLabels = append(setLabels, setLabel{RECT{pad, y, pad + fullW, y + sc(20)}, text, dim})
		y += sc(22)
	}
	ctl := func(id int, class string, style uintptr, x, w int32, text string) uintptr {
		c, _, _ := pCreateWindowExW.Call(0, u16p(class), u16p(text), WS_CHILD|WS_VISIBLE|WS_TABSTOP|style,
			uintptr(x), uintptr(y), uintptr(w), uintptr(rowH), h, uintptr(id), inst, 0)
		pSendMessageW.Call(c, WM_SETFONT, setFont, 1)
		setCtl[id] = c
		return c
	}
	edit := func(id int, style uintptr, x, w int32, text string) uintptr {
		// borderless edit inside a painted rounded field
		setFields = append(setFields, RECT{x, y, x + w, y + rowH})
		inset := sc(8)
		eh := sc(18)
		c, _, _ := pCreateWindowExW.Call(0, u16p("EDIT"), u16p(text),
			WS_CHILD|WS_VISIBLE|WS_TABSTOP|ES_AUTOHSCROLL|style,
			uintptr(x+inset), uintptr(y+(rowH-eh)/2), uintptr(w-2*inset), uintptr(eh), h, uintptr(id), inst, 0)
		pSendMessageW.Call(c, WM_SETFONT, setFont, 1)
		setCtl[id] = c
		return c
	}

	c := app.cfg
	label(T("Translation engine"), false)
	setEngineSel = map[string]int{"offline": 0, "deepl": 1, "azure": 2}[app.engineMode()]
	ctl(idEngine, "BUTTON", BS_OWNERDRAW, pad, fullW, "")
	y += rowH + sc(12)

	setKeyDeepL, setKeyAzure, setRegion = app.key, app.azKey, c.AzureRegion
	keyLabel = len(setLabels)
	label("", false)
	btnW := sc(120)
	keyField = len(setFields)
	setKeyRight = pad + fullW - btnW - sc(8)
	edit(idKey, ES_PASSWORD, pad, fullW-btnW-sc(8), "")
	regionField = len(setFields)
	edit(idRegion, 0, pad, sc(10), "")
	pSendMessageW.Call(setCtl[idRegion], 0x1501 /*EM_SETCUEBANNER*/, 1, uintptr(unsafe.Pointer(u16(T("region")))))
	ctl(idGetKey, "BUTTON", BS_OWNERDRAW, pad+fullW-btnW, btnW, T("Get a key"))
	y += rowH + sc(4)
	usageLabel = len(setLabels)
	label("", true)
	y += sc(8)
	setKeyShows = 0
	showKeyFor(setEngineSel)

	yRow := y
	label(T("Translate chat into"), false)
	skinX := pad + sc(236)
	setLabels = append(setLabels, setLabel{RECT{skinX, yRow, pad + fullW, yRow + sc(20)}, T("Skin"), false})
	setSkinSel, setLangSel = 0, 0
	for i, s := range skins {
		if s.ID == skin.ID {
			setSkinSel = i
		}
	}
	for i, l := range targetLangs {
		if l.Code == c.MyLang {
			setLangSel = i
		}
	}
	ctl(idSkin, "BUTTON", BS_OWNERDRAW, skinX, pad+fullW-skinX, "")
	ctl(idLang, "BUTTON", BS_OWNERDRAW, pad, sc(220), "")
	y += rowH + sc(12)

	label(T("Interface language"), false)
	setUISel = 0
	for i, l := range uiLanguages {
		if l.Code == c.UILang {
			setUISel = i + 1
		}
	}
	ctl(idUILang, "BUTTON", BS_OWNERDRAW, pad, fullW, "")
	y += rowH + sc(12)

	label(T("WoW folder"), false)
	edit(idWow, 0, pad, fullW-btnW-sc(8), c.WowPath)
	ctl(idInstall, "BUTTON", BS_OWNERDRAW, pad+fullW-btnW, btnW, T("Install addon"))
	y += rowH + sc(12)

	half := (fullW - sc(16)) / 2
	yLabel := y
	label(T("Text size (10-32)"), false)
	y = yLabel
	setLabels = append(setLabels, setLabel{RECT{pad + half + sc(16), y, pad + fullW, y + sc(20)}, T("Overlay opacity % (40-100)"), false})
	y += sc(22)
	edit(idFont, ES_NUMBER, pad, half, strconv.Itoa(c.FontSize))
	edit(idOpacity, ES_NUMBER, pad+half+sc(16), half, strconv.Itoa(c.Opacity))
	y += rowH + sc(12)

	label(T("Alert me when a message mentions (your character is included)"), false)
	edit(idAlert, 0, pad, fullW, c.AlertWords)
	pSendMessageW.Call(setCtl[idAlert], 0x1501, 1, uintptr(unsafe.Pointer(u16(T("e.g. Deadmines, healer, WTB")))))
	y += rowH + sc(12)

	label(T("Never translate these names or words"), false)
	edit(idKeepWords, 0, pad, fullW, c.KeepWords)
	pSendMessageW.Call(setCtl[idKeepWords], 0x1501, 1, uintptr(unsafe.Pointer(u16(T("e.g. your guild name, nicknames")))))
	y += rowH + sc(12)

	setOrigOn = c.ShowOriginal
	ctl(idOrig, "BUTTON", BS_OWNERDRAW, pad, fullW, T("Show the original text under each translation"))
	y += rowH + sc(10)
	label(T("Reply hotkey: Ctrl+Shift+T. Esc goes back to WoW."), true)

	y = r.Bottom - rowH - sc(16)
	bw := sc(100)
	ctl(idCancel, "BUTTON", BS_OWNERDRAW, r.Right-pad-bw, bw, T("Cancel"))
	ctl(idSave, "BUTTON", BS_OWNERDRAW, r.Right-pad-2*bw-sc(10), bw, T("Save"))
}

func settingsProc(h, msg, wp, lp uintptr) uintptr {
	switch msg {
	case WM_TIMER:
		if wp == tipTimerID {
			pollSettingsTip()
		}
		return 0
	case msgUsage:
		app.mu.Lock()
		txt := usageResult
		app.mu.Unlock()
		if txt != "" && usageLabel >= 0 && usageLabel < len(setLabels) {
			setLabels[usageLabel].text = txt
			invalidate(h)
		}
		return 0
	case WM_ERASEBKGND:
		return 1
	case WM_PAINT:
		var ps PAINTSTRUCT
		hdc, _, _ := pBeginPaint.Call(h, uintptr(unsafe.Pointer(&ps)))
		cr := clientRect(h)
		fillRect(hdc, cr, colBg)
		drawHeaderBg(hdc, cr.Right, setHeadH())
		pSetBkMode.Call(hdc, TRANSPARENT)
		tx := sc(14)
		is := sc(24)
		if drawIcon(hdc, 1, RECT{tx, (setHeadH() - is) / 2, tx + is, (setHeadH()-is)/2 + is}) {
			tx += is + sc(8)
		}
		pSelectObject.Call(hdc, setTitleFont)
		pSetTextColor.Call(hdc, skin.Title)
		ttl := RECT{tx, 0, cr.Right - sc(50), setHeadH()}
		drawText(hdc, T("Parley settings"), &ttl, DT_SINGLELINE|DT_VCENTER|DT_NOPREFIX)
		drawHeaderButton(hdc, setCloseRect(cr.Right), "\uE8BB", setCloseHot, true)
		if skin.Chrome == "plain" {
			fillRect(hdc, RECT{0, setHeadH() - 1, cr.Right, setHeadH()}, lighten(colBg, 14))
		}
		drawMenuFrame(hdc, cr.Right, cr.Bottom)
		for _, f := range setFields {
			if f.Right > f.Left {
				drawField(hdc, f)
			}
		}
		pSetBkMode.Call(hdc, TRANSPARENT)
		pSelectObject.Call(hdc, setFont)
		for _, l := range setLabels {
			col := colText
			if l.dim {
				col = colDim
			}
			pSetTextColor.Call(hdc, col)
			lr := l.r
			drawText(hdc, l.text, &lr, DT_SINGLELINE|DT_NOPREFIX|DT_END_ELLIPSIS)
		}
		pEndPaint.Call(h, uintptr(unsafe.Pointer(&ps)))
		return 0
	case WM_NCHITTEST:
		pt := POINT{loword(lp), hiword(lp)}
		pScreenToClient.Call(h, uintptr(unsafe.Pointer(&pt)))
		if pt.Y < setHeadH() && !inRect(setCloseRect(clientRect(h).Right), pt.X, pt.Y) {
			return HTCAPTION
		}
		return 1 // HTCLIENT
	case 0x0200: // WM_MOUSEMOVE: close-button hover
		hot := inRect(setCloseRect(clientRect(h).Right), loword(lp), hiword(lp))
		if hot != setCloseHot {
			setCloseHot = hot
			invalidateHeader(h)
		}
		if !setTracking {
			tme := TRACKMOUSEEVENT{DwFlags: 2, HwndTrack: h}
			tme.CbSize = uint32(unsafe.Sizeof(tme))
			pTrackMouseEvent.Call(uintptr(unsafe.Pointer(&tme)))
			setTracking = true
		}
		return 0
	case WM_MOUSELEAVE:
		setTracking = false
		if setCloseHot {
			setCloseHot = false
			invalidateHeader(h)
		}
		return 0
	case WM_LBUTTONUP:
		if inRect(setCloseRect(clientRect(h).Right), loword(lp), hiword(lp)) {
			pPostMessageW.Call(h, WM_COMMAND, idCancel, 0)
		}
		return 0
	case WM_CTLCOLOREDIT:
		pSetTextColor.Call(wp, colText)
		pSetBkColor.Call(wp, colField)
		return setBrushField
	case WM_CTLCOLORLISTBOX:
		pSetTextColor.Call(wp, colText)
		pSetBkColor.Call(wp, colField)
		return setBrushField
	case WM_CTLCOLORSTATIC, WM_CTLCOLORBTN:
		pSetTextColor.Call(wp, colText)
		pSetBkColor.Call(wp, colBg)
		return setBrushBg
	case WM_DRAWITEM:
		di := (*DRAWITEMSTRUCT)(unsafe.Pointer(lp))
		fillRect(di.HDC, di.RcItem, colBg)
		switch id := int(di.CtlID); {
		case isDropdown(id):
			drawDropdownControl(di.HDC, di.RcItem, dropdownText(id), di.ItemState&ODS_SELECTED != 0)
			return 1
		case id == idOrig:
			drawCheckbox(di.HDC, di.RcItem, setOrigOn, getText(di.HwndItem))
			return 1
		}
		txt := drawButton(di.HDC, di.RcItem, di.ItemState&ODS_SELECTED != 0)
		pSetBkMode.Call(di.HDC, TRANSPARENT)
		pSetTextColor.Call(di.HDC, txt)
		pSelectObject.Call(di.HDC, setFont)
		r := di.RcItem
		drawText(di.HDC, getText(di.HwndItem), &r, DT_SINGLELINE|DT_VCENTER|DT_CENTER|DT_NOPREFIX)
		return 1
	case WM_COMMAND:
		id := int(wp & 0xffff)
		if isDropdown(id) {
			openDropdown(id)
			return 0
		}
		switch id {
		case idOrig:
			setOrigOn = !setOrigOn
			invalidate(setCtl[idOrig])
		case idGetKey:
			if setKeyShows == 2 {
				openURL("https://learn.microsoft.com/azure/ai-services/translator/create-translator-resource")
			} else {
				openURL("https://www.deepl.com/pro-api")
			}
		case idInstall:
			path := strings.TrimSpace(getText(setCtl[idWow]))
			if dests, err := installAddonAll(path); err != nil {
				skinAlert(h, T("Couldn't install the addon"), alertLines(localizeMsg(err.Error()), "",
					T("If Windows blocked the write, copy the addon\\Parley folder from the Parley folder into Interface\\AddOns yourself.")))
			} else {
				skinAlert(h, T("Addon installed"), alertLines(append(dests, "", T("Type /reload in WoW if it's running."))...))
			}
		case idSave, 1: // 1 = IDOK (Enter)
			saveSettings()
			pDestroyWindow.Call(h)
		case idCancel, 2: // 2 = IDCANCEL (Esc)
			applySkin(app.cfg.Skin) // undo any preview
			pDestroyWindow.Call(h)
		}
		return 0
	case WM_CLOSE:
		applySkin(app.cfg.Skin)
		pDestroyWindow.Call(h)
		return 0
	case WM_DESTROY:
		settingsHwnd = 0
		settingsTip.reset()
		setCtl = map[int]uintptr{}
		if setFont != 0 {
			pDeleteObject.Call(setFont)
			setFont = 0
		}
		if setTitleFont != 0 {
			pDeleteObject.Call(setTitleFont)
			setTitleFont = 0
		}
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(h, msg, wp, lp)
	return r
}

func saveSettings() {
	switch setKeyShows { // pick up what's in the key row now
	case 1:
		setKeyDeepL = strings.TrimSpace(getText(setCtl[idKey]))
	case 2:
		setKeyAzure = strings.TrimSpace(getText(setCtl[idKey]))
		setRegion = strings.TrimSpace(getText(setCtl[idRegion]))
	}
	key, azKey, region := setKeyDeepL, setKeyAzure, setRegion
	alertWords := strings.TrimSpace(getText(setCtl[idAlert]))
	keepWords := strings.TrimSpace(getText(setCtl[idKeepWords]))
	sel := setLangSel
	fontSize, _ := strconv.Atoi(getText(setCtl[idFont]))
	opacity, _ := strconv.Atoi(getText(setCtl[idOpacity]))

	wow := strings.TrimSpace(getText(setCtl[idWow]))
	skinSel, engSel := setSkinSel, setEngineSel
	uiSel := setUISel

	app.mu.Lock()
	app.key = key
	app.cfg.DeepLKeyEnc = protect([]byte(key))
	app.azKey = azKey
	app.cfg.AzureKeyEnc = protect([]byte(azKey))
	app.cfg.AzureRegion = region
	app.cfg.AlertWords = alertWords
	app.cfg.KeepWords = keepWords
	if sel >= 0 && sel < len(targetLangs) {
		app.cfg.MyLang = targetLangs[sel].Code
	}
	if fontSize >= 10 && fontSize <= 32 {
		app.cfg.FontSize = fontSize
	}
	if opacity >= 40 && opacity <= 100 {
		app.cfg.Opacity = opacity
	}
	app.cfg.ShowOriginal = setOrigOn
	app.cfg.Engine = [...]string{"offline", "deepl", "azure"}[engSel]
	engineMode := app.cfg.Engine
	oldUI := app.cfg.UILang
	app.cfg.UILang = ""
	if uiSel > 0 {
		app.cfg.UILang = uiLanguages[uiSel-1].Code
	}
	uiChanged := oldUI != app.cfg.UILang
	if skinSel >= 0 && skinSel < len(skins) {
		app.cfg.Skin = skins[skinSel].ID
	}
	wowChanged := wow != app.cfg.WowPath
	app.cfg.WowPath = wow
	cfg := app.cfg
	app.mu.Unlock()

	saveConfig(cfg)
	setKeepWords(cfg.KeepWords)
	if uiChanged {
		setUILang(cfg.UILang)
		app.mu.Lock()
		app.status = T("Looking for WoW…")
		app.mu.Unlock()
		copy(app.tray.SzTip[:], make([]uint16, len(app.tray.SzTip)))
		copy(app.tray.SzTip[:], syscall.StringToUTF16(T("Parley: WoW chat translator")))
		pShellNotifyIconW.Call(NIM_MODIFY, uintptr(unsafe.Pointer(&app.tray)))
	}
	applyOpacity()
	updateDPI()
	invalidate(app.overlay)
	switch {
	case engineMode == "deepl" && key == "":
		app.setNotice(T("DeepL needs an API key. Until you add one, Parley translates offline."))
	case engineMode == "azure" && azKey == "":
		app.setNotice(T("Azure needs a key. Until you add one, Parley translates offline."))
	default:
		app.setNotice(T("Settings saved."))
	}
	if wowChanged {
		if _, err := os.Stat(wow); err == nil && addonInstalledVersion(wow) == "" {
			if _, err := installAddonAll(wow); err == nil {
				app.setNotice(T("Settings saved and the addon was installed. /reload in WoW."))
			}
		}
	}
}

func invalidateHeader(h uintptr) {
	r := RECT{0, 0, clientRect(h).Right, setHeadH()}
	pInvalidateRect.Call(h, uintptr(unsafe.Pointer(&r)), 0)
}

func settingsTitleFace() string {
	if nonLatinUI() {
		return "Segoe UI"
	}
	return pickFace(skin.TitleFaces)
}
