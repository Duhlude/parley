package main

// Retail / Dragonflight title areas, message rows and dropdowns.

func wowChrome() bool { return skin.Chrome == "retail" || skin.Chrome == "dragon" }

// drawWowHeader lays out and paints the Retail / Dragonflight title area
// and sets the title-bar button rectangles.
func drawWowHeader(hdc uintptr, w, headH int32, status string, kind statusKind, dot uintptr) {
	g := newGfx(hdc)
	pr := portraitRect()
	switch skin.Chrome {
	case "retail":
		// title strip, like the top bar of retail windows
		strip := RECT{pr.Left + (pr.Right-pr.Left)/2, sc(4), w - sc(5), sc(27)}
		g.fillRound(strip, sc(2), argb(rgb(58, 58, 60), 255), argb(rgb(22, 22, 23), 255))
		g.strokeRound(strip, sc(2), argb(rgb(128, 128, 130), 255), argb(rgb(34, 34, 34), 255), 1)
		g.Close()
		bs := sc(19)
		by := strip.Top + (strip.Bottom-strip.Top-bs)/2
		btnHide = RECT{strip.Right - bs - sc(3), by, strip.Right - sc(3), by + bs}
		btnClear = RECT{btnHide.Left - bs - sc(3), by, btnHide.Left - sc(3), by + bs}
		btnSettings = RECT{btnClear.Left - bs - sc(3), by, btnClear.Left - sc(3), by + bs}
		pSelectObject.Call(hdc, fBold)
		pSetTextColor.Call(hdc, gold)
		tr := RECT{pr.Right, strip.Top, btnSettings.Left, strip.Bottom}
		drawText(hdc, "Parley", &tr, DT_SINGLELINE|DT_VCENTER|DT_CENTER|DT_NOPREFIX)
		// status line under the strip, like "Level 13 Arms Warrior"
		sr := RECT{pr.Right + sc(6), strip.Bottom + sc(2), w - sc(10), headH - sc(3)}
		drawStatusLine(hdc, sr, status, dot, true)
		fillRect(hdc, RECT{sc(4), headH - 2, w - sc(4), headH - 1}, black)
		fillRect(hdc, RECT{sc(4), headH - 1, w - sc(4), headH}, rgb(60, 60, 62))
	case "dragon":
		// name plate next to the portrait, like the player unit frame
		plate := RECT{pr.Right - sc(14), sc(7), w - sc(8), sc(28)}
		g.fillRound(plate, sc(3), argb(rgb(26, 26, 27), 255), argb(rgb(14, 14, 15), 255))
		g.strokeRound(plate, sc(3), argb(rgb(112, 112, 114), 255), argb(rgb(40, 40, 40), 255), 1.2)
		bs := sc(20)
		by := plate.Bottom + sc(4)
		btnHide = RECT{w - sc(8) - bs, by, w - sc(8), by + bs}
		btnClear = RECT{btnHide.Left - bs - sc(4), by, btnHide.Left - sc(4), by + bs}
		btnSettings = RECT{btnClear.Left - bs - sc(4), by, btnClear.Left - sc(4), by + bs}
		// status bar, like the health bar: green when live
		bar := RECT{pr.Right - sc(14), by + sc(4), btnSettings.Left - sc(8), by + bs - sc(4)}
		top, bot := rgb(110, 110, 112), rgb(56, 56, 58)
		if kind == statusLive {
			top, bot = rgb(120, 230, 70), rgb(40, 150, 30)
		}
		if kind == statusWarn || dot == colWarn {
			top, bot = rgb(240, 190, 70), rgb(160, 100, 20)
		}
		g.fillRound(RECT{bar.Left - 1, bar.Top - 1, bar.Right + 1, bar.Bottom + 1}, sc(2), argb(rgb(8, 8, 8), 255), argb(rgb(8, 8, 8), 255))
		g.fillRound(bar, sc(2), argb(top, 255), argb(bot, 255))
		g.strokeRound(RECT{bar.Left - 2, bar.Top - 2, bar.Right + 2, bar.Bottom + 2}, sc(3), argb(rgb(100, 100, 102), 255), argb(rgb(30, 30, 30), 255), 1)
		g.Close()
		pSelectObject.Call(hdc, fTitle)
		pSetTextColor.Call(hdc, gold)
		tr := RECT{pr.Right + sc(6), plate.Top, plate.Right - sc(8), plate.Bottom}
		drawText(hdc, "Parley", &tr, DT_SINGLELINE|DT_VCENTER|DT_NOPREFIX)
		// status on the right of the plate, like the level number
		pSelectObject.Call(hdc, fSmall)
		pSetTextColor.Call(hdc, rgb(236, 236, 236))
		sr := RECT{pr.Right + sc(80), plate.Top, plate.Right - sc(8), plate.Bottom}
		drawText(hdc, status, &sr, DT_SINGLELINE|DT_VCENTER|DT_RIGHT|DT_NOPREFIX|DT_END_ELLIPSIS)
	default:
		g.Close()
	}
	for i, b := range []struct {
		r     RECT
		glyph string
	}{{btnSettings, ""}, {btnClear, ""}, {btnHide, ""}} {
		drawHeaderButton(hdc, b.r, b.glyph, hoverBtn == i+1, i == 2)
	}
}

