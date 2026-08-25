/**
 * Tier 13 Phase 3 — Typed REST client for api-gateway.
 * Reads BACKEND_URL from expo-constants; falls back to dev URL.
 */
import Constants from 'expo-constants';

const BACKEND_URL =
  (Constants.expoConfig?.extra as { backendUrl?: string } | undefined)?.backendUrl ??
  'http://192.168.0.115:8080';

let authToken: string | null = null;

export function setAuthToken(t: string | null) {
  authToken = t;
}

export interface ApiError {
  error: string;
  code?: string;
  status: number;
}

export class ApiClientError extends Error {
  constructor(public readonly error: ApiError) {
    super(error.error);
  }
}

interface FetchOpts {
  method?: 'GET' | 'POST' | 'PATCH' | 'DELETE';
  body?: unknown;
  signal?: AbortSignal;
}

export async function apiFetch<T = unknown>(path: string, opts: FetchOpts = {}): Promise<T> {
  const { method = 'GET', body, signal } = opts;
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  if (authToken) headers.Authorization = `Bearer ${authToken}`;

  const res = await fetch(`${BACKEND_URL}${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
    signal,
  });

  const text = await res.text();
  let parsed: unknown;
  try {
    parsed = text ? JSON.parse(text) : null;
  } catch {
    parsed = { error: text || res.statusText };
  }

  if (!res.ok) {
    throw new ApiClientError({ ...(parsed as ApiError), status: res.status });
  }
  return parsed as T;
}

// Convenience helpers per endpoint shape.
export const mobile = {
  login: (email: string, password: string) =>
    apiFetch<{ token: string; user: { id: string; email: string; full_name: string } }>(
      '/api/v1/auth/login',
      { method: 'POST', body: { email, password } },
    ),
  registerPush: (body: { platform: 'android' | 'ios'; fcm_token: string; app_version?: string; device_name?: string }) =>
    apiFetch('/api/v1/mobile/push/register', { method: 'POST', body }),
  pushDevices: () => apiFetch<{ devices: unknown[] }>('/api/v1/mobile/push/devices'),
  pushLog: () => apiFetch<{ log: unknown[] }>('/api/v1/mobile/push/log'),
  devices: () => apiFetch<{ devices: unknown[] }>('/api/v1/mobile/devices'),
  sessions: () => apiFetch<{ sessions: unknown[] }>('/api/v1/mobile/sessions'),
  testPush: (body: { title: string; body?: string }) =>
    apiFetch('/api/v1/mobile/test-push', { method: 'POST', body }),
};
