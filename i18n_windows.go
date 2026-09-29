package main

import (
	"syscall"
	"unsafe"
)

var pGetUserDefaultLocaleName = syscall.NewLazyDLL("kernel32.dll").NewProc("GetUserDefaultLocaleName")
var pGetUserDefaultUILanguage = syscall.NewLazyDLL("kernel32.dll").NewProc("GetUserDefaultUILanguage")
var pLCIDToLocaleName = syscall.NewLazyDLL("kernel32.dll").NewProc("LCIDToLocaleName")

// systemLocaleName is the Windows display language as a locale name such
// as "pt-BR" (falling back to the user's regional format).
func systemLocaleName() string {
	buf := make([]uint16, 85)
	if lid, _, _ := pGetUserDefaultUILanguage.Call(); lid != 0 && pLCIDToLocaleName.Find() == nil {
		if n, _, _ := pLCIDToLocaleName.Call(lid, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), 0); n > 0 {
			return syscall.UTF16ToString(buf)
		}
	}
	if n, _, _ := pGetUserDefaultLocaleName.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf))); n > 0 {
		return syscall.UTF16ToString(buf)
	}
	return "en-US"
}

func systemUILang() string { return uiLangForLocale(systemLocaleName()) }
