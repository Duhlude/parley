# Parley User Guide

## The overlay

Parley's window sits on top of WoW and has three parts:

- **Title bar**: the status (green **Live** means Parley is reading the game), plus the settings (gear), clear-messages and hide buttons.
- **Messages**: foreign-language chat, translated. Each message shows the channel and sender, the translation, and (optionally) the original text underneath.
- **Reply bar**: who you're replying to, the reply-language button, the text box and the microphone.

Drag the title bar to move the window and drag its edges to resize it. Keep it away from WoW's top-left corner, because that's where the addon's pixel strip appears.

Tray icon: **left-click** shows or hides the overlay. **Right-click** opens the menu with Settings, Skin, Voice input, Install/update addon, Clear messages and Quit.

## Message menu (right-click)

Right-click any message for:

- **Copy translation** / **Copy original**
- **Reply to …**: selects the sender and puts the cursor in the reply box.
- **Mute …**: hides everything that player sends from now on (gold sellers, spammers). Undo with tray icon → **Unmute everyone**.

## Click-through overlay

Tray icon → **Click-through overlay**: the overlay stays visible but mouse clicks go straight through to WoW, so you can't click it by accident mid-fight. Press **Ctrl+Shift+T** to use it (reply, scroll, click); pressing **Esc** or clicking back into WoW makes it click-through again.

## Chat history

Translated messages and your replies are saved to a daily text file in `%APPDATA%\Parley\history` (one file per day), so you can look up what someone said later. Tray icon → **Chat history** opens the folder or turns saving off.

## Replying

1. **Click the message** you want to answer. The line above the reply box shows where your reply goes, e.g. *To Carlos (whisper)* or *To 4. LookingForGroup channel, answering Иван*. Click the message again, or the ✕, to clear it.
2. **Type your reply** in your own language and press **Enter**. From anywhere, **Ctrl+Shift+T** jumps to the reply box.
3. Parley translates it, adds the right chat command (`/w Name`, `/p`, `/ra`, `/g`, `/4` …), copies it to the clipboard and switches back to WoW.
4. In WoW press **Enter, Ctrl+V, Enter**.

The **Reply in …** button sets the reply language. It follows the language of the person you're replying to automatically; pick a language from the list to lock it, or choose *Match their language automatically* to go back. **Tab** in the reply box cycles through recent languages, and **Esc** returns to WoW.

WoW chat lines are limited to 255 characters; Parley warns you if a translation is longer.

### Quick replies

The speech-bubble button next to the microphone opens ready-made replies: *Invite me, please*, *On my way!*, *One moment, please*, *Thanks for the group!*, *Thank you!*, *Sure, no problem*, *No, thank you*, *Sorry, I don't speak your language well*, *Where are you?* and *Good luck, have fun!*. They're already translated into Portuguese, Spanish, Russian, German, French, Italian, Korean, Chinese and Japanese, so they're copied instantly (other languages go through the translation engine). The menu shows each translation next to the phrase.

## Alerts

Messages that mention **your character** or one of your **alert words** get an orange bar and a soft Windows notification sound. Set the words under Settings → **Alert me when a message mentions**, separated by commas (e.g. `Deadmines, healer, WTB`). Your character's name is added automatically once the addon has told the app who you're playing. Turn the sound off under tray icon → **Alert sound**.

## Auto-hide

Tray icon → **Auto-hide overlay** → pick a delay (10 seconds to 2 minutes). When no new foreign chat has arrived for that long, the overlay fades out and gets out of the way completely: it doesn't block your screen or catch clicks. It fades back in when:

- a new foreign-language message arrives (or an alert / notice appears),
- you press **Ctrl+Shift+T**, or
- you click the tray icon.

It never hides while your mouse is over it, while you're typing a reply, or while one of its menus or the settings window is open. Choose **Off** to keep it visible all the time (the default).

## Updates

About once a day Parley checks GitHub for a newer version. When there is one, the overlay says so and the tray menu gets a **Download Parley x.y.z** item that opens the release page. Turn it off under tray icon → **Check for updates**.

## Voice replies

Click the **microphone**, or press **Ctrl+Shift+Y** from anywhere, and speak. Recording stops about a second after you stop talking (or when you click the mic again). By default the reply is then translated and copied automatically, so you only press Enter, Ctrl+V, Enter in WoW.

- Speech recognition runs **on your PC** with whisper.cpp. There's no popup, no chime, and no audio sent anywhere.
- The first use downloads a speech model into `%APPDATA%\Parley\models`: **Accurate** (466 MB, default) or **Fast** (148 MB). Switch under tray icon → **Voice input**.
- Under **Voice input** you can also turn off *Send automatically after speaking* (to review the text first), switch to Windows voice typing (Win+H) instead, or open the voice log for troubleshooting.
- Windows must allow desktop apps to use the microphone: Settings → Privacy & security → Microphone.

