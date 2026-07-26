#!/usr/bin/env sh

set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
docs_dir=$(dirname "$script_dir")
theme_version=$(tr -d '\r\n' < "$docs_dir/THEME_VERSION")
theme_sha256=$(tr -d '\r\n' < "$docs_dir/THEME_SHA256")
theme_dir="$docs_dir/themes/hugo-geekdoc"
version_file="$theme_dir/.javaup-theme-version"

if [ -f "$version_file" ] && [ "$(tr -d '\r\n' < "$version_file")" = "$theme_version" ]; then
  printf 'hugo-geekdoc %s is already installed.\n' "$theme_version"
  exit 0
fi

case "$theme_version" in
  v[0-9]*) ;;
  *)
    printf 'Invalid theme version: %s\n' "$theme_version" >&2
    exit 1
    ;;
esac

temp_dir=$(mktemp -d)
trap 'rm -rf "$temp_dir"' EXIT HUP INT TERM

archive="$temp_dir/hugo-geekdoc.tar.gz"
staged_theme="$temp_dir/hugo-geekdoc"
download_url="https://github.com/thegeeklab/hugo-geekdoc/releases/download/$theme_version/hugo-geekdoc.tar.gz"

mkdir -p "$staged_theme" "$docs_dir/themes"
curl --fail --location --silent --show-error "$download_url" --output "$archive"
printf '%s  %s\n' "$theme_sha256" "$archive" | sha256sum --check --status
tar -xzf "$archive" -C "$staged_theme"

rm -rf "$theme_dir"
mv "$staged_theme" "$theme_dir"
printf '%s\n' "$theme_version" > "$version_file"
printf 'Installed hugo-geekdoc %s.\n' "$theme_version"
