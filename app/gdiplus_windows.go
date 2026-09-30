package main

// Tiny GDI+ wrapper for anti-aliased shapes and gradients used by the skins.
// (Floats are passed as their bit patterns; Go's Windows call path copies
// the first four arguments into XMM registers, and later ones go on the
// stack, which is what the GDI+ flat API expects for REAL parameters.)

import (
	"math"
	"syscall"
	"unsafe"
)

var (
	gdiplus                  = syscall.NewLazyDLL("gdiplus.dll")
	pGdiplusStartup          = gdiplus.NewProc("GdiplusStartup")
	pGdipCreateFromHDC       = gdiplus.NewProc("GdipCreateFromHDC")
	pGdipDeleteGraphics      = gdiplus.NewProc("GdipDeleteGraphics")
	pGdipSetSmoothingMode    = gdiplus.NewProc("GdipSetSmoothingMode")
	pGdipSetPixelOffsetMode  = gdiplus.NewProc("GdipSetPixelOffsetMode")
	pGdipCreateSolidFill     = gdiplus.NewProc("GdipCreateSolidFill")
	pGdipCreateLineBrushI    = gdiplus.NewProc("GdipCreateLineBrushI")
	pGdipDeleteBrush         = gdiplus.NewProc("GdipDeleteBrush")
	pGdipCreatePen1          = gdiplus.NewProc("GdipCreatePen1")
	pGdipCreatePen2          = gdiplus.NewProc("GdipCreatePen2")
	pGdipDeletePen           = gdiplus.NewProc("GdipDeletePen")
	pGdipCreatePath          = gdiplus.NewProc("GdipCreatePath")
	pGdipDeletePath          = gdiplus.NewProc("GdipDeletePath")
	pGdipAddPathArcI         = gdiplus.NewProc("GdipAddPathArcI")
	pGdipAddPathLineI        = gdiplus.NewProc("GdipAddPathLineI")
	pGdipClosePathFigure     = gdiplus.NewProc("GdipClosePathFigure")
	pGdipFillPath            = gdiplus.NewProc("GdipFillPath")
	pGdipDrawPath            = gdiplus.NewProc("GdipDrawPath")
	pGdipFillEllipseI        = gdiplus.NewProc("GdipFillEllipseI")
	pGdipDrawEllipseI        = gdiplus.NewProc("GdipDrawEllipseI")
	pGdipDrawLineI           = gdiplus.NewProc("GdipDrawLineI")
	pGdipFillRectangleI      = gdiplus.NewProc("GdipFillRectangleI")
	pGdipSetPenLineCap197819 = gdiplus.NewProc("GdipSetPenLineCap197819")

	gpReady bool
)

func f32(v float32) uintptr { return uintptr(math.Float32bits(v)) }

func gpStart() bool {
	if gpReady {
		return true
	}
	if pGdiplusStartup.Find() != nil {
		return false
	}
	in := struct {
		Version  uint32
		_        uint32
		Callback uintptr
		NoThread int32
		NoCodecs int32
	}{Version: 1}
	var token uintptr
	if r, _, _ := pGdiplusStartup.Call(uintptr(unsafe.Pointer(&token)), uintptr(unsafe.Pointer(&in)), 0); r != 0 {
		return false
	}
	gpReady = true
	return true
}

// argb builds a GDI+ colour from a GDI COLORREF plus alpha.
func argb(c uintptr, a uint8) uintptr {
	r, g, b := c&0xff, (c>>8)&0xff, (c>>16)&0xff
	return uintptr(a)<<24 | r<<16 | g<<8 | b
}

type gfx struct{ g uintptr }

func newGfx(hdc uintptr) *gfx {
	if !gpStart() {
		return nil
	}
	var g uintptr
	if r, _, _ := pGdipCreateFromHDC.Call(hdc, uintptr(unsafe.Pointer(&g))); r != 0 {
		return nil
	}
	pGdipSetSmoothingMode.Call(g, 4)   // anti-alias
	pGdipSetPixelOffsetMode.Call(g, 4) // half pixel, crisper edges
	return &gfx{g}
}

func (x *gfx) Close() {
	if x != nil {
		pGdipDeleteGraphics.Call(x.g)
	}
}

func (x *gfx) solid(c uintptr) uintptr {
	var b uintptr
	pGdipCreateSolidFill.Call(c, uintptr(unsafe.Pointer(&b)))
	return b
}

