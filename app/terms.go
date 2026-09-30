package main

// WoW place names and role verbs across client languages.
//
// Machine translation mangles localized place names ("Мертвые копи" ->
// "Dead Copies", "Ventobravo" -> "Ventbrave") and gamer verbs ("I can tank"
// -> "posso entrar no “tank”"). Before translating, Parley swaps every known
// name for the one the reader's WoW client uses and marks it so neither
// engine touches it; the same goes for "tank"/"heal" used as verbs in your
// replies.

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// keepOpen/keepClose mark text neither engine may translate. toHTML and
// toXML turn them into the engines' "don't translate" tags.
const keepOpen, keepClose = "\x02", "\x03"

func keep(s string) string { return keepOpen + s + keepClose }

var keepRe = regexp.MustCompile("\x02([^\x02\x03]*)\x03")

func stripKeep(s string) string { return keepRe.ReplaceAllString(s, "$1") }

// Locale columns: en, pt (ptBR), es (esES/esMX), de, fr, it, ru, zhCN, zhTW, ko.
// The first spelling in each column is what that language's readers see; the
// rest are accepted spellings. Russian entries are regular expressions so
// case endings match ("Мертвые копи", "в Мертвых копях"). Japanese players
// use the English client, so katakana spellings live in "ja" for matching only.
type place struct{ loc map[string][]string }

