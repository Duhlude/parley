#!/usr/bin/env bash
# Cross-compiles parley-mt.exe (offline translation engine) on Linux with MinGW-w64.
# Needs: cmake, ninja, g++-mingw-w64-x86-64-posix, curl.
set -euo pipefail
cd "$(dirname "$0")"
W=${WORK:-$PWD/.work}
mkdir -p "$W" && cd "$W"
get() { # repo commit dest
  mkdir -p "$3"; curl -sL "https://codeload.github.com/$1/tar.gz/$2" | tar xz -C "$3" --strip-components=1; }
if [ ! -d bt ]; then
  get browsermt/bergamot-translator 9271618ebbdc5d21ac4dc4df9e72beb7ce644774 bt
  get browsermt/marian-dev 2781d735d4a10dca876d61be587afdab2726293c bt/3rd_party/marian-dev
  get browsermt/ssplit-cpp a311f9865ade34db1e8e080e6cc146f55dafb067 bt/3rd_party/ssplit-cpp
  M=bt/3rd_party/marian-dev/src/3rd_party
  get browsermt/sentencepiece ae41b7740d7006596bb9257e83340b2620db9d00 $M/sentencepiece
  get kpu/intgemm f7401513da71758dacce52fed1c7855549abee59 $M/intgemm
  get browsermt/onnxjs 924924b08e9596b41aeebada4a172f026be95f5a $M/onnxjs
  get google/ruy 2d950b3bfa7ebfbe7a97ecb44b1cc4da5ac1d6f0 $M/ruy
  get browsermt/simd_utils d0793d86aea9036a5bc77b9ca7791dff024168ca $M/simd_utils
  get pytorch/cpuinfo 5916273f79a21551890fd3d56fc5375a78d1598d $M/ruy/third_party/cpuinfo
  mkdir -p bt/3rd_party/ssplit-cpp/src/3rd-party
  curl -sL https://github.com/PhilipHazel/pcre2/releases/download/pcre2-10.39/pcre2-10.39.tar.gz | tar xz -C bt/3rd_party/ssplit-cpp/src/3rd-party
  # CLD2 language detection (Apache-2.0), from the pycld2 source release
  curl -sL https://files.pythonhosted.org/packages/89/81/d4d348e224ac0994234992c6d5fb3a78cc9b660337d5bd17c67fac1baf6a/pycld2-0.42.tar.gz | tar xz
  mkdir -p bt/3rd_party/cld2 && cp -r pycld2-0.42/cld2/internal pycld2-0.42/cld2/public pycld2-0.42/LICENSE bt/3rd_party/cld2/
  (cd bt && patch -p1 < ../../bergamot-mingw.patch)
  cp ../parley-mt.cpp bt/app/
  mkdir -p bt/3rd_party/marian-dev/.git/logs && touch bt/3rd_party/marian-dev/.git/logs/HEAD
  echo '#define GIT_REVISION "2781d73"' > bt/3rd_party/marian-dev/src/common/git_revision.h
fi
# MinGW headers are lower-case; marian includes <Windows.h> and "DbgHelp.h".
mkdir -p shim && echo '#include <windows.h>' > shim/Windows.h && echo '#include <dbghelp.h>' > shim/DbgHelp.h
export CPLUS_INCLUDE_PATH="$W/shim"
mkdir -p build && cd build
cmake -G Ninja ../bt -DCMAKE_TOOLCHAIN_FILE="$(cd ../.. && pwd)/mingw.cmake" -DCMAKE_BUILD_TYPE=Release \
  -DBUILD_ARCH=x86-64-v2 -DUSE_MKL=OFF -DUSE_RUY=ON -DUSE_RUY_SGEMM=ON -DSSPLIT_USE_INTERNAL_PCRE2=ON \
  -DCOMPILE_TESTS=OFF -DUSE_NCCL=OFF -DUSE_DOXYGEN=OFF -DCOMPILE_CUDA=OFF -DUSE_STATIC_LIBS=ON \
  -DCMAKE_CXX_FLAGS="-Wno-format -DPCRE2_STATIC" \
  -DCMAKE_EXE_LINKER_FLAGS="-static -static-libgcc -static-libstdc++" -DCMAKE_CXX_STANDARD_LIBRARIES="-lshlwapi"
ninja parley-mt
x86_64-w64-mingw32-strip -o ../../../bin/parley-mt.exe app/parley-mt.exe
echo "Built bin/parley-mt.exe"
