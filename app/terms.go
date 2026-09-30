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
	"sync"
	"unicode"
	"unicode/utf8"
)

// keepOpen/keepClose mark text neither engine may translate. toHTML and
// toXML turn them into the engines' "don't translate" tags.
const keepOpen, keepClose = "\x02", "\x03"

func keep(s string) string { return keepOpen + s + keepClose }

var keepRe = regexp.MustCompile("\x02([^\x02\x03]*)\x03")

// lockOpen/lockClose mark the player's own "never translate" words (guild
// names, nicknames). Unlike keep, which the online engines may still see as
// plain text, locked text is always sent inside a no-translate tag.
const lockOpen, lockClose = "\x04", "\x05"

func lock(s string) string { return lockOpen + s + lockClose }

var lockRe = regexp.MustCompile("\x04([^\x04\x05]*)\x05")

func stripKeep(s string) string {
	return lockRe.ReplaceAllString(keepRe.ReplaceAllString(s, "$1"), "$1")
}

// userKeep matches the words from Settings → "Never translate".
var userKeep struct {
	sync.RWMutex
	re *regexp.Regexp
}

// keepWordList splits the settings field ("Sombra Eterna, Bob; Kek").
func keepWordList(words string) []string {
	var out []string
	seen := map[string]bool{}
	for _, w := range strings.FieldsFunc(words, func(r rune) bool { return r == ',' || r == ';' || r == '\n' }) {
		w = strings.TrimSpace(w)
		if w == "" || seen[strings.ToLower(w)] {
			continue
		}
		seen[strings.ToLower(w)] = true
		out = append(out, w)
	}
	return out
}

// setKeepWords installs the player's never-translate list.
func setKeepWords(words string) {
	list := keepWordList(words)
	sort.SliceStable(list, func(a, b int) bool { return len(list[a]) > len(list[b]) })
	var parts []string
	for _, w := range list {
		parts = append(parts, strings.ReplaceAll(regexp.QuoteMeta(w), " ", `\s+`))
	}
	var re *regexp.Regexp
	if len(parts) > 0 {
		re = regexp.MustCompile(`(?i)(?:` + strings.Join(parts, "|") + `)`)
	}
	userKeep.Lock()
	userKeep.re = re
	userKeep.Unlock()
}

