package main

// Decoder for the pixel strip painted by the Parley addon.
// See the protocol notes at the top of addon/Parley/Parley.lua.

import (
	"errors"
	"strings"
)

const (
	stripCols   = 64
	stripRows   = 6
	stripCells  = stripCols * stripRows
	stripHeader = 24
)

// Pixels gives access to a captured image in 0..255 RGB.
type Pixels interface {
	At(x, y int) (r, g, b uint8)
	Size() (w, h int)
}

// ChatMessage is one decoded chat line.
type ChatMessage struct {
	Seq      int
	Type     string // W B S Y E P R G O C I
	ChanNum  string
	ChanName string
	Sender   string
	Text     string
}

var (
	errNoMarker    = errors.New("no strip")
	errCalibration = errors.New("colours too distorted")
	errChecksum    = errors.New("checksum mismatch")
)

// FindCellSize returns the cell size (1-4) whose marker is present, or 0.
func FindCellSize(p Pixels, preferred int) int {
	try := []int{preferred, 2, 1, 3, 4}
	for _, cs := range try {
		if cs >= 1 && cs <= 4 && hasMarker(p, cs) {
			return cs
		}
	}
	return 0
}

func sample(p Pixels, cs, cell int) (r, g, b uint8) {
	x := (cell%stripCols)*cs + cs/2
	y := (cell/stripCols)*cs + cs/2
	return p.At(x, y)
}

func hasMarker(p Pixels, cs int) bool {
	w, h := p.Size()
	if w < stripCols*cs || h < stripRows*cs {
		return false
	}
	want := [4][3]bool{{true, false, true}, {false, true, false}, {true, true, false}, {false, false, true}}
	for i, m := range want {
		r, g, b := sample(p, cs, i)
		got := [3]uint8{r, g, b}
		for ch := 0; ch < 3; ch++ {
			if m[ch] && got[ch] < 150 || !m[ch] && got[ch] > 105 {
				return false
			}
		}
	}
	// calibration greys must increase
	var prev int = -1
	for i := 0; i < 16; i++ {
		r, g, b := sample(p, cs, 4+i)
		v := int(r) + int(g) + int(b)
		if v < prev {
			return false
		}
		prev = v
	}
	return true
}

// Decode reads one frame. The caller supplies the cell size from FindCellSize.
func Decode(p Pixels, cs int) (ChatMessage, error) {
	if !hasMarker(p, cs) {
		return ChatMessage{}, errNoMarker
	}
	// Per-channel calibration tables: observed value for each level.
	var cal [3][16]int
	for i := 0; i < 16; i++ {
		r, g, b := sample(p, cs, 4+i)
		cal[0][i], cal[1][i], cal[2][i] = int(r), int(g), int(b)
	}
	for ch := 0; ch < 3; ch++ {
		for i := 1; i < 16; i++ {
			if cal[ch][i]-cal[ch][i-1] < 3 {
				return ChatMessage{}, errCalibration
			}
		}
	}
	level := func(ch int, v uint8) int {
		best, bestD := 0, 1<<30
		for i := 0; i < 16; i++ {
			d := int(v) - cal[ch][i]
			if d < 0 {
				d = -d
			}
			if d < bestD {
				best, bestD = i, d
			}
		}
		return best
	}
	cellLevels := func(cell int) (int, int, int) {
		r, g, b := sample(p, cs, cell)
		return level(0, r), level(1, g), level(2, b)
	}
	read12 := func(cell int) int {
		a, b, c := cellLevels(cell)
		return a<<8 | b<<4 | c
	}
	seq := read12(20)
	n := read12(21)
	ca, cb := read12(22), read12(23)
	if n > (stripCells-stripHeader)*3/2 {
		return ChatMessage{}, errChecksum
	}
	data := make([]byte, n)
	for i := 0; i < n; i++ {
		hi := nibble(cellLevels, stripHeader*3+i*2)
		lo := nibble(cellLevels, stripHeader*3+i*2+1)
		data[i] = byte(hi<<4 | lo)
	}
	a, b := fletcher(data)
	if a != ca || b != cb {
		return ChatMessage{}, errChecksum
	}
	parts := strings.SplitN(string(data), "\x1f", 5)
	if len(parts) != 5 {
		return ChatMessage{}, errChecksum
	}
	return ChatMessage{Seq: seq, Type: parts[0], ChanNum: parts[1], ChanName: parts[2],
		Sender: parts[3], Text: parts[4]}, nil
}

func nibble(cellLevels func(int) (int, int, int), idx int) int {
	r, g, b := cellLevels(idx / 3)
	switch idx % 3 {
	case 0:
		return r
	case 1:
		return g
	}
	return b
}

func fletcher(d []byte) (int, int) {
	a, b := 0, 0
	for _, c := range d {
		a = (a + int(c)) % 4095
		b = (b + a) % 4095
	}
	return a, b
}
