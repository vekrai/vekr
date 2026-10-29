#!/usr/bin/env bash
# Vendors the browser JS packages pinned in package.json into web/static/js/.
# Run after bumping a version in package.json: mise run js:vendor
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
vendor_dir="$repo_root/web/static/js"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

# package name -> path to the built file inside its npm tarball
dist_file_hotwired_turbo="dist/turbo.es2017-esm.js"
dist_file_hotwired_stimulus="dist/stimulus.js"

# package name -> vendored output filename
out_name_hotwired_turbo="turbo.min.js"
out_name_hotwired_stimulus="stimulus.min.js"

packages=$(jq -r '.dependencies | keys[]' "$repo_root/package.json")

for pkg in $packages; do
  version=$(jq -r --arg p "$pkg" '.dependencies[$p]' "$repo_root/package.json")
  key=$(echo "$pkg" | tr '@/' '_')
  dist_file_var="dist_file${key}"
  out_name_var="out_name${key}"
  dist_file="${!dist_file_var:?no dist-file mapping for $pkg in this script}"
  out_name="${!out_name_var:?no output-name mapping for $pkg in this script}"

  echo "Vendoring $pkg@$version..."

  pkg_tmp="$tmp_dir/$(echo "$pkg" | tr '/' '_')"
  mkdir -p "$pkg_tmp"
  ( cd "$pkg_tmp" && npm pack "$pkg@$version" --silent >/dev/null )
  tar xzf "$pkg_tmp"/*.tgz -C "$pkg_tmp"

  cp "$pkg_tmp/package/$dist_file" "$vendor_dir/$out_name"
done

echo "Done — review the updated files in $vendor_dir and commit them."
