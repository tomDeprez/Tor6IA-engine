#!/bin/sh
# Tor6IA engine installer for Linux and macOS.
#   curl -fsSL https://raw.githubusercontent.com/tomDeprez/Tor6IA-engine/main/install.sh | sh
set -eu

REPO="tomDeprez/Tor6IA-engine"

os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$(uname -m)" in
  x86_64 | amd64) arch=amd64 ;;
  arm64 | aarch64) arch=arm64 ;;
  *) echo "Unsupported processor: $(uname -m)" >&2; exit 1 ;;
esac
case "$os" in
  linux | darwin) ;;
  *) echo "Unsupported system: $os. On Windows, download tor6ia-engine-windows-amd64.exe from https://github.com/$REPO/releases/latest" >&2; exit 1 ;;
esac

bin="$HOME/.local/bin"
mkdir -p "$bin"
echo "Downloading the Tor6IA engine..."
curl -fL "https://github.com/$REPO/releases/latest/download/tor6ia-engine-$os-$arch" -o "$bin/tor6ia-engine.new"
chmod +x "$bin/tor6ia-engine.new"
mv -f "$bin/tor6ia-engine.new" "$bin/tor6ia-engine"
echo "Installed: $bin/tor6ia-engine"
echo "Try it:    $bin/tor6ia-engine demo --model gemma4:12b   (a .gguf file, or a model Ollama has)"
