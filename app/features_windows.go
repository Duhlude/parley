package main

// 1.6 features on the Windows side: update check, alert chime and the
// quick-reply menu.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"
)

var pPlaySoundW = winmm.NewProc("PlaySoundW")

// updateLoop checks GitHub right after Parley opens, then every 6 hours for
// people who leave it running. If the check fails (e.g. Parley started with
// Windows before the network was up) it retries every few minutes.
func updateLoop() {
	time.Sleep(3 * time.Second) // let the overlay come up first
	for {
		app.mu.Lock()
		on := app.cfg.UpdateCheck != "off"
		app.mu.Unlock()
		wait := 6 * time.Hour
		if on {
			r, err := latestRelease()
			switch {
			case err != nil:
				wait = 3 * time.Minute
			case newerVersion(r.Version, appVersion):
				app.mu.Lock()
				first := app.update.Version != r.Version
				app.update = r
				app.mu.Unlock()
				if first {
					if canSelfUpdate(r) {
						app.setNotice(Tf("Parley %s is available. Right-click the tray icon and choose Update.", r.Version))
					} else {
						app.setNotice(Tf("Parley %s is available. Right-click the tray icon to download it.", r.Version))
					}
				}
			}
		}
		time.Sleep(wait)
	}
}

// updating is set while an update downloads (GUI thread reads it for the menu).
var updating atomic.Bool

// installedCopy reports whether this Parley.exe is the one Parley-Setup
// installed (%LOCALAPPDATA%\Programs\Parley), which the installer can replace.
func installedCopy() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	if r, err := filepath.EvalSymlinks(exe); err == nil {
		exe = r
	}
	want := filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Parley")
	return os.Getenv("LOCALAPPDATA") != "" && strings.EqualFold(filepath.Clean(filepath.Dir(exe)), filepath.Clean(want))
}

// canSelfUpdate: the release can be installed from inside Parley.
func canSelfUpdate(r release) bool {
	return r.canInstall() && (installedCopy() || os.Getenv("PARLEY_UPDATE_URL") != "")
}

// startUpdate downloads the new installer, checks it against GitHub's
// checksum, runs it in update mode and quits; the installer replaces the
// files and starts the new Parley. On any problem it opens the release page.
func startUpdate(r release) {
	if !updating.CompareAndSwap(false, true) {
		return
	}
	go func() {
		dir := filepath.Join(os.TempDir(), "Parley-update")
		os.RemoveAll(dir) // old downloads
		app.setNotice(Tf("Downloading Parley %s…", r.Version))
		path, err := downloadSetup(r, dir, func(pct int) {
			app.setNotice(Tf("Downloading Parley %s… %d%%", r.Version, pct))
		})
		if err == nil {
			app.setNotice(Tf("Installing Parley %s…", r.Version))
			cmd := exec.Command(path, "/update")
			cmd.Dir = dir
			err = cmd.Start()
		}
		if err != nil {
			vlog("update failed: %v", err)
			app.setNotice(T("Couldn't install the update automatically, so its download page is opening instead."))
			openURL(r.URL)
			updating.Store(false)
			return
		}
		// The installer waits for Parley to close, updates it and starts it again.
		pPostMessageW.Call(app.overlay, WM_APP+9, 0, 0)
	}()
}

// chime plays Windows' notification sound for an alert, at most every 3 s.
func (a *App) chime() {
	a.mu.Lock()
	on := a.cfg.AlertSound != "off"
	ok := time.Since(a.lastChime) > 3*time.Second
	if on && ok {
		a.lastChime = time.Now()
	}
	a.mu.Unlock()
	if on && ok {
		const sndAlias, sndAsync, sndNoDefault = 0x00010000, 0x0001, 0x0002
		if r, _, _ := pPlaySoundW.Call(uintptr(unsafe.Pointer(u16("SystemNotification"))), 0, sndAlias|sndAsync|sndNoDefault); r == 0 {
			pMessageBeep.Call(0x40)
		}
	}
}

var pMessageBeep = user32.NewProc("MessageBeep")

// quickReplyMenu opens the phrasebook above the quick-reply button.
func quickReplyMenu() {
	app.mu.Lock()
	lang := app.reply.Lang
	app.mu.Unlock()
	items := []MenuItem{{Label: T("Quick replies"), Title: true}}
	for i, p := range phrasebook {
		it := MenuItem{Label: T(p.EN), ID: 1300 + i}
		if lang != "" {
			if t := p.ready(lang); t != "" && t != T(p.EN) {
				it.Hint = t
			}
		}
		items = append(items, it)
	}
	if lang == "" {
		items = append(items, MenuItem{Sep: true}, MenuItem{Label: T("Pick a reply language first: click a message, or the language button."), Disabled: true})
	}
	pt := POINT{chipRect.Right + sc(8), qrRect.Top - sc(8)} // opens over the reply field
	pClientToScreen.Call(app.overlay, uintptr(unsafe.Pointer(&pt)))
	cmd := showMenu(items, pt.X, pt.Y, true)
	if cmd >= 1300 && cmd < 1300+len(phrasebook) {
		app.sendPhrase(phrasebook[cmd-1300])
	}
}

// cleanUpdateLeftovers deletes the *.old copies the installer leaves when it
// replaces files that were still in use (the old Parley was still closing).
func cleanUpdateLeftovers() {
	time.Sleep(5 * time.Second)
	exe, err := os.Executable()
	if err != nil || !installedCopy() {
		return
	}
	old, _ := filepath.Glob(filepath.Join(filepath.Dir(exe), "*.old"))
	for _, f := range old {
		os.Remove(f)
	}
}
