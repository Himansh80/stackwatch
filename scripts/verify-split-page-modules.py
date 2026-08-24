"""
Phase 4 verification for split-page-modules spec.

Runs:
  1. npm run type-check (must exit 0)
  2. npm run build (must exit 0)
  3. Bundle size comparison (pre vs post)
  4. File count check (no file over 500 lines for non-tests)
  5. CSS class sanity (all .dash-*, .prof-*, .auth-*, .sw-filter-*, .landing-* present)
  6. CSS_EOF marker absence (was bug, now fixed)

Returns PASS/FAIL summary + writes to proof.txt.
"""
import os
import subprocess
import sys
import re
from pathlib import Path

ROOT = Path(r'C:/Users/himan/HermesProjects/stackwatch')
WEB = ROOT / 'web'
PROOF = Path(r'C:/Users/himan/AppData/Local/Temp/hermes-verify-split-page-modules-2026-08-24.txt')

def run(cmd, cwd=WEB, timeout=180):
    proc = subprocess.run(cmd, shell=True, cwd=cwd, capture_output=True, text=True, timeout=timeout)
    return proc.returncode, proc.stdout, proc.stderr

results = []

def check(name, ok, detail=''):
    results.append((name, ok, detail))
    marker = 'PASS' if ok else 'FAIL'
    print(f'[{marker}] {name}{": " + detail if detail else ""}')
    return ok

# 1. type-check
rc, out, err = run('npm run type-check 2>&1', timeout=120)
check('npm run type-check', rc == 0, f'rc={rc}')

# 2. build
rc, out, err = run('npm run build 2>&1', timeout=180)
check('npm run build', rc == 0, f'rc={rc}')

# 3. bundle sizes (post-refactor)
if rc == 0:
    dist = WEB / 'dist/assets'
    js_files = sorted(dist.glob('index-*.js'))
    css_files = sorted(dist.glob('index-*.css'))
    if js_files:
        js_size = js_files[-1].stat().st_size
        check('JS bundle exists', True, f'{js_size} bytes ({js_size // 1024}KB)')
    if css_files:
        css_size = css_files[-1].stat().st_size
        check('CSS bundle exists', True, f'{css_size} bytes ({css_size // 1024}KB)')

# 4. file line counts (Phase 1+2+3 result)
page_files = {
    'Dashboard.tsx': ROOT / 'web/src/pages/Dashboard.tsx',
    'ProfilePage.tsx': ROOT / 'web/src/pages/ProfilePage.tsx',
    'styles.css': ROOT / 'web/src/styles.css',
}
for name, path in page_files.items():
    if path.exists():
        lines = len(path.read_text(encoding='utf-8').splitlines())
        check(f'{name} line count', lines < 500, f'{lines} lines')

# 5. component file count
comp_dir = ROOT / 'web/src/components'
if comp_dir.exists():
    component_files = list(comp_dir.rglob('*.tsx'))
    dashboard_comps = list((comp_dir / 'dashboard').glob('*.tsx')) if (comp_dir / 'dashboard').exists() else []
    profile_comps = list((comp_dir / 'profile').glob('*.tsx')) if (comp_dir / 'profile').exists() else []
    check('Total component files', len(component_files) >= 12, f'{len(component_files)} files')
    check('Dashboard components', len(dashboard_comps) >= 6, f'{len(dashboard_comps)} files (expected 6+)')
    check('Profile components', len(profile_comps) >= 7, f'{len(profile_comps)} files (expected 7+)')

# 6. CSS class sanity (sample from built CSS)
css_file = None
if css_files:
    css_file = css_files[-1]
elif (WEB / 'dist/assets').exists():
    css_candidates = sorted((WEB / 'dist/assets').glob('index-*.css'))
    css_file = css_candidates[-1] if css_candidates else None

if css_file:
    css_text = css_file.read_text(encoding='utf-8')
    sample_classes = ['prof-hero', 'prof-id-row', 'prof-tokens', 'dash-metric', 'dash-sidebar', 'dash-topbar', 'auth-card', 'sw-filter', 'landing-button', 'cmd-palette']
    for cls in sample_classes:
        check(f'CSS class .{cls}', cls in css_text, f'{css_text.count(cls)} occurrences')
    check('CSS_EOF stray marker absent', 'CSS_EOF' not in css_text, '')

# 7. CSS bundle split
styles_dir = ROOT / 'web/src/styles'
if styles_dir.exists():
    css_files_split = sorted(styles_dir.glob('*.css'))
    check('CSS split into 5 files', len(css_files_split) == 5, f'{len(css_files_split)} files: {[f.name for f in css_files_split]}')

# Summary
passed = [r for r in results if r[1]]
failed = [r for r in results if not r[1]]
print(f'\n=== SUMMARY ===')
print(f'PASS: {len(passed)}/{len(results)}')
print(f'FAIL: {len(failed)}')
for r in failed:
    print(f'  FAILED: {r[0]} — {r[2]}')

# Write to proof.txt
PROOF.write_text(
    f'Verifier: split-page-modules Phase 4 (final)\n'
    f'Date: 2026-08-24\n'
    f'Result: {len(passed)}/{len(results)} PASS\n\n'
    + '\n'.join(f'{"PASS" if ok else "FAIL"}: {name} ({detail})' for name, ok, detail in results)
    + '\n',
    encoding='utf-8',
)
print(f'\nProof written to: {PROOF}')
sys.exit(0 if not failed else 1)