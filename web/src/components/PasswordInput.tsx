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
 * Password input with a live strength meter and per-rule checklist.
 *
 * The strength meter + rules are LIVE feedback — they evaluate the
 * password on every keystroke (synchronous, sub-millisecond). When the
 * backend rejects a submission with one of the `password_*` codes,
 * pass that code via `errorCode` to highlight the matching rule.
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
        <button
          type="button"
          className="pwd-toggle"
          onClick={() => setShow((v) => !v)}
          aria-label={show ? 'Hide password' : 'Show password'}
          tabIndex={-1}
        >
          {show ? 'Hide' : 'Show'}
        </button>
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
