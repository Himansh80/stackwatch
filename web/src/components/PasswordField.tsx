import { useId, useState } from 'react';

/**
 * Minimal password input with an eye-icon show/hide toggle inside the
 * field. Use this wherever the user is entering a password they're
 * not choosing — sign-in, "current password" on change-password,
 * confirm-password fields, etc.
 *
 * The toggle only renders once the field has at least one character;
 * empty fields don't need it.
 *
 * For sign-up / change-password where the user IS choosing a new
 * password, use <PasswordInput/> instead so they get the strength
 * meter + rule checklist.
 */

interface PasswordFieldProps {
  value: string;
  onChange: (next: string) => void;
  /** Optional placeholder override. */
  placeholder?: string;
  /** Optional aria-label override. */
  ariaLabel?: string;
  /** Required for form submission. */
  required?: boolean;
  /** Name attribute. */
  name?: string;
  /** Auto-complete attribute. "current-password" for sign-in / old-password fields; "new-password" for confirm fields. */
  autoComplete?: string;
  /** Auto-focus on mount. */
  autoFocus?: boolean;
  /** Optional minLength attribute. Defaults to 0 (no client-side minimum — we don't enforce policy on existing passwords). */
  minLength?: number;
  /** Optional maxLength attribute. */
  maxLength?: number;
  /** Optional aria-invalid value (when the field has an external validation error). */
  ariaInvalid?: 'true' | 'false' | 'grammar' | 'spelling';
  /** Extra class on the wrapper. */
  className?: string;
}

function EyeIcon({ open }: { open: boolean }) {
  return open ? (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7-10-7-10-7Z" />
      <circle cx="12" cy="12" r="3" />
    </svg>
  ) : (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M17.94 17.94A10.94 10.94 0 0 1 12 19c-6.5 0-10-7-10-7a18.45 18.45 0 0 1 4.06-5.18" />
      <path d="M9.9 4.24A10.94 10.94 0 0 1 12 4c6.5 0 10 7 10 7a18.5 18.5 0 0 1-2.16 3.19" />
      <path d="M14.12 14.12A3 3 0 0 1 9.88 9.88" />
      <path d="M2 2l20 20" />
    </svg>
  );
}

export default function PasswordField({
  value,
  onChange,
  placeholder = 'Password',
  ariaLabel,
  required,
  name = 'password',
  autoComplete = 'current-password',
  autoFocus,
  minLength,
  maxLength,
  ariaInvalid,
  className,
}: PasswordFieldProps) {
  const id = useId();
  const [show, setShow] = useState(false);
  const showToggle = value.length > 0;

  return (
    <div className={`pwd-input ${className ?? ''}`}>
      <div className="pwd-field">
        <input
          id={id}
          name={name}
          type={show ? 'text' : 'password'}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder={placeholder}
          aria-label={ariaLabel ?? 'Password'}
          required={required}
          autoFocus={autoFocus}
          autoComplete={autoComplete}
          spellCheck={false}
          {...(minLength !== undefined ? { minLength } : {})}
          {...(maxLength !== undefined ? { maxLength } : {})}
          aria-invalid={ariaInvalid}
        />
        {showToggle && (
          <button
            type="button"
            className="pwd-toggle pwd-toggle-eye"
            onClick={() => setShow((v) => !v)}
            aria-label={show ? 'Hide password' : 'Show password'}
            aria-pressed={show}
            title={show ? 'Hide password' : 'Show password'}
            tabIndex={-1}
          >
            <EyeIcon open={show} />
          </button>
        )}
      </div>
    </div>
  );
}
