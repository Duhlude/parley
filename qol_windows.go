package main

// Quality-of-life features: translation menu, per-message menu (copy, mute,
// reply), chat history file, and downloading languages ahead of time.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// translationMenu builds the tray "Translation" submenu.
func translationMenu() []MenuItem {
	mode := app.engineMode()
	var dl []MenuItem
	seen := map[string]bool{}
	for i, l := range targetLangs {
		ff := ffCode(l.Code)
		if ff == "en" || seen[ff] {
			continue
		}
		seen[ff] = true
		name := langName(ff)
		if ff == "zh-Hans" || ff == "zh-Hant" {
			name = l.Name
		}
		it := MenuItem{Label: name, ID: cmdLang0 + i}
		if app.off.languageReady(ff) {
			it.Checked, it.Hint = true, "ready"
		}
		dl = append(dl, it)
	}
	pairs, size := app.off.installedPairs()
	folder := MenuItem{Label: T("Open offline models folder"), ID: cmdModelsFolder, Tip: T("Where downloaded languages are stored. Delete a folder there to free space; Parley downloads it again when needed.")}
	if len(pairs) > 0 {
		folder.Hint = fmt.Sprintf("%d MB", size>>20)
	}
	return []MenuItem{
		{Label: T("Offline (free, on this PC)"), ID: cmdEngOffline, Checked: mode == "offline", Radio: true,
			Tip: T("Free and private: translates on your PC. Each language downloads once (about 50 MB).")},
		{Label: T("DeepL (API key)"), ID: cmdEngDeepL, Checked: mode == "deepl", Radio: true,
			Tip: T("The best quality. Needs a DeepL API key in Settings. Falls back to offline if DeepL fails.")},
		{Label: T("Azure Translator (API key)"), ID: cmdEngAzure, Checked: mode == "azure", Radio: true,
			Tip: T("Very good quality, 2 million free characters a month. Needs an Azure key in Settings. Falls back to offline if Azure fails.")},
		{Sep: true},
		{Label: T("Download a language now"), Sub: dl},
		folder,
	}
}

// languageReady reports whether both directions to/from English are downloaded.
func (o *Offline) languageReady(ff string) bool {
	return pairReady(o.pairDir(ff, "en")) && pairReady(o.pairDir("en", ff))
}

// prefetch downloads both directions for a language in the background.
func (a *App) prefetch(ff string) {
	if ff == "" || ff == "en" {
		return
	}
	if a.off.languageReady(ff) {
		a.setNotice(Tf("%s is already downloaded.", langName(ff)))
		return
	}
	go func() {
		for _, p := range [][2]string{{ff, "en"}, {"en", ff}} {
			if _, err := a.off.ensurePair(p[0], p[1]); err != nil {
				a.setNotice(Tf("Couldn't download %s: %s", langName(ff), localizeMsg(err.Error())))
				return
			}
		}
		a.setNotice(Tf("%s is ready for offline translation.", langName(ff)))
	}()
}

// prefetchMyLanguage quietly downloads the reader's own language when it
// isn't English, since every translation into it needs en→it.
func (a *App) prefetchMyLanguage() {
	a.mu.Lock()
	ff := ffCode(a.cfg.MyLang)
	a.mu.Unlock()
	if ff == "en" || a.engineMode() != "offline" || a.off.Engine == nil {
		return
	}
	if !pairReady(a.off.pairDir("en", ff)) {
		go a.off.ensurePair("en", ff)
	}
}

// setClickThrough makes the overlay ignore (or take) mouse clicks.
func setClickThrough(on bool) {
	on = on || autoHide.hidden // a faded-out overlay never catches clicks
	const wsExTransparent = 0x20
	gwlExStyle := -20
	idx := uintptr(gwlExStyle)
	st, _, _ := pGetWindowLongPtrW.Call(app.overlay, idx)
	nst := st &^ wsExTransparent
	if on {
		nst |= wsExTransparent
	}
	if nst != st {
		pSetWindowLongPtrW.Call(app.overlay, idx, nst)
	}
}

