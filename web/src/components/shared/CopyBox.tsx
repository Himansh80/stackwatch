import { useCallback, useState } from 'react';
import { motion, buttonSpring, useReducedMotion } from '../../lib/motion';

/**
 * CopyBox — Tier 9.2 (Phase 2) shared utility for showing a one-shot
 * copyable value. Used by ScimSection to display the plaintext SCIM
 * token returned by POST /scim/tokens (the only time the token is
 * ever visible to the operator).
 *
 * Extracted from ScimSection so that file stays under the 400-LOC
 * cap. The visual contract: monospace accent-tinted code box with a
 * "Copy" / "Copied" toggle that uses navigator.clipboard.writeText
 * when available (HTTPS / localhost) and falls back to the legacy
 * document.execCommand path for plain-HTTP contexts.
 */

interface CopyBoxProps {
  value: string;
}

export default function CopyBox({ value }: CopyBoxProps) {
  const reduce = useReducedMotion();
  const [copied, setCopied] = useState(false);
  const copy = useCallback(() => {
    if (navigator.clipboard) {
      void navigator.clipboard.writeText(value).then(() => {
        setCopied(true);
        window.setTimeout(() => setCopied(false), 2000);
      });
    } else {
      // Fallback for non-secure-context browsers (e.g. http://192.168.x.x).
      const el = document.createElement('textarea');
      el.value = value;
      document.body.appendChild(el);
      el.select();
      document.execCommand('copy');
      document.body.removeChild(el);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 2000);
    }
  }, [value]);
  return (
    <div
      style={{
        display: 'flex',
        gap: 8,
        alignItems: 'center',
        background: 'var(--surface-2)',
        border: '1px solid var(--border)',
        borderRadius: 'var(--radius-md)',
        padding: 12,
        fontFamily: 'var(--font-mono, monospace)',
        fontSize: 12,
      }}
    >
      <code
        style={{
          flex: 1,
          wordBreak: 'break-all',
          color: 'var(--accent)',
        }}
      >
        {value}
      </code>
      <motion.button
        type="button"
        className="threat-card-resolve-btn"
        onClick={copy}
        whileHover={reduce ? undefined : buttonSpring.whileHover}
        whileTap={reduce ? undefined : buttonSpring.whileTap}
        transition={buttonSpring.transition}
      >
        {copied ? 'Copied' : 'Copy'}
      </motion.button>
    </div>
  );
}