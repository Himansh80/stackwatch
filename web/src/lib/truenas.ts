import { api } from './api';

export type TNHost = {
  id: string;
  name: string;
  base_url: string;
  username?: string;
  verify_tls?: boolean;
  status?: string;
  last_error?: string;
};

export type TNRow = Record<string, unknown>;

export async function listTrueNASHosts(): Promise<TNHost[]> {
  const payload = await api<{ hosts?: TNHost[] }>('GET', '/api/v1/truenas/hosts');
  return payload.hosts ?? [];
}

export async function createTrueNASHost(input: Record<string, unknown>): Promise<TNHost> {
  const payload = await api<{ host: TNHost }>('POST', '/api/v1/truenas/hosts', input);
  return payload.host;
}

export async function deleteTrueNASHost(id: string): Promise<void> {
  await api('DELETE', `/api/v1/truenas/hosts/${encodeURIComponent(id)}`);
}

export async function testTrueNASHost(id: string): Promise<void> {
  await api('POST', `/api/v1/truenas/hosts/${encodeURIComponent(id)}/test`);
}

export async function truenasCall<T = unknown>(path: string, body: Record<string, unknown>, method = 'POST'): Promise<T> {
  return api<T>(method, `/api/v1/truenas${path}`, body);
}
