package main

// Skinned popup menus. Windows' own menus can't be themed, so Parley draws
// its menus itself (tray menu, message right-click menu, reply-language
// picker) in the current skin: WoW's dropdown-menu look for the WoW skins,
// a dark rounded panel for the modern skin.
//
// showMenu runs a small modal loop, like TrackPopupMenu: the first menu
// window captures the mouse and keyboard, submenus open on hover, and the
// call returns the chosen item's ID (0 = cancelled).

import (
	"syscall"
	"unsafe"
)

type MenuItem struct {
	Label    string
	Hint     string // right-aligned text, e.g. a shortcut
	ID       int
	Checked  bool
	Radio    bool // draw a dot instead of a tick when checked
	Sep      bool
	Title    bool // gold, unclickable section heading
	Disabled bool
	Sub      []MenuItem
}

func (it MenuItem) selectable() bool { return !it.Sep && !it.Title && !it.Disabled }

type menuLevel struct {
	hwnd     uintptr
	items    []MenuItem
	rect     RECT   // on screen
	rows     []RECT // client coordinates, before scrolling
	hover    int
	scroll   int32
	contentH int32
	parent   int // item index in the level below that opened this one
}

var (
	menuStack     []*menuLevel
	menuResult    int
	menuDone      bool
	menuClassDone bool
	menuFont      uintptr
	menuFontAlt   uintptr
	menuFontTitle uintptr
	menuPending   = -1 // level whose hover change is waiting to open/close submenus
	// one-shot options for the next showMenu (used by dropdowns)
	menuMinWidth   int32
	menuStartHover = -1
)

const (
	menuTimerID  = 7
	wmCapChanged = 0x0215
	wmRButtonDn  = 0x0204
	wmLButtonDn  = 0x0201
	wmMouseMove  = 0x0200
	wmActivateAp = 0x001C
	vkUp, vkDown = 0x26, 0x28
	vkLeft       = 0x25
	vkRight      = 0x27
	vkHome       = 0x24
	vkEnd        = 0x23
)

var (
	pLoadImageW      = user32.NewProc("LoadImageW")
	pDrawIconEx      = user32.NewProc("DrawIconEx")
	iconCache        = map[[2]int]uintptr{}
	pSetCapture      = user32.NewProc("SetCapture")
	pReleaseCapture  = user32.NewProc("ReleaseCapture")
	pMonitorFromPt   = user32.NewProc("MonitorFromPoint")
	pGetMonitorInfoW = user32.NewProc("GetMonitorInfoW")
)

type monitorInfo struct {
	CbSize  uint32
	Monitor RECT
	Work    RECT
	Flags   uint32
}

func workAreaAt(x, y int32) RECT {
	pt := uintptr(uint32(x)) | uintptr(uint32(y))<<32
	mon, _, _ := pMonitorFromPt.Call(pt, 2 /*MONITOR_DEFAULTTONEAREST*/)
	mi := monitorInfo{CbSize: uint32(unsafe.Sizeof(monitorInfo{}))}
	if mon != 0 {
		if r, _, _ := pGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&mi))); r != 0 {
			return mi.Work
		}
	}
	w, _, _ := pGetSystemMetrics.Call(0)
	h, _, _ := pGetSystemMetrics.Call(1)
	return RECT{0, 0, int32(w), int32(h)}
}

