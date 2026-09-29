package main

// Cheap offline language pre-check so English chat never costs API quota.

import (
	"regexp"
	"strings"
	"unicode"
)

// Guess classifies a message. english=true means "don't bother translating".
// hint is a best-effort DeepL source code ("" = let DeepL detect).
func Guess(text string) (english bool, hint string) {
	var latin, cyr, han, kana, hangul, greek, arabic, other, accented int
	for _, r := range text {
		switch {
		case r < 128:
			if unicode.IsLetter(r) {
				latin++
			}
		case unicode.Is(unicode.Cyrillic, r):
			cyr++
		case unicode.Is(unicode.Hangul, r):
			hangul++
		case unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r):
			kana++
		case unicode.Is(unicode.Han, r):
			han++
		case unicode.Is(unicode.Greek, r):
			greek++
		case unicode.Is(unicode.Arabic, r):
			arabic++
		case unicode.Is(unicode.Latin, r):
			latin++
			accented++
		case unicode.IsLetter(r):
			other++
		}
	}
	switch {
	case cyr > 0 && cyr*2 >= latin:
		return false, "" // RU or UK; let DeepL decide
	case kana > 0:
		return false, "JA"
	case hangul > 0:
		return false, "KO"
	case han > 0:
		return false, "ZH"
	case greek > 0:
		return false, "EL"
	case arabic > 0:
		return false, "AR"
	case other > 0:
		return false, ""
	case accented > 0:
		return false, ""
	case latin == 0:
		return true, "" // numbers, symbols, emotes
	}

	// Plain ASCII: English unless enough words look like another language.
	words := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && r != '\''
	})
	if len(words) == 0 {
		return true, ""
	}
	eng, foreign := 0, 0
	for _, w := range words {
		if isLaugh(w) {
			foreign++
		} else if englishWords[w] {
			eng++
		} else if foreignWords[w] {
			foreign++
		}
	}
	if foreign > 0 && foreign >= eng {
		return false, ""
	}
	if foreign == 0 {
		// Plain ASCII with no recognisable foreign words is treated as
		// English: slang, names, song lyrics, "ahhhh". This avoids DeepL
		// "translating" English banter from exotic languages.
		return true, ""
	}
	return false, ""
}

var englishWords = set(`a about after again all also am an and any anyone are around as at away back bad be because been before being best better big bit both but buy by can cant can't come could coming day did didnt didn't do does dont don't done down each even every everyone for from get gets getting give go going good got great group guild guys had has have he hello help her here hey hi him his how i i'm im if in into is it it's its just keep know last late let lets let's like little look looking lol lot make many may me more most much must my need needs new next nice no not now of off ok okay on one only or other our out over pls please plz quest really right run same see she should so some someone something soon sorry still sure take thank thanks that that's thats the their them then there they thing think this those though time to today too two up us very want wants was way we well went were what when where which who why will with without would yeah yes you you're your ty thx np gg wp brb afk lf lfg lfm lf1m lf2m lf3m wts wtb wtt pst inv invite tank heal healer dps need greed ress rez buff buffs mage warrior rogue priest hunter druid paladin shaman warlock dungeon raid boss mob mobs pull pulling oom omw gz grats gratz summon sum port portal quick fast run runs selling buying price gold g ah hc heroic`)

var foreignWords = set(`hola que qué por para con los las una uno pero muy bien gracias alguien quiere quieres necesito necesitamos tengo tienes vamos donde dónde como cómo esta está estoy soy eres somos amigo gente ayuda mazmorra grupo busco buscando falta faltan você voce obrigado obrigada alguém alguem preciso tenho onde ajuda missão missao estou não nao também tambem ja já pra aqui agora cara galera der das und ich nicht ist ein eine mit auf für fur bitte danke suche gruppe noch wer hast sind le la les des une est pas je tu il nous vous avec pour merci bonjour salut cherche groupe qui oui très tres ciao grazie sono cerco gruppo andiamo aiuto jest nie tak dziekuje szukam grupy kto vc vcs voces vocês pq porque tbm tb blz beleza mano mto muito mt td tudo hj hoje obg obgd vlw valeu tmj flw falou nois nóis bora vamo quero queria sabe tava tá tô tou aí oi eae eai iae salve kd cadê cade agr dps depois entao então msm mesmo ngm ninguem ninguém alguma algum gnt fazer faz fazendo quem qual quando ainda nada coisa mais menos bom noite dia tarde masmorra upar upando ajudar ajudem precisa precisamos procurando bixo bicho mlk moleque slc sla ik het een niet wat maar jij mijn ook zijn hebben goed dank bedankt hallo zoek iemand wil kan groep auch aber oder habe mir mich dich kann gibt jemand hier jetzt kommt gut schon mal doch besoin quelqu'un aide suis veux avoir faire allez ici maintenant aussi quien bueno tambien también ahora aquí puedo puedes quiero hacer ayudar eu meu minha vendendo vendo vende vendem compro comprando compra troco trocando bolsa bolsas prata ouro cobre cada manda mande mensagem chama chamar privado pv barato barata preço preco quanto custa custo pago pagando grátis gratis vendiendo compro plata oro cobre cada mensaje barato precio cuanto cuánto cuesta pago`)

