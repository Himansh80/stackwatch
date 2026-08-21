// Password policy helpers — mirror the server-side rules in
// internal/auth/password_policy.go. Both sides share the same error
// codes so the frontend can show the same friendly messages whether
// validation fails client-side or server-side.

import { COMMON_PASSWORDS } from './password-blocklist';

export const MIN_LENGTH = 10;
export const MAX_LENGTH = 72;

// All rule codes the backend can return. Keep this list in sync
// with internal/auth/password_policy.go (Err* sentinels).
export type PasswordCode =
  | 'password_too_short'
  | 'password_too_long'
  | 'password_whitespace_only'
  | 'password_has_leading_trailing_space'
  | 'password_in_blocklist'
  | 'password_needs_letter_and_digit_or_symbol'
  | 'password_must_differ';

export interface PasswordCheck {
  id: PasswordCode;
  label: string;
  passed: boolean;
}

/**
 * Evaluate a password against every rule. Returns the list of
 * checks (one per rule) plus a strength classification derived
 * from how many rules pass and the minimum-length gate.
 *
 * Pure function — no DOM, no network. Safe to call on every
 * keystroke for a live strength meter.
 */
export function evaluatePassword(plain: string): {
  checks: PasswordCheck[];
  strength: 'empty' | 'weak' | 'ok' | 'strong';
  passedCount: number;
  totalCount: number;
} {
  const pwd = plain ?? '';
  const trimmed = pwd.trim();
  const lower = pwd.toLowerCase();

  const hasLetter = /[A-Za-z]/.test(pwd);
  const hasDigit = /\d/.test(pwd);
  const hasSymbol = /[^A-Za-z0-9\s]/.test(pwd);

  const checks: PasswordCheck[] = [
    {
      id: 'password_too_short',
      label: `At least ${MIN_LENGTH} characters`,
      passed: pwd.length >= MIN_LENGTH,
    },
    {
      id: 'password_too_long',
      label: `${MAX_LENGTH} characters or fewer`,
      passed: pwd.length > 0 && pwd.length <= MAX_LENGTH,
    },
    {
      id: 'password_whitespace_only',
      label: 'Not only spaces',
      passed: trimmed.length > 0,
    },
    {
      id: 'password_has_leading_trailing_space',
      label: 'No spaces at the start or end',
      passed: pwd === trimmed || pwd.length === 0,
    },
    {
      id: 'password_in_blocklist',
      label: 'Not in the common-password list',
      passed: pwd.length === 0 || !COMMON_PASSWORDS.has(lower),
    },
    {
      id: 'password_needs_letter_and_digit_or_symbol',
      label: 'Has a letter and either a digit or a symbol',
      passed:
        hasLetter && (hasDigit || hasSymbol),
    },
  ];

  const passedCount = checks.filter((c) => c.passed).length;
  const totalCount = checks.length;

  let strength: 'empty' | 'weak' | 'ok' | 'strong' = 'empty';
  if (pwd.length > 0) {
    // The length rule is the gate: pass it = at least 3 rules.
    // Once past 10 chars, every additional rule lifts the meter.
    if (!checks[0].passed) {
      strength = 'weak';
    } else if (passedCount <= 4) {
      strength = 'weak';
    } else if (passedCount <= 5) {
      strength = 'ok';
    } else {
      strength = 'strong';
    }
  }

  return { checks, strength, passedCount, totalCount };
}

/**
 * Synchronous client-side validation. Returns the FIRST violation
 * as `{ code, message }`, or `null` if everything passes.
 *
 * Mirrors the server-side ValidatePassword (Go). The server is
 * still the source of truth — this is purely for fast UX feedback.
 */
export function validatePasswordClient(plain: string): {
  code: PasswordCode;
  message: string;
} | null {
  const pwd = plain ?? '';
  if (pwd.length < MIN_LENGTH) {
    return {
      code: 'password_too_short',
      message: `Password must be at least ${MIN_LENGTH} characters.`,
    };
  }
  if (pwd.length > MAX_LENGTH) {
    return {
      code: 'password_too_long',
      message: `Password must be ${MAX_LENGTH} characters or fewer.`,
    };
  }
  if (pwd.trim().length === 0) {
    return {
      code: 'password_whitespace_only',
      message: 'Password cannot be only spaces.',
    };
  }
  if (pwd !== pwd.trim()) {
    return {
      code: 'password_has_leading_trailing_space',
      message: 'Remove the spaces at the start and end of your password.',
    };
  }
  if (COMMON_PASSWORDS.has(pwd.toLowerCase())) {
    return {
      code: 'password_in_blocklist',
      message: 'That password is too common. Pick something less guessable.',
    };
  }
  const hasLetter = /[A-Za-z]/.test(pwd);
  const hasDigit = /\d/.test(pwd);
  const hasSymbol = /[^A-Za-z0-9\s]/.test(pwd);
  if (!(hasLetter && (hasDigit || hasSymbol))) {
    return {
      code: 'password_needs_letter_and_digit_or_symbol',
      message: 'Add at least one letter and either a digit or a symbol.',
    };
  }
  return null;
}

/**
 * Friendly message for a server-side password_* error code.
 * Use this when the API returns one of the codes from
 * internal/auth/password_policy.go.
 */
export function friendlyPasswordMessage(
  code: string | undefined,
  fallback: string,
): string {
  switch (code) {
    case 'password_too_short':
      return `Password must be at least ${MIN_LENGTH} characters.`;
    case 'password_too_long':
      return `Password must be ${MAX_LENGTH} characters or fewer.`;
    case 'password_whitespace_only':
      return 'Password cannot be only spaces.';
    case 'password_has_leading_trailing_space':
      return 'Remove the spaces at the start and end of your password.';
    case 'password_in_blocklist':
      return 'That password is too common. Pick something less guessable.';
    case 'password_needs_letter_and_digit_or_symbol':
      return 'Add at least one letter and either a digit or a symbol.';
    case 'password_must_differ':
      return 'New password must be different from your current password.';
    default:
      return fallback;
  }
}
