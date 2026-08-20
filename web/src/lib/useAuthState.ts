// Hook: react to token changes so route guards update immediately.
//
// Problem: <App/> used `isLoggedIn()` which only reads localStorage at
// render time. When Login calls setToken() then nav('/dashboard') in
// the same tick, React doesn't re-evaluate the App's <Routes> until the
// next state change. So the user gets bounced back to /login even
// though the token was just written.
//
// Fix: subscribe to a custom 'stackwatch:auth' event that setToken /
// clearToken fire whenever the localStorage key changes. App re-renders
// immediately and /dashboard sees isLoggedIn() === true.

import { useEffect, useState } from 'react';
import { getToken } from '../lib/api';

const AUTH_EVENT = 'stackwatch:auth';

export function useAuthState(): { isLoggedIn: boolean; token: string | null } {
  const [token, setLocalToken] = useState<string | null>(() => getToken());

  useEffect(() => {
    function onChange() {
      setLocalToken(getToken());
    }
    window.addEventListener(AUTH_EVENT, onChange);
    // 'storage' fires for cross-tab changes; harmless here.
    window.addEventListener('storage', onChange);
    return () => {
      window.removeEventListener(AUTH_EVENT, onChange);
      window.removeEventListener('storage', onChange);
    };
  }, []);

  return { isLoggedIn: !!token, token };
}
