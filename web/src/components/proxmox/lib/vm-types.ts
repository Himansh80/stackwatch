/**
 * Tier 14 Phase 14.6 — VM creation spec (extended with multi-NIC + Cloud-Init + EFI).
 */
export interface Nic {
  bridge: string;
  netModel: 'virtio' | 'e1000' | 'vmxnet3';
  vlanTag: string;
}

export interface VmSpec {
  // Step 1: ISO
  iso: string;
  // Step 2: Hardware
  vmid: number;
  name: string;
  cores: number;
  memory: number;
  disk: number;
  storage: string;
  bios: 'seabios' | 'ovmf';
  machine: 'q35' | 'i440fx';
  cpuType: string;
  osType: 'l26' | 'win11' | 'other';
  cloudInit: boolean;
  efiDisk: boolean;
  tpmState: boolean;
  // Step 3: Network (now multi-NIC)
  nics: Nic[];
  // Step 4: lifecycle
  startAfterCreate: boolean;
}

export const MAX_NICS = 4;

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
  osType: 'l26',
  cloudInit: false,
  efiDisk: false,
  tpmState: false,
  nics: [
    { bridge: 'vmbr0', netModel: 'virtio', vlanTag: '' },
  ],
  startAfterCreate: true,
};