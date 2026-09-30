# Privacy

Parley has no servers, accounts, analytics or telemetry. Everything runs on your PC. The only things that leave your PC are listed below.

## What is sent, and where

| Data | Sent to | When |
|---|---|---|
| Nothing (download only) | **Mozilla** (firefox.settings.services.mozilla.com, firefox-settings-attachments.cdn.mozilla.net) | Once per language, when an offline translation model is first needed, and a model list refresh about once a week |
| Nothing (download only) | **Hugging Face** (huggingface.co) | Once, when the speech model is first downloaded |
| Text of chat messages that look non-English | **DeepL** (api.deepl.com / api-free.deepl.com), using **your** API key | **Only if you choose the DeepL engine**, when a message needs translating |
| Text of replies you ask Parley to translate | **DeepL** | **Only with the DeepL engine**, when you press Enter or finish a voice reply |
| Text of non-English chat messages and your replies | **Microsoft Azure Translator** (api.cognitive.microsofttranslator.com), using **your** key | **Only if you choose the Azure engine** |
| Nothing (a normal web request, no personal data) | **GitHub** (api.github.com) | About once a day, to see whether a newer Parley exists. Turn it off under tray icon → Check for updates |

With the default **offline** engine, chat text never leaves your PC: messages are translated on your computer by `parley-mt.exe`. With DeepL, the sender's name and channel are **not** sent, only the message text, and English messages are filtered out on your PC first. If DeepL fails (no internet, allowance used up), Parley translates that message offline instead.

How DeepL handles text depends on your DeepL plan and their terms. See [DeepL's privacy policy](https://www.deepl.com/privacy) for details. The same applies to Azure: Microsoft says Translator doesn't store the text you send; see [Microsoft's data and privacy notes for Translator](https://learn.microsoft.com/legal/cognitive-services/translator/data-privacy-security).

## What stays on your PC

- **Voice**: your microphone is recorded only while the mic button is active. The audio is transcribed locally by whisper.cpp; the temporary audio file is deleted immediately, and audio is never uploaded.
- **Settings** (`%APPDATA%\Parley\settings.json`): your preferences, with your DeepL and Azure keys encrypted using Windows' built-in protection (DPAPI), so only your Windows account can read it.
- **Voice log** (`%APPDATA%\Parley\voice.log`): timing and error information, plus the transcribed text, for troubleshooting. Delete it any time.
- **Offline translation models** (`%APPDATA%\Parley\models\translate`): about 25–45 MB per direction. Delete the folder any time; Parley downloads what it needs again.
- **Chat history** (`%APPDATA%\Parley\history\YYYY-MM-DD.txt`): translated messages and your replies, one file per day, so you can look things up later. Turn it off under tray icon → Chat history; delete the files any time.
- **Muted players**: the names you muted, in `settings.json`.
- **Character name**: the addon tells the app which character you're playing so it can alert you when someone mentions you. It's kept in `settings.json` and never sent anywhere.
- **Translation cache**: kept in memory only, and cleared when Parley exits.
- **The WoW addon** stores its settings in WoW's SavedVariables (`ParleyDB`) and sends nothing anywhere. It only draws on your screen.

## Screen capture

Parley reads only a small rectangle, 256 × 24 pixels, at the top-left corner of the WoW window. It doesn't take or store screenshots.
