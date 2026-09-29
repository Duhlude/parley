package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"
)

var pGetShortPathNameW = syscall.NewLazyDLL("kernel32.dll").NewProc("GetShortPathNameW")

// asciiPath returns an equivalent path without non-ASCII characters (the
// 8.3 short name) for C/C++ tools that open files with narrow strings, e.g.
// when the Windows user name is "João". Falls back to the original path.
func asciiPath(p string) string {
	if isASCII(p) {
		return p
	}
	in, err := syscall.UTF16PtrFromString(p)
	if err != nil {
		return p
	}
	buf := make([]uint16, 1024)
	n, _, _ := pGetShortPathNameW.Call(uintptr(unsafe.Pointer(in)), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n == 0 || int(n) >= len(buf) {
		return p
	}
	return syscall.UTF16ToString(buf[:n])
}

// offlineModelsDir picks where offline models live: %APPDATA%\Parley\models\translate,
// or, if that path can't be written in plain ASCII, %ProgramData%\Parley\models\translate.
func offlineModelsDir() string {
	dir := filepath.Join(configDir(), "models", "translate")
	if isASCII(dir) {
		return dir
	}
	os.MkdirAll(dir, 0o755)
	if s := asciiPath(dir); isASCII(s) {
		return s
	}
	if pd := os.Getenv("ProgramData"); pd != "" && isASCII(pd) {
		return filepath.Join(pd, "Parley", "models", "translate")
	}
	return dir
}

// hideWindow keeps the engine's console from flashing up.
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
}