// lockUserWords marks the player's never-translate words in text. Words in
// Latin or Cyrillic script must stand alone ("Bob" doesn't match "Bobby");
// text already marked (place names) is left alone.
func lockUserWords(text string) string {
	userKeep.RLock()
	re := userKeep.re
	userKeep.RUnlock()
	if re == nil {
		return text
	}
	var b strings.Builder
	last := 0
	for _, m := range re.FindAllStringIndex(text, -1) {
		start, end := m[0], m[1]
		if start < last || strings.Count(text[:start], keepOpen) != strings.Count(text[:start], keepClose) {
			continue // inside a marked place name
		}
		r, _ := utf8.DecodeRuneInString(text[start:])
		if !unicode.In(r, unicode.Han, unicode.Hangul, unicode.Katakana, unicode.Hiragana) {
			if start > 0 {
				if p, _ := utf8.DecodeLastRuneInString(text[:start]); isWordRune(p) {
					continue
				}
			}
			if end < len(text) {
				if n, _ := utf8.DecodeRuneInString(text[end:]); isWordRune(n) {
					continue
				}
			}
		}
		b.WriteString(text[last:start])
		b.WriteString(lock(text[start:end]))
		last = end
	}
	if last == 0 {
		return text
	}
	b.WriteString(text[last:])
	return b.String()
}

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
	// Burning Crusade (Anniversary realms). Names from the game's own
	// localizations, via LibBabble-SubZone-3.0.
	{map[string][]string{"en": {"Outland"}, "pt": {"Terralém"}, "es": {"Terrallende"}, "de": {"Scherbenwelt"}, "fr": {"Outreterre"}, "it": {"Terre Esterne"}, "ru": {"Запр[её]д[её]ль\\p{L}*"}, "zhCN": {"外域"}, "zhTW": {"外域"}, "ko": {"아웃랜드"}}},
	{map[string][]string{"en": {"Shattrath City", "Shattrath"}, "es": {"Ciudad de Shattrath"}, "ru": {"Шаттрат\\p{L}*"}, "zhCN": {"沙塔斯城"}, "zhTW": {"撒塔斯城"}, "ko": {"샤트라스"}}},
	{map[string][]string{"en": {"Silvermoon City", "Silvermoon"}, "pt": {"Luaprata"}, "es": {"Ciudad de Lunargenta"}, "de": {"Silbermond"}, "fr": {"Lune-d’Argent", "Lune-d'Argent"}, "it": {"Lunargenta"}, "ru": {"Луносв[её]т\\p{L}*"}, "zhCN": {"银月城"}, "zhTW": {"銀月城"}, "ko": {"실버문"}}},
	{map[string][]string{"en": {"The Exodar", "Exodar"}, "es": {"El Exodar", "Exodar"}, "de": {"Die Exodar", "Exodar"}, "fr": {"L’Exodar", "L'Exodar"}, "ru": {"Экзодар\\p{L}*"}, "zhCN": {"埃索达"}, "zhTW": {"艾克索達"}, "ko": {"엑소다르"}}},
	{map[string][]string{"en": {"Hellfire Peninsula"}, "pt": {"Península Fogo do Inferno"}, "es": {"Península del Fuego Infernal"}, "de": {"Höllenfeuerhalbinsel"}, "fr": {"Péninsule des Flammes infernales"}, "it": {"Penisola del Fuoco Infernale"}, "ru": {"Полуостров\\p{L}* Адског\\p{L}* Плам[её]н\\p{L}*"}, "zhCN": {"地狱火半岛"}, "zhTW": {"地獄火半島"}, "ko": {"지옥불 반도"}}},
	{map[string][]string{"en": {"Zangarmarsh"}, "pt": {"Pântano Zíngaro"}, "es": {"Marisma de Zangar"}, "de": {"Zangarmarschen"}, "fr": {"Marécage de Zangar"}, "it": {"Paludi di Zangar"}, "ru": {"Зангартоп\\p{L}*"}, "zhCN": {"赞加沼泽"}, "zhTW": {"贊格沼澤"}, "ko": {"장가르 습지대"}}},
	{map[string][]string{"en": {"Terokkar Forest"}, "pt": {"Mata Terokkar"}, "es": {"Bosque de Terokkar"}, "de": {"Wälder von Terokkar"}, "fr": {"Forêt de Terokkar"}, "it": {"Foresta di Terokk"}, "ru": {"Л[её]с Т[её]роккар\\p{L}*"}, "zhCN": {"泰罗卡森林"}, "zhTW": {"泰洛卡森林"}, "ko": {"테로카르 숲"}}},
	{map[string][]string{"en": {"Nagrand"}, "ru": {"Награнд\\p{L}*"}, "zhCN": {"纳格兰"}, "zhTW": {"納葛蘭"}, "ko": {"나그란드"}}},
	{map[string][]string{"en": {"Blade's Edge Mountains", "Blade's Edge"}, "pt": {"Montanhas da Lâmina Afiada"}, "es": {"Montañas Filospada"}, "de": {"Schergrat"}, "fr": {"Les Tranchantes", "Tranchantes"}, "it": {"Montagne Spinaguzza"}, "ru": {"Острогорь\\p{L}*"}, "zhCN": {"刀锋山"}, "zhTW": {"劍刃山脈"}, "ko": {"칼날 산맥"}}},
	{map[string][]string{"en": {"Netherstorm"}, "pt": {"Eternévoa"}, "es": {"Tormenta Abisal"}, "de": {"Nethersturm"}, "fr": {"Raz-de-Néant"}, "it": {"Landa Fatua"}, "ru": {"Пустов[её]рт\\p{L}*"}, "zhCN": {"虚空风暴"}, "zhTW": {"虛空風暴"}, "ko": {"황천의 폭풍"}}},
	{map[string][]string{"en": {"Shadowmoon Valley"}, "pt": {"Vale da Lua Negra"}, "es": {"Valle Sombraluna"}, "de": {"Schattenmondtal"}, "fr": {"Vallée d’Ombrelune", "Vallée d'Ombrelune"}, "it": {"Valle di Torvaluna"}, "ru": {"Долин\\p{L}* Призрачно\\p{L}* Лун\\p{L}*"}, "zhCN": {"影月谷"}, "zhTW": {"影月谷"}, "ko": {"어둠달 골짜기"}}},
	{map[string][]string{"en": {"Isle of Quel'Danas", "Quel'Danas"}, "pt": {"Ilha de Quel'Danas"}, "es": {"Isla de Quel'Danas"}, "de": {"Insel von Quel'Danas"}, "fr": {"Île de Quel’Danas", "Île de Quel'Danas"}, "it": {"Isola di Quel'Danas"}, "ru": {"Остров\\p{L}* К[её]ль'Данас\\p{L}*"}, "zhCN": {"奎尔丹纳斯岛"}, "zhTW": {"奎爾達納斯之島"}, "ko": {"쿠엘다나스 섬"}}},
	{map[string][]string{"en": {"Hellfire Ramparts"}, "pt": {"Muralha Fogo do Inferno"}, "es": {"Murallas del Fuego Infernal"}, "de": {"Höllenfeuerbollwerk"}, "fr": {"Remparts des Flammes infernales"}, "it": {"Bastioni del Fuoco Infernale"}, "ru": {"Бастион\\p{L}* Адског\\p{L}* Плам[её]н\\p{L}*"}, "zhCN": {"地狱火城墙"}, "zhTW": {"地獄火壁壘"}, "ko": {"지옥불 성루"}}},
	{map[string][]string{"en": {"The Blood Furnace", "Blood Furnace"}, "pt": {"Fornalha de Sangue"}, "es": {"El Horno de Sangre", "Horno de Sangre"}, "de": {"Der Blutkessel", "Blutkessel"}, "fr": {"La Fournaise du sang", "Fournaise du sang"}, "it": {"Forgia del Sangue"}, "ru": {"Кузн\\p{L}* Кров\\p{L}*"}, "zhCN": {"鲜血熔炉"}, "zhTW": {"血熔爐"}, "ko": {"피의 용광로"}}},
	{map[string][]string{"en": {"The Shattered Halls", "Shattered Halls"}, "pt": {"Salões Despedaçados"}, "es": {"Las Salas Arrasadas", "Salas Arrasadas"}, "de": {"Die Zerschmetterten Hallen", "Zerschmetterten Hallen"}, "fr": {"Les salles Brisées", "salles Brisées"}, "it": {"Sale della Devastazione"}, "ru": {"Разруш[её]нны\\p{L}* зал\\p{L}*"}, "zhCN": {"破碎大厅"}, "zhTW": {"破碎大廳"}, "ko": {"으스러진 손의 전당"}}},
	{map[string][]string{"en": {"The Slave Pens", "Slave Pens"}, "pt": {"Pátio dos Escravos"}, "es": {"Recinto de los Esclavos"}, "de": {"Die Sklavenunterkünfte", "Sklavenunterkünfte"}, "fr": {"Les enclos aux esclaves", "enclos aux esclaves"}, "it": {"Fosse degli Schiavi"}, "ru": {"Узилищ\\p{L}*"}, "zhCN": {"奴隶围栏"}, "zhTW": {"奴隸監獄"}, "ko": {"강제 노역소"}}},
	{map[string][]string{"en": {"The Underbog", "Underbog"}, "pt": {"Brejo Oculto"}, "es": {"La Sotiénaga", "Sotiénaga"}, "de": {"Der Tiefensumpf", "Tiefensumpf"}, "fr": {"La Basse-tourbière", "Basse-tourbière"}, "it": {"Torbiera Sotterranea"}, "ru": {"Ниж[её]топ\\p{L}*"}, "zhCN": {"幽暗沼泽"}, "zhTW": {"深幽泥沼"}, "ko": {"지하수렁"}}},
	{map[string][]string{"en": {"The Steamvault", "Steamvault"}, "pt": {"Câmara dos Vapores"}, "es": {"La Cámara de Vapor", "Cámara de Vapor"}, "de": {"Die Dampfkammer", "Dampfkammer"}, "fr": {"Le caveau de la Vapeur", "caveau de la Vapeur"}, "it": {"Antro dei Vapori"}, "ru": {"Парово\\p{L}* подз[её]м[её]ль\\p{L}*"}, "zhCN": {"蒸汽地窟"}, "zhTW": {"蒸汽洞窟"}, "ko": {"증기 저장고"}}},
	{map[string][]string{"en": {"Mana-Tombs"}, "pt": {"Tumbas de Mana"}, "es": {"Tumbas de Maná"}, "de": {"Managruft"}, "fr": {"Tombes-mana"}, "it": {"Tombe del Mana"}, "ru": {"Гробниц\\p{L}* Ман\\p{L}*"}, "zhCN": {"法力陵墓"}, "zhTW": {"法力墓地"}, "ko": {"마나 무덤"}}},
	{map[string][]string{"en": {"Auchenai Crypts"}, "pt": {"Catacumbas Auchenai"}, "es": {"Criptas Auchenai"}, "de": {"Auchenaikrypta"}, "fr": {"Cryptes Auchenaï"}, "it": {"Cripte degli Auchenai"}, "ru": {"Аук[её]найски\\p{L}* гробниц\\p{L}*"}, "zhCN": {"奥金尼地穴"}, "zhTW": {"奧奇奈地穴"}, "ko": {"아키나이 납골당"}}},
	{map[string][]string{"en": {"Sethekk Halls"}, "pt": {"Salões dos Sethekk"}, "es": {"Salas Sethekk"}, "de": {"Sethekkhallen"}, "fr": {"Les salles des Sethekk", "salles des Sethekk"}, "it": {"Sale dei Sethekk"}, "ru": {"С[её]т[её]ккски\\p{L}* зал\\p{L}*"}, "zhCN": {"塞泰克大厅"}, "zhTW": {"塞司克大廳"}, "ko": {"세데크 전당"}}},
	{map[string][]string{"en": {"Shadow Labyrinth"}, "pt": {"Labirinto Soturno"}, "es": {"Laberinto de las Sombras"}, "de": {"Schattenlabyrinth"}, "fr": {"Labyrinthe des Ombres"}, "it": {"Labirinto delle Ombre"}, "ru": {"Т[её]мны\\p{L}* лабиринт\\p{L}*"}, "zhCN": {"暗影迷宫"}, "zhTW": {"暗影迷宮"}, "ko": {"어둠의 미궁"}}},
	{map[string][]string{"en": {"The Mechanar", "Mechanar"}, "pt": {"Mecanar"}, "es": {"El Mechanar", "Mechanar"}, "de": {"Die Mechanar", "Mechanar"}, "fr": {"Le Méchanar", "Méchanar"}, "it": {"Mecanar"}, "ru": {"М[её]ханар\\p{L}*"}, "zhCN": {"能源舰"}, "zhTW": {"麥克納爾"}, "ko": {"메카나르"}}},
	{map[string][]string{"en": {"The Botanica", "Botanica"}, "pt": {"Jardim Botânico"}, "es": {"El Invernáculo", "Invernáculo"}, "de": {"Die Botanika", "Botanika"}, "fr": {"La Botanica", "Botanica"}, "ru": {"Ботаник\\p{L}*"}, "zhCN": {"生态船"}, "zhTW": {"波塔尼卡"}, "ko": {"신록의 정원"}}},
	{map[string][]string{"en": {"The Arcatraz", "Arcatraz"}, "es": {"El Arcatraz", "Arcatraz"}, "de": {"Die Arkatraz", "Arkatraz"}, "fr": {"L’Arcatraz", "L'Arcatraz"}, "ru": {"Аркатрац\\p{L}*"}, "zhCN": {"禁魔监狱"}, "zhTW": {"亞克崔茲"}, "ko": {"알카트라즈"}}},
	{map[string][]string{"en": {"Old Hillsbrad Foothills", "Old Hillsbrad"}, "pt": {"Antigo Contraforte de Eira dos Montes"}, "es": {"Antiguas Laderas de Trabalomas"}, "de": {"Vorgebirge des Alten Hügellands"}, "fr": {"Contreforts de Hautebrande d’antan", "Contreforts de Hautebrande d'antan"}, "it": {"Passato di Colletorto"}, "ru": {"Стары\\p{L}* пр[её]дгорь\\p{L}* Хилсбрад\\p{L}*"}, "zhCN": {"旧希尔斯布莱德丘陵"}, "zhTW": {"希爾斯布萊德丘陵舊址"}, "ko": {"옛 언덕마루 구릉지"}}},
	{map[string][]string{"en": {"The Black Morass", "Black Morass"}, "pt": {"Lamaçal Negro"}, "es": {"La Ciénaga Negra", "Ciénaga Negra"}, "de": {"Der Schwarze Morast", "Schwarze Morast"}, "fr": {"Le Noir marécage", "Noir marécage"}, "it": {"Palude Nera"}, "ru": {"Ч[её]рны\\p{L}* топ\\p{L}*"}, "zhCN": {"黑色沼泽"}, "zhTW": {"黑色沼澤"}, "ko": {"검은늪"}}},
	{map[string][]string{"en": {"Magisters' Terrace"}, "pt": {"Terraço dos Magísteres"}, "es": {"Bancal del Magister"}, "de": {"Terrasse der Magister"}, "fr": {"Terrasse des Magistères"}, "it": {"Terrazza dei Magisteri"}, "ru": {"Т[её]ррас\\p{L}* Магистров\\p{L}*"}, "zhCN": {"魔导师平台"}, "zhTW": {"博學者殿堂"}, "ko": {"마법학자의 정원"}}},
	{map[string][]string{"en": {"Karazhan"}, "ru": {"Каражан\\p{L}*"}, "zhCN": {"卡拉赞"}, "zhTW": {"卡拉贊"}, "ko": {"카라잔"}}},
	{map[string][]string{"en": {"Gruul's Lair"}, "pt": {"Covil de Gruul"}, "es": {"Guarida de Gruul"}, "de": {"Gruuls Unterschlupf"}, "fr": {"Repaire de Gruul"}, "it": {"Antro di Gruul"}, "ru": {"Логов\\p{L}* Груул\\p{L}*"}, "zhCN": {"格鲁尔的巢穴"}, "zhTW": {"戈魯爾之巢"}, "ko": {"그룰의 둥지"}}},
	{map[string][]string{"en": {"Magtheridon's Lair"}, "pt": {"Covil de Magtheridon"}, "es": {"Guarida de Magtheridon"}, "de": {"Magtheridons Kammer"}, "fr": {"Le repaire de Magtheridon", "repaire de Magtheridon"}, "it": {"Antro di Magtheridon"}, "ru": {"Логов\\p{L}* Магт[её]ридон\\p{L}*"}, "zhCN": {"玛瑟里顿的巢穴"}, "zhTW": {"瑪瑟里頓的巢穴"}, "ko": {"마그테리돈의 둥지"}}},
	{map[string][]string{"en": {"Serpentshrine Cavern"}, "pt": {"Caverna do Serpentário"}, "es": {"Caverna Santuario Serpiente"}, "de": {"Höhle des Schlangenschreins"}, "fr": {"Caverne du sanctuaire du Serpent"}, "it": {"Caverna di Sacrespire"}, "ru": {"Зм[её]ино\\p{L}* святилищ\\p{L}*"}, "zhCN": {"毒蛇神殿"}, "zhTW": {"毒蛇神殿洞穴"}, "ko": {"불뱀 제단"}}},
	{map[string][]string{"en": {"Tempest Keep"}, "pt": {"Bastilha da Tormenta"}, "es": {"El Castillo de la Tempestad", "Castillo de la Tempestad"}, "de": {"Festung der Stürme"}, "fr": {"Donjon de la Tempête"}, "it": {"Forte Tempesta"}, "ru": {"Кр[её]пост\\p{L}* Бур\\p{L}*"}, "zhCN": {"风暴要塞"}, "zhTW": {"風暴要塞"}, "ko": {"폭풍우 요새"}}},
	{map[string][]string{"en": {"Hyjal Summit"}, "pt": {"Pico Hyjal"}, "es": {"La Cima Hyjal", "Cima Hyjal"}, "de": {"Hyjalgipfel"}, "fr": {"Sommet d’Hyjal", "Sommet d'Hyjal"}, "it": {"Sommità di Hyjal"}, "ru": {"В[её]ршин\\p{L}* Хиджал\\p{L}*"}, "zhCN": {"海加尔峰"}, "zhTW": {"海加爾山之巔"}, "ko": {"하이잘 정상"}}},
	{map[string][]string{"en": {"Black Temple"}, "pt": {"Templo Negro"}, "es": {"Templo Oscuro"}, "de": {"Der Schwarze Tempel", "Schwarze Tempel", "Schwarzen Tempel"}, "fr": {"Temple Noir"}, "it": {"Tempio Nero"}, "ru": {"Ч[её]рны\\p{L}* храм\\p{L}*"}, "zhCN": {"黑暗神殿"}, "zhTW": {"黑暗神廟"}, "ko": {"검은 사원"}}},
	{map[string][]string{"en": {"Zul'Aman"}, "fr": {"Zul’Aman", "Zul'Aman"}, "ru": {"Зул'Аман\\p{L}*"}, "zhCN": {"祖阿曼"}, "zhTW": {"祖阿曼"}, "ko": {"줄아만"}}},
	{map[string][]string{"en": {"Sunwell Plateau"}, "pt": {"Platô da Nascente do Sol"}, "es": {"Meseta de La Fuente del Sol"}, "de": {"Sonnenbrunnenplateau"}, "fr": {"Plateau du Puits de soleil"}, "it": {"Cittadella del Pozzo Solare"}, "ru": {"Плат\\p{L}* Солн[её]чног\\p{L}* Колодц\\p{L}*"}, "zhCN": {"太阳之井高地"}, "zhTW": {"太陽之井高地"}, "ko": {"태양샘 고원"}}},
	// Mists of Pandaria
	{map[string][]string{"en": {"Pandaria"}, "pt": {"Pandária"}, "fr": {"Pandarie"}, "ru": {"Пандари\\p{L}*"}, "zhCN": {"潘达利亚"}, "zhTW": {"潘達利亞"}, "ko": {"판다리아"}}},
	{map[string][]string{"en": {"The Jade Forest", "Jade Forest"}, "pt": {"Floresta de Jade"}, "es": {"El Bosque de Jade", "Bosque de Jade"}, "de": {"Der Jadewald", "Jadewald"}, "fr": {"La forêt de Jade", "forêt de Jade"}, "it": {"Foresta di Giada"}, "ru": {"Н[её]фритовы\\p{L}* л[её]с"}, "zhCN": {"翡翠林"}, "zhTW": {"翠玉林"}, "ko": {"비취 숲"}}},
	{map[string][]string{"en": {"Valley of the Four Winds"}, "pt": {"Vale dos Quatro Ventos"}, "es": {"Valle de los Cuatro Vientos"}, "de": {"Tal der Vier Winde"}, "fr": {"Vallée des Quatre vents"}, "it": {"Valle dei Quattro Venti"}, "ru": {"Долин\\p{L}* Ч[её]тыр[её]х\\p{L}* В[её]тров\\p{L}*"}, "zhCN": {"四风谷"}, "zhTW": {"四風峽"}, "ko": {"네 바람의 계곡"}}},
	{map[string][]string{"en": {"Krasarang Wilds"}, "pt": {"Selva de Krasarang"}, "es": {"Espesura Krasarang"}, "de": {"Krasarangwildnis"}, "fr": {"Étendues sauvages de Krasarang"}, "it": {"Giungla di Krasarang"}, "ru": {"Красарангски\\p{L}* джунгл\\p{L}*"}, "zhCN": {"卡桑琅丛林"}, "zhTW": {"喀撒朗蠻荒"}, "ko": {"크라사랑 밀림"}}},
	{map[string][]string{"en": {"Kun-Lai Summit", "Kun-Lai"}, "pt": {"Monte Kun-Lai"}, "es": {"Cima Kun-Lai"}, "de": {"Kun-Lai-Gipfel"}, "fr": {"Sommet de Kun-Lai"}, "it": {"Massiccio del Kun-Lai"}, "ru": {"В[её]ршин\\p{L}* Кунь-Ла\\p{L}*"}, "zhCN": {"昆莱山"}, "zhTW": {"崑萊峰"}, "ko": {"쿤라이 봉우리"}}},
	{map[string][]string{"en": {"Townlong Steppes"}, "pt": {"Estepes de Taolong"}, "es": {"Estepas de Tong Long"}, "de": {"Tonlongsteppe"}, "fr": {"Steppes de Tanglong"}, "it": {"Steppe di Tong Long"}, "ru": {"Танлунски\\p{L}* ст[её]п\\p{L}*"}, "zhCN": {"螳螂高原"}, "zhTW": {"螳螂荒原"}, "ko": {"탕랑 평원"}}},
	{map[string][]string{"en": {"Dread Wastes"}, "pt": {"Ermo do Medo"}, "es": {"Desierto del Pavor"}, "de": {"Schreckensöde"}, "fr": {"Terres de l’Angoisse", "Terres de l'Angoisse"}, "it": {"Distese del Terrore"}, "ru": {"Жутки\\p{L}* пустош\\p{L}*"}, "zhCN": {"恐惧废土"}, "zhTW": {"悚然荒野"}, "ko": {"공포의 황무지"}}},
	{map[string][]string{"en": {"Vale of Eternal Blossoms"}, "pt": {"Vale das Flores Eternas"}, "es": {"Valle de la Flor Eterna"}, "de": {"Tal der Ewigen Blüten"}, "fr": {"Val de l’Éternel printemps", "Val de l'Éternel printemps"}, "it": {"Vallata dell'Eterna Primavera"}, "ru": {"В[её]чноцв[её]тущи\\p{L}* дол"}, "zhCN": {"锦绣谷"}, "zhTW": {"恆春谷"}, "ko": {"영원꽃 골짜기"}}},
	{map[string][]string{"en": {"Shrine of Seven Stars"}, "pt": {"Santuário das Sete Estrelas"}, "es": {"Santuario de las Siete Estrellas"}, "de": {"Schrein der Sieben Sterne"}, "fr": {"Sanctuaire des Sept-Étoiles"}, "it": {"Santuario delle Sette Stelle"}, "ru": {"Святилищ\\p{L}* С[её]м\\p{L}* Зв[её]зд\\p{L}*"}, "zhCN": {"七星殿"}, "zhTW": {"七星廟"}, "ko": {"일곱 별의 제단"}}},
	{map[string][]string{"en": {"Shrine of Two Moons"}, "pt": {"Santuário das Duas Luas"}, "es": {"Santuario de las Dos Lunas"}, "de": {"Schrein der Zwei Monde"}, "fr": {"Sanctuaire des Deux-Lunes"}, "it": {"Santuario delle Due Lune"}, "ru": {"Святилищ\\p{L}* Двух\\p{L}* Лун"}, "zhCN": {"双月殿"}, "zhTW": {"雙月廟"}, "ko": {"두 달의 제단"}}},
	{map[string][]string{"en": {"Timeless Isle"}, "pt": {"Ilha Perene"}, "es": {"Isla Intemporal"}, "de": {"Zeitlose Insel"}, "fr": {"Île du Temps figé"}, "it": {"Isola Senza Tempo"}, "ru": {"Вн[её]вр[её]м[её]нны\\p{L}* остров\\p{L}*"}, "zhCN": {"永恒岛"}, "zhTW": {"永恆之島"}, "ko": {"영원의 섬"}}},
	{map[string][]string{"en": {"Isle of Thunder"}, "pt": {"Ilha do Trovão"}, "es": {"Isla del Trueno"}, "de": {"Insel des Donners"}, "fr": {"Île du Tonnerre"}, "it": {"Isola del Tuono"}, "ru": {"Остров\\p{L}* Гром\\p{L}*"}, "zhCN": {"雷神岛"}, "zhTW": {"雷王島"}, "ko": {"천둥의 섬"}}},
	{map[string][]string{"en": {"Temple of the Jade Serpent"}, "pt": {"Templo da Serpente de Jade"}, "es": {"Templo del Dragón de Jade"}, "de": {"Tempel der Jadeschlange"}, "fr": {"Temple du Serpent de jade"}, "it": {"Tempio della Serpe di Giada"}, "ru": {"Храм\\p{L}* Н[её]фритово\\p{L}* Зм[её]\\p{L}*"}, "zhCN": {"青龙寺"}, "zhTW": {"玉蛟寺"}, "ko": {"옥룡사"}}},
	{map[string][]string{"en": {"Stormstout Brewery"}, "pt": {"Cervejaria Malte do Trovão"}, "es": {"Cervecería del Trueno"}, "de": {"Brauerei Sturmbräu"}, "fr": {"Brasserie Brune d’Orage", "Brasserie Brune d'Orage"}, "it": {"Birrificio Triplo Malto"}, "ru": {"Хм[её]л[её]варн\\p{L}* Буйных\\p{L}* Порт[её]ров\\p{L}*"}, "zhCN": {"风暴烈酒酿造厂"}, "zhTW": {"風暴烈酒酒坊"}, "ko": {"스톰스타우트 양조장"}}},
	{map[string][]string{"en": {"Shado-Pan Monastery"}, "pt": {"Monastério Shado-pan"}, "es": {"Monasterio del Shadopan"}, "de": {"Das Shado-Pan-Kloster", "Shado-Pan-Kloster"}, "fr": {"Monastère des Pandashan"}, "it": {"Monastero degli Shandaren"}, "ru": {"Монастыр\\p{L}* Шадо-Пан\\p{L}*"}, "zhCN": {"影踪禅院"}, "zhTW": {"影潘僧院"}, "ko": {"음영파 수도원"}}},
	{map[string][]string{"en": {"Mogu'shan Palace"}, "pt": {"Palácio Mogu'shan"}, "es": {"Palacio Mogu'shan"}, "de": {"Mogu'shanpalast"}, "fr": {"Palais Mogu’shan", "Palais Mogu'shan"}, "it": {"Palazzo Mogu'shan"}, "ru": {"Двор[её]ц\\p{L}* Могу'шан\\p{L}*"}, "zhCN": {"魔古山宫殿"}, "zhTW": {"魔古山宮"}, "ko": {"모구샨 궁전"}}},
	{map[string][]string{"en": {"Gate of the Setting Sun"}, "pt": {"Portal do Sol Poente"}, "es": {"Puerta del Sol Poniente"}, "de": {"Das Tor der Untergehenden Sonne", "Tor der Untergehenden Sonne"}, "fr": {"Porte du Soleil couchant"}, "it": {"Porta del Sole Calante"}, "ru": {"Врат\\p{L}* Заходящ[её]г\\p{L}* Солнц\\p{L}*"}, "zhCN": {"残阳关"}, "zhTW": {"落陽關"}, "ko": {"석양문"}}},
	{map[string][]string{"en": {"Siege of Niuzao Temple"}, "pt": {"Cerco ao Templo Niuzao"}, "es": {"Asedio del Templo de Niuzao"}, "de": {"Belagerung des Niuzaotempels"}, "fr": {"Siège du temple de Niuzao"}, "it": {"Assedio al Tempio di Niuzao"}, "ru": {"Осад\\p{L}* храм\\p{L}* Нюцза\\p{L}*"}, "zhCN": {"围攻砮皂寺"}, "zhTW": {"圍攻怒兆寺"}, "ko": {"니우짜오 사원 공성전투"}}},
	{map[string][]string{"en": {"Mogu'shan Vaults"}, "pt": {"Galerias Mogu'shan"}, "es": {"Cámaras Mogu'shan"}, "de": {"Mogu'shangewölbe"}, "fr": {"Caveaux Mogu’shan", "Caveaux Mogu'shan"}, "it": {"Segrete Mogu'shan"}, "ru": {"Подз[её]м[её]ль\\p{L}* Могу'шан\\p{L}*"}, "zhCN": {"魔古山宝库"}, "zhTW": {"魔古山寶庫"}, "ko": {"모구샨 금고"}}},
	{map[string][]string{"en": {"Heart of Fear"}, "pt": {"Coração do Medo"}, "es": {"Corazón del Miedo"}, "de": {"Das Herz der Angst", "Herz der Angst"}, "fr": {"Cœur de la peur"}, "it": {"Cuore della Paura"}, "ru": {"С[её]рдц\\p{L}* Страх\\p{L}*"}, "zhCN": {"恐惧之心"}, "zhTW": {"恐懼之心"}, "ko": {"공포의 심장"}}},
	{map[string][]string{"en": {"Terrace of Endless Spring"}, "pt": {"Terraço da Primavera Eterna"}, "es": {"Veranda de la Primavera Eterna"}, "de": {"Terrasse des Endlosen Frühlings"}, "fr": {"Terrasse Printanière"}, "it": {"Terrazza dell'Eterna Primavera"}, "ru": {"Т[её]ррас\\p{L}* В[её]чно\\p{L}* В[её]сн\\p{L}*"}, "zhCN": {"永春台"}, "zhTW": {"豐泉台"}, "ko": {"영원한 봄의 정원"}}},
	{map[string][]string{"en": {"Throne of Thunder"}, "pt": {"Trono do Trovão"}, "es": {"Solio del Trueno"}, "de": {"Thron des Donners"}, "fr": {"Trône du tonnerre"}, "it": {"Regno del Tuono"}, "ru": {"Пр[её]стол\\p{L}* Гроз\\p{L}*"}, "zhCN": {"雷电王座"}, "zhTW": {"雷霆王座"}, "ko": {"천둥의 왕좌"}}},
	{map[string][]string{"en": {"Siege of Orgrimmar"}, "pt": {"Cerco a Orgrimmar"}, "es": {"Asedio de Orgrimmar"}, "de": {"Schlacht um Orgrimmar"}, "fr": {"Siège d’Orgrimmar", "Siège d'Orgrimmar"}, "it": {"Assedio di Orgrimmar"}, "ru": {"Осад\\p{L}* Оргриммар\\p{L}*"}, "zhCN": {"决战奥格瑞玛"}, "zhTW": {"圍攻奧格瑪"}, "ko": {"오그리마 공성전"}}},
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

