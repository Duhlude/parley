package main

// Quick replies: common chat phrases, already translated, so they go out
// instantly and read naturally. Languages not listed here are translated by
// the current engine like a typed reply.

import "strings"

type phrase struct {
	EN string
	in map[string]string // phrase column (see phraseCol) -> text
}

var phrasebook = []phrase{
	{"Invite me, please", map[string]string{"pt": "Me convida, por favor", "es": "Invítame, por favor", "ru": "Пригласи меня, пожалуйста", "de": "Lad mich bitte ein", "fr": "Invite-moi, s'il te plaît", "it": "Invitami, per favore", "ko": "초대 부탁드려요", "zhCN": "请邀请我一下", "zhTW": "請邀請我一下", "ja": "招待お願いします"}},
	{"On my way!", map[string]string{"pt": "Estou indo!", "es": "¡Voy para allá!", "ru": "Уже иду!", "de": "Bin unterwegs!", "fr": "J'arrive !", "it": "Sto arrivando!", "ko": "가는 중이에요!", "zhCN": "马上来！", "zhTW": "馬上來！", "ja": "今向かってます！"}},
	{"One moment, please", map[string]string{"pt": "Um momento, por favor", "es": "Un momento, por favor", "ru": "Минутку, пожалуйста", "de": "Einen Moment, bitte", "fr": "Un instant, s'il te plaît", "it": "Un momento, per favore", "ko": "잠시만요", "zhCN": "请稍等", "zhTW": "請稍等", "ja": "少々お待ちください"}},
	{"Thanks for the group!", map[string]string{"pt": "Valeu pelo grupo!", "es": "¡Gracias por el grupo!", "ru": "Спасибо за группу!", "de": "Danke für die Gruppe!", "fr": "Merci pour le groupe !", "it": "Grazie per il gruppo!", "ko": "파티 감사합니다!", "zhCN": "谢谢组队！", "zhTW": "謝謝組隊！", "ja": "パーティーありがとうございました！"}},
	{"Thank you!", map[string]string{"pt": "Obrigado!", "es": "¡Gracias!", "ru": "Спасибо!", "de": "Danke!", "fr": "Merci !", "it": "Grazie!", "ko": "감사합니다!", "zhCN": "谢谢！", "zhTW": "謝謝！", "ja": "ありがとう！"}},
	{"Sure, no problem", map[string]string{"pt": "Claro, sem problema", "es": "Claro, sin problema", "ru": "Конечно, без проблем", "de": "Klar, kein Problem", "fr": "Bien sûr, pas de souci", "it": "Certo, nessun problema", "ko": "네, 문제없어요", "zhCN": "当然，没问题", "zhTW": "當然，沒問題", "ja": "もちろん、大丈夫です"}},
	{"No, thank you", map[string]string{"pt": "Não, obrigado", "es": "No, gracias", "ru": "Нет, спасибо", "de": "Nein, danke", "fr": "Non, merci", "it": "No, grazie", "ko": "아니요, 괜찮아요", "zhCN": "不用了，谢谢", "zhTW": "不用了，謝謝", "ja": "いいえ、結構です"}},
	{"Sorry, I don't speak your language well", map[string]string{"pt": "Desculpa, não falo bem português", "es": "Lo siento, no hablo bien español", "ru": "Извини, я плохо говорю по-русски", "de": "Sorry, ich spreche nicht gut Deutsch", "fr": "Désolé, je ne parle pas bien français", "it": "Scusa, non parlo bene l'italiano", "ko": "죄송해요, 한국어를 잘 못해요", "zhCN": "抱歉，我中文不太好", "zhTW": "抱歉，我中文不太好", "ja": "すみません、日本語があまり話せません"}},
	{"Where are you?", map[string]string{"pt": "Onde você está?", "es": "¿Dónde estás?", "ru": "Где ты?", "de": "Wo bist du?", "fr": "Où es-tu ?", "it": "Dove sei?", "ko": "어디 계세요?", "zhCN": "你在哪儿？", "zhTW": "你在哪裡？", "ja": "どこにいますか？"}},
	{"Good luck, have fun!", map[string]string{"pt": "Boa sorte, divirta-se!", "es": "¡Buena suerte, diviértete!", "ru": "Удачи и хорошей игры!", "de": "Viel Glück und viel Spaß!", "fr": "Bonne chance, amuse-toi bien !", "it": "Buona fortuna, divertiti!", "ko": "행운을 빌어요, 즐겜하세요!", "zhCN": "祝好运，玩得开心！", "zhTW": "祝好運，玩得開心！", "ja": "頑張って、楽しんで！"}},
}

// phraseCol picks the phrasebook column for a reply language code.
func phraseCol(code string) string {
	c := strings.ToUpper(code)
	if baseLang(c) == "JA" {
		return "ja"
	}
	if baseLang(c) == "EN" {
		return "en"
	}
	col := placeLocale(c)
	if col == "en" { // a language the phrasebook doesn't have
		return ""
	}
	return col
}

// ready returns the built-in translation for a reply language ("" if none).
func (p phrase) ready(code string) string {
	col := phraseCol(code)
	if col == "en" {
		return p.EN
	}
	return p.in[col]
}
