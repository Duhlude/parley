package main

import "unsafe"

// Skin "chrome": frames, title bars, buttons, fields and selection
// highlights, drawn with GDI+ to echo WoW's own UI:
//   classic - tooltip/dialog style: navy panel, grey rounded edge, red
//             buttons with gold trim, centred gold title.
//   retail  - modern retail windows: metal frame with gold corner pieces,
//             dark title strip with a centred gold title, round gold-ringed
//             portrait, red close box, red-and-gold buttons, gold glow on
//             the selected row.
//   dragon  - Dragonflight UI kit: charcoal panels, bevelled slot fields,
//             gold-ringed portrait with a name plate, dark bevelled buttons
//             with gold text and icons.
// Everything is drawn by Parley; no Blizzard art is used.

var (
	gold      = rgb(255, 209, 0)
	goldLight = rgb(248, 226, 140)
	goldDark  = rgb(140, 104, 34)
	metalHi   = rgb(170, 170, 172)
	metalLo   = rgb(62, 62, 64)
	black     = rgb(0, 0, 0)
)

// headerLeft is where header text may start (after the portrait).
func headerLeft() int32 {
	if skin.Portrait {
		return portraitRect().Right + sc(8)
	}
	if resIcon(1, 32) != 0 {
		return headerLogoRect().Right + sc(7)
	}
	return sc(10)
}

// headerLogoRect is the small logo before the title in skins without a portrait.
func headerLogoRect() RECT {
	s := sc(20)
	top := (sc(skin.HeaderH) - s) / 2
	return RECT{sc(9), top, sc(9) + s, top + s}
}

func portraitRect() RECT {
	d := sc(skin.HeaderH) - sc(8)
	return RECT{sc(5), sc(4), sc(5) + d, sc(4) + d}
}

// drawHeaderBg paints the title bar area.
func drawHeaderBg(hdc uintptr, w, headH int32) {
	g := newGfx(hdc)
	defer g.Close()
	switch skin.Chrome {
	case "classic":
		g.fillRound(RECT{0, 0, w, headH}, 0, argb(rgb(30, 30, 62), 255), argb(rgb(12, 12, 28), 255))
		fillRect(hdc, RECT{sc(8), headH - 1, w - sc(8), headH}, rgb(120, 120, 130))
	case "retail":
		g.fillRound(RECT{0, 0, w, headH}, 0, argb(rgb(22, 22, 23), 255), argb(rgb(12, 12, 13), 255))
	case "dragon":
		g.fillRound(RECT{0, 0, w, headH}, 0, argb(rgb(54, 54, 55), 255), argb(rgb(40, 40, 41), 255))
	default:
		fillRect(hdc, RECT{0, 0, w, headH}, colHeader)
	}
}

// drawHeaderButton paints a title-bar icon button (settings, clear, hide).
// The hide button is drawn as WoW's red close box in the WoW skins.
func drawHeaderButton(hdc uintptr, r RECT, glyph string, hover, closeBox bool) {
	if skin.Chrome == "plain" {
		if hover {
			boxBorder(hdc, r, sc(skin.Radius), colSelect, 0)
			pSetTextColor.Call(hdc, colAccent)
		} else {
			pSetTextColor.Call(hdc, colDim)
		}
		pSelectObject.Call(hdc, fIcon)
		br := r
		drawText(hdc, glyph, &br, DT_SINGLELINE|DT_VCENTER|DT_CENTER)
		return
	}
	// square button centred in r
	s := min32(r.Right-r.Left, r.Bottom-r.Top) - sc(4)
	cx, cy := (r.Left+r.Right)/2, (r.Top+r.Bottom)/2
	b := RECT{cx - s/2, cy - s/2, cx + s/2, cy + s/2}
	g := newGfx(hdc)
	if closeBox {
		top, bot := rgb(176, 26, 18), rgb(96, 8, 6)
		if hover {
			top, bot = rgb(214, 44, 30), rgb(128, 14, 10)
		}
		g.fillRound(b, sc(3), argb(top, 255), argb(bot, 255))
		g.strokeRound(b, sc(3), argb(goldLight, 255), argb(goldDark, 255), 1.5)
		m := s / 4
		g.line(b.Left+m, b.Top+m, b.Right-m, b.Bottom-m, argb(rgb(250, 232, 190), 255), 2)
		g.line(b.Right-m, b.Top+m, b.Left+m, b.Bottom-m, argb(rgb(250, 232, 190), 255), 2)
		g.Close()
		return
	}
	switch skin.Chrome {
	case "dragon":
		top, bot := rgb(58, 58, 60), rgb(34, 34, 35)
		if hover {
			top, bot = rgb(74, 74, 76), rgb(44, 44, 45)
		}
		g.fillRound(b, sc(4), argb(top, 255), argb(bot, 255))
		g.strokeRound(b, sc(4), argb(rgb(110, 110, 112), 255), argb(rgb(24, 24, 24), 255), 1)
	default: // classic, retail: small metal plate
		top, bot := rgb(70, 70, 72), rgb(26, 26, 27)
		if hover {
			top, bot = rgb(92, 92, 94), rgb(36, 36, 37)
		}
		g.fillRound(b, sc(2), argb(top, 255), argb(bot, 255))
		g.strokeRound(b, sc(2), argb(metalHi, 255), argb(metalLo, 255), 1)
	}
	g.Close()
	c := goldDark
	if hover {
		c = gold
	} else {
		c = rgb(214, 176, 70)
	}
	pSelectObject.Call(hdc, fIcon)
	pSetTextColor.Call(hdc, c)
	br := b
	drawText(hdc, glyph, &br, DT_SINGLELINE|DT_VCENTER|DT_CENTER)
}