func toggleClickThrough() {
	app.mu.Lock()
	app.cfg.ClickThrough = !app.cfg.ClickThrough
	on := app.cfg.ClickThrough
	cfg := app.cfg
	app.mu.Unlock()
	saveConfig(cfg)
	setClickThrough(on)
	if on {
		app.setNotice(T("Click-through is on: clicks go to WoW. Ctrl+Shift+T to reply; Esc returns to WoW."))
	} else {
		app.setNotice(T("Click-through is off."))
	}
}

// messageMenu is the right-click menu on a message.
func messageMenu(e *Entry) {
	name := shortName(e.Msg.Sender)
	var items []MenuItem
	if name != "" {
		items = append(items, MenuItem{Label: name, Title: true})
	}
	if e.Translated != "" {
		items = append(items, MenuItem{Label: T("Copy translation"), ID: cmdMsgCopy})
	}
	items = append(items, MenuItem{Label: T("Copy original"), ID: cmdMsgCopyOrig})
	if !e.Own {
		items = append(items, MenuItem{Label: Tf("Reply to %s", name), ID: cmdMsgReply, Tip: T("Your next reply goes to this player, in their language.")})
		if e.Msg.Sender != "" {
			items = append(items, MenuItem{Sep: true}, MenuItem{Label: Tf("Mute %s", name), Hint: T("hide their messages"), ID: cmdMsgMute,
				Tip: T("Hide everything this player sends (spam, gold sellers). Tray icon → Unmute everyone to undo.")})
		}
	}
	cmd := showMenuAtCursor(items)
	switch int(cmd) {
	case cmdMsgCopy:
		if setClipboard(app.overlay, e.Translated) {
			app.setNotice(T("Translation copied."))
		}
	case cmdMsgCopyOrig:
		if setClipboard(app.overlay, e.Msg.Text) {
			app.setNotice(T("Original copied."))
		}
	case cmdMsgReply:
		app.mu.Lock()
		selected := app.reply.set() && app.reply.Sender == e.Msg.Sender && app.reply.Type == e.Msg.Type &&
			app.reply.ChanNum == e.Msg.ChanNum
		app.mu.Unlock()
		if !selected {
			app.pickReply(e)
		}
		showOverlay(true, true)
	case cmdMsgMute:
		app.mute(e.Msg.Sender)
	}
}

func (a *App) isMuted(sender string) bool {
	if sender == "" {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, m := range a.cfg.Muted {
		if strings.EqualFold(m, sender) {
			return true
		}
	}
	return false
}

func (a *App) mute(sender string) {
	a.mu.Lock()
	a.cfg.Muted = append(a.cfg.Muted, sender)
	if len(a.cfg.Muted) > 500 {
		a.cfg.Muted = a.cfg.Muted[len(a.cfg.Muted)-500:]
	}
	kept := a.entries[:0]
	for _, e := range a.entries {
		if e.Own || !strings.EqualFold(e.Msg.Sender, sender) {
			kept = append(kept, e)
		}
	}
	a.entries = kept
	if strings.EqualFold(a.reply.Sender, sender) {
		a.clearReplyLocked()
	}
	cfg := a.cfg
	a.mu.Unlock()
	saveConfig(cfg)
	a.setNotice(Tf("%s is muted. Undo: tray icon → Unmute everyone.", shortName(sender)))
	a.refresh()
}

// historyOn is true unless turned off (caller may or may not hold mu; it only reads).
func (a *App) historyOn() bool { return a.cfg.History != "off" }

// logHistory appends a translated line to %APPDATA%\Parley\history\YYYY-MM-DD.txt.
func (a *App) logHistory(e *Entry) {
	a.mu.Lock()
	on := a.historyOn()
	a.mu.Unlock()
	if !on || e == nil {
		return
	}
	dir := filepath.Join(configDir(), "history")
	if os.MkdirAll(dir, 0o755) != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(dir, e.At.Format("2006-01-02")+".txt"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	who := shortName(e.Msg.Sender)
	if e.Own {
		who = Tf("You → %s", who)
	}
	line := fmt.Sprintf("[%s] [%s] %s (%s)\r\n    %s\r\n    = %s\r\n",
		e.At.Format(time.TimeOnly), entryTag(e), who, e.Detected,
		oneLine(e.Msg.Text), oneLine(e.Translated))
	f.WriteString(line)
}

func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }
