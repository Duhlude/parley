package main

// Minimal Win32 bindings (standard library only, no cgo).

import (
	"strings"
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	crypt32  = syscall.NewLazyDLL("crypt32.dll")
	uxtheme  = syscall.NewLazyDLL("uxtheme.dll")
	dwmapi   = syscall.NewLazyDLL("dwmapi.dll")

	pRegisterClassExW              = user32.NewProc("RegisterClassExW")
	pCreateWindowExW               = user32.NewProc("CreateWindowExW")
	pDefWindowProcW                = user32.NewProc("DefWindowProcW")
	pGetMessageW                   = user32.NewProc("GetMessageW")
	pTranslateMessage              = user32.NewProc("TranslateMessage")
	pDispatchMessageW              = user32.NewProc("DispatchMessageW")
	pIsDialogMessageW              = user32.NewProc("IsDialogMessageW")
	pPostQuitMessage               = user32.NewProc("PostQuitMessage")
	pShowWindow                    = user32.NewProc("ShowWindow")
	pInvalidateRect                = user32.NewProc("InvalidateRect")
	pBeginPaint                    = user32.NewProc("BeginPaint")
	pEndPaint                      = user32.NewProc("EndPaint")
	pGetClientRect                 = user32.NewProc("GetClientRect")
	pGetWindowRect                 = user32.NewProc("GetWindowRect")
	pSetWindowPos                  = user32.NewProc("SetWindowPos")
	pMoveWindow                    = user32.NewProc("MoveWindow")
	pSetLayeredWindowAttributes    = user32.NewProc("SetLayeredWindowAttributes")
	pLoadCursorW                   = user32.NewProc("LoadCursorW")
	pGetDC                         = user32.NewProc("GetDC")
	pReleaseDC                     = user32.NewProc("ReleaseDC")
	pEnumWindows                   = user32.NewProc("EnumWindows")
	pGetWindowThreadProcessId      = user32.NewProc("GetWindowThreadProcessId")
	pClientToScreen                = user32.NewProc("ClientToScreen")
	pScreenToClient                = user32.NewProc("ScreenToClient")
	pIsWindowVisible               = user32.NewProc("IsWindowVisible")
	pIsWindow                      = user32.NewProc("IsWindow")
	pIsIconic                      = user32.NewProc("IsIconic")
	pGetForegroundWindow           = user32.NewProc("GetForegroundWindow")
	pSetForegroundWindow           = user32.NewProc("SetForegroundWindow")
	pRegisterHotKey                = user32.NewProc("RegisterHotKey")
	pSendMessageW                  = user32.NewProc("SendMessageW")
	pPostMessageW                  = user32.NewProc("PostMessageW")
	pSetTimer                      = user32.NewProc("SetTimer")
	pSetWindowTextW                = user32.NewProc("SetWindowTextW")
	pGetWindowTextW                = user32.NewProc("GetWindowTextW")
	pGetWindowTextLengthW          = user32.NewProc("GetWindowTextLengthW")
	pSetFocus                      = user32.NewProc("SetFocus")
	pGetFocus                      = user32.NewProc("GetFocus")
	pCreatePopupMenu               = user32.NewProc("CreatePopupMenu")
	pAppendMenuW                   = user32.NewProc("AppendMenuW")
	pTrackPopupMenu                = user32.NewProc("TrackPopupMenu")
	pDestroyMenu                   = user32.NewProc("DestroyMenu")
	pGetCursorPos                  = user32.NewProc("GetCursorPos")
	pOpenClipboard                 = user32.NewProc("OpenClipboard")
	pEmptyClipboard                = user32.NewProc("EmptyClipboard")
	pSetClipboardData              = user32.NewProc("SetClipboardData")
	pCloseClipboard                = user32.NewProc("CloseClipboard")
	pDestroyWindow                 = user32.NewProc("DestroyWindow")
	pSetWindowLongPtrW             = user32.NewProc("SetWindowLongPtrW")
	pGetWindowLongPtrW             = user32.NewProc("GetWindowLongPtrW")
	pCallWindowProcW               = user32.NewProc("CallWindowProcW")
	pMessageBoxW                   = user32.NewProc("MessageBoxW")
	pCreateIconIndirect            = user32.NewProc("CreateIconIndirect")
	pFillRect                      = user32.NewProc("FillRect")
	pDrawTextW                     = user32.NewProc("DrawTextW")
	pSetProcessDpiAwarenessContext = user32.NewProc("SetProcessDpiAwarenessContext")
	pGetDpiForWindow               = user32.NewProc("GetDpiForWindow")
	pEnableWindow                  = user32.NewProc("EnableWindow")
	pGetSystemMetrics              = user32.NewProc("GetSystemMetrics")
	pTrackMouseEvent               = user32.NewProc("TrackMouseEvent")

	pCreateCompatibleDC     = gdi32.NewProc("CreateCompatibleDC")
	pCreateCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	pCreateDIBSection       = gdi32.NewProc("CreateDIBSection")
	pCreateBitmap           = gdi32.NewProc("CreateBitmap")
	pSelectObject           = gdi32.NewProc("SelectObject")
	pBitBlt                 = gdi32.NewProc("BitBlt")
	pDeleteObject           = gdi32.NewProc("DeleteObject")
	pDeleteDC               = gdi32.NewProc("DeleteDC")
	pCreateSolidBrush       = gdi32.NewProc("CreateSolidBrush")
	pCreatePen              = gdi32.NewProc("CreatePen")
	pCreateFontW            = gdi32.NewProc("CreateFontW")
	pSetTextColor           = gdi32.NewProc("SetTextColor")
	pSetBkMode              = gdi32.NewProc("SetBkMode")
	pSetBkColor             = gdi32.NewProc("SetBkColor")
	pRoundRect              = gdi32.NewProc("RoundRect")
	pGetStockObject         = gdi32.NewProc("GetStockObject")

	pGetModuleHandleW           = kernel32.NewProc("GetModuleHandleW")
	pOpenProcess                = kernel32.NewProc("OpenProcess")
	pQueryFullProcessImageNameW = kernel32.NewProc("QueryFullProcessImageNameW")
	pCloseHandle                = kernel32.NewProc("CloseHandle")
	pGlobalAlloc                = kernel32.NewProc("GlobalAlloc")
	pGlobalLock                 = kernel32.NewProc("GlobalLock")
	pGlobalUnlock               = kernel32.NewProc("GlobalUnlock")
	pLocalFree                  = kernel32.NewProc("LocalFree")
	pCreateMutexW               = kernel32.NewProc("CreateMutexW")

	pShellNotifyIconW = shell32.NewProc("Shell_NotifyIconW")
	pShellExecuteW    = shell32.NewProc("ShellExecuteW")

	pCryptProtectData   = crypt32.NewProc("CryptProtectData")
	pCryptUnprotectData = crypt32.NewProc("CryptUnprotectData")

	pSetWindowTheme        = uxtheme.NewProc("SetWindowTheme")
	pDwmSetWindowAttribute = dwmapi.NewProc("DwmSetWindowAttribute")
)

