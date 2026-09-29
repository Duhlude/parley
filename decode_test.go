package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"testing"
)

type ppm struct {
	w, h int
	px   []byte
}

func (p *ppm) At(x, y int) (uint8, uint8, uint8) {
	i := (y*p.w + x) * 3
	return p.px[i], p.px[i+1], p.px[i+2]
}
func (p *ppm) Size() (int, int) { return p.w, p.h }

func loadPPM(t *testing.T, path string) *ppm {
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	r := bufio.NewReader(f)
	var w, h, max int
	var magic string
	if _, err := fmt.Fscan(r, &magic, &w, &h, &max); err != nil {
		t.Fatal(err)
	}
	r.ReadByte()
	px := make([]byte, w*h*3)
	if _, err := io_ReadFull(r, px); err != nil {
		t.Fatal(err)
	}
	return &ppm{w, h, px}
}

func io_ReadFull(r *bufio.Reader, b []byte) (int, error) {
	n := 0
	for n < len(b) {
		m, err := r.Read(b[n:])
		n += m
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func TestDecodeFromLuaEncoder(t *testing.T) {
	data, err := os.ReadFile("test/out/cases.txt")
	if err != nil {
		t.Skip("run: cd test && lua5.1 gen.lua out")
	}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var n, cs, seq int
		var hx string
		fmt.Sscan(line, &n, &cs, &seq, &hx)
		want, _ := hex.DecodeString(hx)
		img := loadPPM(t, fmt.Sprintf("test/out/case%d.ppm", n))
		got := FindCellSize(img, 2)
		if got != cs {
			t.Fatalf("case %d: cell size %d, want %d", n, got, cs)
		}
		m, err := Decode(img, cs)
		if err != nil {
			t.Fatalf("case %d: %v", n, err)
		}
		joined := strings.Join([]string{m.Type, m.ChanNum, m.ChanName, m.Sender, m.Text}, "\x1f")
		if joined != string(want) || m.Seq != seq {
			t.Fatalf("case %d mismatch:\n got %q\nwant %q", n, joined, want)
		}
		t.Logf("case %d ok: [%s] %s: %s", n, m.Type, m.Sender, m.Text)

		// corrupt one payload pixel -> must be rejected, never garbage
		cx, cy := (25%stripCols)*cs+cs/2, (25/stripCols)*cs+cs/2
		i := (cy*img.w + cx) * 3
		img.px[i] ^= 0x80
		if _, err := Decode(img, cs); err == nil {
			t.Fatalf("case %d: corrupted frame accepted", n)
		}
	}
}

func TestNoStrip(t *testing.T) {
	img := &ppm{400, 60, make([]byte, 400*60*3)}
	if FindCellSize(img, 2) != 0 {
		t.Fatal("found marker in blank image")
	}
}