// placeAlt is one spelling of one place. Each has its own small regexp and a
// literal prefix, so a message is only run against the few spellings whose
// start actually appears in it (one big alternation got slow as the list grew).
type placeAlt struct {
	re   *regexp.Regexp
	lit  string // lower-case literal start of the pattern, for a quick Contains
	idx  int    // index into places
	size int
	word bool // Latin/Cyrillic: must be a whole word
}

var placeAlts = buildPlaceAlts()

func buildPlaceAlts() []placeAlt {
	var alts []placeAlt
	for i, p := range places {
		for loc, forms := range p.loc {
			for _, f := range forms {
				pat := f
				if loc != "ru" {
					pat = regexp.QuoteMeta(f)
				}
				lit := f
				if loc == "ru" {
					if j := strings.IndexAny(f, `[\(`); j >= 0 {
						lit = f[:j]
					}
				}
				if j := strings.IndexByte(lit, ' '); j >= 0 {
					lit = lit[:j]
				}
				pat = strings.ReplaceAll(pat, " ", `\s+`)
				r, _ := utf8.DecodeRuneInString(f)
				word := !unicode.In(r, unicode.Han, unicode.Hangul, unicode.Katakana, unicode.Hiragana)
				alts = append(alts, placeAlt{regexp.MustCompile(`(?i)` + pat), strings.ToLower(lit), i, len(f), word})
			}
		}
	}
	// Longest first so "The Deadmines" beats "Deadmines".
	sort.SliceStable(alts, func(a, b int) bool { return alts[a].size > alts[b].size })
	return alts
}

