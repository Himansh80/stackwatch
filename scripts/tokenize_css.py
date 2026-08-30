#!/usr/bin/env python3
"""Tokenize hardcoded hex/rgba colors in web/src/styles/*.css.

Maps the well-known dark-theme palette to CSS custom property references.
Run from the repo root. Idempotent — already-tokenized values are untouched.
"""
import os, re, sys

BASE = os.path.join(os.path.dirname(__file__), "..", "web", "src", "styles")

COLOR_MAP = {
    '#0a0e1a': 'var(--color-bg)', '#0a0f1c': 'var(--color-bg)',
    '#0a101b': 'var(--color-bg)', '#0a1322': 'var(--color-bg)',
    '#06121e': 'var(--color-bg)', '#04111d': 'var(--color-bg)',
    '#0b1525': 'var(--color-surface)', '#0c1524': 'var(--color-surface)',
    '#0e1626': 'var(--color-surface)', '#0e1726': 'var(--color-surface)',
    '#0f1a2e': 'var(--color-surface)', '#131826': 'var(--color-surface)',
    '#0e1a30': 'var(--color-surface-elevated)', '#101a2e': 'var(--color-surface-elevated)',
    '#101b2d': 'var(--color-surface-elevated)', '#1c2235': 'var(--color-surface-elevated)',
    '#1a2540': 'var(--color-surface-elevated)',
    '#0f1c33': 'var(--color-surface-hover)', '#11203b': 'var(--color-surface-hover)',
    '#112033': 'var(--color-surface-hover)', '#14233a': 'var(--color-surface-hover)',
    '#14233d': 'var(--color-surface-hover)', '#16243b': 'var(--color-surface-hover)',
    '#16243d': 'var(--color-surface-hover)',
    '#f3f7fd': 'var(--color-text)', '#f6f9ff': 'var(--color-text)',
    '#eaf2fc': 'var(--color-text)', '#e7edf7': 'var(--color-text)',
    '#e2ebf5': 'var(--color-text)', '#dae8f7': 'var(--color-text)',
    '#dce8f6': 'var(--color-text)', '#dbe7f4': 'var(--color-text)',
    '#dce7f4': 'var(--color-text)', '#cfe6f7': 'var(--color-text)',
    '#cad5e6': 'var(--color-text)', '#c9d9eb': 'var(--color-text)',
    '#b4c6dd': 'var(--color-text-muted)', '#b6c3da': 'var(--color-text-muted)',
    '#a9bdd7': 'var(--color-text-muted)', '#a9c0e0': 'var(--color-text-muted)',
    '#9eacce': 'var(--color-text-muted)', '#94a3b8': 'var(--color-text-muted)',
    '#8294ae': 'var(--color-text-muted)', '#8195b3': 'var(--color-text-muted)',
    '#6e83a1': 'var(--color-text-muted)', '#6d809d': 'var(--color-text-muted)',
    '#7387a5': 'var(--color-text-subtle)', '#7186a2': 'var(--color-text-subtle)',
    '#7287a3': 'var(--color-text-subtle)', '#68809e': 'var(--color-text-subtle)',
    '#677790': 'var(--color-text-subtle)', '#667a99': 'var(--color-text-subtle)',
    '#657b98': 'var(--color-text-subtle)', '#647994': 'var(--color-text-subtle)',
    '#5e7391': 'var(--color-text-disabled)', '#5e7493': 'var(--color-text-disabled)',
    '#5d6e8c': 'var(--color-text-disabled)', '#5b6c87': 'var(--color-text-disabled)',
    '#67e8f9': 'var(--color-primary)', '#a5f3fc': 'var(--color-primary-hover)',
    '#22d3ee': 'var(--color-primary)', '#38bdf8': 'var(--color-primary)',
    '#34d399': 'var(--color-success)', '#6ee7b7': 'var(--color-success)',
    '#86efac': 'var(--color-success)', '#5ee6ad': 'var(--color-success)',
    '#a7f3d0': 'var(--color-success)', '#10b981': 'var(--color-success)',
    '#f59e0b': 'var(--color-warning)', '#fcd34d': 'var(--color-warning)',
    '#fbbf24': 'var(--color-warning)',
    '#ef4444': 'var(--color-error)', '#fb7185': 'var(--color-error)',
    '#fca5a5': 'var(--color-error)', '#fda4af': 'var(--color-error)',
    '#fecdd3': 'var(--color-error)',
    '#93c5fd': 'var(--color-info)',
    '#18263b': 'var(--color-border)', '#182840': 'var(--color-border)',
    '#1a273d': 'var(--color-border)', '#1b2940': 'var(--color-border)',
    '#1b2c43': 'var(--color-border)', '#1d324a': 'var(--color-border)',
    '#1e3048': 'var(--color-border-strong)', '#20314a': 'var(--color-border-strong)',
    '#20304a': 'var(--color-border-strong)', '#21334b': 'var(--color-border-strong)',
    '#243b59': 'var(--color-border-strong)', '#263b57': 'var(--color-border-strong)',
    '#26344a': 'var(--color-border-strong)', '#2a3b53': 'var(--color-border-strong)',
    '#2a3e5d': 'var(--color-border-strong)', '#2e4a73': 'var(--color-border-strong)',
 '#40749b': 'var(--color-primary)',
 # Second-pass additions
 '#58c4d9': 'var(--color-primary)',
 '#fff': 'var(--color-text-inverse, #fff)',
 '#06111d': 'var(--color-bg)', '#080c16': 'var(--color-bg)',
 '#080d18': 'var(--color-bg)', '#090f1b': 'var(--color-bg)',
 '#0a1018': 'var(--color-bg)', '#0b111e': 'var(--color-bg)',
 '#0b1220': 'var(--color-bg)', '#0d1523': 'var(--color-surface)',
 '#101827': 'var(--color-surface)', '#101b2b': 'var(--color-surface)',
 '#111d2e': 'var(--color-surface)', '#142238': 'var(--color-surface)',
 '#18263a': 'var(--color-border)', '#1a2d47': 'var(--color-surface-elevated)',
 '#1b2638': 'var(--color-surface-elevated)', '#1b2a3e': 'var(--color-border)',
 '#1d2b40': 'var(--color-border)', '#1d3650': 'var(--color-border-strong)',
 '#20324b': 'var(--color-border-strong)', '#213a56': 'var(--color-border-strong)',
 '#24354c': 'var(--color-border-strong)', '#382b18': 'var(--color-warning-muted)',
 '#4d7498': 'var(--color-text-subtle)', '#5965e8': 'var(--color-indigo, #6366f1)',
 '#596d8a': 'var(--color-text-subtle)', '#60a5fa': 'var(--color-info)',
 '#61738f': 'var(--color-text-subtle)', '#6366f1': 'var(--color-indigo, #6366f1)',
 '#637895': 'var(--color-text-subtle)', '#06b6d4': 'var(--color-primary)',
 '#0f1525': 'var(--color-surface)', '#1a2138': 'var(--color-surface-elevated)',
 '#1f2742': 'var(--color-surface-elevated)', '#3b82f6': 'var(--color-info)',
 '#475569': 'var(--color-text-subtle)', '#f0f4fc': 'var(--color-text)',
 '#f87171': 'var(--color-error)', '#16b9d1': 'var(--color-primary)',
 # base.css third pass — text-muted/subtle ramp + indigo accents + light text
 '#70809a': 'var(--color-text-subtle)', '#6d7c96': 'var(--color-text-subtle)',
 '#91a0b9': 'var(--color-text-muted)', '#eff8ff': 'var(--color-text)',
 '#dce8f7': 'var(--color-text)', '#7486a3': 'var(--color-text-subtle)',
 '#7e8da6': 'var(--color-text-muted)', '#7c8da7': 'var(--color-text-subtle)',
 '#7e91ad': 'var(--color-text-muted)', '#9fb1c9': 'var(--color-text-muted)',
 '#7f91ad': 'var(--color-text-muted)', '#71839e': 'var(--color-text-subtle)',
 '#e1ebf7': 'var(--color-text)', '#c4d1e3': 'var(--color-text-muted)',
 '#a8c7e8': 'var(--color-text-muted)', '#e8eef8': 'var(--color-text)',
 '#71829c': 'var(--color-text-subtle)', '#6f87a8': 'var(--color-text-subtle)',
 '#8292ac': 'var(--color-text-muted)', '#f2f7ff': 'var(--color-text)',
 '#a995a2': 'var(--color-error-muted)',
 }

HEX_RE = re.compile(r'#[0-9a-fA-F]{3,8}\b')

def sub(text):
    def repl(m):
        return COLOR_MAP.get(m.group(0).lower(), m.group(0))
    return HEX_RE.sub(repl, text)

def main():
    for fn in ['dashboard.css', 'profile.css', 'shell.css', 'base.css', 'tokens.css']:
        p = os.path.join(BASE, fn)
        if not os.path.exists(p):
            continue
        orig = open(p, encoding='utf-8').read()
        new = sub(orig)
        before = len(HEX_RE.findall(orig))
        after = len(HEX_RE.findall(new))
        remaining = sorted(set(HEX_RE.findall(new)))
        changed = orig != new
        if changed:
            open(p, 'w', encoding='utf-8', newline='\n').write(new)
        print(f"{fn}: {before} -> {after} hex literals (changed={changed})")
        if remaining:
            print("  remaining:", remaining[:30])

if __name__ == '__main__':
    main()
