#!/usr/bin/env bash
set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"

if [ "$#" -gt 0 ]; then
	files=("$@")
else
	files=(
		"CONTRIBUTING.md"
		"docs/audience/contributors.md"
		"docs/audience/contributor-map.md"
	)
fi

missing=()

for file in "${files[@]}"; do
	if [ ! -f "$file" ]; then
		echo "scan file does not exist: $file" >&2
		exit 1
	fi

	dir=$(dirname "$file")
	while IFS= read -r token; do
		candidate="$token"
		if [ -z "$candidate" ]; then
			continue
		fi
		case "$candidate" in
			*" "* | *$'\t'*)
				continue
				;;
			*"://"* | localhost:* | http:* | https:*)
				continue
				;;
			*"{"* | *"}"* | *"*"* | *"|"* | *"..."*)
				continue
				;;
			-*)
				continue
				;;
		esac

		is_path=0
		case "$candidate" in
			./* | ../* | cmd/* | internal/* | docs/* | api/* | scripts/* | .github/* | .devcontainer/* | .agents/* | README.md | CONTRIBUTING.md | AGENTS.md | PLANS.md | Makefile)
				is_path=1
				;;
		esac
		if [ "$is_path" -eq 0 ]; then
			continue
		fi

		resolved="$candidate"
		if [[ "$candidate" == ./* || "$candidate" == ../* ]]; then
			resolved="$dir/$candidate"
		fi

		resolved="${resolved%,}"
		resolved="${resolved%.}"
		resolved="${resolved%/}"

		if [ ! -e "$resolved" ]; then
			missing+=("$file:$candidate")
		fi
	done < <(perl -ne 'while(/`([^`]+)`/g){print "$1\n"}' "$file")
done

if [ "${#missing[@]}" -ne 0 ]; then
	echo "Broken docs paths detected:" >&2
	printf '  %s\n' "${missing[@]}" >&2
	exit 1
fi

echo "Docs path check passed."
