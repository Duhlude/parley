package main

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestSaveConfigConcurrent(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir()) // os.UserConfigDir on Linux
	t.Setenv("APPDATA", os.Getenv("XDG_CONFIG_HOME"))
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c := defaultConfig()
			c.KeepWords = "Sombra Eterna"
			c.FontSize = 10 + i
			if err := saveConfig(c); err != nil {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()
	if c := loadConfig(); c.KeepWords != "Sombra Eterna" {
		t.Fatalf("lost settings: %+v", c)
	}
	left, _ := filepath.Glob(filepath.Join(configDir(), "*.tmp"))
	if len(left) != 0 {
		t.Errorf("temp files left: %v", left)
	}
	// a damaged settings.json falls back to the last good copy
	c := defaultConfig()
	c.AlertWords = "healer"
	saveConfig(c)
	saveConfig(c) // the .bak now holds a good copy with the alert words
	os.WriteFile(filepath.Join(configDir(), "settings.json"), []byte("{broken"), 0o600)
	if got := loadConfig(); got.AlertWords != "healer" {
		t.Fatalf("backup not used: %+v", got)
	}
}
