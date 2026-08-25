/**
 * Shared types for ServiceStatusWidget — split out so
 * ServiceStatusWidget.tsx stays under the 400-LOC cap.
 *
 * Importing from `'./types'` keeps the import graph shallow and
 * avoids the (currently unused but future-friendly) `react` import
 * here — every consumer pulls these shapes via type-only imports.
 */

export type ServiceStatus = 'up' | 'degraded' | 'down' | 'unknown';

export type ServiceKind = 'http' | 'https' | 'tcp' | 'icmp';

export interface ServiceHealth {
  status: ServiceStatus;
  latency_ms?: number | null;
  status_code?: number | null;
  error_message?: string | null;
  checked_at: string;
}

export interface PinnedService {
  id: string;
  name: string;
  url: string;
  kind: ServiceKind;
  icon?: string | null;
  enabled: boolean;
  created_at: string;
  updated_at: string;
  health?: ServiceHealth | null;
}

export interface ServicesResponse {
  services: PinnedService[];
  count: number;
}

export interface AggregateResponse {
  up: number;
  degraded: number;
  down: number;
  unknown: number;
  total: number;
}

export interface PinFormState {
  name: string;
  url: string;
  kind: ServiceKind;
  icon: string;
}

// Re-exported under a `Status` alias for places that read the
// shape without wanting to type out "ServiceStatus" everywhere.
export type Status = ServiceStatus;