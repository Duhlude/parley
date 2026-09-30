package main

// 1.6 features on the Windows side: update check, alert chime and the
// quick-reply menu.

import (
	"time"
	"unsafe"
)

var pPlaySoundW = winmm.NewProc("PlaySoundW")

// updateLoop checks GitHub shortly after start and then once a day.
func updateLoop() {
	time.Sleep(20 * time.Second)
	for {
		app.mu.Lock()
		on := app.cfg.UpdateCheck != "off"
		app.mu.Unlock()
		if on {
			if r, err := latestRelease(); err == nil && newerVersion(r.Version, appVersion) {
				app.mu.Lock()
				first := app.update.Version != r.Version
				app.update = r
				app.mu.Unlock()
				if first {
					app.setNotice(Tf("Parley %s is available. Right-click the tray icon to download it.", r.Version))
				}
			}
		}
		time.Sleep(24 * time.Hour)
	}
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
