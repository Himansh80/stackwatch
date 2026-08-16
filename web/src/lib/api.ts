// API client + token storage.
const TOKEN_KEY = 'stackwatch.token';

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY);
}

export function setToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token);
}

export function clearToken(): void {
  localStorage.removeItem(TOKEN_KEY);
}

export function isLoggedIn(): boolean {
  return !!getToken();
}

export interface LoginResponse {
  token: string;
  user: { id: string; email: string; full_name: string; role: string };
  tenant: { id: string; name: string; slug: string; plan: string };
}

export async function api<T = any>(
  method: string,
  path: string,
  body?: any,
  auth = true,
): Promise<T> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  if (auth) {
    const tok = getToken();
    if (tok) headers['Authorization'] = `Bearer ${tok}`;
  }
  const res = await fetch(path, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`${res.status}: ${text}`);
  }
  return res.json();
}

export async function login(email: string, password: string): Promise<LoginResponse> {
  return api<LoginResponse>('POST', '/api/v1/auth/login', { email, password }, false);
}

export async function me(): Promise<{ user: any; tenant: any }> {
  return api('GET', '/api/v1/auth/me');
}

export async function health(): Promise<{ status: string; db: string; version: string }> {
  return api('GET', '/health', undefined, false);
}