// drawButton paints a text button; returns the text colour to use.
func drawButton(hdc uintptr, r RECT, hot bool) uintptr {
	switch skin.Chrome {
	case "classic", "retail":
		g := newGfx(hdc)
		top, bot := rgb(150, 18, 12), rgb(78, 6, 4)
		if hot {
			top, bot = rgb(188, 34, 22), rgb(104, 12, 8)
		}
		g.fillRound(r, sc(3), argb(top, 255), argb(bot, 255))
		g.strokeRound(r, sc(3), argb(goldLight, 255), argb(goldDark, 255), 1.4)
		g.line(r.Left+sc(4), r.Top+sc(2), r.Right-sc(4), r.Top+sc(2), argb(rgb(255, 150, 120), 70), 1)
		g.Close()
		return gold
	case "dragon":
		g := newGfx(hdc)
		top, bot := rgb(56, 56, 58), rgb(32, 32, 33)
		if hot {
			top, bot = rgb(70, 70, 72), rgb(40, 40, 41)
		}
		g.fillRound(r, sc(5), argb(top, 255), argb(bot, 255))
		g.strokeRound(r, sc(5), argb(rgb(112, 112, 114), 255), argb(rgb(20, 20, 20), 255), 1.2)
		g.Close()
		return gold
	}
	fill := skin.Button
	if hot {
		fill = lighten(fill, 28)
	}
	boxBorder(hdc, r, sc(skin.Radius), fill, skin.ButtonBorder)
	return skin.ButtonText
}

// drawField paints a text-entry well.
func drawField(hdc uintptr, r RECT) {
	switch skin.Chrome {
	case "retail":
		g := newGfx(hdc)
		g.fillRound(r, sc(3), argb(rgb(2, 2, 2), 255), argb(rgb(10, 10, 10), 255))
		g.strokeRound(r, sc(3), argb(rgb(24, 24, 24), 255), argb(rgb(92, 92, 94), 255), 1.2)
		g.Close()
		return
	case "dragon":
		g := newGfx(hdc)
		g.fillRound(r, sc(6), argb(rgb(22, 22, 23), 255), argb(rgb(30, 30, 31), 255))
		g.strokeRound(r, sc(6), argb(rgb(10, 10, 10), 255), argb(rgb(84, 84, 86), 255), 1.6)
		g.Close()
		return
	case "classic":
		g := newGfx(hdc)
		g.fillRound(r, sc(3), argb(black, 255), argb(black, 255))
		g.strokeRound(r, sc(3), argb(rgb(128, 128, 136), 255), 0, 1)
		g.Close()
		return
	}
	boxBorder(hdc, r, sc(skin.Radius), skin.Field, skin.FieldBorder)
}

