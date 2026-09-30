package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestWowInstalls(t *testing.T) {
	root := filepath.Join(t.TempDir(), "World of Warcraft")
	era, retail := filepath.Join(root, "_classic_era_"), filepath.Join(root, "_retail_")
	os.MkdirAll(era, 0o755)
	if got := wowInstalls(era); !reflect.DeepEqual(got, []string{era}) {
		t.Errorf("only era: %v", got)
	}
	os.MkdirAll(retail, 0o755)
	if got := wowInstalls(era); !reflect.DeepEqual(got, []string{era, retail}) {
		t.Errorf("era picked: %v", got)
	}
	if got := wowInstalls(retail); !reflect.DeepEqual(got, []string{retail, era}) {
		t.Errorf("retail picked: %v", got)
	}
	if got := wowInstalls(root); !reflect.DeepEqual(got, []string{era, retail}) {
		t.Errorf("root picked: %v", got)
	}
	mists, anniv := filepath.Join(root, "_classic_"), filepath.Join(root, "_anniversary_")
	os.MkdirAll(mists, 0o755)
	os.MkdirAll(anniv, 0o755)
	if got := wowInstalls(mists); !reflect.DeepEqual(got, []string{mists, era, anniv, retail}) {
		t.Errorf("progression classic picked: %v", got)
	}
	os.RemoveAll(mists)
	os.RemoveAll(anniv)
	dests, err := installAddonAll(root)
	if err != nil || len(dests) != 2 {
		t.Fatalf("installAddonAll: %v %v", dests, err)
	}
	for _, w := range []string{era, retail} {
		if addonInstalledVersion(w) != embeddedAddonVersion() {
			t.Errorf("%s: version %q", w, addonInstalledVersion(w))
		}
		if _, err := os.Stat(filepath.Join(w, "Interface", "AddOns", "Parley", "Locale.lua")); err != nil {
			t.Error(err)
		}
	}
	if _, err := installAddonAll(filepath.Join(root, "nope")); err == nil {
		t.Error("expected an error for a missing folder")
	}
}
