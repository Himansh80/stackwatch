// Password policy helpers — mirror the server-side rules in
// internal/auth/password_policy.go. Both sides share the same error
// codes so the frontend can show the same friendly messages whether
// validation fails client-side or server-side.

import { COMMON_PASSWORDS } from './password-blocklist';

export const MIN_LENGTH = 10;
export const MAX_LENGTH = 72;

// Strength threshold: scores >= this are "strong". Anything below is
// either "ok" (40-69) or "weak" (<40). The strength score combines
// length, character diversity, uniqueness, and pattern avoidance —
// NOT a simple count of how many binary rules pass.
export const STRONG_THRESHOLD = 70;
export const OK_THRESHOLD = 40;

// All rule codes the backend can return. Keep this list in sync
// with internal/auth/password_policy.go (Err* sentinels).
export type PasswordCode =
  | 'password_too_short'
  | 'password_too_long'
  | 'password_whitespace_only'
  | 'password_has_leading_trailing_space'
  | 'password_in_blocklist'
  | 'password_needs_letter_and_digit_or_symbol'
  | 'password_must_differ'
  | 'password_too_weak'
  | 'password_contains_common';

export interface PasswordCheck {
  id: PasswordCode;
  label: string;
  passed: boolean;
}

/**
 * Score a password's actual entropy from 0 to 100.
 *
 * Components:
 *   - Length (0-40 pts): longer is better, with diminishing returns.
 *   - Character diversity (0-25 pts): lowercase / uppercase / digit / symbol.
 *   - Uniqueness (0-25 pts): uniqueChars / length, scaled. "hhhhhhhh1" gets ~3.
 *   - No obvious patterns (0-10 pts): no 4+ char runs, not in common-passwords.
 *
 * This replaces the old "passes N rules → strong" logic which let
 * `hhhhhhhhhhhhhh1` rate as strong because all 6 binary rules passed.
 */
export function scorePassword(plain: string): number {
  const pwd = plain ?? '';
  if (pwd.length === 0) return 0;

  // 1. Length tier (0-40 pts).
  // 10 chars = 15, 12 = 22, 14 = 28, 16 = 33, 20 = 38, 32+ = 40.
  let lengthScore = 0;
  if (pwd.length >= 10 && pwd.length <= 11) lengthScore = 15;
  else if (pwd.length <= 13) lengthScore = 22;
  else if (pwd.length <= 15) lengthScore = 28;
  else if (pwd.length <= 19) lengthScore = 33;
  else if (pwd.length <= 31) lengthScore = 38;
  else lengthScore = 40; // 32-72

  // 2. Character diversity (0-25 pts): 6 pts per class present, +1 bonus
  // if all 4 classes are present. Catches "all letters" / "all digits".
  let diversityScore = 0;
  const hasLower = /[a-z]/.test(pwd);
  const hasUpper = /[A-Z]/.test(pwd);
  const hasDigit = /\d/.test(pwd);
  const hasSymbol = /[^A-Za-z0-9\s]/.test(pwd);
  if (hasLower) diversityScore += 6;
  if (hasUpper) diversityScore += 6;
  if (hasDigit) diversityScore += 6;
  if (hasSymbol) diversityScore += 6;
  if (hasLower && hasUpper && hasDigit && hasSymbol) diversityScore += 1;

  // 3. Uniqueness (0-25 pts). Penalize runs and low diversity
  // heavily. "hhhhhhhhhhhhhh1" has 2 unique chars out of 15
  // (ratio 0.13) → ~3 pts. We use a quadratic curve so that very
  // low ratios (< 0.3) get crushed: uniqueRatio^1.5 * 25.
  const chars = new Set(pwd);
  const uniqueRatio = chars.size / pwd.length;
  const uniquenessScore = Math.round(Math.pow(uniqueRatio, 1.5) * 25);

  // 4. No obvious patterns (0-10 pts).
  let patternScore = 0;
  // Detect any run of 4+ identical chars (hhhh, 1111, aaaa).
  if (!/(.)\1{3,}/.test(pwd)) patternScore += 5;
  // Not in top-1000 common-passwords (case-insensitive exact match).
  if (!COMMON_PASSWORDS.has(pwd.toLowerCase())) patternScore += 5;

  return lengthScore + diversityScore + uniquenessScore + patternScore;
}

/**
 * Does this password contain a top-1000 common password as a substring?
 * Catches "passwordpassword" (contains "password"), "qwertyqwerty",
 * "adminadmin123" etc. that wouldn't trip the exact-match blocklist.
 *
 * 1000 entries * average word length ~7 chars = ~7000 char comparisons
 * worst case. Fast enough to run on every keystroke.
 */
export function containsCommonSubstring(plain: string): boolean {
  const lower = (plain ?? '').toLowerCase();
  if (lower.length === 0) return false;
  for (const word of COMMON_PASSWORDS) {
    if (word.length >= 4 && lower.includes(word)) {
      return true;
    }
  }
  return false;
}

/**
 * Evaluate a password against every policy rule plus the strength gate.
 * Returns the list of checks (one per rule), the strength classification,
 * the numeric score, and the count of passed checks.
 */
export function evaluatePassword(plain: string): {
  checks: PasswordCheck[];
  strength: 'empty' | 'weak' | 'ok' | 'strong';
  score: number;
  passedCount: number;
  totalCount: number;
} {
  const pwd = plain ?? '';
  const trimmed = pwd.trim();
  const lower = pwd.toLowerCase();
  const score = scorePassword(pwd);

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
      label: 'Not a common password',
      passed: pwd.length === 0 || !COMMON_PASSWORDS.has(lower),
    },
    {
      id: 'password_contains_common',
      label: 'Does not contain a common password',
      passed: pwd.length === 0 || !containsCommonSubstring(pwd),
    },
    {
      id: 'password_needs_letter_and_digit_or_symbol',
      label: 'Has a letter and either a digit or a symbol',
      passed: hasLetter && (hasDigit || hasSymbol),
    },
    {
      id: 'password_too_weak',
      label: 'Strong enough to resist guessing',
      // Only flag this once the password passes the basic length gate —
      // otherwise we double-report on the same problem.
      passed:
        pwd.length >= MIN_LENGTH &&
        pwd.length <= MAX_LENGTH &&
        trimmed.length > 0 &&
        pwd === trimmed &&
        score >= OK_THRESHOLD,
    },
  ];

  const passedCount = checks.filter((c) => c.passed).length;
  const totalCount = checks.length;

  let strength: 'empty' | 'weak' | 'ok' | 'strong' = 'empty';
  if (pwd.length > 0) {
    if (score < OK_THRESHOLD) strength = 'weak';
    else if (score < STRONG_THRESHOLD) strength = 'ok';
    else strength = 'strong';
  }

  return { checks, strength, score, passedCount, totalCount };
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
  if (containsCommonSubstring(pwd)) {
    return {
      code: 'password_contains_common',
      message: 'That password contains a commonly-used word. Try mixing it up.',
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
  const score = scorePassword(pwd);
  if (score < OK_THRESHOLD) {
    return {
      code: 'password_too_weak',
      message:
        'That password is too simple — try mixing in some variety (different characters, no repeats, no obvious patterns).',
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
    case 'password_contains_common':
      return 'That password contains a commonly-used word. Try mixing it up.';
    case 'password_needs_letter_and_digit_or_symbol':
      return 'Add at least one letter and either a digit or a symbol.';
    case 'password_must_differ':
      return 'New password must be different from your current password.';
    case 'password_too_weak':
      return 'That password is too simple — try mixing in some variety (different characters, no repeats, no obvious patterns).';
    default:
      return fallback;
  }
}
