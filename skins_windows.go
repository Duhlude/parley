package main

// Skins: colour/font/border sets for the overlay and settings window.
//
// The looks are drawn by Parley itself (GDI), modelled on WoW's own frames:
//   - Classic: the GameTooltip/dialog backdrop (dark navy, 0.09/0.09/0.19),
//     grey bevelled edge, gold "normal font colour" (1, 0.82, 0) titles,
//     red UIPanelButtons with gold text, Arial Narrow chat text.
//   - Retail: Dragonflight-era panels: near-black stone, bronze/gold
//     double border with corner studs, gold headers, red-and-gold buttons.
//   - Chat frame: WoW's plain chat window, black and minimal.
// Blizzard's textures and fonts aren't redistributed. If WoW's font files
// (FRIZQT__.TTF, ARIALN.TTF...) are dropped into Parley's "fonts" folder
// they're used; otherwise the closest Windows fonts stand in.

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

type Skin struct {
	ID, Label string

	Bg, Header, Field, Text, Dim, Accent, Select, Title uintptr
	FieldBorder                                         uintptr // 0 = none
	Button, ButtonText, ButtonBorder                    uintptr // ButtonBorder 0 = none
	Border                                              []uintptr
	Studs                                               uintptr // corner ornaments, 0 = none
	Corner                                              int32   // DWM corner preference
	Radius                                              int

	Chrome      string // plain, classic, retail, dragon
	HeaderH     int
	Portrait    bool
	TitleCenter bool

	TitleFaces, BodyFaces, SmallFaces []string
	TitleWeight, HeadWeight           int
	BodyBoost                         int // extra px for narrow fonts
}

