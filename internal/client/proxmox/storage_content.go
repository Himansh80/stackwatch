package proxmox

import (
	"context"
	"fmt"
	"net/url"
)

// DownloadStorageFile returns the URL to download a file from Proxmox storage.
// Caller should then GET this URL (with auth header) to actually download the file.
// Volume format: "storage:volid" or just "volid" (uses default storage).
func (c *Client) DownloadStorageFile(node, storage, volume string) string {
	return fmt.Sprintf("%s/api2/json/nodes/%s/storage/%s/download/%s",
		c.baseURL,
		url.PathEscape(node),
		url.PathEscape(storage),
		url.PathEscape(volume))
}

// UploadStorageFileToURL returns the URL to POST a file upload to.
// Caller must POST multipart/form-data with field "content" containing the file.
func (c *Client) UploadStorageFileToURL(node, storage string) string {
	return fmt.Sprintf("%s/api2/json/nodes/%s/storage/%s/upload",
		c.baseURL,
		url.PathEscape(node),
		url.PathEscape(storage))
}

// ISCSITarget describes an iSCSI target configuration.
type ISCSITarget struct {
	ID      string `json:"id"`               // target name (e.g. "iqn.2003-01.org.linux-iscsi...")
	Portal  string `json:"portal,omitempty"` // portal address
	Target  string `json:"target,omitempty"`
	Lun     int    `json:"lun,omitempty"`
	Storage string `json:"storage,omitempty"`
}

// ListISCSI returns iSCSI targets on the node.
func (c *Client) ListISCSI(ctx context.Context, node string) ([]ISCSITarget, error) {
	path := fmt.Sprintf("/nodes/%s/iscsi", url.PathEscape(node))
	var out []ISCSITarget
	if err := c.get(ctx, path, &out); err != nil {
		return nil, err
	}
	return out, nil
}