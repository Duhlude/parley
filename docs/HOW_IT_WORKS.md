# How Parley works

## The problem

WoW addons run in a sandbox with no network access, so an addon on its own can't call a translation service. Other ways of getting chat out of the game have serious drawbacks:

| Approach | Problem |
|---|---|
| Chat log file (`/chatlog`) | WoW buffers it; lines can arrive minutes late |
| SavedVariables | Only written on logout or `/reload` |
| Reading game memory | Touches the game process, which carries a real anti-cheat risk |
| Injecting a DLL | Clearly against the Terms of Use |
| Text recognition on the chat window | Slow and error-prone with names, links and non-Latin scripts |

Parley instead uses a **pixel bridge**. The addon encodes each message as coloured squares, and the app reads those exact colours back from the screen.

## Components

```
WoW client ──(chat events)──▶ Parley addon ──(paints strip)──▶ screen
                                                                  │
                            Parley app ◀──(captures 256×24 px)────┘
                               │
                               ├─▶ language pre-check (offline)
                               ├─▶ parley-mt.exe (offline translation, default)
                               │     or DeepL API (optional)
                               ├─▶ overlay window
                               └─▶ clipboard (your reply)
```

- **`addon/Parley/Parley.lua`**: listens to `CHAT_MSG_*` events, filters chat types, strips colour codes and links, and queues messages. It paints one message per frame on a 64×6 grid of textures in the top-left corner of the screen, sized so that one unit equals one physical pixel. The strip only appears while messages are being sent.
- **Parley app** (Go, Windows API only; source in `app/`):
  - `capture` / `decode.go`: finds the WoW window and copies the strip area about 28 times a second with GDI `BitBlt`, then decodes it.
  - `lang.go`: offline check that skips English (and fixes Brazilian shorthand).
  - `offline.go`, `engine.go`: the offline translator. Picks and downloads Firefox Translations models, and talks to `parley-mt.exe`.
  - `deepl.go`, `azure.go`: the optional DeepL and Azure clients, with caching, gaming-term protection and casual tone.
  - `overlay_windows.go`, `chrome_windows.go`, `skins_windows.go`: the UI and skins.
  - `speech_windows.go`: microphone recording (winmm) and local transcription (`whisper-cli.exe`).
  - `installer/`: `Parley-Setup.exe`.

## Strip protocol

The grid has **64 × 6 cells**; each cell is `CELL × CELL` physical pixels (default 2, configurable 1–4). Each cell carries **three 4-bit values**, one per colour channel, drawn as `level × 17` (0, 17, 34 … 255). Using only 16 levels per channel keeps decoding reliable even when colours shift slightly.

| Cells | Contents |
|---|---|
| 0–3 | Marker: (15,0,15) (0,15,0) (15,15,0) (0,0,15) |
| 4–19 | Calibration: cell 4+i = (i,i,i), so the decoder learns how each level actually looks on screen |
| 20 | Sequence number (12 bits), used to skip frames already seen |
| 21 | Payload length in bytes (12 bits) |
| 22–23 | Fletcher checksum (mod 4095) over the payload |
| 24+ | Payload bytes, two 4-bit values per byte, three per cell (R, G, B) |

Payload (UTF-8): `type ␟ channelNumber ␟ channelName ␟ sender ␟ text`, where ␟ is `\x1F` and type is one of `W B S Y E P R G O C I` (whisper, BNet whisper, say, yell, emote, party, raid, guild, officer, channel, instance).

Capacity is 540 bytes per frame, which is more than WoW's 255-byte chat limit plus headers. Frames with a bad marker, bad calibration or a checksum mismatch are discarded, never shown. Each message stays on screen for `hold` seconds (0.1 by default), so the app, which samples every 35 ms, reads it several times.

`go test ./...` runs the real addon code (in Lua 5.1 with a stubbed WoW API) to produce frames, and decodes them, including with gamma distortion and corrupted pixels.

