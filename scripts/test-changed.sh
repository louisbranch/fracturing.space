#!/usr/bin/env bash
set -euo pipefail

changed_go_files="$({
  git diff --name-only --diff-filter=ACMRTUXB HEAD
  git ls-files --others --exclude-standard
} | sort -u | rg '\.go$' || true)"

if [[ -z "${changed_go_files}" ]]; then
  echo "No changed Go files detected; running full unit suite."
  exec go test ./...
fi

mapfile -t changed_dirs < <(printf '%s\n' "${changed_go_files}" | xargs -r -n1 dirname | sort -u)

if [[ "${#changed_dirs[@]}" -eq 0 ]]; then
  echo "No Go package directories detected; running full unit suite."
  exec go test ./...
fi

pkg_candidates=()
for dir in "${changed_dirs[@]}"; do
  if [[ "${dir}" == "." ]]; then
    pkg_candidates+=(".")
  else
    pkg_candidates+=("./${dir}")
  fi
done

mapfile -t changed_pkgs < <(go list "${pkg_candidates[@]}" 2>/dev/null | sort -u || true)

if [[ "${#changed_pkgs[@]}" -eq 0 ]]; then
  echo "No valid Go packages resolved from changed files; running full unit suite."
  exec go test ./...
fi

echo "Testing changed packages:"
printf '  %s\n' "${changed_pkgs[@]}"
go test "${changed_pkgs[@]}"
