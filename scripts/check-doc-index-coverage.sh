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

link_re = re.compile(r'\[[^\]]+\]\(([^)]+)\)')
missing = []

for section_dir in sorted(p for p in docs_root.iterdir() if p.is_dir()):
    index_path = section_dir / 'index.md'
    if not index_path.exists():
        continue

    index_text = index_path.read_text(encoding='utf-8', errors='ignore')
    linked_local_children = set()
    for href in link_re.findall(index_text):
        href = href.strip()
        if not href:
            continue
        target = href.split('#', 1)[0].split('?', 1)[0].strip()
        if not target or target.startswith(('http://', 'https://', 'mailto:', '#', '../', '/')):
            continue
        if '/' in target:
            continue
        if target.endswith('.md'):
            linked_local_children.add(target)

    for child in sorted(section_dir.glob('*.md')):
        if child.name == 'index.md':
            continue

        text = child.read_text(encoding='utf-8', errors='ignore')
        if text.startswith('---\n'):
            end = text.find('\n---\n', 4)
            if end != -1:
                front_matter = text[4:end]
                if re.search(r'^\s*nav_exclude\s*:\s*true\s*$', front_matter, flags=re.MULTILINE):
                    continue

        if child.name not in linked_local_children:
            missing.append(f"{section_dir.relative_to(docs_root)}/index.md missing link to {child.name}")

if missing:
    print('Docs section index coverage check failed:', file=sys.stderr)
    for item in missing:
        print(f'  {item}', file=sys.stderr)
    sys.exit(1)

print('Docs section index coverage check passed.')
PY
