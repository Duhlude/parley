package main

// Auto-hide: when chat has been quiet for a while, the overlay fades out and
// lets clicks through to WoW; a new message (or Ctrl+Shift+T, the tray icon,
// an alert or a notice) fades it back in. Once faded out the window is also
// hidden, so it can't catch clicks or show up on setups without transparency.

import "time"

const (
	autoHideTimerID = 10 // fade animation
	msgWake         = WM_APP + 21
)

var autoHide struct {
	hidden     bool      // faded out by auto-hide
	alpha      int       // current layered alpha, 0-255
	goal       int       // alpha the fade is heading to
	lastActive time.Time // guarded by app.mu
}

var autoHideChoices = []int{0, 10, 30, 60, 120} // seconds; 0 = off

func opacityAlpha() int { return app.cfg.Opacity * 255 / 100 }

// touch records activity (new message, reply, notice…). Any goroutine.
func (a *App) touch() {
	a.mu.Lock()
	autoHide.lastActive = time.Now()
	a.mu.Unlock()
}

// wake records activity and brings an auto-hidden overlay back. Any goroutine.
func (a *App) wake() {
	a.touch()
	if a.overlay != 0 {
		pPostMessageW.Call(a.overlay, msgWake, 0, 0)
	}
}

// overlayBusy: the player is using the overlay, so don't hide it.
func overlayBusy() bool {
	if len(menuStack) > 0 || settingsHwnd != 0 || listening {
		return true
	}
	if getText(overlayEdit) != "" { // a reply is being typed
		return true
	}
	pt := cursorPos()
	return inRect(windowRect(app.overlay), pt.X, pt.Y)
}

// autoHideTick runs every second on the GUI thread.
func autoHideTick() {
	app.mu.Lock()
	secs := app.cfg.AutoHide
	idle := time.Since(autoHide.lastActive)
	app.mu.Unlock()
	if secs <= 0 || !app.visible {
		if autoHide.hidden {
			fadeIn()
		}
		return
	}
	if autoHide.hidden {
		return
	}
	if overlayBusy() {
		app.touch()
		return
	}
	if idle >= time.Duration(secs)*time.Second {
		fadeOut()
	}
}

func fadeOut() {
	if autoHide.hidden {
		return
	}
	autoHide.hidden = true
	overlayTip.reset()
	setClickThrough(true)
	startFade(0)
}

func fadeIn() {
	app.touch()
	if !autoHide.hidden {
		return
	}
	autoHide.hidden = false
	setClickThrough(app.cfg.ClickThrough && !isForeground(app.overlay))
	if app.visible {
		pShowWindow.Call(app.overlay, SW_SHOWNOACTIVATE)
		pSetWindowPos.Call(app.overlay, ^uintptr(0) /*HWND_TOPMOST*/, 0, 0, 0, 0, SWP_NOMOVE|SWP_NOSIZE|SWP_NOACTIVATE)
	}
	startFade(opacityAlpha())
}

func isForeground(h uintptr) bool {
	fg, _, _ := pGetForegroundWindow.Call()
	return fg == h
}

func startFade(goal int) {
	autoHide.goal = goal
	pSetTimer.Call(app.overlay, autoHideTimerID, 15, 0)
}

// fadeStep moves the alpha toward the goal: ~0.4 s out, ~0.15 s in.
func fadeStep() {
	a, g := autoHide.alpha, autoHide.goal
	switch {
	case a > g:
		a -= 12
		if a < g {
			a = g
		}
	case a < g:
		a += 32
		if a > g {
			a = g
		}
	}
	autoHide.alpha = a
	pSetLayeredWindowAttributes.Call(app.overlay, 0, uintptr(a), LWA_ALPHA)
	if a == g {
		pKillTimer.Call(app.overlay, autoHideTimerID)
		if a == 0 && autoHide.hidden {
			pShowWindow.Call(app.overlay, SW_HIDE) // fully faded: take it off screen
		}
	}
}

// setAutoHide stores the delay chosen in the tray menu.
func setAutoHide(secs int) {
	app.mu.Lock()
	app.cfg.AutoHide = secs
	cfg := app.cfg
	app.mu.Unlock()
	saveConfig(cfg)
	app.touch()
	if secs > 0 {
		app.setNotice(T("Auto-hide is on: the overlay fades out when chat is quiet and comes back when a new message arrives. Ctrl+Shift+T shows it any time."))
	} else if autoHide.hidden {
		fadeIn()
	}
}

// autoHideMenu is the tray submenu.
func autoHideMenu() []MenuItem {
	labels := []string{T("Off"), T("After 10 seconds"), T("After 30 seconds"), T("After 1 minute"), T("After 2 minutes")}
	tip := T("Fade the overlay out when chat has been quiet this long. It comes back when a new message arrives, or with Ctrl+Shift+T.")
	var items []MenuItem
	for i, s := range autoHideChoices {
		it := MenuItem{Label: labels[i], ID: cmdAutoHide0 + i, Checked: app.cfg.AutoHide == s, Radio: true}
		if s > 0 {
			it.Tip = tip
		}
		items = append(items, it)
	}
	return items
}
