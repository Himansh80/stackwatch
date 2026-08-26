/**
 * Tier 14 Phase 14.5 — VM creation spec (shared by all 4 wizard steps).
 */
export interface VmSpec {
  // Step 1: ISO
  iso: string; // volid OR empty for "no media"
  // Step 2: Hardware
  vmid: number;
  name: string;
  cores: number;
  memory: number; // MB
  disk: number; // GB
  storage: string; // root disk storage name
  bios: 'seabios' | 'ovmf';
  machine: 'q35' | 'i440fx';
  cpuType: string;
  // Step 3: Network
  bridge: string;
  netModel: 'virtio' | 'e1000' | 'vmxnet3';
  vlanTag: string;
  // Step 4: lifecycle
  startAfterCreate: boolean;
}

export const DEFAULT_VM_SPEC: VmSpec = {
  iso: '',
  vmid: 100,
  name: '',
  cores: 2,
  memory: 2048,
  disk: 32,
  storage: '',
  bios: 'seabios',
  machine: 'q35',
  cpuType: 'host',
  bridge: 'vmbr0',
  netModel: 'virtio',
  vlanTag: '',
  startAfterCreate: true,
};