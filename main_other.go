//go:build !windows

package main

import (
	"fmt"
	"os/exec"
)

// Parley is a Windows app; this stub only lets the portable parts
// (decoder, language check, DeepL client) build and test elsewhere.
func main() { fmt.Println("Parley runs on Windows.") }

func hideWindow(cmd *exec.Cmd) {}

func systemUILang() string     { return "en" }
func systemLocaleName() string { return "en-US" }
