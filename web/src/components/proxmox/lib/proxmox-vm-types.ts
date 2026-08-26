/**
 * Tier 14 Phase 14.2 — Shared types for VM detail components.
 */

export interface VMNetworkInterface {
  name: string;
  hardware_addr?: string;
  ip_addresses?: Array<{
    ip_address: string;
    ip_address_type: string;
    prefix: number;
  }>;
  mtu?: number;
  stats?: {
    rx_bytes?: number;
    tx_bytes?: number;
  };
}

export interface NetworkInterfacesResponse {
  interfaces: VMNetworkInterface[] | null;
  agent_running: boolean;
}