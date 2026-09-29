# Publishing checklist (GitHub + CurseForge)

*This is guidance, not legal advice.*

## Is it allowed?

**Blizzard's UI Add-On Development Policy.** The addon must:
- [x] Be free: no paid versions, and no charging for downloads or services.
- [x] Have public, unobfuscated code. Parley is open source (MIT).
- [x] Not hurt realm performance. Parley sends no chat and no addon messages, and only draws locally.
- [x] Contain no advertising.
- [x] Contain no donation requests **in game**. A donation link on GitHub or CurseForge is fine; keep it out of the addon's chat messages and UI.
- [x] Have "T"-rated content.
- [x] Follow the WoW Terms of Use.

**The companion app.** The WoW Terms of Use prohibit unauthorised third-party software that "facilitates gameplay". Parley doesn't read or modify the game, automate anything or give a gameplay advantage. It reads screen pixels like streaming tools do, and every message is still pasted and sent by a person. Blizzard hasn't ruled on tools like this, so say so plainly on the project page ("use at your own risk; not endorsed by Blizzard"), which the README and User Guide already do. A similar project, BabelChat, publishes its addon on CurseForge with a separate companion app.

**Intellectual property.**
- [x] No Blizzard art, icons or fonts are included. Skins are drawn in code.
- [x] Friz Quadrata is **not** distributed. `fonts/` is git-ignored; don't upload it anywhere.
- [x] Trademark disclaimer is in the README and THIRD_PARTY_NOTICES.
- [x] whisper.cpp MIT license included (`whisper-cli-LICENSE.txt`).
- [x] The name "Parley" doesn't use "WoW" or "Warcraft". Keep the game name out of the project title; CurseForge asks for this too.

**Privacy.** By default translation is offline and chat text never leaves the PC; PRIVACY.md lists the model downloads from Mozilla and what goes to DeepL if a user opts into it. Parley itself collects nothing.

**Model licensing.** The Firefox Translations models are MPL-2.0 and are downloaded by users from Mozilla's servers, not redistributed by Parley. `parley-mt.exe` contains MPL-2.0 code (bergamot-translator); its source is public and Parley's changes are in `engine/`.

## GitHub

1. Create a repository, e.g. `parley`, and push the source. `.gitignore` already keeps build outputs, the installer payload, fonts and test output out of git.
2. Add a description, e.g. "Live chat translation for WoW Classic Era and Retail: addon + Windows companion app", and topics such as `world-of-warcraft`, `wow-addon`, `translation`, `deepl`, `whisper`.
3. Add screenshots (overlay in each skin; a before/after of chat) under `docs/images/` and link them from the README.
4. For each release, run `build.bat`, create a tag like `v1.3.0`, and attach:
   - `Parley-Setup.exe`
   - `Parley-addon-<version>.zip`
   - optionally `Parley.exe` + `whisper-cli.exe` as a portable zip
   Paste the CHANGELOG entry into the release notes.
5. Optional: code-sign the exe files to avoid the SmartScreen warning. This needs a code-signing certificate.

## CurseForge

1. Create a project for World of Warcraft: name **Parley**, category *Chat & Communication*, license **MIT**.
2. Use `docs/CURSEFORGE_DESCRIPTION.md` as the description.
3. Upload **only the addon zip** (`Parley-addon-<version>.zip`, containing a top-level `Parley/` folder), with game versions **Classic Era (1.15.x)** and **Retail (12.x)**. Don't upload `.exe` files to CurseForge.
4. The companion app is linked to the GitHub Releases page from the description. CurseForge asks for external links to go below the main description.
5. Add screenshots and a logo. The app's speech-bubble icon works as a starting point.
6. After the project is approved, add its ID to the TOC as `## X-Curse-Project-ID: <id>` for addon managers.

## Before every release

- [ ] `go test .` passes
- [ ] Addon `## Version` bumped if the addon changed
- [ ] `const version` in `installer/main_windows.go` bumped
- [ ] `## Interface:` lists the current Classic Era and Retail patches (e.g. `11509, 120100`) (see warcraft.wiki.gg/wiki/TOC_format)
- [ ] CHANGELOG updated
- [ ] Fresh install and update tested on a clean Windows user account
