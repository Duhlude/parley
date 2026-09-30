# Parley

**Live chat translation for Classic Era.** Anniversary, Mists of Pandaria Classic and Retail are supported in beta. Read what Brazilian, Russian, Spanish, German and other players are saying as they type, and answer them in their own language.

> **Please read before installing the desktop app.** The addon itself only uses Blizzard's normal addon API. The companion app is an outside program: it never opens the game, reads its memory, injects anything or presses keys, but its purpose is to carry chat out of WoW to be translated, and Blizzard's EULA restricts software that "collects information" from the game. Blizzard has not reviewed or approved Parley and doesn't pre-approve tools like it. We believe the risk is low, but we can't promise it's zero. **Use it at your own risk**, and if you want to be careful, try it on an account you wouldn't mind losing. Details: "Is this safe?" in the [User Guide](https://github.com/Duhlude/parley/blob/main/docs/USER_GUIDE.md#is-this-safe).

## What it does

- Translates whispers, party, raid, guild, say and channel chat into your language as it arrives. Translation runs offline on your PC (the same open models Firefox uses), so there's no account to create and no limit. DeepL or Azure are optional for the best quality.
- Skips English automatically, and keeps gamer terms like *tank*, *LFG*, *WTS* and dungeon names as they are.
- Understands Brazilian chat shorthand (*vc, blz, tbm, kkkk*).
- Reply by typing, or by voice: speech is recognised on your own PC. Parley translates your reply, adds the right chat command and copies it for you to paste, and shows it translated back so you can check what it says.
- Quick replies, alerts when someone mentions you, a conversation view for whispers, and a list of names (your guild, friends) that are never translated.
- An in-game settings window (`/parley` or the minimap button) with one-click presets such as *Friends & group*, *Group only* and *Social*, plus your own saved presets.
- Skins for the overlay: Classic, Retail, Dragonflight or chat-frame style.

## Important: this addon needs the free Parley desktop app

WoW addons can't access the internet, so Parley uses a small Windows companion app for the actual translation. The addon briefly shows each message as a tiny strip of coloured squares in the corner of your screen, and the app reads those squares and translates the text. The app never opens the game process, never touches the game's memory or files, and never presses keys for you. It finds the WoW window and reads a few pixels, like a screen recorder.

**Download the Parley app (installer includes this addon):** see the GitHub Releases page linked below.

No account or API key needed. Each language downloads once (about 50 MB) the first time it shows up in chat.

## Commands

`/parley` (settings window) · `/parley status` · `/parley test` · `/parley preset <name>` · `/parley all` · `/parley on|off`

## Requirements

Windows 10/11 · WoW Classic Era, Anniversary, Mists of Pandaria Classic or Retail in **Windowed (Fullscreen)** mode

**Anniversary, Mists of Pandaria Classic and Retail are in beta**: please report problems on GitHub. On Retail, Blizzard hides chat from all addons during Mythic+ keys, PvP matches and boss fights, so Parley pauses there and resumes afterwards.

---

Source code, installer, documentation and privacy details: **GitHub: https://github.com/Duhlude/parley** (download the app from Releases)

Parley is a fan project, not affiliated with, reviewed or endorsed by Blizzard Entertainment. Using the companion app is at your own risk.
