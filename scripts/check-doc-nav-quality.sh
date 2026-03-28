#!/usr/bin/env bash
set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"

python - <<'PY'
from collections import defaultdict
from pathlib import Path
import re
import sys

repo = Path('.').resolve()
docs_root = repo / 'docs'

FRONT_MATTER_DELIM = '\n---\n'


def normalize(value: str | None) -> str | None:
    if value is None:
        return None
    return value.strip().lower()


def parse_front_matter(text: str):
    if not text.startswith('---\n'):
        return None, 'missing front matter opening delimiter'
    end = text.find(FRONT_MATTER_DELIM, 4)
    if end == -1:
        return None, 'missing front matter closing delimiter'
    return text[4:end], None


def front_matter_value(front_matter: str, key: str):
    m = re.search(rf'^\s*{re.escape(key)}\s*:\s*(.+?)\s*$', front_matter, flags=re.MULTILINE)
    if not m:
        return None
    raw = m.group(1).strip()
    if raw.startswith('"') and raw.endswith('"') and len(raw) >= 2:
        return raw[1:-1]
    return raw


pages = []
pages_by_path = {}
violations = []

for path in sorted(docs_root.rglob('*.md')):
    text = path.read_text(encoding='utf-8', errors='ignore')
    relative_path = path.relative_to(repo).as_posix()
    docs_relative = path.relative_to(docs_root)
    front_matter, front_matter_error = parse_front_matter(text)
    if front_matter_error:
        violations.append(f'{relative_path}: {front_matter_error}')
        front_matter = ''

    title = front_matter_value(front_matter, 'title') if front_matter else None
    if not title:
        violations.append(f'{relative_path}: missing title in front matter')

    parent = front_matter_value(front_matter, 'parent') if front_matter else None
    nav_order = front_matter_value(front_matter, 'nav_order') if front_matter else None
    nav_exclude = (front_matter_value(front_matter, 'nav_exclude') or '').lower() == 'true'

    page = {
        'path': relative_path,
        'docs_relative': docs_relative,
        'docs_relative_path': docs_relative.as_posix(),
        'title': title,
        'parent': parent,
        'nav_order': nav_order,
        'nav_exclude': nav_exclude,
        'is_index': path.name == 'index.md',
        'directory': path.parent,
    }
    pages.append(page)
    pages_by_path[relative_path] = page

# Only enforce nav groups for visible pages.
visible = [p for p in pages if not p['nav_exclude']]
visible_by_title = defaultdict(list)
for p in visible:
    title_key = normalize(p['title'])
    if title_key:
        visible_by_title[title_key].append(p)

def resolve_parent(page):
    parent_key = normalize(page['parent'])
    if not parent_key:
        return None
    matches = visible_by_title.get(parent_key, [])
    if len(matches) != 1:
        return None
    return matches[0]

# Root visibility and required parent rules.
for p in visible:
    rel_parts = p['docs_relative'].parts
    if p['is_index']:
        if not p['parent'] and len(rel_parts) > 2:
            violations.append(f'{p["path"]}: nested index pages must declare a parent')
        continue
    if not p['parent']:
        violations.append(f'{p["path"]}: visible non-index pages must declare a parent')

for p in visible:
    if p['parent']:
        matches = visible_by_title.get(normalize(p['parent']), [])
        if not matches:
            violations.append(f'{p["path"]}: parent "{p["parent"]}" does not match any visible page title')
        elif len(matches) > 1:
            files = ', '.join(match['path'] for match in matches)
            violations.append(f'{p["path"]}: parent "{p["parent"]}" is ambiguous across {files}')
    elif not p['is_index'] and len(p['docs_relative'].parts) > 2:
        violations.append(f'{p["path"]}: only docs/index.md and first-level section indexes may be root-visible')

# Visible directories must have a visible index.md guide.
visible_by_directory = defaultdict(list)
for p in visible:
    visible_by_directory[p['directory']].append(p)

for directory, directory_pages in sorted(visible_by_directory.items()):
    index_path = (directory / 'index.md').relative_to(repo).as_posix()
    index_page = pages_by_path.get(index_path)
    if index_page is None:
        violations.append(
            f'{directory.relative_to(repo).as_posix()}: visible docs directory is missing index.md'
        )
        continue
    if index_page['nav_exclude']:
        violations.append(
            f'{index_page["path"]}: directory index must be visible when the directory contains visible docs'
        )
    if not index_page['title']:
        violations.append(f'{index_page["path"]}: directory index must declare a title')

# Visible non-index pages must route through their directory guide.
for p in visible:
    if p['is_index']:
        continue
    directory_index_path = (p['directory'] / 'index.md').relative_to(repo).as_posix()
    directory_index = pages_by_path.get(directory_index_path)
    if directory_index is None or directory_index['nav_exclude'] or not directory_index['title']:
        continue
    required_title = normalize(directory_index['title'])
    if not required_title:
        continue

    chain_titles = set()
    seen_paths = set()
    current = p
    while current['parent']:
        parent_page = resolve_parent(current)
        if parent_page is None:
            break
        if parent_page['path'] in seen_paths:
            violations.append(f'{p["path"]}: parent chain contains a cycle')
            break
        seen_paths.add(parent_page['path'])
        if parent_page['title']:
            chain_titles.add(normalize(parent_page['title']))
        current = parent_page
    if required_title not in chain_titles:
        violations.append(
            f'{p["path"]}: parent chain does not route through folder guide "{directory_index["title"]}"'
        )

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