const (
	WS_POPUP        = 0x80000000
	WS_CHILD        = 0x40000000
	WS_VISIBLE      = 0x10000000
	WS_CAPTION      = 0x00C00000
	WS_SYSMENU      = 0x00080000
	WS_TABSTOP      = 0x00010000
	WS_VSCROLL      = 0x00200000
	WS_CLIPCHILDREN = 0x02000000

	WS_EX_TOPMOST       = 0x00000008
	WS_EX_TOOLWINDOW    = 0x00000080
	WS_EX_LAYERED       = 0x00080000
	WS_EX_CONTROLPARENT = 0x00010000

	ES_AUTOHSCROLL = 0x0080
	ES_PASSWORD    = 0x0020
	ES_NUMBER      = 0x2000

	BS_AUTOCHECKBOX  = 0x0003
	BS_OWNERDRAW     = 0x000B
	CBS_DROPDOWNLIST = 0x0003
	CBS_HASSTRINGS   = 0x0200

	WM_CREATE          = 0x0001
	WM_DESTROY         = 0x0002
	WM_SIZE            = 0x0005
	WM_ACTIVATE        = 0x0006
	WM_SETFOCUS        = 0x0007
	WM_CLOSE           = 0x0010
	WM_PAINT           = 0x000F
	WM_ERASEBKGND      = 0x0014
	WM_SETCURSOR       = 0x0020
	WM_MOUSEACTIVATE   = 0x0021
	WM_GETMINMAXINFO   = 0x0024
	WM_DRAWITEM        = 0x002B
	WM_SETFONT         = 0x0030
	WM_NCHITTEST       = 0x0084
	WM_KEYDOWN         = 0x0100
	WM_CHAR            = 0x0102
	WM_COMMAND         = 0x0111
	WM_TIMER           = 0x0113
	WM_CTLCOLOREDIT    = 0x0133
	WM_CTLCOLORLISTBOX = 0x0134
	WM_CTLCOLORBTN     = 0x0135
	WM_CTLCOLORSTATIC  = 0x0138
	WM_MOUSEMOVE       = 0x0200
	WM_LBUTTONDOWN     = 0x0201
	WM_LBUTTONUP       = 0x0202
	WM_RBUTTONUP       = 0x0205
	WM_MOUSEWHEEL      = 0x020A
	WM_MOUSELEAVE      = 0x02A3
	WM_HOTKEY          = 0x0312
	WM_DPICHANGED      = 0x02E0
	WM_APP             = 0x8000

	EM_SETSEL    = 0x00B1
	EM_LIMITTEXT = 0x00C5
	BM_GETCHECK  = 0x00F0
	BM_SETCHECK  = 0x00F1
	CB_ADDSTRING = 0x0143
	CB_GETCURSEL = 0x0147
	CB_SETCURSEL = 0x014E

	SW_HIDE           = 0
	SW_SHOW           = 5
	SW_SHOWNOACTIVATE = 4

	HTCLIENT      = 1
	HTCAPTION     = 2
	HTLEFT        = 10
	HTRIGHT       = 11
	HTTOP         = 12
	HTTOPLEFT     = 13
	HTTOPRIGHT    = 14
	HTBOTTOM      = 15
	HTBOTTOMLEFT  = 16
	HTBOTTOMRIGHT = 17

	MA_ACTIVATE   = 1
	MA_NOACTIVATE = 3

	SWP_NOSIZE     = 0x0001
	SWP_NOMOVE     = 0x0002
	SWP_NOZORDER   = 0x0004
	SWP_NOACTIVATE = 0x0010
	SWP_SHOWWINDOW = 0x0040

	LWA_ALPHA = 0x2

	DT_LEFT         = 0x0000
	DT_CENTER       = 0x0001
	DT_RIGHT        = 0x0002
	DT_VCENTER      = 0x0004
	DT_SINGLELINE   = 0x0020
	DT_WORDBREAK    = 0x0010
	DT_CALCRECT     = 0x0400
	DT_NOPREFIX     = 0x0800
	DT_END_ELLIPSIS = 0x8000
	DT_EDITCONTROL  = 0x2000

	TRANSPARENT = 1
	SRCCOPY     = 0x00CC0020
	CAPTUREBLT  = 0x40000000

	NIM_ADD     = 0
	NIM_MODIFY  = 1
	NIM_DELETE  = 2
	NIF_MESSAGE = 0x1
	NIF_ICON    = 0x2
	NIF_TIP     = 0x4
	NIF_INFO    = 0x10

	MF_STRING       = 0x0
	MF_SEPARATOR    = 0x800
	MF_CHECKED      = 0x8
	TPM_RETURNCMD   = 0x100
	TPM_RIGHTBUTTON = 0x2

	MOD_CONTROL  = 0x2
	MOD_SHIFT    = 0x4
	MOD_NOREPEAT = 0x4000

	VK_RETURN = 0x0D
	VK_ESCAPE = 0x1B
	VK_TAB    = 0x09

	CF_UNICODETEXT = 13
	GMEM_MOVEABLE  = 0x2

	IDC_ARROW = 32512
	IDC_HAND  = 32649
	IDC_IBEAM = 32513

	GWLP_WNDPROC = -4

	BN_CLICKED   = 0
	ODS_SELECTED = 0x1

	MB_OK              = 0
	MB_ICONINFORMATION = 0x40
	MB_ICONWARNING     = 0x30
)

