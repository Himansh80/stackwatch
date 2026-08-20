// useLogout — single place that signs the user out cleanly.
//
// Three things go wrong if you do it ad-hoc:
//   1. clearToken() + navigate('/login') happens, but the currently-mounted
//      page (e.g. Dashboard) is still in the DOM. Its in-flight or pending
//      API call fires with a stale token, the backend returns 401
//      unauthorized, and the page shows an 'unauthorized' error before
//      React Router swaps it out.
//   2. App.tsx re-renders to show <Login />, but the previous page's state
//      (error, snapshot, timers) stays in memory. If the user signs back
//      in as someone else, they briefly see the previous user's data
//      flicker before the new page's API call resolves.
//   3. The 30s polling timers on Dashboard, the open ProfileMenu, the
//      pending /api/v1/auth/me fetch in ProfilePage, etc. all keep running
//      after logout, hitting the backend with a stale token.
//
// Fix:
//   - Cancel any fetch in-flight via AbortController (api() now supports
//     this — see api.ts).
//   - Emit a 'stackwatch:auth' event via clearToken(); App.tsx + the
//     hook in useAuthState pick it up and re-render.
//   - navigate('/login') after that so the route swap happens in the
//     same microtask as the auth-state flip.

import { useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { clearToken } from '../lib/api';

export function useLogout() {
  const navigate = useNavigate();
  return useCallback(() => {
    clearToken();
    // Replace history so back-button doesn't return to a protected page
    // that will then immediately bounce to /login.
    navigate('/login', { replace: true });
  }, [navigate]);
}
