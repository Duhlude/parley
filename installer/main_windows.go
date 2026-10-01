package main

// Parley-Setup.exe: a small per-user installer (no admin rights needed).
//   Parley-Setup.exe              install or update
//   Parley-Setup.exe /uninstall   remove Parley
//   Parley-Setup.exe /update      quiet update started by Parley itself: no
//                                 questions, then Parley is started again
// Installs to %LOCALAPPDATA%\Programs\Parley, adds Start menu and desktop
// shortcuts and an entry in Windows' "Installed apps" list.

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

//go:embed payload
var payload embed.FS

const version = "1.9.3"

// wowFlavors: the WoW game folders Parley installs its addon into (keep in
// step with app/config.go).
var wowFlavors = []string{"_classic_era_", "_anniversary_", "_classic_", "_retail_", "_classic_beta_"}

var (
	user32       = syscall.NewLazyDLL("user32.dll")
	pMessageBoxW = user32.NewProc("MessageBoxW")
	pFindWindowW = user32.NewProc("FindWindowW")
	pPostMessage = user32.NewProc("PostMessageW")
)

const (
	mbOK          = 0x0
	mbYesNo       = 0x4
	mbRetryCancel = 0x5
	mbInfo        = 0x40
	mbWarn        = 0x30
	mbQuestion    = 0x20
	idYes         = 6
	idRetry       = 4
)

func u16(s string) uintptr {
	p, _ := syscall.UTF16PtrFromString(s)
	return uintptr(unsafe.Pointer(p))
}

func box(text string, flags uintptr) int {
	r, _, _ := pMessageBoxW.Call(0, u16(text), u16(T("Parley Setup")), flags)
	return int(r)
}

func installDir() string {
	return filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Parley")
}

// quiet keeps a helper program's console window from flashing up.
func quiet(cmd *exec.Cmd) *exec.Cmd {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	return cmd
}

func hidden(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	return cmd.Run()
}

// closeParley asks a running Parley to quit and waits for it.
func closeParley() bool {
	for i := 0; i < 3; i++ {
		h, _, _ := pFindWindowW.Call(u16("ParleyOverlay"), 0)
		if h == 0 {
			return true
		}
		pPostMessage.Call(h, 0x8000+9, 0, 0)
		for j := 0; j < 20; j++ {
			time.Sleep(150 * time.Millisecond)
			if h, _, _ := pFindWindowW.Call(u16("ParleyOverlay"), 0); h == 0 {
				time.Sleep(300 * time.Millisecond) // let it release its files
				return true
			}
		}
		if box(T("Parley is still running. Quit it from its tray icon, then click Retry."), mbRetryCancel|mbWarn) != idRetry {
			return false
		}
	}
	return false
}

func shortcut(lnk, target, dir string) error {
	ps := "$s=(New-Object -ComObject WScript.Shell).CreateShortcut('" + strings.ReplaceAll(lnk, "'", "''") + "');" +
		"$s.TargetPath='" + strings.ReplaceAll(target, "'", "''") + "';" +
		"$s.WorkingDirectory='" + strings.ReplaceAll(dir, "'", "''") + "';" +
		"$s.Description='Parley: live chat translation for WoW';$s.Save()"
	return hidden("powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", ps)
}

func desktopDir() string {
	out, err := quiet(exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command",
		"[Environment]::GetFolderPath('Desktop')")).Output()
	if err == nil {
		if d := strings.TrimSpace(string(out)); d != "" {
			return d
		}
	}
	return filepath.Join(os.Getenv("USERPROFILE"), "Desktop")
}

func startMenuLnk() string {
	return filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs", "Parley.lnk")
}

const uninstKey = `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\Parley`
const runKey = `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`

