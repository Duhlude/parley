package main

// skinAlert is a skinned replacement for MessageBox: a small panel in the
// current skin with a title, the message and an OK button.

import (
	"strings"
	"syscall"
	"unsafe"
)

var (
	alertClassDone bool
	alertOpen      bool
	alertTitle     string
	alertText      string
	alertOK        RECT
	alertHot       bool
)

func skinAlert(owner uintptr, title, text string) {
	if alertOpen {
		return
	}
	inst, _, _ := pGetModuleHandleW.Call(0)
	if !alertClassDone {
		cur, _, _ := pLoadCursorW.Call(0, IDC_ARROW)
		wc := WNDCLASSEX{Style: 0x00020000 /*CS_DROPSHADOW*/, LpfnWndProc: syscall.NewCallback(alertProc),
			HInstance: inst, HCursor: cur, LpszClassName: u16("ParleyAlert"), HIcon: app.icon, HIconSm: app.iconSmall}
		wc.CbSize = uint32(unsafe.Sizeof(wc))
		pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
		alertClassDone = true
	}
	makeMenuFonts()
	alertTitle, alertText = title, text

	// size to the text
	w := sc(380)
	hdc, _, _ := pGetDC.Call(0)
	pSelectObject.Call(hdc, menuFont)
	tr := RECT{0, 0, w - sc(40), 0}
	drawText(hdc, text, &tr, DT_CALCRECT|DT_WORDBREAK|DT_NOPREFIX|DT_EDITCONTROL)
	pReleaseDC.Call(0, hdc)
	h := sc(20) + sc(26) + sc(10) + tr.Bottom + sc(20) + sc(32) + sc(18)

	// centre on the owner (or the screen)
	var cx, cy int32
	if owner != 0 {
		or := windowRect(owner)
		cx, cy = (or.Left+or.Right)/2, (or.Top+or.Bottom)/2
	} else {
		sw, _, _ := pGetSystemMetrics.Call(0)
		shh, _, _ := pGetSystemMetrics.Call(1)
		cx, cy = int32(sw)/2, int32(shh)/2
	}
	wa := workAreaAt(cx, cy)
	x := max32(wa.Left, min32(cx-w/2, wa.Right-w))
	y := max32(wa.Top, min32(cy-h/2, wa.Bottom-h))
	hw, _, _ := pCreateWindowExW.Call(WS_EX_TOPMOST|WS_EX_TOOLWINDOW, uintptr(unsafe.Pointer(u16("ParleyAlert"))),
		u16p(title), WS_POPUP, uintptr(x), uintptr(y), uintptr(w), uintptr(h), owner, 0, inst, 0)
	pref := skin.Corner
	if pDwmSetWindowAttribute.Find() == nil {
		pDwmSetWindowAttribute.Call(hw, 33, uintptr(unsafe.Pointer(&pref)), 4)
	}
	bw := sc(110)
	alertOK = RECT{(w - bw) / 2, h - sc(18) - sc(32), (w + bw) / 2, h - sc(18)}
	alertHot = false
	if owner != 0 {
		pEnableWindow.Call(owner, 0)
	}
	alertOpen = true
	pShowWindow.Call(hw, SW_SHOW)
	pSetForegroundWindow.Call(hw)
	var m MSG
	for alertOpen {
		r, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			pPostQuitMessage.Call(0)
			break
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
	if owner != 0 {
		pEnableWindow.Call(owner, 1)
		pSetForegroundWindow.Call(owner)
	}
	pDestroyWindow.Call(hw)
}

func alertProc(h, msg, wp, lp uintptr) uintptr {
	switch msg {
	case WM_ERASEBKGND:
		return 1
	case WM_PAINT:
		var ps PAINTSTRUCT
		hdc, _, _ := pBeginPaint.Call(h, uintptr(unsafe.Pointer(&ps)))
		cr := clientRect(h)
		w, ht := cr.Right, cr.Bottom
		mem, _, _ := pCreateCompatibleDC.Call(hdc)
		bmp, _, _ := pCreateCompatibleBitmap.Call(hdc, uintptr(w), uintptr(ht))
		old, _, _ := pSelectObject.Call(mem, bmp)
		pSetBkMode.Call(mem, TRANSPARENT)
		drawMenuBackground(mem, w, ht)
		// logo + title
		ix := sc(20)
		if drawIcon(mem, 1, RECT{ix, sc(18), ix + sc(24), sc(42)}) {
			ix += sc(32)
		}
		pSelectObject.Call(mem, menuFontTitle)
		pSetTextColor.Call(mem, skin.Title)
		tr := RECT{ix, sc(18), w - sc(20), sc(42)}
		drawText(mem, alertTitle, &tr, DT_SINGLELINE|DT_VCENTER|DT_NOPREFIX|DT_END_ELLIPSIS)
		pSelectObject.Call(mem, menuFont)
		pSetTextColor.Call(mem, colText)
		br := RECT{sc(20), sc(56), w - sc(20), alertOK.Top - sc(10)}
		drawText(mem, alertText, &br, DT_WORDBREAK|DT_NOPREFIX|DT_EDITCONTROL)
		txt := drawButton(mem, alertOK, alertHot)
		pSetTextColor.Call(mem, txt)
		ok := alertOK
		drawText(mem, "OK", &ok, DT_SINGLELINE|DT_VCENTER|DT_CENTER|DT_NOPREFIX)
		drawMenuFrame(mem, w, ht)
		pBitBlt.Call(hdc, 0, 0, uintptr(w), uintptr(ht), mem, 0, 0, SRCCOPY)
		pSelectObject.Call(mem, old)
		pDeleteObject.Call(bmp)
		pDeleteDC.Call(mem)
		pEndPaint.Call(h, uintptr(unsafe.Pointer(&ps)))
		return 0
	case wmMouseMove:
		hot := inRect(alertOK, loword(lp), hiword(lp))
		if hot != alertHot {
			alertHot = hot
			invalidate(h)
		}
		return 0
	case WM_LBUTTONUP:
		if inRect(alertOK, loword(lp), hiword(lp)) {
			alertOpen = false
		}
		return 0
	case wmLButtonDn: // drag the panel by its body
		if !inRect(alertOK, loword(lp), hiword(lp)) {
			pReleaseCapture.Call()
			pSendMessageW.Call(h, 0x00A1 /*WM_NCLBUTTONDOWN*/, 2 /*HTCAPTION*/, 0)
		}
		return 0
	case WM_KEYDOWN:
		if wp == VK_RETURN || wp == VK_ESCAPE || wp == 0x20 {
			alertOpen = false
		}
		return 0
	case WM_CLOSE:
		alertOpen = false
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(h, msg, wp, lp)
	return r
}

// alertLines joins message lines the way MessageBox text used to read.
func alertLines(lines ...string) string { return strings.Join(lines, "\n") }
