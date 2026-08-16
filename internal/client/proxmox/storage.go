package proxmox

import (
	"context"
	"fmt"
	"net/url"
)

// StorageType is the Proxmox storage backend kind.
type StorageType string

const (
	StorageDir    StorageType = "dir"    // directory
	StorageLVM    StorageType = "lvm"    // LVM
	StorageLVMThin StorageType = "lvmthin" // LVM-Thin
	StorageZFS    StorageType = "zfspool" // ZFS pool
	StorageNFS    StorageType = "nfs"    // NFS share
	StorageCIFS   StorageType = "cifs"   // CIFS/SMB share
	StorageCephFS StorageType = "cephfs" // CephFS
	StorageRBD    StorageType = "rbd"    // Ceph RBD
	StoragePBS     StorageType = "pbs"    // Proxmox Backup Server
)

// StorageSpec describes a storage to create on a Proxmox node.
type StorageSpec struct {
	Name        string      `json:"name"`
	StorageType StorageType `json:"type"`
	Content     []string    `json:"content,omitempty"` // ["iso", "vztmpl", "backup", "images", "rootdir", "snippets"]
	Path        string      `json:"path,omitempty"`     // for dir: filesystem path; for zfspool: pool name
	Server      string      `json:"server,omitempty"`   // for NFS/CIFS: hostname
	Export      string      `json:"export,omitempty"`   // for NFS/CIFS: remote path
	Username    string      `json:"username,omitempty"` // for CIFS
	Password    string      `json:"password,omitempty"`
	VGName      string      `json:"vgname,omitempty"`     // for LVM: volume group
	ThinPool    string      `json:"thinpool,omitempty"`   // for LVM-Thin
	Pool        string      `json:"pool,omitempty"`       // for CephFS / RBD
	KRBD        int         `json:"krbd,omitempty"`       // for RBD: 0/1
	Nodes       []string    `json:"nodes,omitempty"`      // restrict to nodes
	Disable     int         `json:"disable,omitempty"`    // 0=enabled, 1=disabled
	PruneBackups string     `json:"prune_backups,omitempty"` // keep-last, etc.
	BackupRetention string  `json:"backup_retention,omitempty"`
}

// StorageEntry is a Proxmox storage entry (from /storage list).
type StorageEntry struct {
	Storage      string  `json:"storage"`
	Type         string  `json:"type"`
	Content      string  `json:"content"`     // comma-separated types
	ContentList  []string `json:"-"`           // parsed from Content
	Path         string  `json:"path,omitempty"`
	Server       string  `json:"server,omitempty"`
	Export       string  `json:"export,omitempty"`
	VGName       string  `json:"vgname,omitempty"`
	ThinPool     string  `json:"thinpool,omitempty"`
	Pool         string  `json:"pool,omitempty"`
	Nodes        string  `json:"nodes,omitempty"` // comma-separated
	Disable      int     `json:"disable,omitempty"`
	Shared       int     `json:"shared,omitempty"`
	Used         int64   `json:"used,omitempty"`
	Avail        int64   `json:"avail,omitempty"`
	Total        int64   `json:"total,omitempty"`
	Active       int     `json:"active,omitempty"`
}

// ListStorageEntries returns all storages (cluster-level).
func (c *Client) ListStorageEntries(ctx context.Context) ([]StorageEntry, error) {
	var out []StorageEntry
	if err := c.get(ctx, "/storage", &out); err != nil {
		return nil, err
	}
	for i := range out {
		if out[i].Content != "" {
			out[i].ContentList = splitCSV(out[i].Content)
		}
	}
	return out, nil
}

// CreateStorage registers a new storage backend. Proxmox returns the created
// storage object directly (no task ID). This is a synchronous operation.
func (c *Client) CreateStorage(ctx context.Context, spec StorageSpec) (string, error) {
	path := "/storage"
	form := url.Values{}
	form.Set("storage", spec.Name)
	form.Set("type", string(spec.StorageType))
	if len(spec.Content) > 0 {
		for _, ct := range spec.Content {
			form.Add("content", ct)
		}
	}
	if spec.Path != "" {
		form.Set("path", spec.Path)
	}
	if spec.Server != "" {
		form.Set("server", spec.Server)
	}
	if spec.Export != "" {
		form.Set("export", spec.Export)
	}
	if spec.Username != "" {
		form.Set("username", spec.Username)
	}
	if spec.Password != "" {
		form.Set("password", spec.Password)
	}
	if spec.VGName != "" {
		form.Set("vgname", spec.VGName)
	}
	if spec.ThinPool != "" {
		form.Set("thinpool", spec.ThinPool)
	}
	if spec.Pool != "" {
		form.Set("pool", spec.Pool)
	}
	if spec.KRBD > 0 {
		form.Set("krbd", "1")
	}
	if len(spec.Nodes) > 0 {
		form.Set("nodes", joinCSV(spec.Nodes))
	}
	if spec.Disable > 0 {
		form.Set("disable", "1")
	}
	if spec.PruneBackups != "" {
		form.Set("prune_backups", spec.PruneBackups)
	}
	if spec.BackupRetention != "" {
		form.Set("backup_retention", spec.BackupRetention)
	}

	// Use synchronous request — POST /storage returns the storage object, not a UPID.
	if err := c.postForm(ctx, path, form, nil); err != nil {
		return "", err
	}
	return "", nil
}

// DeleteStorage removes a storage entry (not the underlying data).
func (c *Client) DeleteStorage(ctx context.Context, name string) (string, error) {
	path := "/storage/" + url.PathEscape(name)
	if err := c.deleteForm(ctx, path); err != nil {
		return "", err
	}
	return "", nil
}

// Content is a volume/template/backup stored in a storage.
type Content struct {
	Content string `json:"content"` // "iso", "vztmpl", "backup", "images", "rootdir"
	VolID   string `json:"volid"`   // e.g. "local:iso/debian-12.iso"
	Format  string `json:"format,omitempty"`
	Size    int64  `json:"size"`
	CTime   int64  `json:"ctime,omitempty"`
	Notes   string `json:"notes,omitempty"`
}

// ListContent returns the volumes/templates/ISOs in a storage on a node.
func (c *Client) ListContent(ctx context.Context, node, storage string) ([]Content, error) {
	path := fmt.Sprintf("/nodes/%s/storage/%s/content",
		url.PathEscape(node), url.PathEscape(storage))
	var out []Content
	if err := c.get(ctx, path, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteContent removes a specific volume/template/ISO from storage.
func (c *Client) DeleteContent(ctx context.Context, node, storage, volume string) (string, error) {
	path := fmt.Sprintf("/nodes/%s/storage/%s/content/%s",
		url.PathEscape(node), url.PathEscape(storage), url.PathEscape(volume))
	if err := c.deleteForm(ctx, path); err != nil {
		return "", err
	}
	return "", nil
}

func splitCSV(s string) []string {
	out := []string{}
	cur := ""
	for _, c := range s {
		if c == ',' {
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
			continue
		}
		cur += string(c)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func joinCSV(s []string) string {
	out := ""
	for i, v := range s {
		if i > 0 {
			out += ","
		}
		out += v
	}
	return out
}