type RECT struct{ Left, Top, Right, Bottom int32 }
type POINT struct{ X, Y int32 }

type MSG struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
	private uint32
}

type WNDCLASSEX struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type PAINTSTRUCT struct {
	Hdc         uintptr
	FErase      int32
	RcPaint     RECT
	FRestore    int32
	FIncUpdate  int32
	RgbReserved [32]byte
}

type BITMAPINFOHEADER struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}

type ICONINFO struct {
	FIcon    int32
	XHotspot uint32
	YHotspot uint32
	HbmMask  uintptr
	HbmColor uintptr
}

type NOTIFYICONDATA struct {
	CbSize           uint32
	HWnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            uintptr
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	UVersion         uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
	GuidItem         [16]byte
	HBalloonIcon     uintptr
}

type DRAWITEMSTRUCT struct {
	CtlType    uint32
	CtlID      uint32
	ItemID     uint32
	ItemAction uint32
	ItemState  uint32
	HwndItem   uintptr
	HDC        uintptr
	RcItem     RECT
	ItemData   uintptr
}

type MINMAXINFO struct {
	PtReserved, PtMaxSize, PtMaxPosition, PtMinTrackSize, PtMaxTrackSize POINT
}

type TRACKMOUSEEVENT struct {
	CbSize      uint32
	DwFlags     uint32
	HwndTrack   uintptr
	DwHoverTime uint32
}

