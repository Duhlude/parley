package main

import (
	"strings"
	"testing"
)

func TestExpandShortcuts(t *testing.T) {
	got := expandShortcuts("sry w8 plz, brb [Thunderfury] idk")
	for _, want := range []string{"sorry", "wait", "please", "brb", "[Thunderfury]", "I don't know"} {
		if !strings.Contains(strings.ToLower(got), strings.ToLower(want)) {
			t.Errorf("%q missing %q", got, want)
		}
	}
	if expandShortcuts("secret gladly") != "secret gladly" {
		t.Error("expanded inside words")
	}
}

func TestGloss(t *testing.T) {
	g := glossFor("lf1m tank for SM Cath, w2w pull, bring mats", "PT-BR", 5)
	j := strings.Join(g, " | ")
	for _, want := range []string{"lf1m:", "SM Cath: Mosteiro (Catedral)", "w2w:", "mats:"} {
		if !strings.Contains(j, want) {
			t.Errorf("gloss %q missing %q", j, want)
		}
	}
	if len(glossFor("hello there friend", "EN-US", 5)) != 0 {
		t.Error("gloss on plain text")
	}
	if eng, _ := Guess("ty np gg wp"); !eng {
		t.Error("shortcut chat should count as English")
	}
}