var skins = []Skin{
	{
		ID: "parley", Label: "Parley (modern)",
		Bg: rgb(18, 22, 28), Header: rgb(24, 29, 37), Field: rgb(32, 38, 48),
		Text: rgb(228, 234, 240), Dim: rgb(128, 139, 152), Accent: rgb(0x60, 0xCD, 0xFF),
		Select: rgb(28, 40, 52), Title: rgb(0x60, 0xCD, 0xFF),
		Button: rgb(32, 38, 48), ButtonText: rgb(0x60, 0xCD, 0xFF),
		Corner: 2, Radius: 8,
		TitleFaces: []string{"Segoe UI"}, BodyFaces: []string{"Segoe UI"}, SmallFaces: []string{"Segoe UI"},
		TitleWeight: 700, HeadWeight: 600,
	},
	{
		ID: "classic", Label: "Classic WoW", Chrome: "classic", HeaderH: 34, TitleCenter: true,
		Bg: rgb(23, 23, 48), Header: rgb(14, 14, 30), Field: rgb(0, 0, 0),
		FieldBorder: rgb(96, 96, 104),
		Text:        rgb(255, 255, 255), Dim: rgb(170, 170, 170), Accent: rgb(255, 209, 0),
		Select: rgb(44, 44, 80), Title: rgb(255, 209, 0),
		Button: rgb(120, 14, 8), ButtonText: rgb(255, 209, 0), ButtonBorder: rgb(40, 6, 4),
		Border: []uintptr{rgb(12, 12, 12), rgb(210, 210, 210), rgb(150, 150, 150), rgb(70, 70, 70)},
		Corner: 3, Radius: 3,
		TitleFaces:  []string{"Friz Quadrata TT", "Friz Quadrata", "Palatino Linotype", "Georgia"},
		BodyFaces:   []string{"Friz Quadrata TT", "Friz Quadrata", "Arial Narrow", "Arial"},
		SmallFaces:  []string{"Friz Quadrata TT", "Friz Quadrata", "Arial Narrow", "Arial"},
		TitleWeight: 400, HeadWeight: 400, BodyBoost: 0,
	},
	{
		ID: "retail", Label: "Retail WoW (windows)", Chrome: "retail", HeaderH: 62, Portrait: true, TitleCenter: true,
		Bg: rgb(17, 16, 15), Header: rgb(28, 24, 20), Field: rgb(6, 6, 6),
		FieldBorder: rgb(98, 78, 46),
		Text:        rgb(238, 232, 220), Dim: rgb(156, 146, 128), Accent: rgb(255, 209, 0),
		Select: rgb(44, 36, 24), Title: rgb(255, 209, 0),
		Button: rgb(104, 14, 12), ButtonText: rgb(255, 214, 90), ButtonBorder: rgb(186, 146, 76),
		Border: []uintptr{rgb(0, 0, 0), rgb(196, 156, 84), rgb(120, 92, 48), rgb(12, 10, 8)},
		Studs:  rgb(226, 188, 104),
		Corner: 1, Radius: 2,
		TitleFaces:  []string{"Friz Quadrata TT", "Friz Quadrata", "Palatino Linotype", "Georgia"},
		BodyFaces:   []string{"Friz Quadrata TT", "Friz Quadrata", "Arial Narrow", "Arial"},
		SmallFaces:  []string{"Friz Quadrata TT", "Friz Quadrata", "Arial Narrow", "Arial"},
		TitleWeight: 400, HeadWeight: 400, BodyBoost: 0,
	},
	{
		ID: "dragonflight", Label: "Dragonflight UI", Chrome: "dragon", HeaderH: 60, Portrait: true,
		Bg: rgb(38, 38, 39), Header: rgb(33, 33, 34), Field: rgb(26, 26, 27),
		Text: rgb(236, 236, 236), Dim: rgb(158, 156, 150), Accent: rgb(255, 209, 0),
		Select: rgb(48, 44, 32), Title: rgb(255, 209, 0),
		Button: rgb(46, 46, 47), ButtonText: rgb(255, 209, 0), ButtonBorder: rgb(90, 90, 90),
		Corner: 3, Radius: 6,
		TitleFaces:  []string{"Friz Quadrata TT", "Friz Quadrata", "Palatino Linotype", "Georgia"},
		BodyFaces:   []string{"Friz Quadrata TT", "Friz Quadrata", "Arial Narrow", "Arial"},
		SmallFaces:  []string{"Friz Quadrata TT", "Friz Quadrata", "Arial Narrow", "Arial"},
		TitleWeight: 400, HeadWeight: 400, BodyBoost: 0,
	},
	{
		ID: "chat", Label: "WoW chat frame (minimal)",
		Bg: rgb(0, 0, 0), Header: rgb(0, 0, 0), Field: rgb(16, 16, 16),
		FieldBorder: rgb(60, 60, 60),
		Text:        rgb(255, 255, 255), Dim: rgb(150, 150, 150), Accent: rgb(255, 209, 0),
		Select: rgb(34, 34, 34), Title: rgb(255, 209, 0),
		Button: rgb(24, 24, 24), ButtonText: rgb(255, 209, 0), ButtonBorder: rgb(60, 60, 60),
		Corner: 3, Radius: 3,
		TitleFaces:  []string{"Friz Quadrata TT", "Friz Quadrata", "Arial Narrow", "Arial"},
		BodyFaces:   []string{"Friz Quadrata TT", "Friz Quadrata", "Arial Narrow", "Arial"},
		SmallFaces:  []string{"Friz Quadrata TT", "Friz Quadrata", "Arial Narrow", "Arial"},
		TitleWeight: 400, HeadWeight: 400, BodyBoost: 0,
	},
}

func init() {
	for i := range skins {
		sk := &skins[i]
		if sk.HeaderH == 0 {
			sk.HeaderH = 32
		}
		if sk.Chrome == "" {
			sk.Chrome = "plain"
		}
	}
}

var skin = skins[0]

func skinByID(id string) Skin {
	for _, s := range skins {
		if s.ID == id {
			return s
		}
	}
	return skins[0]
}

// applySkin switches colours and fonts everywhere and repaints.
func applySkin(id string) {
	skin = skinByID(id)
	colBg, colHeader, colField = skin.Bg, skin.Header, skin.Field
	colText, colDim, colAccent, colSelect = skin.Text, skin.Dim, skin.Accent, skin.Select
	if editBrush != 0 {
		pDeleteObject.Call(editBrush)
	}
	editBrush, _, _ = pCreateSolidBrush.Call(colField)
	if setBrushField != 0 {
		pDeleteObject.Call(setBrushField)
		pDeleteObject.Call(setBrushBg)
		setBrushBg, _, _ = pCreateSolidBrush.Call(colBg)
		setBrushField, _, _ = pCreateSolidBrush.Call(colField)
	}
	if app.overlay != 0 {
		pref := skin.Corner
		if pDwmSetWindowAttribute.Find() == nil {
			pDwmSetWindowAttribute.Call(app.overlay, 33, uintptr(unsafe.Pointer(&pref)), 4)
		}
		updateDPI()
		invalidate(app.overlay)
		invalidate(overlayEdit)
	}
	if settingsHwnd != 0 {
		invalidateAll(settingsHwnd)
	}
}

