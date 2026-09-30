package main

// Client for parley-mt.exe, the offline translation process. One process
// stays running; requests are serialized over its stdin/stdout.

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type procEngine struct {
	Path string

	mu     sync.Mutex
	cmd    *exec.Cmd
	in     io.WriteCloser
	out    *bufio.Reader
	nextID int
	fails  int
	last   time.Time
	timer  *time.Timer
}

// idleAfter frees the engine's memory (about 200 MB per loaded language)
// when nobody has written anything foreign for a while. It restarts in
// well under a second on the next message.
const idleAfter = 10 * time.Minute

// engineExe finds parley-mt.exe next to Parley.exe.
func engineExe() string { return helperExe("parley-mt.exe") }

// helperExe finds a bundled helper next to Parley.exe (installed copy) or in
// bin\ beside it (a build from the repository root). "" if missing.
func helperExe(name string) string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	for _, p := range []string{filepath.Join(filepath.Dir(exe), name), filepath.Join(filepath.Dir(exe), "bin", name)} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func (e *procEngine) start() error {
	if e.cmd != nil {
		return nil
	}
	if e.fails >= 5 {
		return errors.New("the offline translator keeps crashing; see Parley's voice/engine log")
	}
	cmd := exec.Command(e.Path)
	hideWindow(cmd)
	in, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		e.fails++
		return fmt.Errorf("couldn't start the offline translator: %v", err)
	}
	e.cmd, e.in, e.out = cmd, in, bufio.NewReaderSize(outPipe, 64<<10)
	ready := make(chan error, 1)
	go func() {
		line, err := e.out.ReadString('\n')
		if err == nil && !strings.HasPrefix(line, "READY") {
			err = fmt.Errorf("unexpected greeting %q", strings.TrimSpace(line))
		}
		ready <- err
	}()
	select {
	case err = <-ready:
	case <-time.After(20 * time.Second):
		err = errors.New("timed out")
	}
	if err != nil {
		e.stopLocked()
		e.fails++
		return fmt.Errorf("the offline translator didn't start: %v", err)
	}
	return nil
}

func (e *procEngine) stopLocked() {
	if e.cmd == nil {
		return
	}
	e.in.Close()
	if e.cmd.Process != nil {
		e.cmd.Process.Kill()
	}
	e.cmd.Wait()
	e.cmd, e.in, e.out = nil, nil, nil
}

func (e *procEngine) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cmd != nil {
		io.WriteString(e.in, "Q\n")
	}
	e.stopLocked()
}

// call sends one request and reads the reply. The engine is restarted once
// if it died (e.g. killed by the user).
func (e *procEngine) call(header string, payload string) (string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		if err := e.start(); err != nil {
			return "", err
		}
		e.nextID++
		id := strconv.Itoa(e.nextID)
		req := strings.Replace(header, "{id}", id, 1) + "\t" + strconv.Itoa(len(payload)) + "\n" + payload
		if _, err := io.WriteString(e.in, req); err != nil {
			lastErr = err
			e.stopLocked()
			e.fails++
			continue
		}
		status, body, err := e.readReply(id)
		if err != nil {
			lastErr = err
			e.stopLocked()
			e.fails++
			continue
		}
		e.fails = 0
		e.last = time.Now()
		if e.timer == nil {
			e.timer = time.AfterFunc(idleAfter, e.idleCheck)
		} else {
			e.timer.Reset(idleAfter)
		}
		if status != "OK" {
			return "", errors.New(body)
		}
		return body, nil
	}
	return "", fmt.Errorf("offline translator stopped: %v", lastErr)
}

func (e *procEngine) idleCheck() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cmd != nil && time.Since(e.last) >= idleAfter {
		io.WriteString(e.in, "Q\n")
		e.stopLocked()
	}
}

func (e *procEngine) readReply(id string) (string, string, error) {
	for {
		line, err := e.out.ReadString('\n')
		if err != nil {
			return "", "", err
		}
		f := strings.Split(strings.TrimRight(line, "\r\n"), "\t")
		if len(f) != 3 {
			continue
		}
		n, err := strconv.Atoi(f[2])
		if err != nil || n < 0 || n > 1<<24 {
			return "", "", fmt.Errorf("bad reply %q", line)
		}
		buf := make([]byte, n)
		if _, err := io.ReadFull(e.out, buf); err != nil {
			return "", "", err
		}
		if f[1] == id {
			return f[0], string(buf), nil
		}
	}
}

func (e *procEngine) Detect(text string) (string, int, bool, error) {
	out, err := e.call("D\t{id}", text)
	if err != nil {
		return "", 0, false, err
	}
	f := strings.Fields(out)
	if len(f) < 7 {
		return "", 0, false, fmt.Errorf("bad detect reply %q", out)
	}
	pct, _ := strconv.Atoi(f[1])
	return f[0], pct, f[6] == "1", nil
}

func (e *procEngine) Translate(text string, html bool, first, second string) (string, error) {
	h := "0"
	if html {
		h = "1"
	}
	if second == "" {
		second = "-"
	}
	if strings.ContainsAny(first+second, "\t\n") {
		return "", errors.New("bad model folder")
	}
	return e.call("T\t{id}\t"+h+"\t"+first+"\t"+second, text)
}
