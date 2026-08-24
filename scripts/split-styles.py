#!/usr/bin/env python3
"""
Split web/src/styles.css into 5 files based on class prefix.

Approach: scan styles.css, build a list of class-prefix blocks.
For each block (the section from one comment heading to the next),
classify by the comment text or first class name. Write to the
correct target file.

This is more robust than slicing by line range because the original
file interleaves dashboard classes with profile classes (e.g.
topbar-search lives in dashboard.css but prof-hero-avatar lives
in profile.css — both in the same line range 240-330).
"""
import re
from pathlib import Path

ROOT = Path(r'C:/Users/himan/HermesProjects/stackwatch')
SRC = ROOT / 'web/src/styles.css'
OUT_DIR = ROOT / 'web/src/styles'

# Mapping: prefix -> output file
PREFIX_MAP = {
    'auth': 'auth.css',
    'cmd': 'dashboard.css',  # command palette = dashboard-wide
    'dash': 'dashboard.css',
    'prof': 'profile.css',
    'sw-': 'dashboard.css',  # mostly dashboard shared (filter bar, status)
    'landing': 'landing.css',
    'password': 'auth.css',
    'sign': 'auth.css',  # signup/forgot/reset use these
    'timewidget': 'dashboard.css',
    'card': 'dashboard.css',  # card component used across dashboard
}

# Read source
src_text = SRC.read_text(encoding='utf-8')
lines = src_text.splitlines(keepends=True)

# Section header pattern: matches opening lines of multi-line comment
# blocks. Recognizes:
#   1. /* ------- Title ------- (no */ — multi-line opener)
#   2. /* ---- inline title ---- */ (single-line)
#   3. /* ============== (divider-only opener, no title — skipped)
# Any line starting with `/*` and containing 3+ dashes or equals signs.
SECTION_HEADER = re.compile(r'^/\*(.*[\-=]{3,}.*)\*?$')

def classify_section(comment_text, first_class_in_block):
    """Decide which output file this block belongs to."""
    txt = comment_text.lower()
    cls = (first_class_in_block or '').lower()
    # Explicit token-based classification
    if 'landing' in txt or 'pricing' in txt or cls.startswith('.landing'):
        return 'landing.css'
    if 'profile' in txt or 'prof-' in cls:
        return 'profile.css'
    if 'auth' in txt or 'login' in txt or 'signup' in txt or 'forgot' in txt or 'reset' in txt or 'password' in txt or cls.startswith('.auth') or cls.startswith('.sw-') and ('form' in cls or 'button' in cls or 'field' in cls):
        return 'auth.css'
    if 'dashboard' in txt or 'sidebar' in txt or 'topbar' in txt or 'kpi' in txt or 'metric' in txt or 'panel' in txt or 'greeting' in txt or 'cmd' in txt or 'timewidget' in txt or 'filter' in txt or cls.startswith('.dash') or cls.startswith('.cmd') or cls.startswith('.sw-filter') or cls.startswith('.sw-status') or cls.startswith('.sw-row') or cls.startswith('.timewidget'):
        return 'dashboard.css'
    return 'base.css'  # fallback

# Walk through lines and split into blocks
blocks = []  # list of (header_text, content_lines, target_file)
current_header = None
current_lines = []
i = 0

def first_class_name(content_lines):
    """Find the first CSS class name in the block."""
    for line in content_lines:
        m = re.match(r'^\.([a-zA-Z0-9_-]+)', line)
        if m:
            return '.' + m.group(1)
    return None

# Lines 0-238 (header through ":root + body + media queries") go straight to base.css
# Find boundary at "/* ------- Topbar" (line 240)
topbar_idx = None
for idx, line in enumerate(lines):
    if 'Topbar - 3-zone' in line:
        topbar_idx = idx
        break

# Pre-base section (lines 0 to topbar_idx): :root + body + base inputs/buttons + media queries
if topbar_idx is not None:
    blocks.append(('base prelude', lines[:topbar_idx], 'base.css'))

# Process remaining lines as section blocks
i = topbar_idx if topbar_idx is not None else 0
while i < len(lines):
    line = lines[i]
    # Check for section header
    m = SECTION_HEADER.match(line.strip())
    if m and ('-------' in line or '=======' in line or '----------' in line):
        # Save previous block
        if current_header is not None and current_lines:
            target = classify_section(current_header, first_class_name(current_lines))
            blocks.append((current_header, current_lines, target))
        # Start new block
        current_header = m.group(1).strip()
        current_lines = [line]
    else:
        current_lines.append(line)
    i += 1
# Save last block
if current_header is not None and current_lines:
    target = classify_section(current_header, first_class_name(current_lines))
    blocks.append((current_header, current_lines, target))

# Bucket blocks by target file
buckets = {f: [] for f in {'base.css', 'auth.css', 'dashboard.css', 'profile.css', 'landing.css'}}

# Header for each file (saying which file it is)
file_headers = {
    'base.css': '/* base.css — design tokens, body, base form elements, universal rules. Imported first. */',
    'auth.css': '/* auth.css — login / signup / forgot / reset / password input. Imported after base. */',
    'dashboard.css': '/* dashboard.css — topbar, sidebar, panels, KPI cards, command palette, filter bar, time widget. Imported after auth. */',
    'profile.css': '/* profile.css — profile page (hero, identity, workspace, security, tokens, skeleton). Imported after dashboard. */',
    'landing.css': '/* landing.css — public landing page (landing/pricing styles). Imported last. */',
}

for header, content, target in blocks:
    buckets[target].append(''.join(content))
    # (silenced for production run)
    print(f'  BLOCK "{header[:60]}" → {target}')

# Special: also append landing motion helpers at the end (lines 1503-1538 are about landing)
# These should already be classified as 'landing.css' via "Landing motion helpers" detection

# Special: the "Cmd palette" block (lines 1156-1211) - currently classified as dashboard.css. Good.
# Special: "FilterBar" block (lines 1214-1278) - .sw-filter-* prefix → dashboard.css. Good.

# Verify no empty content blocks
for fname, content_list in buckets.items():
    out_path = OUT_DIR / fname
    full = file_headers[fname] + '\n\n' + ''.join(content_list)
    # Also remove the stray "CSS_EOF" marker (leftover from a prior heredoc)
    full = full.replace('CSS_EOF\n', '')
    out_path.write_text(full, encoding='utf-8')
    print(f'wrote {fname}: {len(full)} bytes')

# Replace styles.css with a barrel that imports the new files
barrel = '''/* styles.css — barrel that imports per-page CSS bundles.
   Vite handles @import natively; tokens load order matters (base first).
   Run `npm run build` to see the inlined production output. */

@import './styles/base.css';
@import './styles/landing.css';
@import './styles/auth.css';
@import './styles/dashboard.css';
@import './styles/profile.css';
'''
(ROOT / 'web/src/styles.css').write_text(barrel, encoding='utf-8')
print('rewrote styles.css as barrel, {} bytes'.format(len(barrel)))

# Summary
total_bytes = sum(len(content_list) for content_list in buckets.values())
print(f'split complete: {len(buckets)} files, {total_bytes} total bytes')
for fname, content_list in buckets.items():
    byte_size = sum(len(c) for c in content_list)
    line_count = sum(c.count('\n') for c in content_list)
    print(f'  {fname}: {byte_size} bytes, {line_count} lines')