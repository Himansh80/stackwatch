import { useId, useMemo, useState } from 'react';
import {
  evaluatePassword,
  friendlyPasswordMessage,
  MIN_LENGTH,
  type PasswordCheck,
} from '../lib/password';

interface PasswordInputProps {
  value: string;
  onChange: (next: string) => void;
  /** Optional external error code (from server) to display in the rules list. */
  errorCode?: string;
  /** Disable the strength meter + rules UI (e.g. on login where the user isn't choosing a new password). */
  hideStrength?: boolean;
  /** Set true when the user must change their password and confirm a separate old-password field. */
  confirmOldPassword?: string;
  /** Auto-focus on mount. */
  autoFocus?: boolean;
  /** Optional placeholder override. */
  placeholder?: string;
  /** Optional aria-label override. */
  ariaLabel?: string;
  /** Required for form submission. */
  required?: boolean;
  /** Name attribute. */
  name?: string;
  /** Auto-complete attribute for the underlying input. */
  autoComplete?: string;
  /** Extra class on the wrapper. */
  className?: string;
}

/**
 * Eye SVG used for the show/hide toggle. Two paths:
 *   - "open" eye: visible password (the input renders as type=text)
 *   - "closed" eye: hidden password (type=password, the default)
 *
 * Drawn at 18px in the currentColor stroke so it inherits the button
 * text colour and stays legible on both light and dark backgrounds.
 */
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

/**
 * Password input with a live strength meter and per-rule checklist.
 *
 * The strength meter + rules are LIVE feedback — they evaluate the
 * password on every keystroke (synchronous, sub-millisecond). When the
 * backend rejects a submission with one of the `password_*` codes,
 * pass that code via `errorCode` to highlight the matching rule.
 *
 * The show/hide toggle is an eye icon inside the input on the right.
 * It only renders when the field has at least one character — empty
 * password fields don't need a toggle.
 */
export default function PasswordInput({
  value,
  onChange,
  errorCode,
  hideStrength,
  confirmOldPassword,
  autoFocus,
  placeholder = 'At least 10 characters',
  ariaLabel,
  required,
  name = 'password',
  autoComplete = 'new-password',
  className,
}: PasswordInputProps) {
  const id = useId();
  const [show, setShow] = useState(false);

  const evaluation = useMemo(() => evaluatePassword(value), [value]);

  // External "must differ" rule: if a previous-password value was
  // provided and the current value matches it, treat it as a failed
  // check so the user gets a clear "new password must be different"
  // hint without waiting for a server round-trip.
  const checks: PasswordCheck[] = useMemo(() => {
    if (
      confirmOldPassword !== undefined &&
      value.length > 0 &&
      value === confirmOldPassword
    ) {
      return evaluation.checks.map((c) =>
        c.id === 'password_too_short' ? c : c,
      ).concat([
        {
          id: 'password_must_differ',
          label: 'Different from your current password',
          passed: false,
        },
      ]);
    }
    return evaluation.checks;
  }, [evaluation, value, confirmOldPassword]);

  const erroredRuleId = errorCode || undefined;
  // Only render the toggle once the user has typed something — an
  // empty field doesn't need a show/hide control.
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
          minLength={MIN_LENGTH}
          autoComplete={autoComplete}
          spellCheck={false}
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

      {!hideStrength && value.length > 0 && (
        <div className="pwd-meter" aria-live="polite">
          <div className="pwd-meter-bar">
            <div
              className={`pwd-meter-fill pwd-strength-${evaluation.strength}`}
              style={{ width: `${Math.max(8, (evaluation.passedCount / evaluation.totalCount) * 100)}%` }}
            />
          </div>
          <div className="pwd-meter-label">
            <span className={`pwd-strength-tag pwd-strength-${evaluation.strength}`}>
              {evaluation.strength === 'weak'
                ? 'Weak'
                : evaluation.strength === 'ok'
                ? 'OK'
                : evaluation.strength === 'strong'
                ? 'Strong'
                : ''}
            </span>
            {evaluation.passedCount < evaluation.totalCount && (
              <span className="pwd-meter-help">
                {friendlyPasswordMessage(
                  checks.find((c) => !c.passed)?.id,
                  'Pick a stronger password.',
                )}
              </span>
            )}
          </div>

          <ul className="pwd-rules">
            {checks.map((rule) => (
              <li
                key={rule.id}
                className={
                  'pwd-rule ' +
                  (rule.passed ? 'pwd-rule-passed' : 'pwd-rule-failed') +
                  (erroredRuleId === rule.id ? ' pwd-rule-errored' : '')
                }
              >
                <span className="pwd-rule-icon" aria-hidden="true">
                  {rule.passed ? '✓' : '✕'}
                </span>
                <span className="pwd-rule-label">{rule.label}</span>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
