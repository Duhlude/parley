# Changelog

## App 1.9.0 (2026-09-30)

### Added
- **Back-translation check.** Under each reply Parley shows it translated back into your language (*Reads back as: …*, or *Reads back the same.*), so you can see what the other player will read before you paste it. Tray → *Check replies by translating them back* (on by default; with DeepL or Azure it uses a few extra characters).
- **Conversation view.** Right-click a message → *Conversation with …* shows only that player's messages and your replies to them, and sends your next reply to them. Click the bar at the top to see everything again.

### Fixed
- The mouse pointer over Parley's menus could stay a spinning "busy" or resize cursor.
- WoW skins: "You → PT-BR" and other arrows showed as a gap (the WoW font has no arrow); they now show as "»".
- Clicking the language or quick-reply button to close its menu opened it again straight away; clicking a message to close a menu also selected that message.
- A stuck offline translator could freeze translation and stop Parley from quitting (which also broke one-click updates). Requests now time out, and quitting no longer waits for it. After repeated crashes the offline translator is tried again after two minutes instead of staying off until restart.
- Settings could be lost if two saves happened at the same moment; a damaged settings file now falls back to the last good copy.
- Your never-translate list now also applies to what you type in the reply box, and wins over place names inside it ("Ironforge Brewers" stays as written).
- Finishing a reply no longer erases text you had started typing, e.g. under a quick reply.
- The first message after a /reload in WoW could be skipped.
- The tray icon comes back if Windows Explorer restarts.
- Touchpads with smooth scrolling can scroll the overlay and menus.
- The list no longer looks empty after clearing or muting while scrolled up.
- Language downloads no longer fail on slow connections (the 5-minute limit is gone).
- Installer: an update now replaces files as one unit (if anything fails, the previous version is put back and started again), and no console windows flash up.

### Faster
- The overlay no longer redraws every second when nothing changed, and remembers text sizes instead of measuring every message on every redraw.
- Parley checks for WoW twice a second instead of 28 times while WoW isn't running or is minimized.
- Busy chat is translated a few messages at a time, and messages that already scrolled out of the list are skipped (saving DeepL/Azure characters).

## App 1.8.0 (2026-09-30)

### Added
- **Never-translate list.** Settings → *Never translate these names or words*: guild names, nicknames and other words Parley leaves exactly as written, in messages and replies, with every engine.
- **Burning Crusade and Mists of Pandaria place names.** About 60 zones, cities, dungeons and raids (Shattrath, Karazhan, Hellfire Ramparts, Temple of the Jade Serpent, Siege of Orgrimmar…) are recognised in every WoW language and shown by the name your client uses, like the Classic ones already were. Names from the game's own localizations (via LibBabble-SubZone-3.0). Place-name matching is also much faster.

### Changed
- Settings: the WoW folder label no longer says "Classic Era or Retail", since any supported version works.

## App 1.7.0 · Addon 1.6.0 (2026-09-29)

### Added
- **One-click updates.** When a new version is out, tray → *Update to Parley x.y.z* downloads the installer from GitHub, checks it against GitHub's published SHA-256 checksum, installs it quietly and restarts Parley. Settings are kept and the WoW addon is updated too. If anything goes wrong nothing is installed and the release page opens instead. (Installer: new `/update` mode for this.)
- **More WoW versions (beta).** The addon now also loads on the **Anniversary** realms (`_anniversary_`) and **Mists of Pandaria Classic** (`_classic_`), besides Classic Era and Retail. The app and installer install it into every one they find.

### Changed
- The update check now runs a few seconds after Parley opens (it used to wait until later), then every 6 hours while Parley is running. If the check fails, for example because Parley started with Windows before the network was up, it tries again a few minutes later.
- Repository reorganized: app source in `app/`, prebuilt helpers in `bin/` (see docs/BUILDING.md). No change for players.

### Addon 1.6.0
- Loads on Classic Era, Anniversary, Mists of Pandaria Classic and Retail (`## Interface: 11509, 20506, 50504, 120100`).

## App 1.6.0 · Addon 1.5.0 (2026-09-29)

