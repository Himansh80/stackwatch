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

export class ApiError extends Error {
  status: number;
  code: string;
  body: unknown;

  constructor(status: number, code: string, message: string, body: unknown) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
    this.body = body;
  }

  /** Human-readable message suitable for showing in a UI. */
  get friendlyMessage(): string {
    switch (this.status) {
      case 400:
        return 'Please double-check the form fields and try again.';
      case 401:
        return 'Wrong email or password.';
      case 403:
        if (this.code === 'must_change_password') {
          return 'You must change your password before you can sign in.';
        }
        return 'You do not have permission to do that.';
      case 404:
        return 'We could not find that resource.';
      case 409:
        if (this.code === 'unique_violation' || this.code === 'conflict') {
          return 'An account with that email already exists.';
        }
        return 'That action conflicts with the current state.';
      case 422:
        return 'Please fill in every field correctly.';
      case 429:
        return 'Too many attempts. Please wait a few minutes and try again.';
      case 502:
      case 503:
      case 504:
        return 'The control plane is temporarily unavailable. Please try again in a moment.';
      default:
        if (this.status >= 500) {
          return 'The control plane hit an unexpected error. Please try again.';
        }
        return this.message || 'Something went wrong.';
    }
  }
}

function parseBody(text: string): { code: string; message: string; body: unknown } {
  if (!text) {
    return { code: 'unknown', message: 'Empty response from server.', body: null };
  }
  try {
    const body = JSON.parse(text);
    if (body && typeof body === 'object') {
      const code = typeof body.code === 'string' ? body.code : 'unknown';
      const message = typeof body.error === 'string'
        ? body.error
        : typeof body.message === 'string'
          ? body.message
          : 'Request failed.';
      return { code, message, body };
    }
    return { code: 'unknown', message: String(body), body };
  } catch {
    // Non-JSON body: surface a trimmed snippet, never the full HTML.
    const snippet = text.length > 200 ? `${text.slice(0, 200)}…` : text;
    return { code: 'unknown', message: snippet, body: text };
  }
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
  let res: Response;
  try {
    res = await fetch(path, {
      method,
      headers,
      body: body ? JSON.stringify(body) : undefined,
    });
  } catch (cause: any) {
    throw new ApiError(0, 'network_error', cause?.message || 'Network error.', null);
  }
  const text = await res.text();
  const parsed = parseBody(text);
  if (!res.ok) {
    throw new ApiError(res.status, parsed.code, parsed.message, parsed.body);
  }
  return parsed.body as T;
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
