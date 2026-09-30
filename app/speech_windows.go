package main

// Voice input that runs entirely on this PC: record the microphone with
// winmm, then transcribe with whisper.cpp (whisper-cli.exe next to
// Parley.exe) using a speech model downloaded once on first use.
// Silent: no popup, no chime, no online speech service.

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var (
	winmm                  = syscall.NewLazyDLL("winmm.dll")
	pWaveInOpen            = winmm.NewProc("waveInOpen")
	pWaveInClose           = winmm.NewProc("waveInClose")
	pWaveInPrepareHeader   = winmm.NewProc("waveInPrepareHeader")
	pWaveInUnprepareHeader = winmm.NewProc("waveInUnprepareHeader")
	pWaveInAddBuffer       = winmm.NewProc("waveInAddBuffer")
	pWaveInStart           = winmm.NewProc("waveInStart")
	pWaveInStop            = winmm.NewProc("waveInStop")
	pWaveInReset           = winmm.NewProc("waveInReset")
)

type waveFormat struct {
	FormatTag      uint16
	Channels       uint16
	SamplesPerSec  uint32
	AvgBytesPerSec uint32
	BlockAlign     uint16
	BitsPerSample  uint16
	CbSize         uint16
}

type waveHdr struct {
	Data          *byte
	BufferLength  uint32
	BytesRecorded uint32
	User          uintptr
	Flags         uint32
	Loops         uint32
	Next          uintptr
	Reserved      uintptr
}

const (
	sampleRate = 16000
	chunk      = sampleRate / 10 // 100 ms
	whdrDone   = 0x1
)

type voiceEngine struct {
	reqs chan bool
}

var voice = &voiceEngine{reqs: make(chan bool, 4)}

var errNoSpeech = errors.New("Didn't hear anything. Click the mic and try again.")

// stopRequested drains the request channel; true if "stop" was asked.
func (v *voiceEngine) stopRequested() bool {
	for {
		select {
		case start := <-v.reqs:
			if !start {
				return true
			}
		default:
			return false
		}
	}
}

// record captures one utterance: it starts when you speak and ends after
// about a second of quiet, 20 seconds, or a click on the mic.
func (v *voiceEngine) record() ([]int16, error) {
	wf := waveFormat{FormatTag: 1, Channels: 1, SamplesPerSec: sampleRate,
		AvgBytesPerSec: sampleRate * 2, BlockAlign: 2, BitsPerSample: 16}
	var hwi uintptr
	if r, _, _ := pWaveInOpen.Call(uintptr(unsafe.Pointer(&hwi)), 0xFFFFFFFF, uintptr(unsafe.Pointer(&wf)), 0, 0, 0); r != 0 {
		vlog("waveInOpen failed %d", r)
		return nil, errors.New(T("Couldn't open the microphone. Check Windows Settings → Privacy & security → Microphone (allow desktop apps), and that a mic is plugged in."))
	}
	defer pWaveInClose.Call(hwi)

	const nbuf = 8
	bufs := make([][]byte, nbuf)
	hdrs := make([]*waveHdr, nbuf)
	hsz := unsafe.Sizeof(waveHdr{})
	for i := range bufs {
		bufs[i] = make([]byte, chunk*2)
		hdrs[i] = &waveHdr{Data: &bufs[i][0], BufferLength: uint32(len(bufs[i]))}
		pWaveInPrepareHeader.Call(hwi, uintptr(unsafe.Pointer(hdrs[i])), hsz)
		pWaveInAddBuffer.Call(hwi, uintptr(unsafe.Pointer(hdrs[i])), hsz)
	}
	defer func() {
		pWaveInReset.Call(hwi)
		for i := range hdrs {
			pWaveInUnprepareHeader.Call(hwi, uintptr(unsafe.Pointer(hdrs[i])), hsz)
		}
		runtime.KeepAlive(bufs)
		runtime.KeepAlive(hdrs)
	}()
	pWaveInStart.Call(hwi)
	defer pWaveInStop.Call(hwi)

	var (
		all      []int16
		noise    float64
		chunks   int
		loud     int
		started  bool
		quiet    int
		startIdx int
		begin    = time.Now()
		next     = 0
	)
	for {
		if v.stopRequested() {
			vlog("stopped by user after %d chunks", chunks)
			if !started {
				return nil, nil
			}
			break
		}
		h := hdrs[next]
		if h.Flags&whdrDone == 0 {
			time.Sleep(15 * time.Millisecond)
			if time.Since(begin) > 25*time.Second {
				break
			}
			continue
		}
		n := int(h.BytesRecorded) / 2
		samples := make([]int16, n)
		for i := 0; i < n; i++ {
			samples[i] = int16(binary.LittleEndian.Uint16(bufs[next][i*2:]))
		}
		h.Flags &^= whdrDone
		pWaveInAddBuffer.Call(hwi, uintptr(unsafe.Pointer(h)), hsz)
		next = (next + 1) % nbuf

		all = append(all, samples...)
		chunks++
		rms := rmsOf(samples)
		if chunks <= 3 {
			noise += rms / 3
			continue
		}
		thresh := math.Max(noise*2.5, 350)
		if rms > thresh {
			loud++
			quiet = 0
			if !started && loud >= 2 {
				started = true
				startIdx = max(0, len(all)-sampleRate/2) // keep 0.5 s before speech
				vlog("speech started (rms %.0f, noise %.0f)", rms, noise)
			}
		} else {
			loud = 0
			if started {
				quiet++
				if quiet >= 10 { // ~1 s of quiet
					break
				}
			} else {
				noise = noise*0.9 + rms*0.1
			}
		}
		if !started && time.Since(begin) > 8*time.Second {
			vlog("no speech; noise %.0f", noise)
			return nil, errNoSpeech
		}
		if time.Since(begin) > 20*time.Second {
			break
		}
	}
	if !started {
		return nil, errNoSpeech
	}
	return all[startIdx:], nil
}

