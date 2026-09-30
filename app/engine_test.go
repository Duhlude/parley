package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"
)

// When PARLEY_FAKE_MT is set, the test binary acts as parley-mt.exe.
func TestMain(m *testing.M) {
	if os.Getenv("PARLEY_FAKE_MT") != "" {
		fakeMT()
		return
	}
	os.Exit(m.Run())
}

func fakeMT() {
	in := bufio.NewReader(os.Stdin)
	fmt.Print("READY\tparley-mt\t1\n")
	for {
		line, err := in.ReadString('\n')
		if err != nil {
			return
		}
		f := strings.Split(strings.TrimRight(line, "\n"), "\t")
		if f[0] == "Q" {
			return
		}
		n, _ := strconv.Atoi(f[len(f)-1])
		buf := make([]byte, n)
		io.ReadFull(in, buf)
		text := string(buf)
		var out, status = "", "OK"
		switch {
		case f[0] == "D":
			out = "pt 97 un 0 un 0 1"
		case text == "crash":
			os.Exit(3)
		case text == "fail":
			status, out = "ERR", "model files missing"
		default:
			out = "[" + f[3] + "|" + f[4] + "] " + strings.ToUpper(text)
		}
		fmt.Printf("%s\t%s\t%d\n%s", status, f[1], len(out), out)
	}
}

func TestProcEngine(t *testing.T) {
	exe, _ := os.Executable()
	t.Setenv("PARLEY_FAKE_MT", "1")
	e := &procEngine{Path: exe}
	defer e.Close()

	code, pct, rel, err := e.Detect("olá tudo bem")
	if err != nil || code != "pt" || pct != 97 || !rel {
		t.Fatalf("detect %v %v %v %v", code, pct, rel, err)
	}
	out, err := e.Translate("olá\nmundo ção", true, `C:\m\pt-en`, "")
	if err != nil || out != `[C:\m\pt-en|-] OLÁ
MUNDO ÇÃO` {
		t.Fatalf("translate %q %v", out, err)
	}
	if _, err := e.Translate("fail", true, "a", "b"); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("want engine error, got %v", err)
	}
	// A crash is survived: the engine restarts for the next request.
	e.Translate("crash", true, "a", "")
	if out, err := e.Translate("again", true, "a", ""); err != nil || !strings.HasSuffix(out, "AGAIN") {
		t.Fatalf("after crash %q %v", out, err)
	}
	// Idle shutdown frees the process; the next call starts it again.
	e.last = e.last.Add(-idleAfter)
	e.idleCheck()
	if e.cmd != nil {
		t.Fatal("engine still running after idle check")
	}
	if out, err := e.Translate("back", true, "a", ""); err != nil || !strings.HasSuffix(out, "BACK") {
		t.Fatalf("after idle %q %v", out, err)
	}
	if _, err := e.Translate("x", true, "a\tb", ""); err == nil {
		t.Error("tab in model path must be rejected")
	}
}