## Translation engines

Choose under Settings → **Translation engine**.

- **Offline** (default): translation runs on your PC with Mozilla's Firefox Translations models. No account, no limits, and chat text never leaves your computer. The first time a language shows up, Parley downloads it once (about 25–45 MB per direction; the overlay shows the progress). Models are kept in `%APPDATA%\Parley\models\translate`.
  - Supported: Spanish, Portuguese, French, German, Italian, Dutch, Polish, Russian, Ukrainian, Swedish, Danish, Norwegian, Finnish, Czech, Slovak, Slovenian, Croatian, Serbian, Bulgarian, Romanian, Hungarian, Greek, Turkish, Arabic, Hebrew, Persian, Chinese (Simplified and Traditional), Japanese, Korean, Vietnamese, Indonesian, Hindi and more, to and from English.
  - Two non-English languages (say Russian → Portuguese) are translated through English in two steps.
  - Quality is good for full sentences; heavy slang can come out rougher than DeepL. Parley expands common shorthand first to help.
- **Azure Translator** (Microsoft): 2 million characters a month free, then pay-as-you-go. Create a free Translator resource in the Azure portal ([how](https://learn.microsoft.com/azure/ai-services/translator/create-translator-resource)), then paste its key and region (for example `westeurope`; use `global` if your resource is global) in Settings. Stronger than offline for Chinese, Japanese and Korean. If Azure fails, Parley translates that message offline instead.
- **DeepL**: the best quality, but you need a DeepL API account and key. If DeepL is unreachable, rejects the key or runs out of characters, Parley translates that message offline instead. Settings shows how many DeepL characters you've used, and Parley warns you at 90%.

You can also switch engines from the tray icon → **Translation**. The same menu has **Download a language now**, which fetches a language ahead of time so the first message isn't delayed, and a link to the models folder with its size. If you read chat in a language other than English, Parley downloads that language automatically.

## What gets translated

Only messages that aren't in your language:

- Messages in other alphabets (Cyrillic, Chinese, Korean, Greek, Arabic …) are always translated.
- Messages in plain Latin letters are translated only if they contain recognisable foreign words. English slang, names and "ahhhh" are skipped for free.
- WoW jargon (tank, heal, dps, LFG, WTS, dungeon abbreviations like SM, BRD, UBRS) and `[Item Links]` are kept as they are.
- Brazilian Portuguese chat shorthand (vc, blz, tbm, pq, cmg, hj, vlw, kkkk …) is expanded to full words before translation.
- English chat shortcuts in **your replies** (sry, w8, idk, ty, omw, asap …) are spelled out before translation, so the other player gets proper words.
- **Explain gaming terms** (tray icon menu): adds a *Terms:* line under a translation explaining WoW jargon and dungeon abbreviations in your language, e.g. *lf1m: Looking for 1 more*, *SM Cath: Scarlet Monastery (Cathedral)*. It's on by default when you translate into a language other than English.

Choose which chat types are sent to Parley with the in-game commands below. `/parley preset` limits it to whispers from friends, party and raid chat.

## In-game settings window

Type **`/parley`**, or click the **Parley minimap button**, to open the settings window in WoW. It's also listed under Esc → Options → AddOns → Parley.

- **Send chat to the Parley app**: pauses or resumes translation. Right-clicking the minimap button does the same.
- **Presets**: one click switches which chat gets translated. The active preset is highlighted.

| Preset | Translates |
|---|---|
| Everything | All chat types, including public channels |
| Friends & group | Whispers from friends, party, raid and battlegrounds |
| Group only | Party, raid and battleground chat |
| Social | Whispers, say/yell and guild; no public channels |
| Nearby & whispers | Whispers plus people talking near you |
| Friends only | Only whispers from your friends list |

- **My presets**: set the chat types the way you like, type a name and click **Save current as preset**. You can save up to 8. Click one to apply it, or **X** to delete it.
- **Chat types to translate**: tick exactly what you want. *Only from friends* limits whispers to your friends list (characters and Battle.net friends).
- **Show minimap button**, **Send test messages**, and **Show strip** (a troubleshooting option that keeps the pixel strip visible).

Settings and your presets are saved per account.

## In-game commands

| Command | What it does |
|---|---|
| `/parley` | Open the settings window |
| `/parley status` | Status in chat |
| `/parley on` / `off` | Pause or resume sending chat to the app |
| `/parley preset <name>` | Apply a preset by name, e.g. `/parley preset group only` or one of your own. Without a name it applies Friends & group. |
| `/parley all` | Every chat type again |
| `/parley whisper` · `say` · `party` · `raid` · `guild` · `channel` · `instance` | Toggle one chat type |
| `/parley friendsonly` | Toggle: whispers only from your friends list (character or Battle.net friends) |
| `/parley test` | Send three sample messages (Spanish, Russian, Portuguese) |
| `/parley test all` | Samples in eight languages, including Japanese, Chinese and Korean |
| `/parley test ja` | One sample: `es` `ru` `pt` `de` `fr` `ja` `zh` `ko`, or `asia` for the three Asian languages |
| `/parley show` | Keep the pixel strip visible (troubleshooting) |
| `/parley cell 1-4` | Size of each square in pixels (default 2) |
| `/parley hold <seconds>` | How long each message stays on screen (default 0.1) |


## Skins

Tray icon → **Skin**, or choose one in Settings (it previews live):

- **Parley (modern)**: dark theme with a cyan accent.
- **Classic WoW**: the original tooltip/dialog look, with a navy panel, silver edge, gold title and red buttons.
- **Retail WoW (windows)**: metal frame with gold corners, gold-ringed portrait, red close box, and a gold glow on the selected message.
- **Dragonflight UI**: charcoal panels, bevelled fields, a portrait with a name plate, and gold icons.
- **WoW chat frame (minimal)**: plain black, like the chat window.

WoW's fonts aren't included because they're licensed. If you have Friz Quadrata (`FRIZQT__.TTF`), put it in a `fonts` folder next to `Parley.exe`; the WoW skins use it automatically. Text in other alphabets always falls back to a font that supports it.

## Settings

| Setting | |
|---|---|
| Translation engine | Offline (default), DeepL or Azure, see above |
| DeepL API key / Azure key and region | Optional; the row shows the key for the engine you picked. Stored encrypted with your Windows account (DPAPI) |
| Translate chat into | Your language (set from your Windows language on first start) |
| Interface language | Language of Parley's own menus and windows. *Automatic* follows Windows; English, Deutsch, Español, Français, Italiano, Português, Русский, 한국어, 简体中文 and 繁體中文 are available. The addon follows your WoW client's language |
| Skin | See above |
| WoW folder | Your `_classic_era_` or `_retail_` folder, detected automatically. **Install addon** copies the addon into it and into the other one if you have both |
| Text size / Overlay opacity | Appearance |
| Alert me when a message mentions | Words that get a message highlighted (and a sound), comma-separated. Your character is included automatically |
| Show the original text | Shows the untranslated message under each translation |

Settings live in `%APPDATA%\Parley\settings.json`.

## Troubleshooting

| You see | Fix |
|---|---|
| **Waiting for WoW** | WoW isn't running, or it's minimised. |
| **WoW found · type /parley test in game** | The app can't see the strip yet. Check that the addon is enabled on the character screen and that WoW uses **Windowed (Fullscreen)**. |
| **Colours distorted** | Reset WoW's Gamma, Brightness and Contrast, and turn off Windows HDR while playing. |
| **No DeepL key** | You chose the DeepL engine without a key. Add the key, or switch the engine to Offline. |
| "Downloading the … offline model" | First use of a language. It happens once; the message is translated when the download finishes. |
| "offline translation doesn't support …" | Mozilla has no model for that language yet. The DeepL engine may cover it. |
| "couldn't tell which language this is" | The message was too short or mixed. Nothing is wrong; longer messages work. |
| **DeepL rejected the API key** | Re-copy the key from your DeepL account page. |
| **DeepL character allowance used up** | Parley switches to offline for those messages. Top up in your DeepL account, or set the engine to Offline. |
| "Blizzard hides chat from addons…" (Retail) | During Mythic+ keys, PvP matches and boss fights Blizzard doesn't let any addon read chat. Parley picks up again when it's over. |
| Messages missing in busy cities | Parley sends up to about 10 messages a second. Turn off public channels with `/parley channel`. |
| Voice: "Couldn't open the microphone" | Allow desktop apps to use the microphone in Windows privacy settings, and check your default recording device. |
| Voice: other errors | Tray icon → Voice input → **Open voice log**. |

## Is this safe?

- **The addon** uses only Blizzard's standard addon API. It reads chat events and draws coloured squares, which is ordinary addon behaviour.
- **The app** captures a small area of your screen, like OBS or Discord's overlay. It never reads WoW's memory, injects code, changes game files or sends keystrokes. You always paste and send replies yourself.
- Blizzard's terms broadly prohibit unauthorised third-party software that "facilitates gameplay". Translating chat doesn't automate anything or give an advantage, but Blizzard has not reviewed or approved Parley. Use it at your own risk.