var places = []place{
	// Capitals
	{map[string][]string{"en": {"Stormwind"}, "pt": {"Ventobravo"}, "es": {"Ventormenta"}, "de": {"Sturmwind"}, "fr": {"Hurlevent"}, "it": {"Roccavento"}, "ru": {"Штормград\\p{L}*"}, "zhCN": {"暴风城"}, "zhTW": {"暴風城"}, "ko": {"스톰윈드"}, "ja": {"ストームウィンド"}}},
	{map[string][]string{"en": {"Ironforge"}, "pt": {"Altaforja"}, "es": {"Forjaz"}, "de": {"Eisenschmiede"}, "fr": {"Forgefer"}, "it": {"Forgiardente"}, "ru": {"Стальгорн\\p{L}*"}, "zhCN": {"铁炉堡"}, "zhTW": {"鐵爐堡"}, "ko": {"아이언포지"}, "ja": {"アイアンフォージ"}}},
	{map[string][]string{"en": {"Darnassus"}, "ru": {"Дарнас\\p{L}*"}, "zhCN": {"达纳苏斯"}, "zhTW": {"達納蘇斯"}, "ko": {"다르나서스"}, "ja": {"ダルナサス"}}},
	{map[string][]string{"en": {"Orgrimmar"}, "ru": {"Оргриммар\\p{L}*"}, "zhCN": {"奥格瑞玛"}, "zhTW": {"奧格瑪"}, "ko": {"오그리마"}, "ja": {"オーグリマー", "オグリマー"}}},
	{map[string][]string{"en": {"Thunder Bluff"}, "pt": {"Penhasco do Trovão"}, "es": {"Cima del Trueno"}, "de": {"Donnerfels"}, "fr": {"Les Pitons-du-Tonnerre", "Pitons-du-Tonnerre", "Pitons du Tonnerre"}, "it": {"Picco del Tuono"}, "ru": {"Громов\\p{L}* Ут[её]с\\p{L}*"}, "zhCN": {"雷霆崖"}, "zhTW": {"雷霆崖"}, "ko": {"썬더 블러프", "썬더블러프"}, "ja": {"サンダーブラフ"}}},
	{map[string][]string{"en": {"Undercity"}, "pt": {"Cidade Baixa"}, "es": {"Entrañas"}, "de": {"Unterstadt"}, "fr": {"Fossoyeuse"}, "it": {"Sepulcra"}, "ru": {"Подгород\\p{L}*"}, "zhCN": {"幽暗城"}, "zhTW": {"幽暗城"}, "ko": {"언더시티"}, "ja": {"アンダーシティ"}}},
	// Dungeons
	{map[string][]string{"en": {"Deadmines", "The Deadmines"}, "pt": {"Minas Mortas", "Minas da Morte"}, "es": {"Minas de la Muerte"}, "de": {"Todesminen", "Die Todesminen"}, "fr": {"Mortemines", "Les Mortemines"}, "it": {"Miniere della Morte"}, "ru": {"М[её]ртв\\p{L}* коп\\p{L}*"}, "zhCN": {"死亡矿井", "死矿"}, "zhTW": {"死亡礦坑", "死亡礦井", "死礦"}, "ko": {"죽음의 폐광", "데드마인"}, "ja": {"デッドマインズ", "デッドマイン"}}},
	{map[string][]string{"en": {"Wailing Caverns"}, "pt": {"Caverna Ululante", "Cavernas Ululantes"}, "es": {"Cuevas de los Lamentos"}, "de": {"Höhlen des Wehklagens", "Die Höhlen des Wehklagens"}, "fr": {"Cavernes des Lamentations"}, "ru": {"Пещер\\p{L}* Стенани\\p{L}*"}, "zhCN": {"哀嚎洞穴"}, "zhTW": {"哀嚎洞穴"}, "ko": {"통곡의 동굴"}, "ja": {"ウェイリングキャバーンズ"}}},
	{map[string][]string{"en": {"Shadowfang Keep"}, "pt": {"Bastilha da Presa Negra", "Bastilha Presa Negra"}, "es": {"Castillo de Colmillo Oscuro"}, "de": {"Burg Schattenfang"}, "fr": {"Donjon d'Ombrecroc", "Donjon d’Ombrecroc"}, "ru": {"Крепост\\p{L}* Т[её]мно\\p{L}* Клык\\p{L}*"}, "zhCN": {"影牙城堡"}, "zhTW": {"影牙城堡"}, "ko": {"그림자송곳니 성채"}, "ja": {"シャドウファングキープ"}}},
	{map[string][]string{"en": {"Blackfathom Deeps"}, "pt": {"Profundezas Negras"}, "es": {"Cavernas de Brazanegra"}, "de": {"Tiefschwarze Grotte"}, "fr": {"Profondeurs de Brassenoire"}, "ru": {"Непроглядн\\p{L}* Пучин\\p{L}*"}, "zhCN": {"黑暗深渊"}, "zhTW": {"黑暗深淵"}, "ko": {"검은심연의 나락"}}},
	{map[string][]string{"en": {"Gnomeregan"}, "ru": {"Гномреган\\p{L}*"}, "zhCN": {"诺莫瑞根"}, "zhTW": {"諾姆瑞根"}, "ko": {"놈리건"}, "ja": {"ノームリーガン"}}},
	{map[string][]string{"en": {"Scarlet Monastery"}, "pt": {"Monastério Escarlate"}, "es": {"Monasterio Escarlata"}, "de": {"Scharlachrote Kloster", "Scharlachroten Kloster", "Das Scharlachrote Kloster"}, "fr": {"Monastère écarlate"}, "it": {"Monastero Scarlatto"}, "ru": {"Монастыр\\p{L}* Ал\\p{L}* [Оо]рден\\p{L}*"}, "zhCN": {"血色修道院"}, "zhTW": {"血色修道院"}, "ko": {"붉은십자군 수도원"}, "ja": {"スカーレットモナステリー"}}},
	{map[string][]string{"en": {"Uldaman"}, "ru": {"Ульдаман\\p{L}*"}, "zhCN": {"奥达曼"}, "zhTW": {"奧達曼"}, "ko": {"울다만"}, "ja": {"ウルダマン"}}},
	{map[string][]string{"en": {"Zul'Farrak"}, "ru": {"Зул'Фаррак\\p{L}*"}, "zhCN": {"祖尔法拉克"}, "zhTW": {"祖爾法拉克"}, "ko": {"줄파락"}}},
	{map[string][]string{"en": {"Maraudon"}, "ru": {"Мародон\\p{L}*"}, "zhCN": {"玛拉顿"}, "zhTW": {"瑪拉頓"}, "ko": {"마라우돈"}}},
	{map[string][]string{"en": {"Blackrock Depths"}, "pt": {"Abismo Rocha Negra", "Abismo da Rocha Negra"}, "es": {"Profundidades de Roca Negra"}, "de": {"Schwarzfelstiefen"}, "fr": {"Profondeurs de Rochenoire"}, "ru": {"Глубин\\p{L}* Ч[её]рной [Гг]ор\\p{L}*"}, "zhCN": {"黑石深渊"}, "zhTW": {"黑石深淵"}, "ko": {"검은바위 나락"}, "ja": {"ブラックロックデプス"}}},
	{map[string][]string{"en": {"Blackrock Spire"}, "pt": {"Pico da Rocha Negra"}, "es": {"Cumbre de Roca Negra"}, "de": {"Schwarzfelsspitze"}, "fr": {"Pic Rochenoire", "Pic de Rochenoire"}, "ru": {"Вершин\\p{L}* Ч[её]рной [Гг]ор\\p{L}*"}, "zhCN": {"黑石塔"}, "zhTW": {"黑石塔"}, "ko": {"검은바위 첨탑"}}},
	{map[string][]string{"en": {"Stratholme"}, "ru": {"Стратхольм\\p{L}*"}, "zhCN": {"斯坦索姆"}, "zhTW": {"斯坦索姆"}, "ko": {"스트라솔름"}, "ja": {"ストラトホルム"}}},
	{map[string][]string{"en": {"Scholomance"}, "ru": {"Некроситет\\p{L}*"}, "zhCN": {"通灵学院"}, "zhTW": {"通靈學院"}, "ko": {"스칼로맨스"}}},
	{map[string][]string{"en": {"Dire Maul"}, "pt": {"Gládio Cruel"}, "es": {"La Masacre"}, "de": {"Düsterbruch"}, "fr": {"Hache-Tripes"}, "ru": {"Забыт\\p{L}* город\\p{L}*"}, "zhCN": {"厄运之槌"}, "zhTW": {"厄運之槌"}, "ko": {"혈투의 전장"}}},
	// Raids
	{map[string][]string{"en": {"Molten Core"}, "pt": {"Núcleo Derretido"}, "es": {"Núcleo de Magma"}, "de": {"Geschmolzener Kern", "Geschmolzenen Kern"}, "fr": {"Cœur du Magma", "Coeur du Magma"}, "ru": {"Огненн\\p{L}* Недр\\p{L}*"}, "zhCN": {"熔火之心"}, "zhTW": {"熔火之心"}, "ko": {"화산 심장부"}, "ja": {"モルテンコア"}}},
	{map[string][]string{"en": {"Onyxia's Lair"}, "ru": {"Логов\\p{L}* Ониксии"}, "zhCN": {"奥妮克希亚的巢穴"}, "zhTW": {"奧妮克希亞的巢穴"}, "ko": {"오닉시아의 둥지"}}},
	{map[string][]string{"en": {"Blackwing Lair"}, "pt": {"Covil Asa Negra"}, "es": {"Guarida Alanegra"}, "de": {"Pechschwingenhort"}, "fr": {"Repaire de l'Aile noire", "Repaire de l’Aile noire"}, "ru": {"Логов\\p{L}* Крыла Тьмы"}, "zhCN": {"黑翼之巢"}, "zhTW": {"黑翼之巢"}, "ko": {"검은날개 둥지"}}},
	{map[string][]string{"en": {"Zul'Gurub"}, "ru": {"Зул'Гуруб\\p{L}*"}, "zhCN": {"祖尔格拉布"}, "zhTW": {"祖爾格拉布"}, "ko": {"줄구룹"}}},
	{map[string][]string{"en": {"Naxxramas"}, "ru": {"Наксрамас\\p{L}*"}, "zhCN": {"纳克萨玛斯"}, "zhTW": {"納克薩瑪斯"}, "ko": {"낙스라마스"}}},
	// Popular zones
	{map[string][]string{"en": {"Elwynn Forest"}, "pt": {"Floresta de Elwynn"}, "es": {"Bosque de Elwynn"}, "de": {"Wald von Elwynn"}, "fr": {"Forêt d'Elwynn", "Forêt d’Elwynn"}, "ru": {"Элвиннск\\p{L}* лес\\p{L}*"}, "zhCN": {"艾尔文森林"}, "zhTW": {"艾爾文森林"}, "ko": {"엘윈 숲"}}},
	{map[string][]string{"en": {"Westfall"}, "pt": {"Cerro Oeste"}, "es": {"Páramos de Poniente"}, "fr": {"Marche de l'Ouest", "Marche de l’Ouest"}, "ru": {"Западн\\p{L}* Кра\\p{L}*"}, "zhCN": {"西部荒野"}, "zhTW": {"西部荒野"}, "ko": {"서부 몰락지대"}}},
	{map[string][]string{"en": {"Duskwood"}, "pt": {"Floresta do Crepúsculo"}, "es": {"Bosque del Ocaso"}, "de": {"Dämmerwald"}, "fr": {"Bois de la Pénombre"}, "ru": {"Сумеречн\\p{L}* лес\\p{L}*"}, "zhCN": {"暮色森林"}, "zhTW": {"暮色森林"}, "ko": {"그늘숲"}}},
	{map[string][]string{"en": {"Stranglethorn Vale", "Stranglethorn"}, "pt": {"Selva do Espinhaço"}, "es": {"Vega de Tuercespina"}, "de": {"Schlingendorntal"}, "fr": {"Vallée de Strangleronce", "Strangleronce"}, "ru": {"Тернист\\p{L}* долин\\p{L}*"}, "zhCN": {"荆棘谷"}, "zhTW": {"荊棘谷"}, "ko": {"가시덤불 골짜기"}}},
	{map[string][]string{"en": {"The Barrens", "Barrens"}, "pt": {"Sertões"}, "es": {"Los Baldíos"}, "de": {"Brachland"}, "fr": {"Les Tarides", "Tarides"}, "zhCN": {"贫瘠之地"}, "zhTW": {"貧瘠之地"}, "ko": {"불모의 땅"}}},
	{map[string][]string{"en": {"Ashenvale"}, "pt": {"Vale Gris"}, "es": {"Vallefresno"}, "de": {"Eschental"}, "fr": {"Orneval"}, "ru": {"Ясенев\\p{L}* лес\\p{L}*"}, "zhCN": {"灰谷"}, "zhTW": {"灰谷"}, "ko": {"잿빛 골짜기"}}},
}

