# Third-party notices

## Included with Parley

### whisper.cpp (`whisper-cli.exe`)
Local speech-to-text engine by Georgi Gerganov and contributors, built from source for Windows x64.
- Source: https://github.com/ggml-org/whisper.cpp
- License: MIT. The full text is in `whisper-cli-LICENSE.txt`.

### Offline translation engine (`parley-mt.exe`)
Parley's on-device translator, built from source for Windows x64 (see `engine/`):
- **bergamot-translator**, the engine behind Firefox Translations, by the Bergamot project and Mozilla: https://github.com/browsermt/bergamot-translator (MPL-2.0). Parley's small MinGW build changes are in `engine/bergamot-mingw.patch`.
- **Marian NMT** (MIT), **SentencePiece** (Apache-2.0), **intgemm** (MIT), **ruy** (Apache-2.0), **cpuinfo** (BSD), **PCRE2** (BSD), **ssplit-cpp** (Apache-2.0) and smaller libraries bundled with Marian.
- **CLD2** language detection by Google (Apache-2.0): https://github.com/CLD2Owners/cld2
- Full license texts: `third_party/parley-mt-LICENSES.txt`.

### zstd decoder (`internal/zstd`)
Copied from the Go standard library (BSD-3-Clause, © The Go Authors). License: `third_party/go-zstd-LICENSE.txt`.

### Gaming-term dictionaries (`slangdata.go`)
Chat shortcuts and WoW term explanations in many languages, taken from the data files of two MIT-licensed addons and converted by `tools/gen_slang.py`:
- **BabelChat** by Andrey Yumashev: https://github.com/Yumash/BabelChat (MIT, © 2025-2026 Andrey Yumashev). License: `third_party/BabelChat-LICENSE.txt`.
- **WoW Translator** by Pirson: https://github.com/Pirson-s-Addons/WoW-Translator (MIT, © 2026 Pirson). License: `third_party/WoW-Translator-LICENSE.txt`.

## Downloaded on first use (not included)

### Whisper speech models (`ggml-small.en.bin`, `ggml-base.en.bin`, …)
OpenAI's Whisper model weights, converted to the ggml format by the whisper.cpp project and downloaded from Hugging Face.
- Original: https://github.com/openai/whisper (MIT License)
- Converted models: https://huggingface.co/ggerganov/whisper.cpp

### Firefox Translations models
Translation models by Mozilla, downloaded per language pair from Firefox's public model server (firefox.settings.services.mozilla.com / firefox-settings-attachments.cdn.mozilla.net) and stored in `%APPDATA%\Parley\models\translate`.
- Project: https://github.com/mozilla/translations
- License: MPL-2.0

## Services

### DeepL API (optional)
When the DeepL engine is selected, translations are provided by DeepL SE through the user's own API key and are subject to DeepL's terms: https://www.deepl.com/pro-license

## Not included

### Friz Quadrata, Arial Narrow and other game fonts
These are commercially licensed fonts and are **not distributed** with Parley. Users who have them may place them in a `fonts` folder next to `Parley.exe`.

## Logo artwork

Parley's logo and banners (`assets/logo`, `tools/make_logo.py`) are original artwork. The banner's wordmark is set in **Cinzel** (SIL Open Font License 1.1, © The Cinzel Project Authors) and the 文 glyph in the logo is drawn from **IPAGothic** (IPA Font License); both are only used to render the images, and no font files are distributed.

## Trademarks

World of Warcraft, Warcraft and Blizzard Entertainment are trademarks or registered trademarks of Blizzard Entertainment, Inc. in the U.S. and/or other countries. Parley is an independent fan project and is not affiliated with or endorsed by Blizzard Entertainment. DeepL is a trademark of DeepL SE. Firefox is a trademark of the Mozilla Foundation; Parley is not affiliated with or endorsed by Mozilla.

Parley's skins are drawn in code and inspired by the look of the game's interface; they contain no Blizzard artwork.