// copyPayload writes the app files into dir as one unit: each file being
// replaced is first renamed to *.old (Windows allows that even for a program
// that's still running), and if anything fails the old files are put back,
// so a failed update never leaves a mix of old and new files.
func copyPayload(dir string) error {
	cleanOld(dir)
	type swap struct{ dst, old string }
	var done []swap
	rollback := func() {
		for i := len(done) - 1; i >= 0; i-- {
			os.Remove(done[i].dst)
			if done[i].old != "" {
				os.Rename(done[i].old, done[i].dst)
			}
		}
	}
	err := fs.WalkDir(payload, "payload", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(strings.TrimPrefix(p, "payload"), "/")
		if rel == "" {
			return nil
		}
		dst := filepath.Join(dir, filepath.FromSlash(rel))
		if d.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		b, err := payload.ReadFile(p)
		if err != nil {
			return err
		}
		sw := swap{dst: dst}
		if exists(dst) {
			sw.old = dst + ".old"
			os.Remove(sw.old)
			var rerr error
			for try := 0; try < 20; try++ { // a file may stay locked for a moment
				if rerr = os.Rename(dst, sw.old); rerr == nil {
					break
				}
				time.Sleep(500 * time.Millisecond)
			}
			if rerr != nil {
				return rerr
			}
		}
		done = append(done, sw)
		return os.WriteFile(dst, b, 0o755)
	})
	if err != nil {
		rollback()
		return err
	}
	cleanOld(dir)
	return nil
}

// cleanOld removes the *.old files a previous update left behind (they
// can't be deleted while the old Parley is still running).
func cleanOld(dir string) {
	filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(p, ".old") {
			os.Remove(p)
		}
		return nil
	})
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }

func install(update bool) {
	dir := installDir()
	if !update && box(fmt.Sprintf(T("Install Parley %s (live WoW chat translation)?\n\nIt goes into:\n%s\n\nNo admin rights needed. If Parley is already installed it's updated and your settings are kept."), version, dir),
		mbYesNo|mbQuestion) != idYes {
		return
	}
	if !closeParley() {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		box(T("Couldn't create the install folder:")+"\n"+err.Error(), mbOK|mbWarn)
		return
	}
	if err := copyPayload(dir); err != nil {
		box(T("Install failed while copying files:")+"\n"+err.Error(), mbOK|mbWarn)
		if update { // nothing was changed: bring the old Parley back
			if exe := filepath.Join(dir, "Parley.exe"); exists(exe) {
				cmd := exec.Command(exe)
				cmd.Dir = dir
				cmd.Start()
			}
		}
		return
	}
	// keep a copy of the installer as the uninstaller
	self, _ := os.Executable()
	uninst := filepath.Join(dir, "Uninstall Parley.exe")
	if b, err := os.ReadFile(self); err == nil && !strings.EqualFold(self, uninst) {
		os.WriteFile(uninst, b, 0o755)
	}
	exe := filepath.Join(dir, "Parley.exe")
	addonMsg := installAddonToWow()
	shortcut(startMenuLnk(), exe, dir)
	if desk := filepath.Join(desktopDir(), "Parley.lnk"); !update || exists(desk) { // an update doesn't bring back a deleted shortcut
		shortcut(desk, exe, dir)
	}

	hidden("reg", "add", uninstKey, "/v", "DisplayName", "/d", "Parley", "/f")
	hidden("reg", "add", uninstKey, "/v", "DisplayVersion", "/d", version, "/f")
	hidden("reg", "add", uninstKey, "/v", "Publisher", "/d", "Magichouse Media", "/f")
	hidden("reg", "add", uninstKey, "/v", "DisplayIcon", "/d", exe, "/f")
	hidden("reg", "add", uninstKey, "/v", "InstallLocation", "/d", dir, "/f")
	hidden("reg", "add", uninstKey, "/v", "UninstallString", "/d", `"`+uninst+`" /uninstall`, "/f")
	hidden("reg", "add", uninstKey, "/v", "NoModify", "/t", "REG_DWORD", "/d", "1", "/f")
	hidden("reg", "add", uninstKey, "/v", "NoRepair", "/t", "REG_DWORD", "/d", "1", "/f")

	if update { // keep the start-with-Windows choice; start the new version
		cmd := exec.Command(exe, "--updated")
		cmd.Dir = dir
		if cmd.Start() != nil {
			box(T("Parley is installed."), mbOK|mbInfo)
		}
		return
	}
	if box(T("Start Parley automatically when Windows starts?"), mbYesNo|mbQuestion) == idYes {
		hidden("reg", "add", runKey, "/v", "Parley", "/d", `"`+exe+`"`, "/f")
	} else {
		hidden("reg", "delete", runKey, "/v", "Parley", "/f")
	}
	if box(T("Parley is installed.")+"\n\n"+addonMsg+"\n\n"+T("Next: in WoW type /parley test. Parley translates offline, no account needed; each language downloads once the first time it's used.")+"\n\n"+T("Start Parley now?"), mbYesNo|mbInfo) == idYes {
		cmd := exec.Command(exe)
		cmd.Dir = dir
		cmd.Start()
	}
}

