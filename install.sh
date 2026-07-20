#!/bin/sh

# Installs jup on Linux or macOS.
#
# Usage:
#   curl -fsSL https://github.com/codeboyzhou/javaup/releases/latest/download/install.sh | sh
#
# Optional environment variables:
#   JUP_VERSION         Release version to install, such as v0.1.0 or 0.1.0
#   JUP_INSTALL_DIR     Installation directory; defaults to $HOME/.javaup
#   JUP_NO_MODIFY_PATH  Skip the shell profile update when set to a non-empty value

set -eu

repository="codeboyzhou/javaup"
release_base="https://github.com/$repository/releases"
install_dir=${JUP_INSTALL_DIR:-"$HOME/.javaup"}
no_modify_path=${JUP_NO_MODIFY_PATH:-}
temporary=

write_step() {
  printf '==> %s\n' "$1"
}

die() {
  printf 'error: %s\n' "$1" >&2
  exit 1
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || die "required command not found: $1"
}

detect_platform() {
  case "$(uname -s)" in
    Linux) printf '%s\n' 'linux' ;;
    Darwin) printf '%s\n' 'darwin' ;;
    *) die "unsupported operating system: $(uname -s)" ;;
  esac
}

detect_architecture() {
  case "$(uname -m)" in
    x86_64 | amd64) printf '%s\n' 'amd64' ;;
    arm64 | aarch64) printf '%s\n' 'arm64' ;;
    *) die "unsupported architecture: $(uname -m)" ;;
  esac
}

resolve_version() {
  requested=${JUP_VERSION:-}
  if [ -n "$requested" ]; then
    case "$requested" in
      v*) tag=$requested ;;
      *) tag="v$requested" ;;
    esac
    write_step "Using requested version $tag" >&2
  else
    write_step 'Resolving the latest GitHub release' >&2
    latest_url=$(curl -fsSL -o /dev/null -w '%{url_effective}' "$release_base/latest") ||
      die 'could not resolve the latest GitHub release'
    latest_url=${latest_url%/}
    tag=${latest_url##*/}
    write_step "Latest version: $tag" >&2
  fi

  printf '%s\n' "$tag" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$' ||
    die "invalid release version: $tag"
  printf '%s\n' "$tag"
}

calculate_checksum() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{ print tolower($1) }'
    return
  fi
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{ print tolower($1) }'
    return
  fi
  die 'required checksum command not found: sha256sum or shasum'
}

shell_quote() {
  printf "'"
  printf '%s' "$1" | sed "s/'/'\\\\''/g"
  printf "'"
}

add_to_path() {
  bin_directory=$1
  platform=$2

  if [ -n "$no_modify_path" ]; then
    write_step 'Skipping PATH update because JUP_NO_MODIFY_PATH is set'
    return
  fi

  case ":${PATH:-}:" in
    *":$bin_directory:"*)
      write_step "$bin_directory is already in PATH"
      return
      ;;
  esac

  shell_path=${SHELL:-}
  shell_name=${shell_path##*/}
  case "$shell_name" in
    zsh) profile="$HOME/.zshrc" ;;
    bash)
      if [ "$platform" = 'darwin' ]; then
        profile="$HOME/.bash_profile"
      else
        profile="$HOME/.bashrc"
      fi
      ;;
    fish) profile="$HOME/.config/fish/config.fish" ;;
    *) profile="$HOME/.profile" ;;
  esac

  quoted_directory=$(shell_quote "$bin_directory")
  if [ "$shell_name" = 'fish' ]; then
    path_line="fish_add_path $quoted_directory"
  else
    path_line="export PATH=$quoted_directory:\$PATH"
  fi
  if [ -f "$profile" ] && grep -Fqx "$path_line" "$profile"; then
    write_step "$bin_directory is already configured in $profile"
    return
  fi

  mkdir -p "${profile%/*}"
  printf '\n# Added by the javaup installer\n%s\n' "$path_line" >>"$profile"
  write_step "Added $bin_directory to PATH in $profile"
}

cleanup() {
  if [ -n "$temporary" ] && [ -d "$temporary" ]; then
    rm -rf "$temporary"
  fi
}

require_command curl
require_command grep
require_command awk
require_command sed
require_command tar
require_command find
require_command mktemp
require_command uname

platform=$(detect_platform)
architecture=$(detect_architecture)
tag=$(resolve_version)
version=${tag#v}
archive_name="javaup-$version-$platform-$architecture.tar.gz"
download_base="$release_base/download/$tag"

temporary=$(mktemp -d "${TMPDIR:-/tmp}/javaup-install.XXXXXX")
trap cleanup 0
trap 'exit 1' HUP INT TERM

archive="$temporary/$archive_name"
checksums="$temporary/checksums.txt"
expanded="$temporary/expanded"

write_step "Downloading $archive_name"
curl -fL --retry 3 --retry-delay 1 -o "$archive" "$download_base/$archive_name"
curl -fL --retry 3 --retry-delay 1 -o "$checksums" "$download_base/checksums.txt"

write_step 'Verifying SHA-256 checksum'
expected=$(awk -v name="$archive_name" '$2 == name || $2 == "*" name { print tolower($1); exit }' "$checksums")
printf '%s\n' "$expected" | grep -Eq '^[a-f0-9]{64}$' ||
  die "checksum for $archive_name was not found"
actual=$(calculate_checksum "$archive")
[ "$actual" = "$expected" ] || die "checksum mismatch: expected $expected, got $actual"

mkdir -p "$expanded"
tar -xzf "$archive" -C "$expanded"
binaries=$(find "$expanded" -type f -name 'jup' -print)
binary=$(printf '%s\n' "$binaries" | sed -n '1p')
extra_binary=$(printf '%s\n' "$binaries" | sed -n '2p')
[ -n "$binary" ] || die 'release archive does not contain jup'
[ -z "$extra_binary" ] || die 'release archive contains more than one jup binary'

bin_directory="$install_dir/bin"
mkdir -p "$bin_directory"
bin_directory=$(cd "$bin_directory" && pwd -P)
staged="$bin_directory/.jup-$$.tmp"
cp "$binary" "$staged"
chmod 0755 "$staged"
mv -f "$staged" "$bin_directory/jup"

add_to_path "$bin_directory" "$platform"
write_step "Installed jup $tag to $bin_directory/jup"
write_step 'Open a new terminal and run: jup version'