// placeLocale maps a Parley language code to the place-name column its
// readers know. Japanese, and anything unknown, reads English names.
func placeLocale(code string) string {
	c := strings.ToUpper(code)
	switch {
	case strings.HasPrefix(c, "ZH-HANT"), c == "ZH-TW":
		return "zhTW"
	case strings.HasPrefix(c, "ZH"):
		return "zhCN"
	}
	switch baseLang(c) {
	case "PT":
		return "pt"
	case "ES":
		return "es"
	case "DE":
		return "de"
	case "FR":
		return "fr"
	case "IT":
		return "it"
	case "RU":
		return "ru"
	case "KO":
		return "ko"
	}
	return "en"
}

type placeMatcher struct {
	re    *regexp.Regexp
	names []int // capture group i+1 belongs to places[names[i]]
	words []bool
}

var placeRe = buildPlaceRe()

func buildPlaceRe() placeMatcher {
	type alt struct {
		pat  string
		idx  int
		size int
		word bool
	}
	var alts []alt
	for i, p := range places {
		for loc, forms := range p.loc {
			for _, f := range forms {
				pat := f
				if loc != "ru" {
					pat = regexp.QuoteMeta(f)
				}
				pat = strings.ReplaceAll(pat, " ", `\s+`)
				r, _ := utf8.DecodeRuneInString(f)
				word := !unicode.In(r, unicode.Han, unicode.Hangul, unicode.Katakana, unicode.Hiragana)
				alts = append(alts, alt{pat, i, len(f), word})
			}
		}
	}
	// Longest first so "The Deadmines" beats "Deadmines".
	sort.SliceStable(alts, func(a, b int) bool { return alts[a].size > alts[b].size })
	m := placeMatcher{}
	var parts []string
	for _, a := range alts {
		parts = append(parts, "("+a.pat+")")
		m.names = append(m.names, a.idx)
		m.words = append(m.words, a.word)
	}
	m.re = regexp.MustCompile(`(?i)` + strings.Join(parts, "|"))
	return m
}