// showMenu opens a skinned menu with its top-left at (x, y), or its
// bottom-left there when above is true, and returns the chosen ID.
func showMenu(items []MenuItem, x, y int32, above bool) int {
	if len(items) == 0 || len(menuStack) > 0 {
		return 0
	}
	registerMenuClass()
	makeMenuFonts()
	menuDone, menuResult, menuPending = false, 0, -1
	openMenuLevel(items, RECT{x, y, x, y}, above, -1)
	menuMinWidth = 0
	if menuStartHover >= 0 && menuStartHover < len(items) {
		lv := menuStack[0]
		lv.hover = menuStartHover
		ensureMenuRowVisible(lv, menuStartHover)
	}
	menuStartHover = -1
	root := menuStack[0].hwnd
	pSetForegroundWindow.Call(root)
	pSetFocus.Call(root)
	pSetCapture.Call(root)

	var m MSG
	for !menuDone {
		r, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			pPostQuitMessage.Call(0)
			break
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
	pKillTimer.Call(root, menuTimerID)
	stack := menuStack
	menuStack = nil
	pReleaseCapture.Call()
	for i := len(stack) - 1; i >= 0; i-- {
		pDestroyWindow.Call(stack[i].hwnd)
	}
	return menuResult
}

// showMenuAtCursor opens a menu at the mouse pointer (tray and right-click menus).
func showMenuAtCursor(items []MenuItem) int {
	var pt POINT
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	return showMenu(items, pt.X, pt.Y, false)
}

func registerMenuClass() {
	if menuClassDone {
		return
	}
	inst, _, _ := pGetModuleHandleW.Call(0)
	cur, _, _ := pLoadCursorW.Call(0, IDC_ARROW)
	wc := WNDCLASSEX{Style: 0x00020000 /*CS_DROPSHADOW*/, LpfnWndProc: syscall.NewCallback(menuProc),
		HInstance: inst, HCursor: cur, LpszClassName: u16("ParleyMenu")}
	wc.CbSize = uint32(unsafe.Sizeof(wc))
	pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	menuClassDone = true
}

func makeMenuFonts() {
	for _, f := range []uintptr{menuFont, menuFontAlt, menuFontTitle} {
		if f != 0 {
			pDeleteObject.Call(f)
		}
	}
	size := app.cfg.FontSize - 1
	if size > 16 {
		size = 16
	}
	if size < 12 {
		size = 12
	}
	face, tface := pickFace(skin.BodyFaces), pickFace(skin.TitleFaces)
	if nonLatinUI() {
		face, tface = "Segoe UI", "Segoe UI"
	}
	menuFont = newFont(int(sc(size)), 400, face)
	menuFontAlt = newFont(int(sc(size)), 400, "Segoe UI")
	menuFontTitle = newFont(int(sc(size)), max(skin.HeadWeight, 600), tface)
}

// Metrics.
func menuPadY() int32   { return sc(6) }
func menuRowH() int32   { return sc(app.cfg.FontSize) + sc(11) }
func menuSepH() int32   { return sc(9) }
func menuGutter() int32 { return sc(26) }
func menuArrowW() int32 { return sc(24) }

// openMenuLevel lays out and shows one menu window next to anchor.
func openMenuLevel(items []MenuItem, anchor RECT, above bool, parent int) {
	lv := &menuLevel{items: items, hover: -1, parent: parent}
	// measure
	hdc, _, _ := pGetDC.Call(0)
	textW, hintW := int32(0), int32(0)
	for _, it := range items {
		if it.Sep {
			continue
		}
		f := menuFont
		if it.Title {
			f = menuFontTitle
		} else if !latinOnly(it.Label) {
			f = menuFontAlt
		}
		pSelectObject.Call(hdc, f)
		r := RECT{}
		drawText(hdc, it.Label, &r, DT_CALCRECT|DT_SINGLELINE|DT_NOPREFIX)
		textW = max32(textW, r.Right)
		if it.Hint != "" {
			hr := RECT{}
			pSelectObject.Call(hdc, menuFont)
			drawText(hdc, it.Hint, &hr, DT_CALCRECT|DT_SINGLELINE|DT_NOPREFIX)
			hintW = max32(hintW, hr.Right+sc(24))
		}
	}
	pReleaseDC.Call(0, hdc)
	w := menuGutter() + textW + hintW + menuArrowW() + sc(6)
	w = max32(w, sc(150))
	if parent < 0 {
		w = max32(w, menuMinWidth)
	}
	y := menuPadY()
	for _, it := range items {
		h := menuRowH()
		if it.Sep {
			h = menuSepH()
		}
		lv.rows = append(lv.rows, RECT{menuPadY(), y, w - menuPadY(), y + h})
		y += h
	}
	lv.contentH = y + menuPadY()
	h := lv.contentH

	// position on screen, flipping or clamping to the monitor
	wa := workAreaAt(anchor.Left, anchor.Top)
	if h > wa.Bottom-wa.Top {
		h = wa.Bottom - wa.Top
	}
	var x, top int32
	if parent < 0 { // root: at the point
		x, top = anchor.Left, anchor.Top
		if above || top+h > wa.Bottom {
			top = anchor.Top - h
		}
		if x+w > wa.Right {
			x = anchor.Left - w
		}
	} else { // submenu: beside the parent row
		x, top = anchor.Right-sc(3), anchor.Top-menuPadY()
		if x+w > wa.Right {
			x = anchor.Left - w + sc(3)
		}
		if top+h > wa.Bottom {
			top = wa.Bottom - h
		}
	}
	x = max32(wa.Left, min32(x, wa.Right-w))
	top = max32(wa.Top, min32(top, wa.Bottom-h))
	lv.rect = RECT{x, top, x + w, top + h}

	inst, _, _ := pGetModuleHandleW.Call(0)
	hw, _, _ := pCreateWindowExW.Call(WS_EX_TOPMOST|WS_EX_TOOLWINDOW, uintptr(unsafe.Pointer(u16("ParleyMenu"))), 0,
		WS_POPUP, uintptr(x), uintptr(top), uintptr(w), uintptr(h), app.overlay, 0, inst, 0)
	lv.hwnd = hw
	pref := skin.Corner
	if pDwmSetWindowAttribute.Find() == nil {
		pDwmSetWindowAttribute.Call(hw, 33, uintptr(unsafe.Pointer(&pref)), 4)
	}
	menuStack = append(menuStack, lv)
	pShowWindow.Call(hw, 4 /*SW_SHOWNOACTIVATE*/)
	invalidate(hw)
}

func closeMenuLevelsAbove(depth int) {
	for len(menuStack) > depth+1 {
		top := menuStack[len(menuStack)-1]
		menuStack = menuStack[:len(menuStack)-1]
		pDestroyWindow.Call(top.hwnd)
	}
}

// menuHit finds the level and row under a screen point (-1, -1 if none).
func menuHit(x, y int32) (int, int) {
	for d := len(menuStack) - 1; d >= 0; d-- {
		lv := menuStack[d]
		if !inRect(lv.rect, x, y) {
			continue
		}
		cy := y - lv.rect.Top + lv.scroll
		for i, r := range lv.rows {
			if cy >= r.Top && cy < r.Bottom {
				return d, i
			}
		}
		return d, -1
	}
	return -1, -1
}

func setMenuHover(d, i int) {
	lv := menuStack[d]
	if lv.hover == i {
		return
	}
	lv.hover = i
	invalidate(lv.hwnd)
	// a deeper level showing a different item's submenu closes after a pause
	menuPending = d
	pSetTimer.Call(menuStack[0].hwnd, menuTimerID, 220, 0)
}

// syncSubmenu opens the hovered item's submenu (or closes stale ones) for level d.
func syncSubmenu(d int) {
	if d < 0 || d >= len(menuStack) {
		return
	}
	lv := menuStack[d]
	if len(menuStack) > d+1 && menuStack[d+1].parent == lv.hover {
		return // already open
	}
	closeMenuLevelsAbove(d)
	if lv.hover >= 0 && len(lv.items[lv.hover].Sub) > 0 && !lv.items[lv.hover].Disabled {
		r := lv.rows[lv.hover]
		anchor := RECT{lv.rect.Left, lv.rect.Top + r.Top - lv.scroll, lv.rect.Right, lv.rect.Top + r.Bottom - lv.scroll}
		openMenuLevel(lv.items[lv.hover].Sub, anchor, false, lv.hover)
	}
}

func activateMenuItem(d, i int) {
	if d < 0 || i < 0 {
		return
	}
	it := menuStack[d].items[i]
	if !it.selectable() {
		return
	}
	if len(it.Sub) > 0 {
		menuStack[d].hover = i
		syncSubmenu(d)
		if len(menuStack) > d+1 {
			moveMenuHover(d+1, 1, true)
		}
		return
	}
	menuResult, menuDone = it.ID, true
}

// moveMenuHover steps the highlight in level d (dir +1/-1), skipping
// separators and headings. fromEdge starts at the top/bottom.
func moveMenuHover(d, dir int, fromEdge bool) {
	lv := menuStack[d]
	n := len(lv.items)
	i := lv.hover
	if fromEdge || i < 0 {
		if dir > 0 {
			i = -1
		} else {
			i = n
		}
	}
	for k := 0; k < n; k++ {
		i += dir
		if i < 0 || i >= n {
			return
		}
		if lv.items[i].selectable() {
			lv.hover = i
			ensureMenuRowVisible(lv, i)
			invalidate(lv.hwnd)
			closeMenuLevelsAbove(d)
			return
		}
	}
}

func ensureMenuRowVisible(lv *menuLevel, i int) {
	viewH := lv.rect.Bottom - lv.rect.Top
	r := lv.rows[i]
	if r.Top-lv.scroll < menuPadY() {
		lv.scroll = r.Top - menuPadY()
	} else if r.Bottom-lv.scroll > viewH-menuPadY() {
		lv.scroll = r.Bottom - viewH + menuPadY()
	}
	lv.scroll = max32(0, min32(lv.scroll, lv.contentH-viewH))
}

func menuProc(h, msg, wp, lp uintptr) uintptr {
	switch msg {
	case WM_PAINT:
		for _, lv := range menuStack {
			if lv.hwnd == h {
				paintMenu(h, lv)
				return 0
			}
		}
		var ps PAINTSTRUCT
		pBeginPaint.Call(h, uintptr(unsafe.Pointer(&ps)))
		pEndPaint.Call(h, uintptr(unsafe.Pointer(&ps)))
		return 0
	case WM_ERASEBKGND:
		return 1
	case 0x0021: // WM_MOUSEACTIVATE: don't steal activation from the root
		return 3 // MA_NOACTIVATE
	}
	isMenu := false
	for _, lv := range menuStack {
		isMenu = isMenu || lv.hwnd == h
	}
	if !isMenu {
		r, _, _ := pDefWindowProcW.Call(h, msg, wp, lp)
		return r
	}
	// The root menu has the capture and focus; submenus forward their mouse
	// input here too, so every menu window is handled the same way.
	screenPt := func() (int32, int32) {
		pt := POINT{loword(lp), hiword(lp)}
		pClientToScreen.Call(h, uintptr(unsafe.Pointer(&pt)))
		return pt.X, pt.Y
	}
	switch msg {
	case wmMouseMove:
		x, y := screenPt()
		d, i := menuHit(x, y)
		if d >= 0 {
			if d < len(menuStack)-1 && i == menuStack[d+1].parent {
				menuPending = -1 // back on the item that owns the open submenu
			}
			if i >= 0 && !menuStack[d].items[i].selectable() {
				i = -1
			}
			setMenuHover(d, i)
			if d < len(menuStack)-1 && i == menuStack[d+1].parent {
				menuPending = -1
			}
		}
		return 0
	case wmLButtonDn, wmRButtonDn:
		x, y := screenPt()
		if d, _ := menuHit(x, y); d < 0 {
			menuDone = true // click outside cancels
		}
		return 0
	case WM_LBUTTONUP, WM_RBUTTONUP:
		x, y := screenPt()
		d, i := menuHit(x, y)
		activateMenuItem(d, i)
		return 0
	case WM_MOUSEWHEEL:
		var pt POINT
		pGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
		if d, _ := menuHit(pt.X, pt.Y); d >= 0 {
			lv := menuStack[d]
			viewH := lv.rect.Bottom - lv.rect.Top
			lv.scroll -= hiword(wp) / 120 * menuRowH() * 3
			lv.scroll = max32(0, min32(lv.scroll, lv.contentH-viewH))
			closeMenuLevelsAbove(d)
			invalidate(lv.hwnd)
		}
		return 0
	case WM_TIMER:
		if wp == menuTimerID {
			pKillTimer.Call(h, menuTimerID)
			if menuPending >= 0 {
				syncSubmenu(menuPending)
				menuPending = -1
			}
		}
		return 0
	case WM_KEYDOWN:
		d := len(menuStack) - 1
		lv := menuStack[d]
		switch wp {
		case vkDown:
			moveMenuHover(d, 1, false)
		case vkUp:
			moveMenuHover(d, -1, false)
		case vkHome:
			moveMenuHover(d, 1, true)
		case vkEnd:
			moveMenuHover(d, -1, true)
		case vkRight:
			if lv.hover >= 0 && len(lv.items[lv.hover].Sub) > 0 {
				activateMenuItem(d, lv.hover)
			}
		case vkLeft:
			if d > 0 {
				closeMenuLevelsAbove(d - 1)
			}
		case VK_RETURN, 0x20: // Enter, Space
			activateMenuItem(d, lv.hover)
		case VK_ESCAPE:
			if d > 0 {
				closeMenuLevelsAbove(d - 1)
			} else {
				menuDone = true
			}
		}
		return 0
	case wmCapChanged:
		for _, lv := range menuStack {
			if lp == lv.hwnd {
				return 0
			}
		}
		menuDone = true
		return 0
	case wmActivateAp:
		if wp == 0 {
			menuDone = true
		}
		return 0
	case WM_ACTIVATE:
		if loword(wp) == 0 {
			menuDone = true
		}
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(h, msg, wp, lp)
	return r
}

// ---------------------------------------------------------------------------
// Painting
// ---------------------------------------------------------------------------

func paintMenu(h uintptr, lv *menuLevel) {
	var ps PAINTSTRUCT
	hdc, _, _ := pBeginPaint.Call(h, uintptr(unsafe.Pointer(&ps)))
	defer pEndPaint.Call(h, uintptr(unsafe.Pointer(&ps)))
	cr := clientRect(h)
	w, ht := cr.Right, cr.Bottom
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
	drawMenuBackground(mem, w, ht)

	pIntersectClipRect.Call(mem, uintptr(sc(2)), uintptr(sc(2)), uintptr(w-sc(2)), uintptr(ht-sc(2)))
	for i, it := range lv.items {
		r := lv.rows[i]
		r.Top -= lv.scroll
		r.Bottom -= lv.scroll
		if r.Bottom < 0 || r.Top > ht {
			continue
		}
		if it.Sep {
			drawMenuSeparator(mem, r)
			continue
		}
		hot := i == lv.hover && it.selectable()
		open := len(menuStack) > 0 && lvIndex(lv) < len(menuStack)-1 && menuStack[lvIndex(lv)+1].parent == i
		if hot || open {
			drawMenuHighlight(mem, r)
		}
		col := colText
		switch {
		case it.Title:
			col = colAccent
		case it.Disabled:
			col = colDim
		case hot && skin.Chrome == "plain" && skin.ID == "chat":
			col = colAccent
		}
		if it.Checked {
			drawMenuCheck(mem, RECT{r.Left, r.Top, r.Left + menuGutter(), r.Bottom}, it.Radio)
		}
		f := menuFont
		if it.Title {
			f = menuFontTitle
		} else if !latinOnly(it.Label) {
			f = menuFontAlt
		}
		pSelectObject.Call(mem, f)
		pSetTextColor.Call(mem, col)
		tr := RECT{r.Left + menuGutter(), r.Top, r.Right - menuArrowW(), r.Bottom}
		drawText(mem, it.Label, &tr, DT_SINGLELINE|DT_VCENTER|DT_NOPREFIX|DT_END_ELLIPSIS)
		if it.Hint != "" {
			pSelectObject.Call(mem, menuFont)
			pSetTextColor.Call(mem, colDim)
			hr := RECT{r.Left, r.Top, r.Right - menuArrowW(), r.Bottom}
			drawText(mem, it.Hint, &hr, DT_SINGLELINE|DT_VCENTER|DT_RIGHT|DT_NOPREFIX)
		}
		if len(it.Sub) > 0 {
			drawMenuArrow(mem, RECT{r.Right - menuArrowW(), r.Top, r.Right, r.Bottom}, hot || open)
		}
	}
	// scroll hints
	if lv.contentH > ht {
		if lv.scroll > 0 {
			drawMenuScrollMark(mem, w/2, sc(4), true)
		}
		if lv.scroll < lv.contentH-ht {
			drawMenuScrollMark(mem, w/2, ht-sc(4), false)
		}
	}
	pSelectClipRgn.Call(mem, 0)
	drawMenuFrame(mem, w, ht)
}

func lvIndex(lv *menuLevel) int {
	for i, l := range menuStack {
		if l == lv {
			return i
		}
	}
	return -1
}

func drawMenuBackground(hdc uintptr, w, h int32) {
	g := newGfx(hdc)
	defer g.Close()
	all := RECT{0, 0, w, h}
	switch skin.Chrome {
	case "classic": // tooltip/dropdown backdrop: deep navy
		g.fillRound(all, 0, argb(rgb(22, 22, 50), 255), argb(rgb(10, 10, 26), 255))
	case "retail":
		g.fillRound(all, 0, argb(rgb(24, 22, 20), 255), argb(rgb(10, 9, 8), 255))
	case "dragon":
		g.fillRound(all, 0, argb(rgb(44, 44, 45), 255), argb(rgb(32, 32, 33), 255))
	default:
		fillRect(hdc, all, lighten(colBg, 6))
	}
}

func drawMenuFrame(hdc uintptr, w, h int32) {
	switch skin.Chrome {
	case "classic":
		g := newGfx(hdc)
		g.strokeRound(RECT{1, 1, w - 2, h - 2}, sc(5), argb(rgb(230, 230, 232), 255), argb(rgb(140, 140, 146), 255), 2)
		g.Close()
		frameRect(hdc, RECT{0, 0, w, h}, black)
	case "retail":
		g := newGfx(hdc)
		g.strokeRound(RECT{1, 1, w - 2, h - 2}, 0, argb(metalHi, 255), argb(metalLo, 255), 2)
		g.Close()
		frameRect(hdc, RECT{0, 0, w, h}, black)
		l := sc(8)
		gc := rgb(206, 166, 72)
		for _, c := range [][2]int32{{0, 0}, {w, 0}, {0, h}, {w, h}} {
			x0, y0 := c[0], c[1]
			dx, dy := int32(1), int32(1)
			if x0 == w {
				dx = -1
			}
			if y0 == h {
				dy = -1
			}
			fillRect(hdc, RECT{min32(x0, x0+dx*l), min32(y0, y0+dy*2), max32(x0, x0+dx*l), max32(y0, y0+dy*2)}, gc)
			fillRect(hdc, RECT{min32(x0, x0+dx*2), min32(y0, y0+dy*l), max32(x0, x0+dx*2), max32(y0, y0+dy*l)}, gc)
		}
	case "dragon":
		frameRect(hdc, RECT{0, 0, w, h}, rgb(8, 8, 8))
		fillRect(hdc, RECT{1, 1, w - 1, 2}, rgb(92, 92, 94))
		fillRect(hdc, RECT{1, 1, 2, h - 1}, rgb(72, 72, 74))
	default:
		if len(skin.Border) > 0 {
			for i, c := range skin.Border {
				frameRect(hdc, RECT{int32(i), int32(i), w - int32(i), h - int32(i)}, c)
			}
		} else if skin.FieldBorder != 0 {
			frameRect(hdc, RECT{0, 0, w, h}, skin.FieldBorder)
		} else {
			frameRect(hdc, RECT{0, 0, w, h}, lighten(colBg, 26))
		}
	}
}

func frameRect(hdc uintptr, r RECT, c uintptr) {
	fillRect(hdc, RECT{r.Left, r.Top, r.Right, r.Top + 1}, c)
	fillRect(hdc, RECT{r.Left, r.Bottom - 1, r.Right, r.Bottom}, c)
	fillRect(hdc, RECT{r.Left, r.Top, r.Left + 1, r.Bottom}, c)
	fillRect(hdc, RECT{r.Right - 1, r.Top, r.Right, r.Bottom}, c)
}

// drawMenuHighlight marks the hovered row the way each UI does.
func drawMenuHighlight(hdc uintptr, r RECT) {
	g := newGfx(hdc)
	defer g.Close()
	switch skin.Chrome {
	case "classic": // WoW dropdown menus: a soft blue-white glow bar
		g.fillRound(r, sc(2), argb(rgb(120, 150, 230), 120), argb(rgb(70, 90, 170), 70))
	case "retail", "dragon": // gold glow like the selected message
		g.fillRound(r, sc(3), argb(rgb(70, 58, 20), 220), argb(rgb(38, 32, 12), 220))
		g.strokeRound(r, sc(3), argb(rgb(240, 204, 96), 200), argb(rgb(160, 120, 36), 200), 1.2)
	default:
		if skin.ID == "chat" {
			g.fillRound(r, 0, argb(rgb(52, 52, 52), 255), argb(rgb(40, 40, 40), 255))
			return
		}
		g.fillRound(r, sc(4), argb(colSelect, 255), argb(colSelect, 255))
		g.fillRound(RECT{r.Left, r.Top + sc(4), r.Left + sc(3), r.Bottom - sc(4)}, 0, argb(colAccent, 255), argb(colAccent, 255))
	}
}

func drawMenuSeparator(hdc uintptr, r RECT) {
	y := (r.Top + r.Bottom) / 2
	c := lighten(colBg, 24)
	switch skin.Chrome {
	case "classic":
		c = rgb(90, 90, 110)
	case "retail":
		c = rgb(96, 76, 40)
	case "dragon":
		c = rgb(70, 70, 72)
	}
	fillRect(hdc, RECT{r.Left + sc(8), y, r.Right - sc(8), y + 1}, c)
}

func drawMenuCheck(hdc uintptr, r RECT, radio bool) {
	g := newGfx(hdc)
	defer g.Close()
	c := colAccent
	cx, cy := (r.Left+r.Right)/2, (r.Top+r.Bottom)/2
	if radio {
		d := sc(4)
		g.fillEllipse(RECT{cx - d, cy - d, cx + d, cy + d}, argb(c, 255), argb(c, 255))
		return
	}
	a := sc(4)
	g.line(cx-a, cy, cx-a/3, cy+a*2/3, argb(c, 255), 2.2)
	g.line(cx-a/3, cy+a*2/3, cx+a, cy-a*2/3, argb(c, 255), 2.2)
}

func drawMenuArrow(hdc uintptr, r RECT, hot bool) {
	g := newGfx(hdc)
	defer g.Close()
	c := colDim
	if hot || skin.Chrome != "plain" {
		c = colAccent
	}
	cx, cy := (r.Left+r.Right)/2, (r.Top+r.Bottom)/2
	a := sc(4)
	g.line(cx-a/2, cy-a, cx+a/2, cy, argb(c, 255), 2)
	g.line(cx+a/2, cy, cx-a/2, cy+a, argb(c, 255), 2)
}

func drawMenuScrollMark(hdc uintptr, cx, y int32, up bool) {
	g := newGfx(hdc)
	defer g.Close()
	a := sc(4)
	d := a
	if up {
		d = -a
	}
	g.line(cx-a, y-d/2, cx, y+d/2, argb(colAccent, 255), 1.8)
	g.line(cx, y+d/2, cx+a, y-d/2, argb(colAccent, 255), 1.8)
}