func isWordRune(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }

// localizePlaces rewrites every known WoW place name in text to the name the
// reader of language target knows, marked so the engines keep it.
func localizePlaces(text, target string) string {
	type hit struct{ start, end, idx, size int }
	var hits []hit
	lower := strings.ToLower(text)
	for _, a := range placeAlts {
		if !strings.Contains(lower, a.lit) {
			continue
		}
		for _, m := range a.re.FindAllStringIndex(text, -1) {
			start, end := m[0], m[1]
			if a.word {
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
			hits = append(hits, hit{start, end, a.idx, a.size})
		}
	}
	if len(hits) == 0 {
		return text
	}
	// Never touch the player's own never-translate words.
	if locked := lockRe.FindAllStringIndex(text, -1); len(locked) > 0 {
		kept := hits[:0]
		for _, h := range hits {
			inside := false
			for _, l := range locked {
				if h.start < l[1] && h.end > l[0] {
					inside = true
					break
				}
			}
			if !inside {
				kept = append(kept, h)
			}
		}
		if hits = kept; len(hits) == 0 {
			return text
		}
	}
	// Leftmost first; at the same spot the longest spelling wins.
	sort.SliceStable(hits, func(a, b int) bool {
		if hits[a].start != hits[b].start {
			return hits[a].start < hits[b].start
		}
		return hits[a].end-hits[a].start > hits[b].end-hits[b].start
	})
	col := placeLocale(target)
	var b strings.Builder
	last := 0
	for _, h := range hits {
		if h.start < last { // overlaps a name already replaced
			continue
		}
		p := places[h.idx]
		name := p.loc["en"][0]
		if f, ok := p.loc[col]; ok && col != "ru" {
			name = f[0]
		} else if col == "ru" {
			if f, ok := ruDisplay[name]; ok {
				name = f
			}
		}
		b.WriteString(text[last:h.start])
		b.WriteString(keep(name))
		last = h.end
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
	"Outland":                    "Запределье",
	"Shattrath City":             "Шаттрат",
	"Silvermoon City":            "Луносвет",
	"The Exodar":                 "Экзодар",
	"Hellfire Peninsula":         "Полуостров Адского Пламени",
	"Zangarmarsh":                "Зангартопь",
	"Terokkar Forest":            "Лес Тероккар",
	"Nagrand":                    "Награнд",
	"Blade's Edge Mountains":     "Острогорье",
	"Netherstorm":                "Пустоверть",
	"Shadowmoon Valley":          "Долина Призрачной Луны",
	"Isle of Quel'Danas":         "Остров Кель'Данас",
	"Hellfire Ramparts":          "Бастионы Адского Пламени",
	"The Blood Furnace":          "Кузня Крови",
	"The Shattered Halls":        "Разрушенные залы",
	"The Slave Pens":             "Узилище",
	"The Underbog":               "Нижетопь",
	"The Steamvault":             "Паровое подземелье",
	"Mana-Tombs":                 "Гробницы Маны",
	"Auchenai Crypts":            "Аукенайские гробницы",
	"Sethekk Halls":              "Сетеккские залы",
	"Shadow Labyrinth":           "Темный лабиринт",
	"The Mechanar":               "Механар",
	"The Botanica":               "Ботаника",
	"The Arcatraz":               "Аркатрац",
	"Old Hillsbrad Foothills":    "Старые предгорья Хилсбрада",
	"The Black Morass":           "Черные топи",
	"Magisters' Terrace":         "Терраса Магистров",
	"Karazhan":                   "Каражан",
	"Gruul's Lair":               "Логово Груула",
	"Magtheridon's Lair":         "Логово Магтеридона",
	"Serpentshrine Cavern":       "Змеиное святилище",
	"Tempest Keep":               "Крепость Бурь",
	"Hyjal Summit":               "Вершина Хиджала",
	"Black Temple":               "Черный храм",
	"Zul'Aman":                   "Зул'Аман",
	"Sunwell Plateau":            "Плато Солнечного Колодца",
	"Pandaria":                   "Пандария",
	"The Jade Forest":            "Нефритовый лес",
	"Valley of the Four Winds":   "Долина Четырех Ветров",
	"Krasarang Wilds":            "Красарангские джунгли",
	"Kun-Lai Summit":             "Вершина Кунь-Лай",
	"Townlong Steppes":           "Танлунские степи",
	"Dread Wastes":               "Жуткие пустоши",
	"Vale of Eternal Blossoms":   "Вечноцветущий дол",
	"Shrine of Seven Stars":      "Святилище Семи Звезд",
	"Shrine of Two Moons":        "Святилище Двух Лун",
	"Timeless Isle":              "Вневременный остров",
	"Isle of Thunder":            "Остров Грома",
	"Temple of the Jade Serpent": "Храм Нефритовой Змеи",
	"Stormstout Brewery":         "Хмелеварня Буйных Портеров",
	"Shado-Pan Monastery":        "Монастырь Шадо-Пан",
	"Mogu'shan Palace":           "Дворец Могу'шан",
	"Gate of the Setting Sun":    "Врата Заходящего Солнца",
	"Siege of Niuzao Temple":     "Осада храма Нюцзао",
	"Mogu'shan Vaults":           "Подземелья Могу'шан",
	"Heart of Fear":              "Сердце Страха",
	"Terrace of Endless Spring":  "Терраса Вечной Весны",
	"Throne of Thunder":          "Престол Гроз",
	"Siege of Orgrimmar":         "Осада Оргриммара",
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
	return localizePlaces(lockUserWords(text), target)
}

func prepareReply(text, target string) string { // text is English
	return localizePlaces(lockUserWords(localizeRoleVerbs(text, target)), target)
}

// sameMeaning: the back-translation matches what you typed, ignoring case,
// spaces and punctuation.
func sameMeaning(a, b string) bool {
	norm := func(s string) string {
		return strings.Map(func(r rune) rune {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				return unicode.ToLower(r)
			}
			return -1
		}, s)
	}
	return norm(a) == norm(b)
}
