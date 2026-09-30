package main

// Translation benchmark: `Parley.exe --benchmark` runs a fixed set of real
// WoW chat lines (bench/benchmark.json) through the offline engine and, if a
// key is saved, DeepL, and writes the outputs and timings to a JSON file
// next to Parley.exe (or in %APPDATA%\Parley\benchmark). Nothing here runs
// during normal use.

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"syscall"
	"time"
	"unsafe"
)

//go:embed bench/benchmark.json
var benchCases []byte

type benchCase struct {
	ID    string `json:"id"`
	Lang  string `json:"lang"`
	From  string `json:"from"`
	To    string `json:"to"`
	Text  string `json:"text"`
	Ref   string `json:"ref"`
	Reply bool   `json:"reply,omitempty"`
}

type benchRun struct {
	Text     string  `json:"text"`
	Detected string  `json:"detected,omitempty"`
	Err      string  `json:"err,omitempty"`
	Ms       float64 `json:"ms"`
	Cold     bool    `json:"cold,omitempty"`
}

type benchResult struct {
	benchCase
	Input   string    `json:"input"`
	Offline *benchRun `json:"offline,omitempty"`
	DeepL   *benchRun `json:"deepl,omitempty"`
	Azure   *benchRun `json:"azure,omitempty"`
}

type benchReport struct {
	When          string             `json:"when"`
	CPU           string             `json:"cpu"`
	Threads       int                `json:"threads"`
	DeepLUsed     bool               `json:"deepl_used"`
	AzureUsed     bool               `json:"azure_used"`
	DeepLNote     string             `json:"deepl_note,omitempty"`
	PairSetupMs   map[string]float64 `json:"offline_pair_setup_ms"`
	EngineRAMMB   float64            `json:"offline_engine_peak_ram_mb"`
	EngineStartMs float64            `json:"offline_engine_start_ms"`
	Results       []benchResult      `json:"results"`
}

var (
	pAllocConsole         = kernel32.NewProc("AllocConsole")
	pGetProcessMemoryInfo = syscall.NewLazyDLL("psapi.dll").NewProc("GetProcessMemoryInfo")
	pOpenProcessBench     = kernel32.NewProc("OpenProcess")
	pCloseHandleBench     = kernel32.NewProc("CloseHandle")
)

func benchConsole() {
	pAllocConsole.Call()
	if f, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0); err == nil {
		os.Stdout, os.Stderr = f, f
	}
	if f, err := os.OpenFile("CONIN$", os.O_RDONLY, 0); err == nil {
		os.Stdin = f
	}
}

// peakRAM returns the peak working set of a process in MB.
func peakRAM(pid int) float64 {
	h, _, _ := pOpenProcessBench.Call(0x1000|0x0010, 0, uintptr(pid)) // QUERY_LIMITED_INFORMATION | VM_READ
	if h == 0 {
		return 0
	}
	defer pCloseHandleBench.Call(h)
	var pmc [10]uintptr // PROCESS_MEMORY_COUNTERS: cb, PageFaultCount, PeakWorkingSetSize, ...
	*(*uint32)(unsafe.Pointer(&pmc[0])) = uint32(unsafe.Sizeof(pmc))
	if r, _, _ := pGetProcessMemoryInfo.Call(h, uintptr(unsafe.Pointer(&pmc[0])), unsafe.Sizeof(pmc)); r == 0 {
		return 0
	}
	return float64(pmc[2]) / (1 << 20)
}

func msSince(t time.Time) float64 { return float64(time.Since(t).Microseconds()) / 1000 }