func rmsOf(s []int16) float64 {
	if len(s) == 0 {
		return 0
	}
	var sum float64
	for _, v := range s {
		f := float64(v)
		sum += f * f
	}
	return math.Sqrt(sum / float64(len(s)))
}

func writeWav(path string, s []int16) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	data := uint32(len(s) * 2)
	w := func(v any) { binary.Write(f, binary.LittleEndian, v) }
	f.WriteString("RIFF")
	w(36 + data)
	f.WriteString("WAVEfmt ")
	w(uint32(16))
	w(uint16(1))
	w(uint16(1))
	w(uint32(sampleRate))
	w(uint32(sampleRate * 2))
	w(uint16(2))
	w(uint16(16))
	f.WriteString("data")
	w(data)
	return binary.Write(f, binary.LittleEndian, s)
}

// ---------------------------------------------------------------------------
// Model + transcription
// ---------------------------------------------------------------------------

// modelFor picks the whisper model: "fast" = base (148 MB), otherwise
// "accurate" = small (466 MB, about 3x slower but much better with names
// and mumbled speech). English-only variants are used for English.
func modelFor(lang string) (file string, sizeMB int) {
	app.mu.Lock()
	fast := app.cfg.VoiceModel == "fast"
	app.mu.Unlock()
	name, size := "small", 466
	if fast {
		name, size = "base", 148
	}
	if baseLang(lang) == "EN" {
		return "ggml-" + name + ".en.bin", size
	}
	return "ggml-" + name + ".bin", size
}