func isWordRune(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }

// localizePlaces rewrites every known WoW place name in text to the name the
// reader of language target knows, marked so the engines keep it.
func localizePlaces(text, target string) string {
	col := placeLocale(target)
	var b strings.Builder
	last := 0
	for _, loc := range placeRe.re.FindAllStringSubmatchIndex(text, -1) {
		g := 0
		for i := 1; i < len(loc)/2; i++ {
			if loc[2*i] >= 0 {
				g = i
				break
			}
		}
		if g == 0 {
			continue
		}
		start, end := loc[0], loc[1]
		if placeRe.words[g-1] { // Latin/Cyrillic names must be whole words
			if start > 0 {
				if r, _ := utf8.DecodeLastRuneInString(text[:start]); isWordRune(r) {
					continue
				}
			}
			if end < len(text) {
				if r, _ := utf8.DecodeRuneInString(text[end:]); isWordRune(r) {
					continue
				}
			}
		}
		p := places[placeRe.names[g-1]]
		name := p.loc["en"][0]
		if f, ok := p.loc[col]; ok && col != "ru" {
			name = f[0]
		} else if col == "ru" {
			if f, ok := ruDisplay[name]; ok {
				name = f
			}
		}
		b.WriteString(text[last:start])
		b.WriteString(keep(name))
		last = end
	}
	if last == 0 {
		return text
	}
	b.WriteString(text[last:])
	return b.String()
}