// drawSelection highlights the message you're replying to.
func drawSelection(hdc uintptr, r RECT) {
	switch skin.Chrome {
	case "retail", "dragon":
		g := newGfx(hdc)
		g.fillRound(r, sc(4), argb(rgb(52, 44, 16), 200), argb(rgb(30, 26, 10), 200))
		g.strokeRound(r, sc(4), argb(rgb(255, 222, 110), 255), argb(rgb(200, 150, 40), 255), 1.6)
		g.Close()
		return
	}
	fillRect(hdc, r, colSelect)
	fillRect(hdc, RECT{r.Left, r.Top, r.Left + sc(3), r.Bottom}, colAccent)
}

// drawFrameBorder draws the outer frame and portrait on top of everything.
func drawFrameBorder(hdc uintptr, w, h int32) {
	switch skin.Chrome {
	case "classic":
		g := newGfx(hdc)
		outer := RECT{1, 1, w - 2, h - 2}
		g.strokeRound(outer, sc(7), argb(rgb(230, 230, 232), 255), argb(rgb(150, 150, 156), 255), 2.5)
		g.strokeRound(RECT{3, 3, w - 4, h - 4}, sc(6), argb(rgb(70, 70, 80), 255), 0, 1)
		g.Close()
		fillRect(hdc, RECT{0, 0, w, 1}, black)
		fillRect(hdc, RECT{0, h - 1, w, h}, black)
		fillRect(hdc, RECT{0, 0, 1, h}, black)
		fillRect(hdc, RECT{w - 1, 0, w, h}, black)
	case "retail":
		g := newGfx(hdc)
		g.strokeRound(RECT{1, 1, w - 2, h - 2}, 0, argb(metalHi, 255), argb(metalLo, 255), 3)
		g.Close()
		fillRect(hdc, RECT{0, 0, w, 1}, black)
		fillRect(hdc, RECT{0, h - 1, w, h}, black)
		fillRect(hdc, RECT{0, 0, 1, h}, black)
		fillRect(hdc, RECT{w - 1, 0, w, h}, black)
		// inner shadow line
		fillRect(hdc, RECT{3, 3, w - 3, 4}, black)
		fillRect(hdc, RECT{3, h - 4, w - 3, h - 3}, black)
		fillRect(hdc, RECT{3, 3, 4, h - 3}, black)
		fillRect(hdc, RECT{w - 4, 3, w - 3, h - 3}, black)
		// gold corner pieces
		l, t := sc(12), int32(3)
		for _, c := range [][2]int32{{0, 0}, {w, 0}, {0, h}, {w, h}} {
			x0, y0 := c[0], c[1]
			dx, dy := int32(1), int32(1)
			if x0 == w {
				dx = -1
			}
			if y0 == h {
				dy = -1
			}
			hx := RECT{min32(x0, x0+dx*l), min32(y0, y0+dy*t), max32(x0, x0+dx*l), max32(y0, y0+dy*t)}
			vx := RECT{min32(x0, x0+dx*t), min32(y0, y0+dy*l), max32(x0, x0+dx*t), max32(y0, y0+dy*l)}
			fillRect(hdc, hx, rgb(206, 166, 72))
			fillRect(hdc, vx, rgb(206, 166, 72))
		}
	case "dragon":
		fillRect(hdc, RECT{0, 0, w, 1}, rgb(8, 8, 8))
		fillRect(hdc, RECT{0, h - 1, w, h}, rgb(8, 8, 8))
		fillRect(hdc, RECT{0, 0, 1, h}, rgb(8, 8, 8))
		fillRect(hdc, RECT{w - 1, 0, w, h}, rgb(8, 8, 8))
		fillRect(hdc, RECT{1, 1, w - 1, 2}, rgb(84, 84, 86))
		fillRect(hdc, RECT{1, 1, 2, h - 1}, rgb(70, 70, 72))
	default:
		for i, c := range skin.Border {
			o := int32(i)
			fillRect(hdc, RECT{o, o, w - o, o + 1}, c)
			fillRect(hdc, RECT{o, h - o - 1, w - o, h - o}, c)
			fillRect(hdc, RECT{o, o, o + 1, h - o}, c)
			fillRect(hdc, RECT{w - o - 1, o, w - o, h - o}, c)
		}
	}
	if skin.Portrait {
		drawPortrait(hdc)
	}
}

