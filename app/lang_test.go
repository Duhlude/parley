package main

import "testing"

func TestPortuguese(t *testing.T) {
	cases := map[string]bool{ // text -> should translate
		"vc quer ir na dungeon cmg?":           true,
		"alguem pra fazer deadmines? kkkkk":    true,
		"blz mano, vlw pela ajuda":             true,
		"Alguém pode me ajudar com a missão?":  true,
		"lf tank for deadmines pst":            false,
		"anyone want to do the quest with me?": false,
		"tô upando, bora?":                     true,
	}
	for txt, want := range cases {
		eng, _ := Guess(txt)
		if eng == want {
			t.Errorf("%q: english=%v", txt, eng)
		}
	}
	if got := ExpandSlang("vc quer ir cmg? kkkk blz"); got != "você quer ir comigo? haha beleza" {
		t.Errorf("expand: %q", got)
	}
}

func TestEnglishSlangSkipped(t *testing.T) {
	for _, txt := range []string{"crank dat soulja boy", "o kume bye ahhhh", "don't die", "ta mate", "lf dungeon lvl 20"} {
		if eng, _ := Guess(txt); !eng {
			t.Errorf("%q should be treated as English", txt)
		}
	}
	for _, txt := range []string{"hola alguien quiere ir", "ik zoek een groep", "je cherche un groupe", "bora fazer dg mano"} {
		if eng, _ := Guess(txt); eng {
			t.Errorf("%q should be translated", txt)
		}
	}
	if !sameText("Crank dat, Soulja Boy!", "crank dat soulja boy") {
		t.Error("sameText")
	}
}

func TestTradeChatNotEnglish(t *testing.T) {
	for _, s := range []string{
		"Vendendo bolsas de seda, 40 prata cada. Me manda mensagem!",
		"vendo bolsa 10 slots barato chama pv",
		"Vendiendo bolsas, 40 plata cada, mensaje",
	} {
		if en, _ := Guess(s); en {
			t.Errorf("Guess(%q) says English", s)
		}
	}
	for _, s := range []string{"WTS silk bags 40s each pst", "selling bags cheap, whisper me"} {
		if en, _ := Guess(s); !en {
			t.Errorf("Guess(%q) says foreign", s)
		}
	}
	if g := guessLatinLang("Vendendo bolsas de seda, 40 prata cada. Me manda mensagem!"); g != "pt" {
		t.Errorf("guessLatinLang = %q", g)
	}
}
