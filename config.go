package main

import (
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	DeepLKeyEnc  []byte   `json:"deeplKeyEnc,omitempty"` // DPAPI-encrypted
	MyLang       string   `json:"myLang"`
	WowPath      string   `json:"wowPath"`
	FontSize     int      `json:"fontSize"`
	Opacity      int      `json:"opacity"` // percent
	ShowOriginal bool     `json:"showOriginal"`
	CellSize     int      `json:"cellSize"`
	Skin         string   `json:"skin"`
	VoiceMode    string   `json:"voiceMode"`             // "silent" or "typing" (Win+H)
	VoiceModel   string   `json:"voiceModel"`            // "accurate" (small) or "fast" (base)
	VoiceManual  bool     `json:"voiceManual"`           // true = review before sending
	ExplainTerms string   `json:"explainTerms"`          // ""/auto, "on", "off"
	Engine       string   `json:"engine"`                // ""/auto, "offline", "deepl", "azure"
	Muted        []string `json:"muted,omitempty"`       // senders whose messages are hidden
	History      string   `json:"history"`               // ""/on, "off": save translated chat to a daily text file
	ClickThrough bool     `json:"clickThrough"`          // overlay ignores the mouse until Ctrl+Shift+T
	UILang       string   `json:"uiLang"`                // interface language; ""/auto = Windows
	AzureKeyEnc  []byte   `json:"azureKeyEnc,omitempty"` // DPAPI-encrypted
	AzureRegion  string   `json:"azureRegion,omitempty"`
	AlertWords   string   `json:"alertWords,omitempty"`  // comma-separated; the character name is added automatically
	AlertSound   string   `json:"alertSound,omitempty"`  // ""/on, "off"
	CharName     string   `json:"charName,omitempty"`    // last character the addon reported
	UpdateCheck  string   `json:"updateCheck,omitempty"` // ""/on, "off"
	AutoHide     int      `json:"autoHide,omitempty"`    // seconds of quiet before the overlay fades out; 0 = off
	X, Y, W, H   int
}

func defaultConfig() Config {
	return Config{
		MyLang:       "EN-US",
		WowPath:      `C:\Program Files (x86)\World of Warcraft\_classic_era_`,
		FontSize:     15,
		Opacity:      92,
		ShowOriginal: true,
		CellSize:     2,
		Skin:         "parley",
		X:            40, Y: 260, W: 460, H: 320,
	}
}

func configDir() string {
	d, err := os.UserConfigDir()
	if err != nil {
		d = "."
	}
	return filepath.Join(d, "Parley")
}

func loadConfig() Config {
	c := defaultConfig()
	b, err := os.ReadFile(filepath.Join(configDir(), "settings.json"))
	if err == nil {
		json.Unmarshal(b, &c)
	} else {
		// first run: read chat in the Windows language
		c.MyLang = chatLangForLocale(systemLocaleName())
	}
	if c.FontSize < 10 || c.FontSize > 32 {
		c.FontSize = 15
	}
	if c.Opacity < 40 || c.Opacity > 100 {
		c.Opacity = 92
	}
	if c.MyLang == "" {
		c.MyLang = "EN-US"
	}
	return c
}

func saveConfig(c Config) error {
	if err := os.MkdirAll(configDir(), 0o755); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(c, "", "  ")
	tmp := filepath.Join(configDir(), "settings.json.tmp")
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(configDir(), "settings.json"))
}

//go:embed addon/Parley
var addonFS embed.FS

// installAddon copies the embedded addon into <wowPath>\Interface\AddOns\Parley.
func installAddon(wowPath string) (string, error) {
	wowPath = strings.TrimSpace(wowPath)
	if wowPath == "" {
		return "", errors.New("set your WoW folder first")
	}
	if st, err := os.Stat(wowPath); err != nil || !st.IsDir() {
		return "", errors.New("WoW folder not found: " + wowPath)
	}
	dest := filepath.Join(wowPath, "Interface", "AddOns", "Parley")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return "", err
	}
	err := fs.WalkDir(addonFS, "addon/Parley", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := addonFS.ReadFile(p)
		if err != nil {
			return err
		}
		out := filepath.Join(dest, filepath.FromSlash(strings.TrimPrefix(p, "addon/Parley/")))
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		return os.WriteFile(out, b, 0o644)
	})
	return dest, err
}

// wowFlavors are the game folders Parley supports, under the World of Warcraft folder.
var wowFlavors = []string{"_classic_era_", "_retail_"}

// wowInstalls returns the folder you picked plus the other supported game
// folders next to it (e.g. _retail_ beside _classic_era_) that exist.
func wowInstalls(wowPath string) []string {
	wowPath = strings.TrimRight(strings.TrimSpace(wowPath), `\/`)
	if wowPath == "" {
		return nil
	}
	out := []string{wowPath}
	base := strings.ToLower(filepath.Base(wowPath))
	root := filepath.Dir(wowPath)
	if !strings.HasPrefix(base, "_") || !strings.HasSuffix(base, "_") {
		root = wowPath // they picked the World of Warcraft folder itself
		out = nil
	}
	for _, fl := range wowFlavors {
		p := filepath.Join(root, fl)
		if strings.EqualFold(p, wowPath) {
			continue
		}
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			out = append(out, p)
		}
	}
	return out
}

// installAddonAll installs the addon into every supported game folder found
// from wowPath and returns where it went. It fails only if nothing worked.
func installAddonAll(wowPath string) ([]string, error) {
	var dests []string
	var firstErr error
	for _, w := range wowInstalls(wowPath) {
		d, err := installAddon(w)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		dests = append(dests, d)
	}
	if len(dests) == 0 {
		if firstErr == nil {
			firstErr = errors.New("WoW folder not found: " + wowPath)
		}
		return nil, firstErr
	}
	return dests, nil
}

// addonInstalledVersion returns the Version line of the installed TOC, or "".
func addonInstalledVersion(wowPath string) string {
	b, err := os.ReadFile(filepath.Join(wowPath, "Interface", "AddOns", "Parley", "Parley.toc"))
	if err != nil {
		return ""
	}
	return tocVersion(string(b))
}

func embeddedAddonVersion() string {
	b, _ := addonFS.ReadFile("addon/Parley/Parley.toc")
	return tocVersion(string(b))
}

func tocVersion(s string) string {
	for _, l := range strings.Split(s, "\n") {
		if strings.HasPrefix(l, "## Version:") {
			return strings.TrimSpace(strings.TrimPrefix(l, "## Version:"))
		}
	}
	return ""
}

func sleepMs(n int) { time.Sleep(time.Duration(n) * time.Millisecond) }
