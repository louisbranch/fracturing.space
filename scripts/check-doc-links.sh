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

files = sorted(docs_root.rglob('*.md')) + [
    repo / 'README.md',
    repo / 'CONTRIBUTING.md',
    repo / 'AGENTS.md',
]

link_re = re.compile(r'\[[^\]]+\]\(([^)]+)\)')
broken = []
policy_violations = []


def clean_fenced_blocks(text: str) -> str:
    cleaned_lines = []
    in_fence = False
    for line in text.splitlines():
        if line.strip().startswith('```'):
            in_fence = not in_fence
            cleaned_lines.append('')
            continue
        cleaned_lines.append('' if in_fence else line)
    return '\n'.join(cleaned_lines)


for file_path in files:
    text = file_path.read_text(encoding='utf-8', errors='ignore')
    cleaned = clean_fenced_blocks(text)
    in_docs_tree = docs_root in file_path.parents

    for href in link_re.findall(cleaned):
        href = href.strip()
        if not href:
            continue
        if href.startswith(('http://', 'https://', 'mailto:', '#')):
            continue

        target = href.split('#', 1)[0].split('?', 1)[0].strip()
        if not target:
            continue

        # README badge assets are published from a separate branch path.
        if target.startswith('../../raw/'):
            continue

        if target.startswith('/'):
            resolved = (repo / target.lstrip('/')).resolve()
            relative_link = False
        else:
            resolved = (file_path.parent / target).resolve()
            relative_link = True

        if not resolved.exists():
            broken.append(f"{file_path.relative_to(repo)}:{href}")
            continue

        # Docs pages must not link via relative markdown paths outside docs/.
        if in_docs_tree and relative_link and resolved.suffix == '.md':
            if docs_root not in resolved.parents and resolved != docs_root:
                policy_violations.append(
                    f"{file_path.relative_to(repo)}:{href} -> {resolved.relative_to(repo)}"
                )

if broken:
    print('Broken markdown links detected:', file=sys.stderr)
    for item in broken:
        print(f'  {item}', file=sys.stderr)
    sys.exit(1)

if policy_violations:
    print('Docs relative-link policy violations detected:', file=sys.stderr)
    for item in policy_violations:
        print(f'  {item}', file=sys.stderr)
    print('Use docs-internal relative links or absolute https links for repo-root markdown files.', file=sys.stderr)
    sys.exit(1)

print('Docs markdown link check passed.')
PY
