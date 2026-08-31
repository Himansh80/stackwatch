export type Section = { id: string; label: string; path: string };

export const sections: Section[] = [
  { id: 'overview', label: 'Overview', path: '/system/info' },
  { id: 'pools', label: 'ZFS Pools', path: '/pools/list' },
  { id: 'datasets', label: 'Datasets', path: '/datasets/list' },
  { id: 'nfs', label: 'NFS Shares', path: '/nfs/list' },
  { id: 'smb', label: 'SMB Shares', path: '/smb/list' },
  { id: 'iscsi', label: 'iSCSI', path: '/iscsi/extents/list' },
  { id: 'snapshots', label: 'Snapshots', path: '/snapshots/list' },
  { id: 'disks', label: 'Disk Health', path: '/disks/list' },
  { id: 'users', label: 'Users & Groups', path: '/users/list' },
  { id: 'system', label: 'System Services', path: '/system/services/list' },
  { id: 'cloud', label: 'Cloud Sync', path: '/cloud/sync/list' },
];

export const actionPaths: Record<string, string> = {
  pools: '/pools/create', datasets: '/datasets/create', nfs: '/nfs/create', smb: '/smb/create',
  snapshots: '/snapshots/create', users: '/users/create', cloud: '/cloud/sync/create',
};