func ensureModel(lang string) (string, error) {
	file, size := modelFor(lang)
	if exe, err := os.Executable(); err == nil { // a model placed next to Parley.exe wins
		p := filepath.Join(filepath.Dir(exe), "models", file)
		if st, err := os.Stat(p); err == nil && st.Size() > 50<<20 {
			return p, nil
		}
	}
	dir := filepath.Join(configDir(), "models")
	path := filepath.Join(dir, file)
	if st, err := os.Stat(path); err == nil && st.Size() > 50<<20 {
		return path, nil
	}
	os.MkdirAll(dir, 0o755)
	url := "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/" + file
	vlog("downloading %s", url)
	app.setNotice(Tf("First-time setup: downloading the voice model (%d MB)…", size))
	resp, err := http.Get(url)
	if err != nil {
		return "", errors.New(Tf("Couldn't download the voice model: %v", err))
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", errors.New(Tf("Couldn't download the voice model (HTTP %d).", resp.StatusCode))
	}
	tmp := path + ".part"
	out, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	total := resp.ContentLength
	var got int64
	buf := make([]byte, 1<<20)
	lastPct := -1
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := out.Write(buf[:n]); werr != nil {
				out.Close()
				return "", werr
			}
			got += int64(n)
			if total > 0 {
				if pct := int(got * 100 / total); pct != lastPct && pct%5 == 0 {
					lastPct = pct
					app.setNotice(Tf("First-time setup: downloading the voice model… %d%%", pct))
				}
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			out.Close()
			return "", errors.New(Tf("Voice model download interrupted: %v", rerr))
		}
	}
	out.Close()
	if total > 0 && got != total {
		return "", errors.New(T("Voice model download was incomplete. Try the mic again."))
	}
	if err := os.Rename(tmp, path); err != nil {
		return "", err
	}
	vlog("model ready %s (%d bytes)", path, got)
	return path, nil
}

func whisperExe() (string, error) {
	p := helperExe("whisper-cli.exe")
	if p == "" {
		return "", errors.New("whisper-cli.exe is missing from the Parley folder.")
	}
	return p, nil
}

var noiseTags = regexp.MustCompile(`\[[^\]]*\]|\([^)]*\)|\*[^*]*\*`)

func transcribe(wav, model, lang string) (string, error) {
	cli, err := whisperExe()
	if err != nil {
		return "", err
	}
	threads := runtime.NumCPU()
	if threads > 8 {
		threads = 8
	}
	l := strings.ToLower(baseLang(lang))
	cmd := exec.Command(cli, "-m", asciiPath(model), "-f", asciiPath(wav), "-l", l, "-nt", "-np", "-t", strconv.Itoa(threads))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	out, err := cmd.Output()
	if err != nil {
		var stderr string
		if ee, ok := err.(*exec.ExitError); ok {
			stderr = string(ee.Stderr)
		}
		vlog("whisper failed: %v %s", err, stderr)
		return "", errors.New(T("Transcription failed. Open the voice log from the tray menu for details."))
	}
	text := noiseTags.ReplaceAllString(string(out), " ")
	return strings.Join(strings.Fields(text), " "), nil
}

// voiceLoop handles one recording at a time on its own goroutine.
func voiceLoop() {
	for start := range voice.reqs {
		if !start {
			continue
		}
		vlog("start requested")
		app.mu.Lock()
		lang := app.cfg.MyLang
		app.mu.Unlock()

		model, err := ensureModel(lang)
		if err != nil {
			vlog("model: %v", err)
			app.voiceDone("", err)
			continue
		}
		app.setNotice(T("Listening… speak your reply. (Click the mic again to stop.)"))
		samples, err := voice.record()
		if err != nil || samples == nil {
			app.voiceDone("", err)
			continue
		}
		app.setNotice(T("Got it, transcribing…"))
		wav := filepath.Join(os.TempDir(), "parley-voice.wav")
		if err := writeWav(wav, samples); err != nil {
			app.voiceDone("", err)
			continue
		}
		t0 := time.Now()
		text, err := transcribe(wav, model, lang)
		os.Remove(wav)
		vlog("transcribed %.1fs of audio in %v: %q err=%v", float64(len(samples))/sampleRate, time.Since(t0), text, err)
		if err == nil && text == "" {
			err = errNoSpeech
		}
		app.voiceDone(text, err)
	}
}

// vlog appends to %APPDATA%\Parley\voice.log for troubleshooting.
func vlog(format string, args ...any) {
	f, err := os.OpenFile(filepath.Join(configDir(), "voice.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s  %s\n", time.Now().Format("2006-01-02 15:04:05.000"), fmt.Sprintf(format, args...))
}
