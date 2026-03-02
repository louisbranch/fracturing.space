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

pages = []
for path in sorted(docs_root.rglob('*.md')):
    text = path.read_text(encoding='utf-8', errors='ignore')
    if not text.startswith('---\n'):
        continue
    end = text.find('\n---\n', 4)
    if end == -1:
        continue
    fm = text[4:end]

    def get(key: str):
        m = re.search(rf'^\s*{re.escape(key)}\s*:\s*(.+?)\s*$', fm, flags=re.MULTILINE)
        if not m:
            return None
        raw = m.group(1).strip()
        if raw.startswith('"') and raw.endswith('"') and len(raw) >= 2:
            return raw[1:-1]
        return raw

    title = get('title')
    parent = get('parent')
    nav_order = get('nav_order')
    nav_exclude = (get('nav_exclude') or '').lower() == 'true'

    pages.append(
        {
            'path': path.relative_to(repo).as_posix(),
            'title': title,
            'parent': parent,
            'nav_order': nav_order,
            'nav_exclude': nav_exclude,
        }
    )

violations = []

# Only enforce nav groups for visible pages.
visible = [p for p in pages if not p['nav_exclude']]

# Duplicate sibling title and nav_order checks by parent.
by_parent = {}
for p in visible:
    parent = p['parent'] or '<root>'
    by_parent.setdefault(parent, []).append(p)

for parent, siblings in sorted(by_parent.items()):
    title_seen = {}
    order_seen = {}
    for p in siblings:
        if p['title']:
            key = p['title'].strip().lower()
            title_seen.setdefault(key, []).append(p)
        if p['nav_order']:
            order_seen.setdefault(p['nav_order'], []).append(p)

    for _, group in title_seen.items():
        if len(group) > 1:
            files = ', '.join(g['path'] for g in group)
            violations.append(f'duplicate sibling title under parent "{parent}": {group[0]["title"]} ({files})')

    for order, group in order_seen.items():
        if len(group) > 1:
            files = ', '.join(g['path'] for g in group)
            violations.append(f'duplicate sibling nav_order under parent "{parent}": {order} ({files})')

# Parent/child title collision check.
children_by_parent_name = {}
for p in visible:
    if p['parent']:
        children_by_parent_name.setdefault(p['parent'].strip().lower(), []).append(p)

for p in visible:
    if not p['title']:
        continue
    key = p['title'].strip().lower()
    for child in children_by_parent_name.get(key, []):
        if child['title'] and child['title'].strip().lower() == key:
            violations.append(
                f'parent/child title collision: parent title "{p["title"]}" and child "{child["path"]}" share same title'
            )

if violations:
    print('Docs nav quality check failed:', file=sys.stderr)
    for v in violations:
        print(f'  {v}', file=sys.stderr)
    sys.exit(1)

print('Docs nav quality check passed.')
PY
