package main

import (
	"strings"
	"testing"
)

func TestLocalizePlaces(t *testing.T) {
	cases := []struct{ in, target, want string }{
		{"ищу группу в Мертвые копи, нужен танк", "EN-US", "ищу группу в \x02Deadmines\x03, нужен танк"},
		{"кто идет в Мертвых копях?", "EN-US", "кто идет в \x02Deadmines\x03?"},
		{"alguém sabe onde fica o treinador de mago em Ventobravo?", "EN-US", "alguém sabe onde fica o treinador de mago em \x02Stormwind\x03?"},
		{"alguien me puede abrir un portal a Forjaz? pago", "EN-US", "alguien me puede abrir un portal a \x02Ironforge\x03? pago"},
		{"cherche groupe pour les Mortemines", "EN-US", "cherche groupe pour \x02Deadmines\x03"},
		{"有人去死亡矿井吗？", "EN-US", "有人去\x02Deadmines\x03吗？"},
		{"탱커 구합니다 데드마인 가실 분", "EN-US", "탱커 구합니다 \x02Deadmines\x03 가실 분"},
		{"タンク募集中です、デッドマインズ行きます", "EN-US", "タンク募集中です、\x02Deadmines\x03行きます"},
		{"LFG Deadmines, meet at Stormwind", "PT-BR", "LFG \x02Minas Mortas\x03, meet at \x02Ventobravo\x03"},
		{"meet me in Ironforge", "RU", "meet me in \x02Стальгорн\x03"},
		{"meet me in Ironforge", "ZH-HANT", "meet me in \x02鐵爐堡\x03"},
		{"Forjazmente", "EN-US", "Forjazmente"}, // not a whole word
		{"nothing here", "PT-BR", "nothing here"},
		// Burning Crusade and Mists of Pandaria
		{"bora Karazhan hoje? preciso de tank", "EN-US", "bora \x02Karazhan\x03 hoje? preciso de tank"},
		{"alguém pra Muralha Fogo do Inferno?", "EN-US", "alguém pra \x02Hellfire Ramparts\x03?"},
		{"suche Gruppe für den Schwarzen Tempel", "EN-US", "suche Gruppe für den \x02Black Temple\x03"},
		{"кто в Каражан? нужен хил", "EN-US", "кто в \x02Karazhan\x03? нужен хил"},
		{"LFM Siege of Orgrimmar", "DE", "LFM \x02Schlacht um Orgrimmar\x03"},
		{"meet at Shattrath", "PT-BR", "meet at \x02Shattrath City\x03"},
		{"on va au Temple du Serpent de jade ?", "EN-US", "on va au \x02Temple of the Jade Serpent\x03 ?"},
		{"青龙寺来人", "EN-US", "\x02Temple of the Jade Serpent\x03来人"},
		{"meet in the Vale of Eternal Blossoms", "KO", "meet in the \x02영원꽃 골짜기\x03"},
		{"who's in Outland?", "RU", "who's in \x02Запределье\x03?"},
	}
	for _, c := range cases {
		if got := localizePlaces(c.in, c.target); got != c.want {
			t.Errorf("localizePlaces(%q, %s) = %q, want %q", c.in, c.target, got, c.want)
		}
	}
}

func TestLocalizeRoleVerbs(t *testing.T) {
	cases := []struct{ in, target, want string }{
		{"Sure, I can tank. Send me an invite.", "PT-BR", "Sure, I can \x02tankar\x03. Send me an invite."},
		{"I'll heal, you tank", "DE", "I'll \x02heilen\x03, you \x02tanken\x03"},
		{"we need a tank", "RU", "we need a tank"}, // noun stays
		{"can tank", "JA", "can tank"},
	}
	for _, c := range cases {
		if got := localizeRoleVerbs(c.in, c.target); got != c.want {
			t.Errorf("localizeRoleVerbs(%q, %s) = %q, want %q", c.in, c.target, got, c.want)
		}
	}
	if got := toHTML(keep("Deadmines") + " & tank"); got != "<code>Deadmines</code> &amp; <code>tank</code>" {
		t.Errorf("toHTML keep: %q", got)
	}
	if got := toXML("in "+keep("Stormwind")+" tank", "EN-US"); got != "in Stormwind tank" {
		t.Errorf("toXML keep: %q", got)
	}
}

func TestFromXMLGlue(t *testing.T) {
	if got := fromXML("look in<x>[Linen Cloth]</x>now"); got != "look in [Linen Cloth] now" {
		t.Errorf("fromXML glue: %q", got)
	}
}

func TestKeepWords(t *testing.T) {
	defer setKeepWords("")
	setKeepWords("Sombra Eterna, Bob; 暗影 , ,Bob")
	if got := keepWordList("Sombra Eterna, Bob; 暗影 , ,Bob"); len(got) != 3 {
		t.Errorf("list %q", got)
	}
	cases := []struct{ in, want string }{
		{"entrem na sombra eterna, galera", "entrem na \x04sombra eterna\x05, galera"},
		{"Sombra  Eterna recruta!", "\x04Sombra  Eterna\x05 recruta!"},
		{"Bob e Bobby vão", "\x04Bob\x05 e Bobby vão"},
		{"欢迎加入暗影公会", "欢迎加入\x04暗影\x05公会"},
		{"nada aqui", "nada aqui"},
	}
	for _, c := range cases {
		if got := prepareIncoming(c.in, "EN-US"); got != c.want {
			t.Errorf("prepareIncoming(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	// the player's words win over place names inside them
	setKeepWords("Ironforge Brewers")
	if got := prepareReply("join Ironforge Brewers in Ironforge", "DE"); got != "join \x04Ironforge Brewers\x05 in \x02Eisenschmiede\x03" {
		t.Errorf("place inside a kept name: %q", got)
	}
	setKeepWords("Sombra Eterna")
	in := prepareReply("join Sombra Eterna, we raid", "PT-BR")
	if stripKeep(in) != "join Sombra Eterna, we raid" {
		t.Errorf("strip %q", stripKeep(in))
	}
	if x := toXML(in, "EN-US"); x != "join <x>Sombra Eterna</x>, we <x>raid</x>" && x != "join <x>Sombra Eterna</x>, we raid" {
		t.Errorf("toXML %q", x)
	}
	if h := toHTML(in); !strings.Contains(h, "<code>Sombra Eterna</code>") {
		t.Errorf("toHTML %q", h)
	}
	if a := toAzureHTML(in, "PT-BR"); !strings.Contains(a, `<span translate="no">Sombra Eterna</span>`) {
		t.Errorf("azure %q", a)
	}
}

func TestSameMeaning(t *testing.T) {
	if !sameMeaning("Sure, I can tank!", "sure i can tank") || sameMeaning("I can tank", "I can heal") {
		t.Error("sameMeaning")
	}
}
