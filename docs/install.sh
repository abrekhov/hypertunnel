#!/usr/bin/env sh
set -eu

REPO="abrekhov/hypertunnel"
BIN_NAME="ht"
INSTALL_NAME="ht"

info() {
  printf '%s\n' "$*" >&2
}

fail() {
  info "Error: $*"
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1
}

download() {
  url="$1"
  out="$2"

  if need_cmd curl; then
    curl -fsSL "$url" -o "$out"
  elif need_cmd wget; then
    wget -qO "$out" "$url"
  else
    fail "need curl or wget"
  fi
}

sha256_file() {
  f="$1"

  if need_cmd sha256sum; then
    sha256sum "$f" | awk '{print $1}'
  elif need_cmd shasum; then
    shasum -a 256 "$f" | awk '{print $1}'
  else
    return 1
  fi
}

os=""
case "$(uname -s)" in
  Linux) os="linux" ;;
  Darwin) os="darwin" ;;
  *) fail "unsupported OS: $(uname -s)" ;;
esac

arch=""
case "$(uname -m)" in
  x86_64|amd64) arch="amd64" ;;
  aarch64|arm64) arch="arm64" ;;
  *) fail "unsupported architecture: $(uname -m)" ;;
esac

asset="${BIN_NAME}_${os}_${arch}"
url="https://github.com/${REPO}/releases/latest/download/${asset}"
checksums_url="https://github.com/${REPO}/releases/latest/download/checksums.txt"

tmp_dir=""
if tmp_dir=$(mktemp -d 2>/dev/null); then
  :
else
  tmp_dir=$(mktemp -d -t hypertunnel)
fi

trap 'rm -rf "$tmp_dir"' EXIT

tmp_bin="$tmp_dir/$INSTALL_NAME"
tmp_checksums="$tmp_dir/checksums.txt"

info "Downloading ${asset}..."
download "$url" "$tmp_bin"
chmod +x "$tmp_bin"

if download "$checksums_url" "$tmp_checksums" 2>/dev/null; then
  expected_sha=$(grep " ${asset}$" "$tmp_checksums" 2>/dev/null | awk '{print $1}' | head -n 1 || true)
  if [ -n "$expected_sha" ]; then
    actual_sha=$(sha256_file "$tmp_bin" 2>/dev/null || true)
    if [ -n "$actual_sha" ]; then
      if [ "$expected_sha" != "$actual_sha" ]; then
        fail "checksum mismatch for ${asset}"
      fi
      info "Checksum verified."
    else
      info "Skipping checksum verification (no sha256 tool found)."
    fi
  else
    info "Skipping checksum verification (no checksum entry for ${asset})."
  fi
else
  info "Skipping checksum verification (checksums.txt not available)."
fi

install_to="$HOME/.local/bin"
dest_dir="/usr/local/bin"
dest_path="$dest_dir/$INSTALL_NAME"

if [ "$(id -u)" -eq 0 ]; then
  mkdir -p "$dest_dir"
  mv "$tmp_bin" "$dest_path"
  info "Installed to $dest_path"
  info "Run: $INSTALL_NAME --help"
  exit 0
fi

if [ -d "$dest_dir" ] && [ -w "$dest_dir" ]; then
  mv "$tmp_bin" "$dest_path"
  info "Installed to $dest_path"
  info "Run: $INSTALL_NAME --help"
  exit 0
fi

if need_cmd sudo; then
  sudo mkdir -p "$dest_dir"
  sudo mv "$tmp_bin" "$dest_path"
  info "Installed to $dest_path"
  info "Run: $INSTALL_NAME --help"
  exit 0
fi

mkdir -p "$install_to"
dest_path="$install_to/$INSTALL_NAME"
mv "$tmp_bin" "$dest_path"

info "Installed to $dest_path"
info "Make sure $install_to is in your PATH."
info "Run: $INSTALL_NAME --help"
