#!/usr/bin/env bash
# coverpkg-list.sh — outputs a comma-separated -coverpkg package list.
#
# Usage:
#   coverpkg-list.sh <tags> <pattern> [pattern...]
#
# The COVER_EXCLUDE_REGEX env var (if set) filters out packages whose
# import path matches the regex. The output is suitable for passing to
# go test -coverpkg=<output>.
set -euo pipefail

tags="${1:?usage: coverpkg-list.sh <tags> <pattern> [pattern...]}"
shift

exclude="${COVER_EXCLUDE_REGEX:-}"

mapfile -t pkgs < <(go list -tags="$tags" "$@" 2>/dev/null)

if [[ ${#pkgs[@]} -eq 0 ]]; then
	echo "coverpkg-list.sh: no packages matched" >&2
	exit 1
fi

if [[ -n "$exclude" ]]; then
	filtered=()
	for pkg in "${pkgs[@]}"; do
		if ! printf '%s\n' "$pkg" | grep -Eq "$exclude"; then
			filtered+=("$pkg")
		fi
	done
	pkgs=("${filtered[@]}")
fi

if [[ ${#pkgs[@]} -eq 0 ]]; then
	echo "coverpkg-list.sh: all packages excluded by COVER_EXCLUDE_REGEX" >&2
	exit 1
fi

printf '%s' "${pkgs[0]}"
for pkg in "${pkgs[@]:1}"; do
	printf ',%s' "$pkg"
done
printf '\n'
