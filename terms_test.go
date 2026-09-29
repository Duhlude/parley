package main

import "testing"

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