func set(s string) map[string]bool {
	m := map[string]bool{}
	for _, w := range strings.Fields(s) {
		m[w] = true
	}
	return m
}

// Brazilian (and some Spanish) chat shorthand that DeepL may not know.
// Only applied once a message already looks non-English.
var slang = map[string]string{
	// Russian gamer slang
	"спс": "спасибо", "пж": "пожалуйста", "плз": "пожалуйста", "пжл": "пожалуйста", "нзч": "не за что",
	"лс": "личные сообщения", "личку": "личные сообщения", "личка": "личные сообщения",
	"инвайт": "приглашение", "инвайтни": "пригласи", "инвайтните": "пригласите", "пати": "группа",
	"агрить": "провоцировать", "агрите": "провоцируйте", "агри": "провоцируй", "сагрил": "спровоцировал",
	"сагрили": "спровоцировали", "агрнул": "спровоцировал",
	"хил": "лекарь", "хила": "лекаря", "хилу": "лекарю", "хилом": "лекарем", "хилы": "лекари", "хилов": "лекарей",
	"хилер": "лекарь", "хилера": "лекаря", "хилеры": "лекари", "хилить": "лечить",
	"данж": "подземелье", "данжа": "подземелья", "данжи": "подземелья", "го": "пойдём",
	"vc": "você", "vcs": "vocês", "pq": "porque", "tbm": "também", "tb": "também",
	"blz": "beleza", "mto": "muito", "mt": "muito", "td": "tudo", "hj": "hoje",
	"obg": "obrigado", "obgd": "obrigado", "vlw": "obrigado", "valeu": "obrigado", "bora": "vamos", "pras": "para as", "pros": "para os", "flw": "falou", "tmj": "estamos juntos",
	"kd": "cadê", "agr": "agora", "msm": "mesmo", "ngm": "ninguém", "gnt": "gente",
	"sla": "sei lá", "slc": "sem condições", "q": "que", "n": "não", "eh": "é", "tá": "está",
	"ta": "está", "tô": "estou", "dps": "depois", "cmg": "comigo", "ctg": "contigo",
	"qnd": "quando", "qlq": "qualquer", "fds": "fim de semana", "pfv": "por favor", "pfvr": "por favor",
	"vdd": "verdade", "pdc": "pode crer", "tlgd": "tá ligado", "fmz": "firmeza", "dnv": "de novo",
	"ctz": "certeza", "oq": "o que", "qm": "quem", "qro": "quero", "abs": "abraços", "mds": "meu deus",
	"nd": "nada", "tds": "todos", "cm": "com", "vms": "vamos", "vamo": "vamos", "mn": "mano",
	"upar": "subir de nível", "upando": "subindo de nível", "xq": "porque", "tmb": "también", "porfa": "por favor", "grax": "gracias",
}

var wordRe = regexp.MustCompile(`[\p{L}']+`)

// ExpandSlang rewrites chat shorthand to full words before translation.
func ExpandSlang(text string) string {
	return wordRe.ReplaceAllStringFunc(text, func(w string) string {
		lw := strings.ToLower(w)
		if isLaugh(lw) {
			return "haha"
		}
		if full, ok := slang[lw]; ok {
			return full
		}
		return w
	})
}

// kkkk / kkkkk is Brazilian laughter; "rsrs" too.
func isLaugh(w string) bool {
	if len(w) >= 3 && strings.Trim(w, "k") == "" {
		return true
	}
	return len(w) >= 4 && strings.Trim(w, "rs") == ""
}

// plausibleForASCII reports whether DeepL's detected language is one people
// commonly type in plain ASCII. Anything else on ASCII text is likely a
// misdetection of English slang.
func plausibleForASCII(code string) bool {
	switch baseLang(code) {
	case "ES", "PT", "FR", "DE", "IT", "NL", "PL", "SV", "DA", "NB", "FI", "CS", "RO", "TR", "HU", "ID", "SK", "SL", "ET", "LV", "LT", "RU", "UK", "BG":
		return true
	}
	return false
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 128 {
			return false
		}
	}
	return true
}

// sameText is true when a "translation" is really just the original.
func sameText(a, b string) bool {
	norm := func(s string) string {
		return strings.Join(strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r)
		}), " ")
	}
	return norm(a) == norm(b)
}