var pRedrawWindow = user32.NewProc("RedrawWindow")

func invalidateAll(h uintptr) {
	pRedrawWindow.Call(h, 0, 0, 0x1|0x4|0x80|0x100) // INVALIDATE|ERASE|ALLCHILDREN|UPDATENOW
}

// ---------------------------------------------------------------------------
// Fonts
// ---------------------------------------------------------------------------

var (
	pAddFontResourceExW = gdi32.NewProc("AddFontResourceExW")
	pGetTextFaceW       = gdi32.NewProc("GetTextFaceW")
	fontsLoaded         bool
)

// loadExtraFonts makes WoW's own font files usable if the player provides
// them: <Parley folder>\fonts\*.ttf or <WoW folder>\Fonts\*.ttf.
func loadExtraFonts() {
	if fontsLoaded {
		return
	}
	fontsLoaded = true
	var dirs []string
	if exe, err := os.Executable(); err == nil {
		dirs = append(dirs, filepath.Join(filepath.Dir(exe), "fonts"))
	}
	dirs = append(dirs, filepath.Join(app.cfg.WowPath, "Fonts"))
	for _, d := range dirs {
		ents, err := os.ReadDir(d)
		if err != nil {
			continue
		}
		for _, e := range ents {
			n := strings.ToLower(e.Name())
			if strings.HasSuffix(n, ".ttf") || strings.HasSuffix(n, ".otf") {
				pAddFontResourceExW.Call(u16p(filepath.Join(d, e.Name())), 0x10 /*FR_PRIVATE*/, 0)
			}
		}
	}
}

var faceCache = map[string]string{}

// pickFace returns the first installed font family from the list.
func pickFace(faces []string) string {
	key := strings.Join(faces, "|")
	if f, ok := faceCache[key]; ok {
		return f
	}
	dc, _, _ := pCreateCompatibleDC.Call(0)
	defer pDeleteDC.Call(dc)
	for _, want := range faces {
		f := newFont(14, 400, want)
		old, _, _ := pSelectObject.Call(dc, f)
		buf := make([]uint16, 64)
		pGetTextFaceW.Call(dc, 64, uintptr(unsafe.Pointer(&buf[0])))
		pSelectObject.Call(dc, old)
		pDeleteObject.Call(f)
		if strings.EqualFold(syscall.UTF16ToString(buf), want) {
			faceCache[key] = want
			return want
		}
	}
	faceCache[key] = faces[len(faces)-1]
	return faces[len(faces)-1]
}

// ---------------------------------------------------------------------------
// Skinned drawing helpers
// ---------------------------------------------------------------------------

func boxBorder(hdc uintptr, r RECT, radius int32, fill, border uintptr) {
	br, _, _ := pCreateSolidBrush.Call(fill)
	pc := fill
	if border != 0 {
		pc = border
	}
	pen, _, _ := pCreatePen.Call(0, 1, pc)
	ob, _, _ := pSelectObject.Call(hdc, br)
	op, _, _ := pSelectObject.Call(hdc, pen)
	pRoundRect.Call(hdc, uintptr(r.Left), uintptr(r.Top), uintptr(r.Right), uintptr(r.Bottom),
		uintptr(radius), uintptr(radius))
	pSelectObject.Call(hdc, ob)
	pSelectObject.Call(hdc, op)
	pDeleteObject.Call(br)
	pDeleteObject.Call(pen)
}

func lighten(c uintptr, d int) uintptr {
	f := func(v uintptr) uint8 {
		n := int(v&0xff) + d
		if n > 255 {
			n = 255
		}
		return uint8(n)
	}
	return rgb(f(c), f(c>>8), f(c>>16))
}

func abs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

func borderWidth() int32 { return int32(len(skin.Border)) }

// setSkin switches skin and remembers it.
func setSkin(id string) {
	app.mu.Lock()
	app.cfg.Skin = id
	cfg := app.cfg
	app.mu.Unlock()
	saveConfig(cfg)
	applySkin(id)
}