## Other WoW versions

The same addon loads on Classic Era, the Anniversary realms, Mists of Pandaria Classic and Retail (`## Interface: 11509, 20506, 50504, 120100`), and the app installs it into every one of those game folders it finds (`_classic_era_`, `_anniversary_`, `_classic_`, `_retail_`). All four run the modern client, so the addon code and the strip protocol are identical.

### Retail

Since Midnight, Retail passes chat to addons as *secret values* while a Mythic+ key, a PvP match or a boss encounter is active: an addon may show them but can't read them, so they can't be painted into the strip. The addon checks every chat argument with `issecretvalue` first. When chat is hidden it sends a single `X` frame (at most once every 30 seconds) instead of the message, and the app shows a note that Parley will pick up again afterwards. Everything else, including the strip protocol, is identical.

## Why screen capture is considered safe

The app never opens a handle to the WoW process, never reads its memory, and never sends it input. It reads a few hundred pixels of the desktop, which is the same thing streaming and screenshot tools do. The addon uses only documented API calls (`CreateFrame`, `SetColorTexture`, chat events).

## Offline translation

`parley-mt.exe` (source in `engine/`) is a small C++ program built from **bergamot-translator**, the engine inside Firefox Translations, plus **CLD2** for language detection. Parley starts it once, at below-normal priority, and talks to it over stdin/stdout.

1. **Detect**: plain-English chat is still skipped by `lang.go`. Everything else goes to CLD2; for very short lines that CLD2 can't place ("vc vai?"), a small word list decides.
2. **Pick models**: Parley reads Mozilla's public model catalog (Remote Settings collection `translations-models-v2`, cached for a week), keeps desktop release records, and takes the newest version of each pair (`pt→en`, `en→pt`, …). Each pair is three files: the model (int8 quantised), a SentencePiece vocabulary and a lexical shortlist.
3. **Download once**: files are zstd-compressed; Parley decompresses them with a zstd decoder copied from Go's standard library, checks each file's SHA-256 against the catalog, and only then marks the pair ready.
4. **Translate**: `[Item Links]` and WoW jargon are wrapped in `<code>` elements, which bergamot copies through untranslated. Languages without a direct model go through English (`ru→en→pt`) inside the engine in one request.

Up to three models stay loaded (enough for a pivot like `ru→en→pt`). If no foreign message arrives for 10 minutes, Parley stops the engine to free its memory; it starts again in well under a second.

## Performance

Measured on a 2.1 GHz server CPU running the Windows build under Wine, so a gaming PC should be as fast or faster.

| What | Cost |
|---|---|
| Reading the strip | One 256 × 24 pixel copy every 35 ms. Decoding a frame takes about 6 µs, and the app sleeps in between. |
| Offline translation, typical chat line | 12–35 ms |
| Offline translation, a full 255-character message | about 0.4 s |
| Loading a language the first time in a session | 0.3–1.2 s |
| Engine memory | about 30 MB idle, plus about 200 MB for each loaded direction (so ~430 MB while chatting in one language both ways). Freed after 10 idle minutes. |
| Disk | about 37 MB per direction (25 MB download) in `%APPDATA%\Parley\models\translate` |
| DeepL | a web request per message, 200–500 ms; almost no local work |

The engine runs at below-normal priority, so WoW keeps priority for the CPU.

## Translation details (DeepL)

- **Pre-filter** (both engines): messages in plain Latin letters need recognisable foreign words to be translated. Results that come back identical to the original, or in an implausible language, are hidden.
- **Protection**: `[Item Links]` and WoW jargon are wrapped in `<x>` tags, which DeepL leaves untranslated (`tag_handling=xml`, `ignore_tags=x`).
- **Quality**: `formality=prefer_less`, `model_type=prefer_quality_optimized` and a free `context` hint ("casual WoW chat"). If a plan rejects these options, Parley retries without them.
- **Cache**: identical messages aren't translated twice (2,000-entry cache).
