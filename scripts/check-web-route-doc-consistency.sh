#!/usr/bin/env bash
set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"

python - <<'PY'
from pathlib import Path
import re
import sys

repo = Path('.').resolve()
architecture_doc = repo / 'docs' / 'architecture' / 'foundations' / 'architecture.md'
web_architecture_doc = repo / 'docs' / 'architecture' / 'platform' / 'web-architecture.md'

architecture_text = architecture_doc.read_text(encoding='utf-8', errors='ignore')
web_architecture_text = web_architecture_doc.read_text(encoding='utf-8', errors='ignore')

line_match = re.search(r'Authenticated surface: canonical `/app/\*` routes \(([^\n]+)\)\.', architecture_text)
if not line_match:
    print('Web route docs consistency check failed:', file=sys.stderr)
    print('  docs/architecture/foundations/architecture.md: missing canonical authenticated surface line.', file=sys.stderr)
    sys.exit(1)

routes_clause = line_match.group(1)
if '/app/invites' in routes_clause:
    print('Web route docs consistency check failed:', file=sys.stderr)
    print('  docs/architecture/foundations/architecture.md: authenticated surface still includes `/app/invites`, which is intentionally unregistered.', file=sys.stderr)
    sys.exit(1)

required_routes = [
    '/app/dashboard',
    '/app/campaigns',
    '/app/campaigns/{id}/*',
    '/app/notifications',
    '/app/settings/*',
]
missing_required = [route for route in required_routes if route not in routes_clause]
if missing_required:
    print('Web route docs consistency check failed:', file=sys.stderr)
    print('  docs/architecture/foundations/architecture.md: authenticated surface line is missing expected route claims:', file=sys.stderr)
    for route in missing_required:
        print(f'    - {route}', file=sys.stderr)
    sys.exit(1)

web_invariant = 'Legacy top-level invites scaffolding (`/app/invites`) remains intentionally\n  unregistered until that area has a production route owner.'
if web_invariant not in web_architecture_text:
    print('Web route docs consistency check failed:', file=sys.stderr)
    print('  docs/architecture/platform/web-architecture.md: missing explicit `/app/invites` unregistered invariant.', file=sys.stderr)
    sys.exit(1)

print('Web route docs consistency check passed.')
PY
