import { api } from './api';

export type JsonObject = Record<string, unknown>;

export interface ProxmoxHost {
  id: string;
  name: string;
  base_url: string;
  verify_tls: boolean;
  node_name?: string;
  status?: string;
  last_check_at?: string;
  last_error?: string;
  created_at?: string;
}

export interface ProxmoxNode extends JsonObject {
  node?: string;
  status?: string;
  uptime?: number;
  maxcpu?: number;
  maxmem?: number;
  mem?: number;
  cpu?: number;
  disk?: number;
  maxdisk?: number;
}

export interface ProxmoxResource extends JsonObject {
  id?: string;
  type?: string;
  node?: string;
  vmid?: number;
  name?: string;
  status?: string;
  cpu?: number;
  mem?: number;
  maxmem?: number;
  disk?: number;
  maxdisk?: number;
  uptime?: number;
}

export function hostPath(hostId: string, suffix = ''): string {
  return `/api/v1/proxmox/hosts/${encodeURIComponent(hostId)}${suffix}`;
}

export function pxGet<T = JsonObject>(hostId: string, suffix: string): Promise<T> {
  return api<T>('GET', hostPath(hostId, suffix));
}

export function pxPost<T = JsonObject>(hostId: string, suffix: string, body?: unknown): Promise<T> {
  return api<T>('POST', hostPath(hostId, suffix), body ?? {});
}

export function pxPut<T = JsonObject>(hostId: string, suffix: string, body?: unknown): Promise<T> {
  return api<T>('PUT', hostPath(hostId, suffix), body ?? {});
}

export function pxDelete<T = JsonObject>(hostId: string, suffix: string, body?: unknown): Promise<T> {
  return api<T>('DELETE', hostPath(hostId, suffix), body);
}

export function listFrom(payload: unknown, ...keys: string[]): JsonObject[] {
  if (Array.isArray(payload)) return payload.filter(isObject);
  if (!isObject(payload)) return [];
  for (const key of keys) {
    const value = payload[key];
    if (Array.isArray(value)) return value.filter(isObject);
  }
  if (isObject(payload.data)) {
    return listFrom(payload.data, ...keys);
  }
  return [];
}

export function objectFrom(payload: unknown): JsonObject {
  if (!isObject(payload)) return {};
  if (isObject(payload.data)) return payload.data;
  return payload;
}

export function isObject(value: unknown): value is JsonObject {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

export function displayValue(value: unknown): string {
  if (value === null || value === undefined || value === '') return '—';
  if (typeof value === 'object') return JSON.stringify(value);
  return String(value);
}

export function formatBytes(value: unknown): string {
  const n = Number(value);
  if (!Number.isFinite(n) || n <= 0) return '—';
  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
  let amount = n;
  let unit = 0;
  while (amount >= 1024 && unit < units.length - 1) {
    amount /= 1024;
    unit += 1;
  }
  return `${amount.toFixed(amount >= 10 || unit === 0 ? 0 : 1)} ${units[unit]}`;
}

export function formatPercent(value: unknown): string {
  const n = Number(value);
  if (!Number.isFinite(n)) return '—';
  return `${(n * 100).toFixed(1)}%`;
}

export function formatDate(value: unknown): string {
  if (!value) return '—';
  const date = new Date(String(value));
  return Number.isNaN(date.getTime()) ? String(value) : date.toLocaleString();
}

/**
 * formatUptime — convert Proxmox uptime (seconds, number) into a human
 * string like "3d 4h", "5m", "12s". Returns "—" for missing/zero.
 */
export function formatUptime(value: unknown): string {
  const n = Number(value);
  if (!Number.isFinite(n) || n <= 0) return '—';
  const days = Math.floor(n / 86400);
  const hours = Math.floor((n % 86400) / 3600);
  const minutes = Math.floor((n % 3600) / 60);
  if (days > 0) return `${days}d ${hours}h`;
  if (hours > 0) return `${hours}h ${minutes}m`;
  if (minutes > 0) return `${minutes}m`;
  return `${Math.floor(n)}s`;
}

/**
 * readString — coerce any unknown value to a trimmed string.
 * Returns '' for null/undefined.
 */
export function readString(value: unknown): string {
  if (value === undefined || value === null) return '';
  return String(value).trim();
}