// Russian columns are patterns; these are the spellings Russian readers see.
var ruDisplay = map[string]string{
	"Stormwind": "Штормград", "Ironforge": "Стальгорн", "Darnassus": "Дарнас", "Orgrimmar": "Оргриммар",
	"Thunder Bluff": "Громовой Утес", "Undercity": "Подгород", "Deadmines": "Мертвые копи",
	"Wailing Caverns": "Пещеры Стенаний", "Shadowfang Keep": "Крепость Темного Клыка",
	"Blackfathom Deeps": "Непроглядная Пучина", "Gnomeregan": "Гномреган", "Scarlet Monastery": "Монастырь Алого ордена",
	"Uldaman": "Ульдаман", "Zul'Farrak": "Зул'Фаррак", "Maraudon": "Мародон", "Blackrock Depths": "Глубины Черной горы",
	"Blackrock Spire": "Вершина Черной горы", "Stratholme": "Стратхольм", "Scholomance": "Некроситет",
	"Dire Maul": "Забытый город", "Molten Core": "Огненные Недра", "Onyxia's Lair": "Логово Ониксии",
	"Blackwing Lair": "Логово Крыла Тьмы", "Zul'Gurub": "Зул'Гуруб", "Naxxramas": "Наксрамас",
	"Elwynn Forest": "Элвиннский лес", "Westfall": "Западный Край", "Duskwood": "Сумеречный лес",
	"Stranglethorn Vale": "Тернистая долина", "The Barrens": "Степи", "Ashenvale": "Ясеневый лес",
}

// Role verbs in your replies: "I can tank" should become "posso tankar",
// not "tank" left in English or put in quotes.
var roleVerbs = map[string]map[string]string{
	"tank": {"pt": "tankar", "es": "tanquear", "de": "tanken", "fr": "tanker", "it": "tankare", "ru": "танковать"},
	"heal": {"pt": "curar", "es": "curar", "de": "heilen", "fr": "soigner", "it": "curare", "ru": "хилить"},
}

var roleVerbRe = regexp.MustCompile(`(?i)(\bcan|\bcould|\bwill|\bwould|\bshould|\bto|'ll|\bgonna|\bwanna|\bme|\bi|\bwe|\byou|\bpls|\bplease|\bcan't|\bcannot|\bwon't)(\s+)(tank|heal)\b`)

// localizeRoleVerbs rewrites "can tank", "I'll heal", "want me to tank" in an
// English reply into the target language's gamer verb, kept as is.
func localizeRoleVerbs(text, target string) string {
	col := placeLocale(target)
	var out strings.Builder
	last := 0
	for _, m := range roleVerbRe.FindAllStringSubmatchIndex(text, -1) {
		verb, ok := roleVerbs[strings.ToLower(text[m[6]:m[7]])][col]
		if !ok {
			continue
		}
		out.WriteString(text[last:m[6]])
		out.WriteString(keep(verb))
		last = m[7]
	}
	out.WriteString(text[last:])
	return out.String()
}

// prepareIncoming and prepareReply are the text fixes run before either
// engine sees a message.
func prepareIncoming(text, target string) string {
	return localizePlaces(text, target)
}

func prepareReply(text, target string) string { // text is English
	return localizePlaces(localizeRoleVerbs(text, target), target)
}
