# Building Parley

## Requirements

- **Go 1.24+**: https://go.dev/dl/ . The app and installer use only Go's standard library and need no cgo.
- **whisper-cli.exe**: prebuilt copy included in the repository; to build it yourself, see below.
- **parley-mt.exe** (offline translator): prebuilt copy included; to rebuild it, run `engine/build-engine.sh` on Linux with MinGW-w64 (see `engine/README.md`).
- Optional: **Lua 5.1** to run the addon-encoder test.

## One command (Windows)

```
build.bat
```

This produces:

| File | What |
|---|---|
| `Parley.exe` | The app (embeds the addon, which it installs into WoW) |
| `Parley-Setup.exe` | Per-user installer (embeds `Parley.exe`, `whisper-cli.exe`, `parley-mt.exe`, the addon and the docs) |
| `Parley-addon-<version>.zip` | The addon alone, ready for CurseForge |

## By hand

```
go build -trimpath -ldflags "-s -w -H windowsgui" -o Parley.exe .
```

Cross-compiling from Linux or macOS works the same with `GOOS=windows GOARCH=amd64`.

The installer embeds everything in `installer/payload/` (which is git-ignored), so fill it first:

```
installer/payload/Parley.exe
installer/payload/whisper-cli.exe
installer/payload/parley-mt.exe
installer/payload/whisper-cli-LICENSE.txt
installer/payload/README.md, INSTALL.txt, LICENSE, THIRD_PARTY_NOTICES.md, PRIVACY.md
installer/payload/addon/Parley/Parley.toc, Parley.lua
```

Then run `cd installer && go build -trimpath -ldflags "-s -w -H windowsgui" -o ../Parley-Setup.exe .`

Update `const version` in `installer/main_windows.go` for each release.

## Logo, icons and Windows resources

`python3 tools/make_logo.py` (needs `cairosvg` and Pillow, plus the Cinzel and IPAGothic fonts) regenerates everything in `assets/logo/` and the addon's `Media/*.tga`. The icons and version info are compiled into the executables from `parley.rc` and `installer/installer.rc`:

```
x86_64-w64-mingw32-windres -O coff -i parley.rc -o rsrc_windows_amd64.syso
cd installer && x86_64-w64-mingw32-windres -O coff -i installer.rc -o rsrc_windows_amd64.syso
```

The `.syso` files are committed, so a normal `go build` picks them up without windres. Update the version numbers in both `.rc` files for each release.

## Translations of the interface

App strings are keyed by their English text. Translations live in `tools/i18n/<lang>.json` (app) and `tools/i18n-addon/<wowLocale>.json` (addon); the installer's few strings are in `installer/i18n_windows.go`. After editing:

```
python3 tools/i18n_keys.py      # checks every catalog has every key
python3 tools/gen_i18n.py       # writes i18n_data.go
python3 tools/gen_addon_i18n.py # writes addon/Parley/Locale.lua
```

## Tests

```
cd test && lua5.1 gen.lua out && cd ..
go test .
```

`gen.lua` runs the real addon code against a stubbed WoW API and writes pixel frames (including gamma distortion). `go test` decodes them, checks that corrupted frames are rejected, and tests language detection, shorthand expansion, the DeepL client (against a local mock server) and the offline model catalog, download, zstd/SHA-256 checks and pivoting (against a mock model server and a fake engine).

## Building whisper-cli.exe

whisper.cpp, static, CPU-only, AVX2, cross-compiled with MinGW-w64:

```
cmake -S whisper.cpp -B build-win -DCMAKE_TOOLCHAIN_FILE=mingw.cmake -DCMAKE_BUILD_TYPE=Release \
  -DBUILD_SHARED_LIBS=OFF -DGGML_NATIVE=OFF -DGGML_AVX=ON -DGGML_AVX2=ON -DGGML_FMA=ON -DGGML_F16C=ON \
  -DGGML_OPENMP=OFF -DWHISPER_BUILD_TESTS=OFF -DWHISPER_BUILD_SERVER=OFF -DWHISPER_SDL2=OFF \
  -DCMAKE_EXE_LINKER_FLAGS="-static -static-libgcc -static-libstdc++"
cmake --build build-win --target whisper-cli -j
```

(With older MinGW headers, the `THREAD_POWER_THROTTLING_STATE` block in `ggml/src/ggml-cpu/ggml-cpu.c` must be disabled.) On Windows with Visual Studio, the standard whisper.cpp CMake build works too.

## Versioning

- Addon: `## Version:` in `addon/Parley/Parley.toc`. The app compares this with the installed copy and updates WoW's AddOns folder automatically when it changes.
- App/installer: `const version` in `installer/main_windows.go`.
- Record every release in `CHANGELOG.md`.