func runBenchmark() {
	benchConsole()
	fmt.Println("Parley translation benchmark")
	fmt.Println("============================")
	var cases []benchCase
	if err := json.Unmarshal(benchCases, &cases); err != nil {
		fmt.Println("bad test set:", err)
		benchWait()
		return
	}
	cfg := loadConfig()
	key := string(unprotect(cfg.DeepLKeyEnc))
	azKey, azRegion := string(unprotect(cfg.AzureKeyEnc)), cfg.AzureRegion

	rep := benchReport{When: time.Now().Format(time.RFC3339), CPU: os.Getenv("PROCESSOR_IDENTIFIER"),
		Threads: runtime.NumCPU(), PairSetupMs: map[string]float64{}}

	exe := engineExe()
	if exe == "" {
		fmt.Println("parley-mt.exe not found next to Parley.exe")
		benchWait()
		return
	}
	eng := &procEngine{Path: exe}
	off := NewOffline(offlineModelsDir(), eng)
	off.Progress = func(m string) { fmt.Println("   ", m) }

	input := func(c benchCase) (string, string) {
		if c.Reply {
			return prepareReply(expandShortcuts(c.Text), c.To), c.From
		}
		_, hint := Guess(c.Text)
		return prepareIncoming(ExpandSlang(c.Text), c.To), hint
	}

	// 1. Download every language once (not timed as translation).
	fmt.Println("\n1/4  Making sure every language is downloaded (first run only, a few hundred MB)...")
	seen := map[string]bool{}
	for _, c := range cases {
		pair := c.Lang + ">" + c.To
		if seen[pair] {
			continue
		}
		seen[pair] = true
		src, hint := input(c)
		t := time.Now()
		if _, err := off.Translate(src, hint, c.To); err != nil {
			fmt.Printf("   %s: %v\n", pair, err)
		}
		rep.PairSetupMs[pair] = msSince(t)
	}
	eng.Close() // unload everything so the next pass measures real cold starts

	// 2. Offline: the first line of each language includes loading its model
	// from disk ("cold"); the rest are normal. (No second pass: the engine
	// caches repeated sentences, which real chat rarely has.)
	fmt.Println("\n2/4  Offline engine...")
	res := make([]benchResult, len(cases))
	sorted := make([]int, len(cases))
	for i := range sorted {
		sorted[i] = i
	}
	sort.SliceStable(sorted, func(a, b int) bool {
		return cases[sorted[a]].Lang+cases[sorted[a]].To < cases[sorted[b]].Lang+cases[sorted[b]].To
	})
	t0 := time.Now()
	loaded := map[string]bool{}
	for round := 1; round <= 1; round++ {
		for _, i := range sorted {
			c := cases[i]
			src, hint := input(c)
			pair := c.Lang + ">" + c.To
			off.clearCache()
			t := time.Now()
			tr, err := off.Translate(src, hint, c.To)
			tr.Text = stripKeep(tr.Text)
			ms := msSince(t)
			if round == 1 {
				if rep.EngineStartMs == 0 {
					rep.EngineStartMs = msSince(t0)
				}
				r := &benchRun{Text: tr.Text, Detected: tr.Detected, Ms: ms, Cold: !loaded[pair]}
				if err != nil {
					r.Err = err.Error()
				}
				loaded[pair] = true
				res[i] = benchResult{benchCase: c, Input: stripKeep(src), Offline: r}
				fmt.Printf("   %-12s %7.0f ms  %s\n", c.ID, ms, tr.Text)
			}
		}
		if eng.cmd != nil && eng.cmd.Process != nil {
			if mb := peakRAM(eng.cmd.Process.Pid); mb > rep.EngineRAMMB {
				rep.EngineRAMMB = mb
			}
		}
	}
	eng.Close()

	// 3. DeepL.
	fmt.Println("\n3/4  Online engines: DeepL...")
	if key == "" {
		rep.DeepLNote = "no DeepL key saved in Parley's settings"
		fmt.Println("   skipped: no DeepL key saved in Parley's settings")
	} else {
		rep.DeepLUsed = true
		dl := NewTranslator()
		for round := 1; round <= 1; round++ {
			for i, c := range cases {
				src, hint := input(c)
				t := time.Now()
				tr, err := dl.Translate(key, src, hint, c.To)
				tr.Text = stripKeep(tr.Text)
				ms := msSince(t)
				if round == 1 {
					r := &benchRun{Text: tr.Text, Detected: tr.Detected, Ms: ms}
					if err != nil {
						r.Err = err.Error()
					}
					res[i].DeepL = r
					fmt.Printf("   %-12s %7.0f ms  %s\n", c.ID, ms, tr.Text)
				}
				time.Sleep(120 * time.Millisecond) // stay well under DeepL's rate limit
			}
		}
	}
	fmt.Println("\n     Azure...")
	if azKey == "" {
		fmt.Println("   skipped: no Azure key saved in Parley's settings")
	} else {
		rep.AzureUsed = true
		az := NewAzure()
		for i, c := range cases {
			src, hint := input(c)
			t := time.Now()
			tr, err := az.Translate(azKey, azRegion, src, hint, c.To)
			tr.Text = stripKeep(tr.Text)
			ms := msSince(t)
			r := &benchRun{Text: tr.Text, Detected: tr.Detected, Ms: ms}
			if err != nil {
				r.Err = err.Error()
			}
			res[i].Azure = r
			fmt.Printf("   %-12s %7.0f ms  %s\n", c.ID, ms, tr.Text)
			time.Sleep(120 * time.Millisecond)
		}
	}
	rep.Results = res

	// 4. Save.
	fmt.Println("\n4/4  Saving...")
	name := "benchmark-" + time.Now().Format("2006-01-02-150405") + ".json"
	b, _ := json.MarshalIndent(rep, "", " ")
	var saved string
	for _, dir := range benchDirs() {
		if os.MkdirAll(dir, 0o755) == nil && os.WriteFile(filepath.Join(dir, name), b, 0o644) == nil {
			saved = filepath.Join(dir, name)
			break
		}
	}
	if saved == "" {
		fmt.Println("   couldn't save the results")
	} else {
		fmt.Println("   saved to", saved)
	}
	fmt.Println("\nDone. Tell Claude it's finished.")
	benchWait()
}

func benchDirs() []string {
	var out []string
	if exe, err := os.Executable(); err == nil {
		out = append(out, filepath.Join(filepath.Dir(exe), "benchmark"))
	}
	return append(out, filepath.Join(configDir(), "benchmark"))
}

func benchWait() {
	fmt.Println("Press Enter to close this window.")
	var s string
	fmt.Scanln(&s)
}
