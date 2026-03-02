#!/usr/bin/env bash
set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"

python - <<'PY'
from pathlib import Path
import os
import sys

repo = Path('.').resolve()
architecture_root = repo / 'docs' / 'architecture'
max_lines = int(os.environ.get('ARCHITECTURE_DOC_MAX_LINES', '150'))

violations = []
for path in sorted(architecture_root.rglob('*.md')):
    if path.name == 'index.md':
        continue
    line_count = len(path.read_text(encoding='utf-8', errors='ignore').splitlines())
    if line_count > max_lines:
        violations.append((path.relative_to(repo).as_posix(), line_count))

if violations:
    print('Architecture page budget check failed:', file=sys.stderr)
    print(f'  max allowed lines per page: {max_lines}', file=sys.stderr)
    for rel, count in violations:
        print(f'  {rel}: {count} lines', file=sys.stderr)
    sys.exit(1)

print('Architecture page budget check passed.')
PY