type DATA_BLOB struct {
	CbData uint32
	PbData *byte
}

func u16(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

// keepAlive holds recent string buffers so the GC can't free them while a
// Win32 call is still reading them through a uintptr.
var (
	keepAlive    [256]*uint16
	keepAliveIdx int
)

func u16p(s string) uintptr {
	p := u16(s)
	keepAlive[keepAliveIdx%len(keepAlive)] = p
	keepAliveIdx++
	return uintptr(unsafe.Pointer(p))
}

func rgb(r, g, b uint8) uintptr { return uintptr(r) | uintptr(g)<<8 | uintptr(b)<<16 }

func loword(v uintptr) int32 { return int32(int16(v & 0xffff)) }
func hiword(v uintptr) int32 { return int32(int16((v >> 16) & 0xffff)) }

func getText(h uintptr) string {
	n, _, _ := pGetWindowTextLengthW.Call(h)
	buf := make([]uint16, n+1)
	pGetWindowTextW.Call(h, uintptr(unsafe.Pointer(&buf[0])), n+1)
	return syscall.UTF16ToString(buf)
}

func setText(h uintptr, s string) { pSetWindowTextW.Call(h, u16p(s)) }

func invalidate(h uintptr) { pInvalidateRect.Call(h, 0, 0) }

func clientRect(h uintptr) RECT {
	var r RECT
	pGetClientRect.Call(h, uintptr(unsafe.Pointer(&r)))
	return r
}

func windowRect(h uintptr) RECT {
	var r RECT
	pGetWindowRect.Call(h, uintptr(unsafe.Pointer(&r)))
	return r
}

func dpiOf(h uintptr) int {
	if pGetDpiForWindow.Find() == nil {
		if d, _, _ := pGetDpiForWindow.Call(h); d > 0 {
			return int(d)
		}
	}
	return 96
}

func newFont(px int, weight int, face string) uintptr {
	f, _, _ := pCreateFontW.Call(uintptr(int32(-px)), 0, 0, 0, uintptr(weight), 0, 0, 0,
		1 /*DEFAULT_CHARSET*/, 0, 0, 5 /*CLEARTYPE_QUALITY*/, 0, u16p(face))
	return f
}

func drawText(hdc uintptr, s string, r *RECT, flags uintptr) int32 {
	if strings.ContainsRune(s, '→') && !hasGlyph(hdc, '→') {
		s = strings.ReplaceAll(s, "→", "»") // Friz Quadrata (the WoW skins) has no arrow
	}
	p, _ := syscall.UTF16FromString(s)
	h, _, _ := pDrawTextW.Call(hdc, uintptr(unsafe.Pointer(&p[0])), uintptr(len(p)-1),
		uintptr(unsafe.Pointer(r)), flags)
	return int32(h)
}

var (
	pGetGlyphIndicesW = gdi32.NewProc("GetGlyphIndicesW")
	glyphCache        = map[string]bool{}
)

// hasGlyph reports whether the font selected into hdc can draw r (cached per font).
func hasGlyph(hdc uintptr, r rune) bool {
	var face [64]uint16 // key on the face name: font handles get reused
	n, _, _ := pGetTextFaceW.Call(hdc, uintptr(len(face)), uintptr(unsafe.Pointer(&face[0])))
	if int(n) > len(face) {
		n = uintptr(len(face))
	}
	k := syscall.UTF16ToString(face[:n]) + string(r)
	if v, ok := glyphCache[k]; ok {
		return v
	}
	in := []uint16{uint16(r)}
	out := []uint16{0}
	pGetGlyphIndicesW.Call(hdc, uintptr(unsafe.Pointer(&in[0])), 1, uintptr(unsafe.Pointer(&out[0])), 1 /*GGI_MARK_NONEXISTING_GLYPHS*/)
	ok := out[0] != 0xFFFF
	glyphCache[k] = ok
	return ok
}

func fillRect(hdc uintptr, r RECT, color uintptr) {
	br, _, _ := pCreateSolidBrush.Call(color)
	pFillRect.Call(hdc, uintptr(unsafe.Pointer(&r)), br)
	pDeleteObject.Call(br)
}

func roundRect(hdc uintptr, r RECT, radius int32, fill uintptr) {
	br, _, _ := pCreateSolidBrush.Call(fill)
	pen, _, _ := pCreatePen.Call(0, 1, fill)
	ob, _, _ := pSelectObject.Call(hdc, br)
	op, _, _ := pSelectObject.Call(hdc, pen)
	pRoundRect.Call(hdc, uintptr(r.Left), uintptr(r.Top), uintptr(r.Right), uintptr(r.Bottom),
		uintptr(radius), uintptr(radius))
	pSelectObject.Call(hdc, ob)
	pSelectObject.Call(hdc, op)
	pDeleteObject.Call(br)
	pDeleteObject.Call(pen)
}

func msgBox(owner uintptr, text, title string, flags uintptr) {
	pMessageBoxW.Call(owner, u16p(text), u16p(title), flags)
}

func openURL(url string) {
	pShellExecuteW.Call(0, u16p("open"), u16p(url), 0, 0, SW_SHOW)
}

func setClipboard(owner uintptr, s string) bool {
	p, _ := syscall.UTF16FromString(s)
	for i := 0; i < 10; i++ {
		if r, _, _ := pOpenClipboard.Call(owner); r != 0 {
			pEmptyClipboard.Call()
			size := uintptr(len(p) * 2)
			h, _, _ := pGlobalAlloc.Call(GMEM_MOVEABLE, size)
			if h == 0 {
				pCloseClipboard.Call()
				return false
			}
			dst, _, _ := pGlobalLock.Call(h)
			copy(unsafe.Slice((*uint16)(unsafe.Pointer(dst)), len(p)), p)
			pGlobalUnlock.Call(h)
			pSetClipboardData.Call(CF_UNICODETEXT, h)
			pCloseClipboard.Call()
			return true
		}
		sleepMs(20)
	}
	return false
}

// DPAPI: encrypt secrets to the current Windows user.
func protect(plain []byte) []byte {
	if len(plain) == 0 {
		return nil
	}
	in := DATA_BLOB{uint32(len(plain)), &plain[0]}
	var out DATA_BLOB
	if r, _, _ := pCryptProtectData.Call(uintptr(unsafe.Pointer(&in)), 0, 0, 0, 0, 0,
		uintptr(unsafe.Pointer(&out))); r == 0 {
		return nil
	}
	defer pLocalFree.Call(uintptr(unsafe.Pointer(out.PbData)))
	return append([]byte(nil), unsafe.Slice(out.PbData, out.CbData)...)
}

func unprotect(enc []byte) []byte {
	if len(enc) == 0 {
		return nil
	}
	in := DATA_BLOB{uint32(len(enc)), &enc[0]}
	var out DATA_BLOB
	if r, _, _ := pCryptUnprotectData.Call(uintptr(unsafe.Pointer(&in)), 0, 0, 0, 0, 0,
		uintptr(unsafe.Pointer(&out))); r == 0 {
		return nil
	}
	defer pLocalFree.Call(uintptr(unsafe.Pointer(out.PbData)))
	return append([]byte(nil), unsafe.Slice(out.PbData, out.CbData)...)
}

// Dark title bar for normal windows (Windows 10 20H1+ / 11).
func darkTitle(h uintptr) {
	if pDwmSetWindowAttribute.Find() != nil {
		return
	}
	on := int32(1)
	pDwmSetWindowAttribute.Call(h, 20, uintptr(unsafe.Pointer(&on)), 4)
}