// vgrad is a vertical gradient brush from top colour to bottom colour.
func (x *gfx) vgrad(top, bottom int32, c1, c2 uintptr) uintptr {
	p1 := [2]int32{0, top - 1}
	p2 := [2]int32{0, bottom + 1}
	var b uintptr
	pGdipCreateLineBrushI.Call(uintptr(unsafe.Pointer(&p1)), uintptr(unsafe.Pointer(&p2)), c1, c2, 0, uintptr(unsafe.Pointer(&b)))
	return b
}

func (x *gfx) pen(c uintptr, w float32) uintptr {
	var p uintptr
	pGdipCreatePen1.Call(c, f32(w), 2, uintptr(unsafe.Pointer(&p)))
	return p
}

func (x *gfx) brushPen(brush uintptr, w float32) uintptr {
	var p uintptr
	pGdipCreatePen2.Call(brush, f32(w), 2, uintptr(unsafe.Pointer(&p)))
	return p
}

func roundPath(r RECT, rad int32) uintptr {
	var p uintptr
	pGdipCreatePath.Call(0, uintptr(unsafe.Pointer(&p)))
	x, y, w, h := r.Left, r.Top, r.Right-r.Left, r.Bottom-r.Top
	if rad <= 0 {
		pGdipAddPathLineI.Call(p, uintptr(x), uintptr(y), uintptr(x+w), uintptr(y))
		pGdipAddPathLineI.Call(p, uintptr(x+w), uintptr(y), uintptr(x+w), uintptr(y+h))
		pGdipAddPathLineI.Call(p, uintptr(x+w), uintptr(y+h), uintptr(x), uintptr(y+h))
		pGdipClosePathFigure.Call(p)
		return p
	}
	d := rad * 2
	if d > w {
		d = w
	}
	if d > h {
		d = h
	}
	arc := func(ax, ay int32, start float32) {
		pGdipAddPathArcI.Call(p, uintptr(ax), uintptr(ay), uintptr(d), uintptr(d), f32(start), f32(90))
	}
	arc(x, y, 180)
	arc(x+w-d, y, 270)
	arc(x+w-d, y+h-d, 0)
	arc(x, y+h-d, 90)
	pGdipClosePathFigure.Call(p)
	return p
}

// fillRound fills a rounded rect with a vertical gradient (c1 top, c2 bottom).
func (x *gfx) fillRound(r RECT, rad int32, c1, c2 uintptr) {
	if x == nil {
		return
	}
	p := roundPath(r, rad)
	b := x.vgrad(r.Top, r.Bottom, c1, c2)
	pGdipFillPath.Call(x.g, b, p)
	pGdipDeleteBrush.Call(b)
	pGdipDeletePath.Call(p)
}

// strokeRound outlines a rounded rect; c2 != 0 makes it a vertical gradient.
func (x *gfx) strokeRound(r RECT, rad int32, c1, c2 uintptr, w float32) {
	if x == nil {
		return
	}
	p := roundPath(r, rad)
	var pen uintptr
	if c2 != 0 {
		b := x.vgrad(r.Top, r.Bottom, c1, c2)
		pen = x.brushPen(b, w)
		pGdipDeleteBrush.Call(b)
	} else {
		pen = x.pen(c1, w)
	}
	pGdipDrawPath.Call(x.g, pen, p)
	pGdipDeletePen.Call(pen)
	pGdipDeletePath.Call(p)
}

func (x *gfx) fillEllipse(r RECT, c1, c2 uintptr) {
	if x == nil {
		return
	}
	b := x.vgrad(r.Top, r.Bottom, c1, c2)
	pGdipFillEllipseI.Call(x.g, b, uintptr(r.Left), uintptr(r.Top), uintptr(r.Right-r.Left), uintptr(r.Bottom-r.Top))
	pGdipDeleteBrush.Call(b)
}

func (x *gfx) strokeEllipse(r RECT, c1, c2 uintptr, w float32) {
	if x == nil {
		return
	}
	b := x.vgrad(r.Top, r.Bottom, c1, c2)
	pen := x.brushPen(b, w)
	pGdipDrawEllipseI.Call(x.g, pen, uintptr(r.Left), uintptr(r.Top), uintptr(r.Right-r.Left), uintptr(r.Bottom-r.Top))
	pGdipDeletePen.Call(pen)
	pGdipDeleteBrush.Call(b)
}

func (x *gfx) line(x1, y1, x2, y2 int32, c uintptr, w float32) {
	if x == nil {
		return
	}
	pen := x.pen(c, w)
	pGdipSetPenLineCap197819.Call(pen, 2, 2, 0) // round caps
	pGdipDrawLineI.Call(x.g, pen, uintptr(x1), uintptr(y1), uintptr(x2), uintptr(y2))
	pGdipDeletePen.Call(pen)
}
