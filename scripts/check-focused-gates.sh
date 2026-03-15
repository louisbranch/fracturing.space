#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$repo_root"

make_cmd="${MAKE:-make}"
print_only=false
if [[ "${1:-}" == "--print-targets" ]]; then
	print_only=true
fi

resolve_base_ref() {
	if [[ -n "${CHECK_BASE_REF:-}" ]] && git rev-parse --verify --quiet "${CHECK_BASE_REF}" >/dev/null; then
		printf '%s\n' "${CHECK_BASE_REF}"
		return 0
	fi

	if git rev-parse --verify --quiet origin/main >/dev/null; then
		printf '%s\n' "origin/main"
		return 0
	fi

	return 1
}

base_ref=""
if base_ref="$(resolve_base_ref)"; then
	base_ref="${base_ref}"
else
	base_ref=""
fi

changed_files="$({
	if [[ -n "${base_ref}" ]]; then
		merge_base="$(git merge-base HEAD "${base_ref}")"
		git diff --name-only --diff-filter=ACMRTUXB "${merge_base}"...HEAD
	fi
	git diff --name-only --diff-filter=ACMRTUXB HEAD
	git ls-files --others --exclude-standard
} | sort -u)"

if [[ -z "${changed_files}" ]]; then
	echo "No changed files detected; skipping focused architecture gates."
	exit 0
fi

targets=()

if printf '%s\n' "${changed_files}" | rg -q '^(internal/services/web/|internal/cmd/web/)'; then
	targets+=("web-architecture-check")
fi

if printf '%s\n' "${changed_files}" | rg -q '^(internal/services/game/|api/proto/game/|api/proto/systems/)'; then
	targets+=("game-architecture-check")
fi

if printf '%s\n' "${changed_files}" | rg -q '^(internal/services/admin/|internal/cmd/admin/)'; then
	targets+=("admin-architecture-check")
fi

if printf '%s\n' "${changed_files}" | rg -q '^(internal/services/play/|internal/cmd/play/)'; then
	targets+=("play-architecture-check")
fi

if [[ "${#targets[@]}" -eq 0 ]]; then
	echo "No focused architecture gates selected."
	exit 0
fi

if [[ "${print_only}" == "true" ]]; then
	printf '%s\n' "${targets[@]}"
	exit 0
fi

echo "Focused architecture gates: ${targets[*]}"
for target in "${targets[@]}"; do
	"${make_cmd}" "${target}"
done
