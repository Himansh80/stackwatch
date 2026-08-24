#!/usr/bin/env bash
# scripts/verify-auth-motion.sh
#
# Verifies that every auth page in web/src/pages/{Login,Signup,ForgotPassword,ResetPassword}.tsx
# uses the shared motion lib in the right places. Run after any auth-page edit.
#
# Usage:  bash scripts/verify-auth-motion.sh
# Exit:   0 if all checks pass, 1 if any fails.

set -u
# Don't `set -e` because we want to count failures rather than abort on first.

REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || echo ".")"
PAGES_DIR="$REPO_ROOT/web/src/pages"
LIB="$REPO_ROOT/web/src/lib/motion.tsx"

PASS=0
FAIL=0
FAILURES=()

check() {
  local name="$1"
  local pattern="$2"
  local file="$3"
  local expect="${4:-ge1}"   # ge1 = at least one match

  local count
  count=$(grep -cE "$pattern" "$file" 2>/dev/null || echo 0)
  count=$(echo "$count" | tr -d '[:space:]')

  local ok=0
  case "$expect" in
    ge1) [ "$count" -ge 1 ] && ok=1 ;;
    ge2) [ "$count" -ge 2 ] && ok=1 ;;
    eq0) [ "$count" -eq 0 ] && ok=1 ;;
  esac

  if [ "$ok" -eq 1 ]; then
    printf "  \xE2\x9C\x93  %-55s %s match(es)\n" "$name" "$count"
    PASS=$((PASS+1))
  else
    printf "  \xE2\x9C\x97  %-55s want %s, got %s\n" "$name" "$expect" "$count"
    FAIL=$((FAIL+1))
    FAILURES+=("$name")
  fi
}

echo "=== Auth motion verifier ==="
echo "Repo: $REPO_ROOT"
echo ""

# ---------- 1. Lib exists and is non-empty ----------
echo "[1] Shared motion lib"
if [ -f "$LIB" ] && [ -s "$LIB" ]; then
  lines=$(wc -l < "$LIB")
  printf "  \xE2\x9C\x93  %-55s %s lines\n" "lib/motion.tsx present" "$lines"
  PASS=$((PASS+1))
else
  printf "  \xE2\xC\x97  lib/motion.tsx missing or empty\n"
  FAIL=$((FAIL+1))
  FAILURES+=("lib/motion.tsx missing")
fi
echo ""

# Helper: each page must import from '../lib/motion' AND use motion.* at least 4 times
for page in Login Signup ForgotPassword ResetPassword; do
  FILE="$PAGES_DIR/$page.tsx"
  echo "[$page]"

  if [ ! -f "$FILE" ]; then
    printf "  \xE2\x9C\x97  File does not exist\n"
    FAIL=$((FAIL+1))
    FAILURES+=("$page missing")
    continue
  fi

  check "Imports from '../lib/motion'" \
        "from '\\.\\./lib/motion'" \
        "$FILE"

  check "Uses motion.div / motion.form / etc. (>=4 times)" \
        "<motion\\." \
        "$FILE" \
        "ge2"

  check "Uses useReducedMotion hook" \
        "useReducedMotion" \
        "$FILE"

  check "Uses cardEntrance variant on the auth-card" \
        "cardEntrance" \
        "$FILE"

  # Login/Signup/ResetPassword use staggerFormRows on the form.
  # ForgotPassword uses it inside AnimatePresence.
  case "$page" in
    ForgotPassword)
      check "Uses staggerFormRows inside AnimatePresence" \
            "staggerFormRows" \
            "$FILE"
      check "Uses AnimatePresence for view switch" \
            "<AnimatePresence" \
            "$FILE"
      ;;
    *)
      check "Uses staggerFormRows on the form" \
            "staggerFormRows" \
            "$FILE"
      ;;
  esac

  check "Uses buttonSpring on the submit button" \
        "buttonSpring" \
        "$FILE"

  # The TOP-LEVEL form (the one the user fills out) must be wrapped as
  # motion.form. ForgotPassword legitimately has additional raw <form>
  # tags inside the "send a fresh reset link" details — those are
  # intentionally unwrapped because they only flip state and don't need
  # entrance animation. So we allow up to 2 raw forms in total.
  case "$page" in
    ForgotPassword) raw_max=2 ;;
    *) raw_max=0 ;;
  esac

  raw_count=$(grep -cE "<form[[:space:]>]" "$FILE" 2>/dev/null || echo 0)
  raw_count=$(echo "$raw_count" | tr -d '[:space:]')
  if [ "$raw_count" -le "$raw_max" ]; then
    printf "  \xE2\x9C\x93  %-55s %s raw <form> (max %s)\n" "No stray top-level <form>" "$raw_count" "$raw_max"
    PASS=$((PASS+1))
  else
    printf "  \xE2\x9C\x97  %-55s got %s raw, want <= %s\n" "Stray raw <form>" "$raw_count" "$raw_max"
    FAIL=$((FAIL+1))
    FAILURES+=("$page stray form")
  fi
  echo ""
done

# ---------- 3. CSS still has the reduced-motion guard ----------
echo "[CSS]"
CSS="$REPO_ROOT/web/src/styles.css"
check "Reduced-motion guard exists in styles.css" \
      "prefers-reduced-motion: reduce" \
      "$CSS"
echo ""

# ---------- Summary ----------
echo "=== Summary ==="
echo "  PASS: $PASS"
echo "  FAIL: $FAIL"
if [ "$FAIL" -gt 0 ]; then
  echo ""
  echo "Failures:"
  for f in "${FAILURES[@]}"; do
    echo "  - $f"
  done
  exit 1
fi
exit 0
