/**
 * CspmSeverityBadge — small pill that color-codes a CSPM finding severity.
 *
 * Used on CspmPage's findings table. Five tones mirror the spec
 * (critical / high / medium / low / info) and reuse the existing
 * .dash-sev-* token classes defined in styles/dashboard.css so we
 * stay consistent with ThreatCard's severity pill:
 *   - critical → red + dash-sev-pulse animation
 *   - high     → red
 *   - medium   → amber
 *   - low      → muted/surface-2
 *   - info     → accent
 *
 * No hex literals — every color comes from a CSS variable.
 */
import type { ReactNode } from 'react';

export type CspmSeverity = 'critical' | 'high' | 'medium' | 'low' | 'info';

export interface CspmSeverityBadgeProps {
  severity: CspmSeverity | string;
  children?: ReactNode;
}

function toneClass(severity: string): string {
  const sev = (severity || '').toLowerCase();
  switch (sev) {
    case 'critical':
      return 'dash-sev dash-sev-critical';
    case 'high':
      return 'dash-sev dash-sev-high';
    case 'medium':
      return 'dash-sev dash-sev-medium';
    case 'low':
      return 'dash-sev dash-sev-low';
    case 'info':
      return 'dash-sev dash-sev-info';
    default:
      return 'dash-sev dash-sev-low';
  }
}

export default function CspmSeverityBadge({ severity, children }: CspmSeverityBadgeProps) {
  const sev = (severity || 'low').toLowerCase();
  return (
    <span className={toneClass(sev)}>
      {children ?? sev}
    </span>
  );
}