#!/usr/bin/env bash
set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"

python - <<'PY'
from pathlib import Path
import re
import sys

repo = Path('.').resolve()
docs_root = repo / 'docs'

checks = [
    (
        'artifact heading',
        re.compile(r'^\s{0,3}#{1,6}\s+.*\b(roadmap|backlog|implementation plan|in progress|work in progress|wip)\b', re.IGNORECASE),
    ),
    (
        'phase tracker line',
        re.compile(r'^\s*phase\s+[0-9]+\s*:', re.IGNORECASE),
    ),
    (
        'legacy title marker',
        re.compile(r'^\s*title\s*:\s*"?.*\(legacy\).*"?\s*$', re.IGNORECASE),
    ),
    (
        'implementation focus heading',
        re.compile(r'^\s{0,3}#{1,6}\s+.*implementation focus', re.IGNORECASE),
    ),
]

violations = []

for path in sorted(docs_root.rglob('*.md')):
    text = path.read_text(encoding='utf-8', errors='ignore')
    for idx, line in enumerate(text.splitlines(), start=1):
        for label, pattern in checks:
            if pattern.search(line):
                violations.append(f"{path.relative_to(repo)}:{idx}: {label}: {line.strip()}")

if violations:
    print('Docs lifecycle policy check failed:', file=sys.stderr)
    for item in violations:
        print(f'  {item}', file=sys.stderr)
    print(
        'Remove roadmap/phase/backlog implementation-tracker artifacts from docs/ and keep working plans in .agents/plans/.',
        file=sys.stderr,
    )
    sys.exit(1)

print('Docs lifecycle policy check passed.')
PY