func uninstall() {
	if box(T("Remove Parley from this PC?")+"\n\n"+T("Your settings in %APPDATA%\\Parley and the WoW addon are left in place."),
		mbYesNo|mbQuestion) != idYes {
		return
	}
	if !closeParley() {
		return
	}
	dir := installDir()
	os.Remove(startMenuLnk())
	os.Remove(filepath.Join(desktopDir(), "Parley.lnk"))
	hidden("reg", "delete", uninstKey, "/f")
	hidden("reg", "delete", runKey, "/v", "Parley", "/f")
	// The uninstaller lives in the folder it deletes, so let cmd finish the job after we exit.
	cmd := exec.Command("cmd", "/c", "ping 127.0.0.1 -n 3 >nul & rmdir /s /q \""+dir+"\"")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	cmd.Start()
	box(T("Parley has been removed."), mbOK|mbInfo)
}

func main() {
	for _, a := range os.Args[1:] {
		if strings.EqualFold(a, "/uninstall") || strings.EqualFold(a, "--uninstall") {
			uninstall()
			return
		}
	}
	update := false
	for _, a := range os.Args[1:] {
		if strings.EqualFold(a, "/update") || strings.EqualFold(a, "--update") {
			update = true
		}
	}
	install(update)
}

// wowFolders lists likely game folders (every supported WoW version): Battle.net's
// registry entry first, then the default locations.
func wowFolders() []string {
	var out []string
	for _, key := range []string{`HKLM\SOFTWARE\WOW6432Node\Blizzard Entertainment\World of Warcraft`,
		`HKLM\SOFTWARE\Blizzard Entertainment\World of Warcraft`} {
		b, err := quiet(exec.Command("reg", "query", key, "/v", "InstallPath")).Output()
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(b), "\n") {
			if i := strings.Index(line, "REG_SZ"); i >= 0 {
				p := strings.TrimSpace(line[i+6:])
				// InstallPath points at e.g. ...\World of Warcraft\_retail_\
				root := filepath.Dir(strings.TrimRight(p, `\`))
				for _, fl := range wowFlavors {
					out = append(out, filepath.Join(root, fl))
				}
			}
		}
	}
	for _, drive := range []string{"C", "D", "E", "F"} {
		for _, rel := range []string{`Program Files (x86)\World of Warcraft`, `Program Files\World of Warcraft`,
			`World of Warcraft`, `Games\World of Warcraft`, `Games\WoW\World of Warcraft`, `Battle.net\World of Warcraft`} {
			for _, fl := range wowFlavors {
				out = append(out, drive+`:\`+rel+`\`+fl)
			}
		}
	}
	return out
}

// installAddonToWow copies the embedded addon into Interface\AddOns\Parley.
func installAddonToWow() string {
	var done []string
	seen := map[string]bool{}
	for _, w := range wowFolders() {
		if seen[strings.ToLower(w)] {
			continue
		}
		seen[strings.ToLower(w)] = true
		if st, err := os.Stat(w); err != nil || !st.IsDir() {
			continue
		}
		dest := filepath.Join(w, "Interface", "AddOns", "Parley")
		if err := os.MkdirAll(dest, 0o755); err != nil {
			continue
		}
		ok, n := true, 0
		fs.WalkDir(payload, "payload/addon/Parley", func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			b, err := payload.ReadFile(p)
			out := filepath.Join(dest, filepath.FromSlash(strings.TrimPrefix(p, "payload/addon/Parley/")))
			if err != nil || os.MkdirAll(filepath.Dir(out), 0o755) != nil || os.WriteFile(out, b, 0o644) != nil {
				ok = false
			}
			n++
			return nil
		})
		if ok && n > 0 {
			done = append(done, dest)
		}
	}
	if len(done) > 0 {
		return T("The WoW addon was installed to:") + "\n" + strings.Join(done, "\n") + "\n" + T("(type /reload if WoW is running).")
	}
	return T("Couldn't find WoW automatically. Parley will offer to install the addon when you set your WoW folder in its settings; a copy is also in the addon folder inside:") + "\n" + installDir()
}