### Added
- **Azure Translator engine.** A second online option next to DeepL: 2 million characters a month free on Microsoft's free tier. Paste the key and region under Settings → Translation engine → Azure. Falls back to offline like DeepL does. The translation benchmark tests it too.
- **Quick replies.** The speech-bubble button in the reply bar opens ten common phrases (*Invite me, please*, *On my way!*, *Thanks for the group!*, *Sorry, I don't speak your language well*…), already translated into Portuguese, Spanish, Russian, German, French, Italian, Korean, Chinese and Japanese, so they're copied instantly.
- **Alerts.** Messages that mention your character or your own alert words (Settings → *Alert me when a message mentions*) get an orange bar and a soft notification sound (tray → Alert sound).
- **Auto-hide.** Tray → *Auto-hide overlay*: after 10 s, 30 s, 1 or 2 minutes without new chat the overlay fades out (and stops catching clicks); it fades back in when a foreign message arrives, on an alert or notice, with Ctrl+Shift+T or a click on the tray icon. It stays put while you hover it, type a reply or have a menu open. Off by default.
- **Tooltips everywhere.** Hover over any menu option, settings field or overlay button for a short explanation of what it does, in your interface language.
- **Update notice.** Parley checks GitHub about once a day and tells you when a new version is out (tray → Download Parley x.y.z; turn off under tray → Check for updates).

### Addon 1.5.0
- Tells the app which character you're playing (at login and every 5 minutes), for name alerts.


## App 1.5.0 · Addon 1.4.0 (2026-09-29)

### Added
- **Retail support (beta).** The addon loads on Retail (12.1) as well as Classic Era, and Parley installs it into both game folders when it finds them. During Mythic+ keys, PvP matches and boss fights Retail hides chat from addons; Parley shows a short note and picks up again afterwards.
- **Offline translation, no account needed.** Parley now translates on your PC with Mozilla's Firefox Translations models (bergamot engine, `parley-mt.exe`). It's the default engine: nothing to sign up for, no character limits, and chat text never leaves your computer.
- 30+ languages to and from English; other pairs (e.g. Russian → Portuguese) are translated through English.
- Each language downloads once (about 25–45 MB per direction) the first time it's needed, with progress in the overlay. Files are checked against Mozilla's published SHA-256 hashes.
- On-device language detection (CLD2).
- Settings → **Translation engine**: Offline or DeepL. DeepL is now optional; if it's unreachable, rejects the key or runs out of characters, Parley falls back to offline for that message.

- **Right-click a message**: copy the translation or original, reply to the sender, or **mute** them (tray → Unmute everyone to undo).
- **Click-through overlay** (tray): clicks go through to WoW; Ctrl+Shift+T to use it, Esc to go back.
- **Chat history**: translated chat and your replies are saved to a daily text file (tray → Chat history).
- Tray → **Translation**: switch engine, **download a language ahead of time**, open the models folder (with its size).
- Your own language is downloaded in the background when it isn't English.
- DeepL usage shown in Settings, with a warning at 90%.
- The offline engine frees its memory after 10 minutes without foreign chat.
- **Logo and icons**: a new Parley logo (two speech bubbles in a gold-ringed badge) for the app, tray, installer, the WoW skins' portrait and the addon (minimap button and AddOns list). Banner art for GitHub and CurseForge is in `assets/logo`.
- **Settings window fully skinned**: its own title bar and frame, dropdowns that open skinned menus, a skinned checkbox, and skinned message dialogs instead of Windows' white boxes.
- **Parley speaks your language.** The app, installer and addon follow your Windows (and WoW client) language: German, Spanish, French, Italian, Portuguese (Brazil), Russian, Korean and Chinese (Simplified and Traditional), with English as the fallback. Change it under Settings → **Interface language**. On first start, "Translate chat into" is also set from your Windows language. Language names in menus are shown in their own language (Español, Русский, 日本語 …).
- **Skinned menus**: the tray menu, the right-click message menu and the reply-language picker are now drawn in the current skin (WoW dropdown style for the WoW skins), with submenus, keyboard navigation and scrolling. The language picker is shorter: automatic, recent languages, then "All languages".

### Addon 1.4.0
- `/parley test all` sends samples in eight languages (Spanish, Russian, Portuguese, German, French, Japanese, Chinese, Korean); `/parley test ja` (or `es`, `zh`, `asia`, …) sends one.

### Fixed
- **Place names**: dungeons, raids, capitals and popular zones written in another client's language ("Мертвые копи", "Ventobravo", "死亡矿井", "데드마인") are shown with the name your client uses (Deadmines, Stormwind…), and your replies use theirs ("Deadmines" → "Minas Mortas" for a Brazilian). Works with both engines.
- **"I can tank" / "I'll heal" in replies** become the gamer verb in their language (tankar, tanquear, танковать, tanken) instead of an untranslated or quoted "tank".
- More chat slang understood: Russian (спс, лс, хил, агрить, инвайт, данж…) and Brazilian (vlw, valeu, bora, pras). "mana" is no longer read as Brazilian slang for "sister".
- Short Russian lines that the language detector couldn't place are treated as Russian.
- Offline translations no longer capitalize a word next to a WoW term ("healer For Deadmines", "raid Tomorrow").
- A short exclamation is no longer doubled ("Sim! Sim! Tenho 5…").
- Plain-ASCII Portuguese and Spanish trade chat ("Vendendo bolsas de seda, 40 prata cada") is no longer mistaken for English and skipped.

### Changed
- No more "add your DeepL key" warning on first start.
- Docs: install steps without DeepL, a "Performance & what it reads" section, and resolution support.

## App 1.4.0 (2026-09-29)

### Added
- Gaming slang dictionary built from BabelChat and WoW Translator (both MIT): 50 chat shortcuts and 155 WoW terms with explanations in up to 19 languages, including all Classic dungeon and raid abbreviations.
- Your replies: shortcuts like *sry, w8, idk, ty, omw* are spelled out before translation, so the other player gets proper words.
- Incoming chat: when you translate into a language other than English, English shortcuts in foreign messages are spelled out, and a **Terms:** line explains WoW jargon in your language (e.g. *SM Cath: Mosteiro (Catedral)*, *w2w: …*). Toggle under tray icon → **Explain gaming terms**.
- Messages made only of gamer shorthand ("ty np gg wp") are recognised as English and skipped.

## App 1.3.1 · Addon 1.2.0 (2026-09-29)

### Added
- In-game settings window: `/parley`, a minimap button, or Esc → Options → AddOns → Parley.
- Six built-in presets (Everything, Friends & group, Group only, Social, Nearby & whispers, Friends only) and up to 8 saved presets of your own.
- Chat-type checkboxes, a pause switch (also right-click on the minimap button), test messages and a strip toggle in the window.
- `/parley preset <name>` applies any preset by name; `/parley status` shows the status in chat.

### Changed
- `/parley` on its own now opens the settings window instead of printing help.
- The installer copies every addon file, now including `ParleyUI.lua`.

## App 1.3.0 · Addon 1.1.1 (2026-09-29)

First public release candidate.

### Added
- Voice replies with on-device speech recognition (whisper.cpp): no popup, no online speech service. Accurate (466 MB) and Fast (148 MB) models, auto-send after speaking, voice log.
- Skins: Classic WoW, Retail WoW (windows), Dragonflight UI, WoW chat frame, Parley (modern). Retail has a title strip, an overlapping gold-ringed portrait, a status line, a red close box, framed message rows and a dropdown-style language picker. Dragonflight has a teardrop unit-frame portrait, a name plate, a green "health bar" status and bevelled slot rows. Optional Friz Quadrata support.
- `Parley-Setup.exe` per-user installer: shortcuts, optional start with Windows, uninstaller, automatic WoW addon install, and in-place updates that close a running Parley.
- Automatic detection of the WoW Classic Era folder (Battle.net registry entry and common locations on any drive).
- Addon: `/parley preset`, `/parley all`, `/parley friendsonly` (whispers only from friends).
- Documentation: install guide, user guide, how it works, privacy, third-party notices.

### Changed
- Replies: you choose the recipient by clicking a message, and the reply language is set separately, following the recipient's language automatically.
- Translation: WoW jargon and item links stay in English, casual tone, a game-chat context hint for DeepL, and Brazilian Portuguese shorthand expansion.
- English slang in plain Latin letters is no longer mis-detected as another language.

### Fixed
- Names and messages in other alphabets now always render with a font that supports them.

## App 1.0.0 · Addon 1.0.0 (2026-09-28)

- First working version: pixel-strip bridge, DeepL translation, overlay, replies via clipboard.
