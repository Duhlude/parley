// Package addon embeds the WoW addon so Parley.exe can install and update it.
package addon

import "embed"

// FS holds the Parley addon folder (Parley/...).
//
//go:embed Parley
var FS embed.FS