// drawPortrait draws Parley's badge in a gold ring, like a unit portrait.
// Dragonflight uses the unit-frame "teardrop" shape (square bottom-right).
func drawPortrait(hdc uintptr) {
	g := newGfx(hdc)
	defer g.Close()
	r := portraitRect()
	d := r.Right - r.Left
	tear := skin.Chrome == "dragon"
	shape := func(rr RECT) uintptr {
		var p uintptr
		pGdipCreatePath.Call(0, uintptr(unsafe.Pointer(&p)))
		x, y, dd := rr.Left, rr.Top, rr.Right-rr.Left
		if tear {
			pGdipAddPathArcI.Call(p, uintptr(x), uintptr(y), uintptr(dd), uintptr(dd), f32(90), f32(270))
			pGdipAddPathLineI.Call(p, uintptr(x+dd), uintptr(y+dd/2), uintptr(x+dd), uintptr(y+dd))
			pGdipAddPathLineI.Call(p, uintptr(x+dd), uintptr(y+dd), uintptr(x+dd/2), uintptr(y+dd))
		} else {
			pGdipAddPathArcI.Call(p, uintptr(x), uintptr(y), uintptr(dd), uintptr(dd), f32(0), f32(360))
		}
		pGdipClosePathFigure.Call(p)
		return p
	}
	fillPath := func(p uintptr, rr RECT, c1, c2 uintptr) {
		b := g.vgrad(rr.Top, rr.Bottom, c1, c2)
		pGdipFillPath.Call(g.g, b, p)
		pGdipDeleteBrush.Call(b)
	}
	strokePath := func(p uintptr, rr RECT, c1, c2 uintptr, wd float32) {
		b := g.vgrad(rr.Top, rr.Bottom, c1, c2)
		pen := g.brushPen(b, wd)
		pGdipDrawPath.Call(g.g, pen, p)
		pGdipDeletePen.Call(pen)
		pGdipDeleteBrush.Call(b)
	}
	outer := shape(RECT{r.Left - 1, r.Top - 1, r.Right + 1, r.Bottom + 1})
	fillPath(outer, r, argb(black, 220), argb(black, 220))
	pGdipDeletePath.Call(outer)
	p := shape(r)
	fillPath(p, r, argb(rgb(40, 60, 98), 255), argb(rgb(10, 16, 32), 255))
	// the logo's speech bubbles (falls back to a drawn bubble)
	in := d * 14 / 100
	if drawIcon(hdc, 2, RECT{r.Left + in, r.Top + in, r.Right - in, r.Bottom - in}) {
		goto ring
	}
	{
		bw, bh := d*52/100, d*38/100
		bx, by := r.Left+(d-bw)/2, r.Top+d*24/100
		bub := RECT{bx, by, bx + bw, by + bh}
		g.fillRound(bub, bh/3, argb(rgb(250, 236, 196), 255), argb(rgb(214, 188, 128), 255))
		tail := RECT{bx + bw/5, by + bh - 2, bx + bw/5 + bw/5, by + bh + d/9}
		g.fillRound(tail, 1, argb(rgb(214, 188, 128), 255), argb(rgb(214, 188, 128), 255))
		dot := d / 14
		if dot < 2 {
			dot = 2
		}
		for i := int32(-1); i <= 1; i++ {
			cx := bx + bw/2 + i*bw/4
			cy := by + bh/2
			g.fillEllipse(RECT{cx - dot, cy - dot, cx + dot, cy + dot}, argb(rgb(60, 40, 16), 255), argb(rgb(60, 40, 16), 255))
		}
	}
ring:
	// gold ring
	ring := sc(3)
	if tear {
		ring = sc(4)
	}
	strokePath(p, r, argb(rgb(255, 234, 150), 255), argb(rgb(146, 106, 30), 255), float32(ring))
	pGdipDeletePath.Call(p)
	if tear { // small gold bracket inside the square corner
		c := RECT{r.Right - d/5, r.Bottom - d/5, r.Right - sc(4), r.Bottom - sc(4)}
		g.line(c.Left, c.Bottom, c.Right, c.Bottom, argb(rgb(236, 200, 110), 255), 1.5)
		g.line(c.Right, c.Top, c.Right, c.Bottom, argb(rgb(236, 200, 110), 255), 1.5)
	}
}
