/**
 * Tier 14 Phase 14.4 — LXC creation spec (shared by all 4 wizard steps).
 */
export interface LxcSpec {
  // Step 1
  ostemplate: string;
  // Step 2
  vmid: number;
  hostname: string;
  cores: number;
  memory: number; // MB
  swap: number; // MB
  disk: number; // GB
  storage: string; // root storage name
  // Step 3
  bridge: string;
  ipv4Mode: 'dhcp' | 'static' | 'none';
  ipv4Addr: string;
  ipv4Gw: string;
  ipv6Mode: 'dhcp' | 'slaac' | 'static' | 'none';
  ipv6Addr: string;
  ipv6Gw: string;
  vlanTag: string;
  // Step 4 (start on create)
  startOnBoot: boolean;
}

export const DEFAULT_LXC_SPEC: LxcSpec = {
  ostemplate: '',
  vmid: 100,
  hostname: '',
  cores: 2,
  memory: 1024,
  swap: 0,
  disk: 8,
  storage: '',
  bridge: 'vmbr0',
  ipv4Mode: 'dhcp',
  ipv4Addr: '',
  ipv4Gw: '',
  ipv6Mode: 'none',
  ipv6Addr: '',
  ipv6Gw: '',
  vlanTag: '',
  startOnBoot: false,
};