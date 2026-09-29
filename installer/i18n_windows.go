// Installer text in the Windows display language (same languages as Parley).

package main

import (
	"strings"
	"syscall"
	"unsafe"
)

var setupText = map[string]map[string]string{
	"de": {
		"Parley Setup": "Parley-Setup",
		"Parley is still running. Quit it from its tray icon, then click Retry.":                                                                                                 "Parley läuft noch. Beende es über das Taskleistensymbol und klicke dann auf „Wiederholen“.",
		"Install Parley %s (live WoW chat translation)?\n\nIt goes into:\n%s\n\nNo admin rights needed. If Parley is already installed it's updated and your settings are kept.": "Parley %s installieren (Live-Übersetzung für den WoW-Chat)?\n\nZielordner:\n%s\n\nKeine Administratorrechte nötig. Ist Parley schon installiert, wird es aktualisiert und deine Einstellungen bleiben erhalten.",
		"Couldn't create the install folder:":             "Der Installationsordner konnte nicht erstellt werden:",
		"Install failed while copying files:":             "Installation beim Kopieren der Dateien fehlgeschlagen:",
		"Start Parley automatically when Windows starts?": "Parley beim Windows-Start automatisch starten?",
		"Parley is installed.":                            "Parley ist installiert.",
		"Next: in WoW type /parley test. Parley translates offline, no account needed; each language downloads once the first time it's used.": "Als Nächstes: Gib in WoW /parley test ein. Parley übersetzt offline, ganz ohne Konto; jede Sprache wird beim ersten Gebrauch einmalig heruntergeladen.",
		"Start Parley now?":           "Parley jetzt starten?",
		"Remove Parley from this PC?": "Parley von diesem PC entfernen?",
		"Your settings in %APPDATA%\\Parley and the WoW addon are left in place.": "Deine Einstellungen in %APPDATA%\\Parley und das WoW-Addon bleiben erhalten.",
		"Parley has been removed.":          "Parley wurde entfernt.",
		"The WoW addon was installed to:":   "Das WoW-Addon wurde installiert in:",
		"(type /reload if WoW is running).": "(gib /reload ein, falls WoW läuft).",
		"Couldn't find WoW automatically. Parley will offer to install the addon when you set your WoW folder in its settings; a copy is also in the addon folder inside:": "WoW wurde nicht automatisch gefunden. Parley bietet die Installation des Addons an, sobald du den WoW-Ordner in den Einstellungen festlegst; eine Kopie liegt außerdem im Ordner addon unter:",
	},
	"es": {
		"Parley Setup": "Instalación de Parley",
		"Parley is still running. Quit it from its tray icon, then click Retry.":                                                                                                 "Parley sigue abierto. Ciérralo desde su icono de la bandeja y pulsa Reintentar.",
		"Install Parley %s (live WoW chat translation)?\n\nIt goes into:\n%s\n\nNo admin rights needed. If Parley is already installed it's updated and your settings are kept.": "¿Instalar Parley %s (traducción en directo del chat de WoW)?\n\nSe instalará en:\n%s\n\nNo hacen falta permisos de administrador. Si Parley ya está instalado, se actualiza y se conservan tus ajustes.",
		"Couldn't create the install folder:":             "No se pudo crear la carpeta de instalación:",
		"Install failed while copying files:":             "La instalación falló al copiar los archivos:",
		"Start Parley automatically when Windows starts?": "¿Iniciar Parley automáticamente con Windows?",
		"Parley is installed.":                            "Parley está instalado.",
		"Next: in WoW type /parley test. Parley translates offline, no account needed; each language downloads once the first time it's used.": "Siguiente paso: en WoW escribe /parley test. Parley traduce sin conexión y sin cuenta; cada idioma se descarga una sola vez la primera vez que se usa.",
		"Start Parley now?":           "¿Iniciar Parley ahora?",
		"Remove Parley from this PC?": "¿Quitar Parley de este PC?",
		"Your settings in %APPDATA%\\Parley and the WoW addon are left in place.": "Tus ajustes en %APPDATA%\\Parley y el addon de WoW se conservan.",
		"Parley has been removed.":          "Parley se ha quitado.",
		"The WoW addon was installed to:":   "El addon de WoW se instaló en:",
		"(type /reload if WoW is running).": "(escribe /reload si WoW está abierto).",
		"Couldn't find WoW automatically. Parley will offer to install the addon when you set your WoW folder in its settings; a copy is also in the addon folder inside:": "No se encontró WoW automáticamente. Parley te ofrecerá instalar el addon cuando indiques la carpeta de WoW en sus ajustes; también hay una copia en la carpeta addon dentro de:",
	},
	"fr": {
		"Parley Setup": "Installation de Parley",
		"Parley is still running. Quit it from its tray icon, then click Retry.":                                                                                                 "Parley est encore lancé. Quittez-le depuis son icône dans la barre des tâches, puis cliquez sur Réessayer.",
		"Install Parley %s (live WoW chat translation)?\n\nIt goes into:\n%s\n\nNo admin rights needed. If Parley is already installed it's updated and your settings are kept.": "Installer Parley %s (traduction en direct du chat de WoW) ?\n\nDossier d’installation :\n%s\n\nAucun droit administrateur requis. Si Parley est déjà installé, il est mis à jour et vos paramètres sont conservés.",
		"Couldn't create the install folder:":             "Impossible de créer le dossier d’installation :",
		"Install failed while copying files:":             "L’installation a échoué pendant la copie des fichiers :",
		"Start Parley automatically when Windows starts?": "Lancer Parley automatiquement au démarrage de Windows ?",
		"Parley is installed.":                            "Parley est installé.",
		"Next: in WoW type /parley test. Parley translates offline, no account needed; each language downloads once the first time it's used.": "Ensuite : dans WoW, tapez /parley test. Parley traduit hors ligne, sans compte ; chaque langue est téléchargée une seule fois, à sa première utilisation.",
		"Start Parley now?":           "Lancer Parley maintenant ?",
		"Remove Parley from this PC?": "Supprimer Parley de ce PC ?",
		"Your settings in %APPDATA%\\Parley and the WoW addon are left in place.": "Vos paramètres dans %APPDATA%\\Parley et l’addon WoW sont conservés.",
		"Parley has been removed.":          "Parley a été supprimé.",
		"The WoW addon was installed to:":   "L’addon WoW a été installé dans :",
		"(type /reload if WoW is running).": "(tapez /reload si WoW est lancé).",
		"Couldn't find WoW automatically. Parley will offer to install the addon when you set your WoW folder in its settings; a copy is also in the addon folder inside:": "WoW introuvable automatiquement. Parley proposera d’installer l’addon quand vous indiquerez le dossier de WoW dans ses paramètres ; une copie se trouve aussi dans le dossier addon de :",
	},
	"it": {
		"Parley Setup": "Installazione di Parley",
		"Parley is still running. Quit it from its tray icon, then click Retry.":                                                                                                 "Parley è ancora aperto. Chiudilo dall'icona nella barra, poi fai clic su Riprova.",
		"Install Parley %s (live WoW chat translation)?\n\nIt goes into:\n%s\n\nNo admin rights needed. If Parley is already installed it's updated and your settings are kept.": "Installare Parley %s (traduzione in tempo reale della chat di WoW)?\n\nVerrà installato in:\n%s\n\nNon servono diritti di amministratore. Se Parley è già installato viene aggiornato e le impostazioni restano.",
		"Couldn't create the install folder:":             "Impossibile creare la cartella di installazione:",
		"Install failed while copying files:":             "Installazione non riuscita durante la copia dei file:",
		"Start Parley automatically when Windows starts?": "Avviare Parley automaticamente all'avvio di Windows?",
		"Parley is installed.":                            "Parley è installato.",
		"Next: in WoW type /parley test. Parley translates offline, no account needed; each language downloads once the first time it's used.": "Ora: in WoW scrivi /parley test. Parley traduce offline, senza account; ogni lingua viene scaricata una sola volta, al primo utilizzo.",
		"Start Parley now?":           "Avviare Parley adesso?",
		"Remove Parley from this PC?": "Rimuovere Parley da questo PC?",
		"Your settings in %APPDATA%\\Parley and the WoW addon are left in place.": "Le impostazioni in %APPDATA%\\Parley e l'addon di WoW restano al loro posto.",
		"Parley has been removed.":          "Parley è stato rimosso.",
		"The WoW addon was installed to:":   "L'addon di WoW è stato installato in:",
		"(type /reload if WoW is running).": "(scrivi /reload se WoW è aperto).",
		"Couldn't find WoW automatically. Parley will offer to install the addon when you set your WoW folder in its settings; a copy is also in the addon folder inside:": "WoW non trovato automaticamente. Parley proporrà di installare l'addon quando imposterai la cartella di WoW nelle impostazioni; una copia si trova anche nella cartella addon in:",
	},
	"pt": {
		"Parley Setup": "Instalação do Parley",
		"Parley is still running. Quit it from its tray icon, then click Retry.":                                                                                                 "O Parley ainda está aberto. Feche-o pelo ícone da bandeja e clique em Repetir.",
		"Install Parley %s (live WoW chat translation)?\n\nIt goes into:\n%s\n\nNo admin rights needed. If Parley is already installed it's updated and your settings are kept.": "Instalar o Parley %s (tradução ao vivo do chat do WoW)?\n\nPasta de instalação:\n%s\n\nNão precisa de permissão de administrador. Se o Parley já estiver instalado, ele é atualizado e suas configurações são mantidas.",
		"Couldn't create the install folder:":             "Não foi possível criar a pasta de instalação:",
		"Install failed while copying files:":             "A instalação falhou ao copiar os arquivos:",
		"Start Parley automatically when Windows starts?": "Iniciar o Parley automaticamente com o Windows?",
		"Parley is installed.":                            "O Parley está instalado.",
		"Next: in WoW type /parley test. Parley translates offline, no account needed; each language downloads once the first time it's used.": "Próximo passo: no WoW, digite /parley test. O Parley traduz offline, sem conta; cada idioma é baixado uma única vez, na primeira vez que for usado.",
		"Start Parley now?":           "Iniciar o Parley agora?",
		"Remove Parley from this PC?": "Remover o Parley deste PC?",
		"Your settings in %APPDATA%\\Parley and the WoW addon are left in place.": "Suas configurações em %APPDATA%\\Parley e o addon do WoW são mantidos.",
		"Parley has been removed.":          "O Parley foi removido.",
		"The WoW addon was installed to:":   "O addon do WoW foi instalado em:",
		"(type /reload if WoW is running).": "(digite /reload se o WoW estiver aberto).",
		"Couldn't find WoW automatically. Parley will offer to install the addon when you set your WoW folder in its settings; a copy is also in the addon folder inside:": "Não foi possível encontrar o WoW automaticamente. O Parley vai oferecer a instalação do addon quando você definir a pasta do WoW nas configurações; também há uma cópia na pasta addon dentro de:",
	},
	"ru": {
		"Parley Setup": "Установка Parley",
		"Parley is still running. Quit it from its tray icon, then click Retry.":                                                                                                 "Parley ещё запущен. Закройте его через значок в трее и нажмите «Повторить».",
		"Install Parley %s (live WoW chat translation)?\n\nIt goes into:\n%s\n\nNo admin rights needed. If Parley is already installed it's updated and your settings are kept.": "Установить Parley %s (перевод чата WoW в реальном времени)?\n\nПапка установки:\n%s\n\nПрава администратора не нужны. Если Parley уже установлен, он обновится, а настройки сохранятся.",
		"Couldn't create the install folder:":             "Не удалось создать папку установки:",
		"Install failed while copying files:":             "Ошибка при копировании файлов:",
		"Start Parley automatically when Windows starts?": "Запускать Parley вместе с Windows?",
		"Parley is installed.":                            "Parley установлен.",
		"Next: in WoW type /parley test. Parley translates offline, no account needed; each language downloads once the first time it's used.": "Дальше: введите в WoW /parley test. Parley переводит офлайн, без учётной записи; каждый язык скачивается один раз, при первом использовании.",
		"Start Parley now?":           "Запустить Parley сейчас?",
		"Remove Parley from this PC?": "Удалить Parley с этого компьютера?",
		"Your settings in %APPDATA%\\Parley and the WoW addon are left in place.": "Настройки в %APPDATA%\\Parley и аддон WoW останутся на месте.",
		"Parley has been removed.":          "Parley удалён.",
		"The WoW addon was installed to:":   "Аддон WoW установлен в:",
		"(type /reload if WoW is running).": "(введите /reload, если WoW запущен).",
		"Couldn't find WoW automatically. Parley will offer to install the addon when you set your WoW folder in its settings; a copy is also in the addon folder inside:": "Не удалось автоматически найти WoW. Parley предложит установить аддон, когда вы укажете папку WoW в настройках; копия также лежит в папке addon внутри:",
	},
	"ko": {
		"Parley Setup": "Parley 설치",
		"Parley is still running. Quit it from its tray icon, then click Retry.":                                                                                                 "Parley가 아직 실행 중입니다. 트레이 아이콘에서 종료한 뒤 다시 시도를 누르세요.",
		"Install Parley %s (live WoW chat translation)?\n\nIt goes into:\n%s\n\nNo admin rights needed. If Parley is already installed it's updated and your settings are kept.": "Parley %s(WoW 채팅 실시간 번역)을(를) 설치할까요?\n\n설치 위치:\n%s\n\n관리자 권한이 필요 없습니다. 이미 설치되어 있으면 업데이트되며 설정은 유지됩니다.",
		"Couldn't create the install folder:":             "설치 폴더를 만들 수 없습니다:",
		"Install failed while copying files:":             "파일을 복사하는 중 설치에 실패했습니다:",
		"Start Parley automatically when Windows starts?": "Windows를 시작할 때 Parley를 자동으로 실행할까요?",
		"Parley is installed.":                            "Parley를 설치했습니다.",
		"Next: in WoW type /parley test. Parley translates offline, no account needed; each language downloads once the first time it's used.": "다음 단계: WoW에서 /parley test를 입력하세요. Parley는 계정 없이 오프라인으로 번역하며, 각 언어는 처음 사용할 때 한 번만 내려받습니다.",
		"Start Parley now?":           "지금 Parley를 실행할까요?",
		"Remove Parley from this PC?": "이 PC에서 Parley를 제거할까요?",
		"Your settings in %APPDATA%\\Parley and the WoW addon are left in place.": "%APPDATA%\\Parley의 설정과 WoW 애드온은 그대로 남습니다.",
		"Parley has been removed.":          "Parley를 제거했습니다.",
		"The WoW addon was installed to:":   "WoW 애드온 설치 위치:",
		"(type /reload if WoW is running).": "(WoW가 실행 중이면 /reload를 입력하세요).",
		"Couldn't find WoW automatically. Parley will offer to install the addon when you set your WoW folder in its settings; a copy is also in the addon folder inside:": "WoW를 자동으로 찾지 못했습니다. 설정에서 WoW 폴더를 지정하면 Parley가 애드온 설치를 제안합니다. 사본은 다음 위치의 addon 폴더에도 있습니다:",
	},
	"zh-Hans": {
		"Parley Setup": "Parley 安装程序",
		"Parley is still running. Quit it from its tray icon, then click Retry.":                                                                                                 "Parley 仍在运行。请从托盘图标退出，然后点击“重试”。",
		"Install Parley %s (live WoW chat translation)?\n\nIt goes into:\n%s\n\nNo admin rights needed. If Parley is already installed it's updated and your settings are kept.": "安装 Parley %s（魔兽世界聊天实时翻译）？\n\n安装位置：\n%s\n\n无需管理员权限。如果已安装 Parley，将进行更新并保留你的设置。",
		"Couldn't create the install folder:":             "无法创建安装文件夹：",
		"Install failed while copying files:":             "复制文件时安装失败：",
		"Start Parley automatically when Windows starts?": "开机时自动启动 Parley？",
		"Parley is installed.":                            "Parley 已安装。",
		"Next: in WoW type /parley test. Parley translates offline, no account needed; each language downloads once the first time it's used.": "下一步：在游戏中输入 /parley test。Parley 离线翻译，无需账号；每种语言在首次使用时下载一次。",
		"Start Parley now?":           "现在启动 Parley？",
		"Remove Parley from this PC?": "从此电脑移除 Parley？",
		"Your settings in %APPDATA%\\Parley and the WoW addon are left in place.": "%APPDATA%\\Parley 中的设置和魔兽世界插件会保留。",
		"Parley has been removed.":          "Parley 已移除。",
		"The WoW addon was installed to:":   "魔兽世界插件已安装到：",
		"(type /reload if WoW is running).": "（如果游戏正在运行，请输入 /reload）。",
		"Couldn't find WoW automatically. Parley will offer to install the addon when you set your WoW folder in its settings; a copy is also in the addon folder inside:": "未能自动找到魔兽世界。在 Parley 设置中指定魔兽世界文件夹后，它会提示安装插件；以下位置的 addon 文件夹中也有一份副本：",
	},
	"zh-Hant": {
		"Parley Setup": "Parley 安裝程式",
		"Parley is still running. Quit it from its tray icon, then click Retry.":                                                                                                 "Parley 仍在執行。請從系統匣圖示結束，然後按「重試」。",
		"Install Parley %s (live WoW chat translation)?\n\nIt goes into:\n%s\n\nNo admin rights needed. If Parley is already installed it's updated and your settings are kept.": "要安裝 Parley %s（魔獸世界聊天即時翻譯）嗎？\n\n安裝位置：\n%s\n\n不需要系統管理員權限。如果已安裝 Parley，會進行更新並保留你的設定。",
		"Couldn't create the install folder:":             "無法建立安裝資料夾：",
		"Install failed while copying files:":             "複製檔案時安裝失敗：",
		"Start Parley automatically when Windows starts?": "要在 Windows 啟動時自動執行 Parley 嗎？",
		"Parley is installed.":                            "Parley 已安裝。",
		"Next: in WoW type /parley test. Parley translates offline, no account needed; each language downloads once the first time it's used.": "下一步：在遊戲中輸入 /parley test。Parley 離線翻譯，不需要帳號；每種語言在第一次使用時下載一次。",
		"Start Parley now?":           "要立即啟動 Parley 嗎？",
		"Remove Parley from this PC?": "要從這台電腦移除 Parley 嗎？",
		"Your settings in %APPDATA%\\Parley and the WoW addon are left in place.": "%APPDATA%\\Parley 中的設定和魔獸世界插件會保留。",
		"Parley has been removed.":          "Parley 已移除。",
		"The WoW addon was installed to:":   "魔獸世界插件已安裝到：",
		"(type /reload if WoW is running).": "（如果遊戲正在執行，請輸入 /reload）。",
		"Couldn't find WoW automatically. Parley will offer to install the addon when you set your WoW folder in its settings; a copy is also in the addon folder inside:": "無法自動找到魔獸世界。在 Parley 設定中指定魔獸世界資料夾後，它會提示安裝插件；以下位置的 addon 資料夾中也有一份副本：",
	},
}

var setupLang = detectLang()

// T translates installer text.
func T(s string) string {
	if t, ok := setupText[setupLang][s]; ok {
		return t
	}
	return s
}

func detectLang() string {
	k := syscall.NewLazyDLL("kernel32.dll")
	buf := make([]uint16, 85)
	lid, _, _ := k.NewProc("GetUserDefaultUILanguage").Call()
	name := ""
	if n, _, _ := k.NewProc("LCIDToLocaleName").Call(lid, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), 0); n > 0 {
		name = strings.ToLower(syscall.UTF16ToString(buf))
	}
	switch {
	case strings.HasPrefix(name, "zh"):
		if strings.Contains(name, "hant") || strings.HasSuffix(name, "-tw") || strings.HasSuffix(name, "-hk") || strings.HasSuffix(name, "-mo") {
			return "zh-Hant"
		}
		return "zh-Hans"
	}
	if i := strings.IndexAny(name, "-_"); i > 0 {
		name = name[:i]
	}
	if _, ok := setupText[name]; ok {
		return name
	}
	return "en"
}
