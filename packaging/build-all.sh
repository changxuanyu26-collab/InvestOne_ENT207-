#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
mkdir -p dist

while read -r goos arch label; do
  [[ -z "$goos" ]] && continue
  dest="dist/stage/$label/InvestOne"
  mkdir -p "$dest"
  cp docs/MULTIPLATFORM.md "$dest/README.md"

  case "$goos" in
    darwin)
      mkdir -p "$dest/InvestOne.app/Contents/MacOS"
      GOOS=darwin GOARCH="$arch" CGO_ENABLED=0 go -C src build -trimpath -ldflags="-s -w" -o "$ROOT/$dest/InvestOne.app/Contents/MacOS/InvestOne" .
      cp packaging/macos/Info.plist "$dest/InvestOne.app/Contents/Info.plist"
      printf "APPL????" > "$dest/InvestOne.app/Contents/PkgInfo"
      if [[ "$arch" == arm64 ]]; then
        cp "packaging/macos/Open InvestOne.command" "$dest/Open InvestOne.command"
      else
        cp "packaging/macos/Open InvestOne Intel.command" "$dest/Open InvestOne.command"
      fi
      cp "packaging/macos/Stop InvestOne.command" "$dest/Stop InvestOne.command"
      chmod +x "$dest/InvestOne.app/Contents/MacOS/InvestOne" "$dest/"*.command
      ;;
    windows)
      GOOS=windows GOARCH="$arch" CGO_ENABLED=0 go -C src build -trimpath -ldflags="-s -w" -o "$ROOT/$dest/InvestOne.exe" .
      cp packaging/windows/*.cmd "$dest/"
      ;;
    linux)
      GOOS=linux GOARCH="$arch" CGO_ENABLED=0 go -C src build -trimpath -ldflags="-s -w" -o "$ROOT/$dest/InvestOne" .
      cp packaging/linux/*.sh "$dest/"
      chmod +x "$dest/InvestOne" "$dest/"*.sh
      ;;
    *) echo "Unknown target: $goos" >&2; exit 1 ;;
  esac

  archive="$ROOT/dist/InvestOne-v2.1.0-rc1-$label.zip"
  (cd "dist/stage/$label" && zip -q -r -9 "$archive" InvestOne)
  unzip -tqq "$archive"
  echo "Built $goos/$arch → $archive"
done <<'TARGETS'
darwin arm64 Mac-AppleSilicon
darwin amd64 Mac-Intel
windows amd64 Windows-x64
windows arm64 Windows-ARM64
linux amd64 Linux-x64
linux arm64 Linux-ARM64
TARGETS

(cd dist && sha256sum InvestOne-v2.1.0-rc1-*.zip > SHA256SUMS.txt)
ls -lh dist/InvestOne-v2.1.0-rc1-*.zip dist/SHA256SUMS.txt