func drawStatusLine(hdc uintptr, r RECT, status string, dot uintptr, centred bool) {
	pSelectObject.Call(hdc, fSmall)
	mr := RECT{0, 0, r.Right - r.Left, 100}
	drawText(hdc, status, &mr, DT_SINGLELINE|DT_CALCRECT|DT_NOPREFIX)
	d := sc(4)
	tw := min32(mr.Right+2*d+sc(6), r.Right-r.Left)
	x := r.Left
	if centred {
		x = r.Left + (r.Right-r.Left-tw)/2
	}
	cy := (r.Top + r.Bottom) / 2
	g := newGfx(hdc)
	g.fillEllipse(RECT{x, cy - d, x + 2*d, cy + d}, argb(lighten(dot, 40), 255), argb(dot, 255))
	g.Close()
	c := gold
	if dot == colWarn {
		c = rgb(255, 170, 60)
	}
	pSetTextColor.Call(hdc, c)
	tr := RECT{x + 2*d + sc(6), r.Top, r.Right, r.Bottom}
	drawText(hdc, status, &tr, DT_SINGLELINE|DT_VCENTER|DT_NOPREFIX|DT_END_ELLIPSIS)
}

// drawRow gives each message its own panel, like WoW list entries.
func drawRow(hdc uintptr, r RECT) {
	g := newGfx(hdc)
	defer g.Close()
	switch skin.Chrome {
	case "retail":
		g.fillRound(r, sc(3), argb(rgb(26, 26, 27), 255), argb(rgb(12, 12, 13), 255))
		g.strokeRound(r, sc(3), argb(rgb(60, 60, 62), 255), argb(rgb(30, 30, 31), 255), 1)
	case "dragon":
		g.fillRound(r, sc(5), argb(rgb(46, 46, 47), 255), argb(rgb(34, 34, 35), 255))
		g.strokeRound(r, sc(5), argb(rgb(98, 98, 100), 255), argb(rgb(18, 18, 18), 255), 1.2)
	}
}

// drawDropdown paints the reply-language picker like a WoW dropdown:
// a dark field with an arrow button. Returns the text colour.
func drawDropdown(hdc uintptr, r RECT, hot bool) uintptr {
	g := newGfx(hdc)
	defer g.Close()
	fillTop, fillBot := rgb(8, 8, 8), rgb(20, 20, 21)
	if hot {
		fillTop, fillBot = rgb(26, 23, 12), rgb(36, 32, 18)
	}
	rad := sc(3)
	if skin.Chrome == "dragon" {
		rad = sc(5)
	}
	g.fillRound(r, rad, argb(fillTop, 255), argb(fillBot, 255))
	g.strokeRound(r, rad, argb(rgb(24, 24, 24), 255), argb(rgb(104, 104, 106), 255), 1.2)
	s := r.Bottom - r.Top - sc(8)
	ab := RECT{r.Right - s - sc(4), r.Top + sc(4), r.Right - sc(4), r.Bottom - sc(4)}
	if skin.Chrome == "retail" {
		g.fillRound(ab, sc(2), argb(rgb(150, 18, 12), 255), argb(rgb(78, 6, 4), 255))
		g.strokeRound(ab, sc(2), argb(goldLight, 255), argb(goldDark, 255), 1)
	} else {
		g.fillRound(ab, sc(3), argb(rgb(58, 58, 60), 255), argb(rgb(32, 32, 33), 255))
		g.strokeRound(ab, sc(3), argb(rgb(110, 110, 112), 255), argb(rgb(20, 20, 20), 255), 1)
	}
	cx, cy := (ab.Left+ab.Right)/2, (ab.Top+ab.Bottom)/2+sc(1)
	a := s / 4
	g.line(cx-a, cy-a/2, cx, cy+a/2, argb(gold, 255), 2)
	g.line(cx, cy+a/2, cx+a, cy-a/2, argb(gold, 255), 2)
	return gold
}
