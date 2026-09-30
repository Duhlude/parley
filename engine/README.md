# parley-mt (offline translation engine)

`parley-mt.exe` is the small helper that Parley starts for offline translation. It wraps
[bergamot-translator](https://github.com/browsermt/bergamot-translator) (the engine behind
Firefox Translations, MPL-2.0) and [CLD2](https://github.com/CLD2Owners/cld2) language
detection (Apache-2.0) behind a line-based protocol on stdin/stdout. See the comment at the
top of `parley-mt.cpp` for the protocol.

Parley downloads Mozilla's translation models (MPL-2.0) from Firefox's model server the first
time a language is needed and keeps them in `%APPDATA%\Parley\models\translate`.

## Building

`build-engine.sh` cross-compiles it on Linux with MinGW-w64 (posix threads). It fetches the
pinned sources, applies `bergamot-mingw.patch` (MinGW fixes: aligned allocation, header case,
no `-Werror`, no git revision step) and writes `bin/parley-mt.exe`.

```
sudo apt install cmake ninja-build g++-mingw-w64-x86-64-posix curl
engine/build-engine.sh
```
