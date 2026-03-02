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


def front_matter(path: Path) -> str:
    text = path.read_text(encoding='utf-8', errors='ignore')
    if not text.startswith('---\n'):
        return ''
    end = text.find('\n---\n', 4)
    if end == -1:
        return ''
    return text[4:end]


def nav_excluded(path: Path) -> bool:
    fm = front_matter(path)
    return bool(re.search(r'^\s*nav_exclude\s*:\s*true\s*$', fm, flags=re.MULTILINE))


def front_matter_value(path: Path, key: str):
    fm = front_matter(path)
    m = re.search(rf'^\s*{re.escape(key)}\s*:\s*(.+?)\s*$', fm, flags=re.MULTILINE)
    if not m:
        return None
    value = m.group(1).strip()
    if value.startswith('"') and value.endswith('"') and len(value) >= 2:
        value = value[1:-1]
    return value


def normalize_local_target(raw: str):
    target = raw.split('#', 1)[0].split('?', 1)[0].strip()
    if not target:
        return None
    if target.startswith(('http://', 'https://', 'mailto:', '#', '/')):
        return None
    target = target[2:] if target.startswith('./') else target
    return target

for section_dir in sorted(p for p in docs_root.rglob('*') if p.is_dir()):
    index_path = section_dir / 'index.md'
    if not index_path.exists():
        continue
    index_title = front_matter_value(index_path, 'title')

    index_text = index_path.read_text(encoding='utf-8', errors='ignore')
    linked_local = set()
    for href in link_re.findall(index_text):
        target = normalize_local_target(href)
        if target is None:
            continue
        linked_local.add(target.rstrip('/'))

    # Immediate markdown children in same directory.
    for child in sorted(section_dir.glob('*.md')):
        if child.name == 'index.md':
            continue
        if nav_excluded(child):
            continue
        child_parent = front_matter_value(child, 'parent')
        if child_parent and index_title and child_parent != index_title:
            # Child is nested under another page in this directory.
            continue
        if child.name not in linked_local:
            missing.append(f"{section_dir.relative_to(docs_root)}/index.md missing link to {child.name}")

    # Immediate child directories that contain index.md.
    for child_dir in sorted(p for p in section_dir.iterdir() if p.is_dir()):
        child_index = child_dir / 'index.md'
        if not child_index.exists():
            continue
        if nav_excluded(child_index):
            continue
        child_parent = front_matter_value(child_index, 'parent')
        if child_parent and index_title and child_parent != index_title:
            continue
        options = {f"{child_dir.name}", f"{child_dir.name}/index.md"}
        if linked_local.isdisjoint(options):
            missing.append(
                f"{section_dir.relative_to(docs_root)}/index.md missing link to {child_dir.name}/"
            )

if missing:
    print('Docs section index coverage check failed:', file=sys.stderr)
    for item in missing:
        print(f'  {item}', file=sys.stderr)
    sys.exit(1)

print('Docs section index coverage check passed.')
PY
