#!/usr/bin/env bash
set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"

if [ "$#" -gt 0 ]; then
	files=("$@")
else
	files=("README.md" "CONTRIBUTING.md" "AGENTS.md")
	while IFS= read -r path; do
		files+=("$path")
	done < <(find docs -type f -name '*.md' | sort)
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
			*"{"* | *"}"* | *"<"* | *">"* | *"*"* | *"|"* | *"..."*)
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

		resolved="${resolved%%#*}"
		resolved="${resolved%%\?*}"
		resolved="$(printf '%s' "$resolved" | perl -pe 's/:[0-9]+(?::[0-9]+)?$//')"
		resolved="${resolved%,}"
		resolved="${resolved%.}"
		resolved="${resolved%/}"

		if [ ! -e "$resolved" ]; then
			base=$(basename "$resolved")
			if [[ "$candidate" != */ ]] \
				&& [ "$candidate" != "README.md" ] \
				&& [ "$candidate" != "CONTRIBUTING.md" ] \
				&& [ "$candidate" != "AGENTS.md" ] \
				&& [ "$candidate" != "PLANS.md" ] \
				&& [ "$candidate" != "Makefile" ]; then
				ext="${base##*.}"
				if [ "$base" = "$ext" ]; then
					continue
				fi
				case "$ext" in
					md|go|sh|json|jsonc|yml|yaml|proto|templ|sql|lua|txt|svg|toml|conf|ini|csv)
						;;
					*)
						continue
						;;
				esac
			fi
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
