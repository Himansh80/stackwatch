package proxmox

import (
	"context"
	"fmt"
	"net/url"
)

// HAResource is a Proxmox HA-managed resource.
type HAResource map[string]any

// RRDPoint is a time-series sample returned by Proxmox RRD endpoints.
type RRDPoint map[string]any

func (c *Client) ListTemplates(ctx context.Context, node, storage string) ([]Content, error) {
	return c.ListContent(ctx, node, storage, "vztmpl")
}

func (c *Client) MarkVMTemplate(ctx context.Context, node string, vmid int, kind string) (string, error) {
	path := fmt.Sprintf("/nodes/%s/%s/%d/template", url.PathEscape(node), url.PathEscape(kind), vmid)
	var out any
	if err := c.postForm(ctx, path, url.Values{}, &out); err != nil {
		return "", err
	}
	return "", nil
}

func (c *Client) ApplyCloudInit(ctx context.Context, node, kind string, vmid int, fields map[string]string) error {
	path := fmt.Sprintf("/nodes/%s/%s/%d/config", url.PathEscape(node), url.PathEscape(kind), vmid)
	form := url.Values{}
	for k, v := range fields {
		if v != "" {
			form.Set(k, v)
		}
	}
	return c.putForm(ctx, path, form, nil)
}

func (c *Client) GetHAResources(ctx context.Context) ([]HAResource, error) {
	var out []HAResource
	if err := c.get(ctx, "/cluster/ha/resources", &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetHAStatus(ctx context.Context) (any, error) {
	var out any
	if err := c.get(ctx, "/cluster/ha/status/current", &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) JoinCluster(ctx context.Context, fields map[string]string) error {
	form := url.Values{}
	for k, v := range fields {
		if v != "" {
			form.Set(k, v)
		}
	}
	return c.postForm(ctx, "/cluster/config/join", form, nil)
}

func (c *Client) LeaveCluster(ctx context.Context) error {
	return c.deleteForm(ctx, "/cluster/config/totem")
}

func (c *Client) Migrate(ctx context.Context, node, kind string, vmid int, target string, online bool) (string, error) {
	path := fmt.Sprintf("/nodes/%s/%s/%d/migrate", url.PathEscape(node), url.PathEscape(kind), vmid)
	form := url.Values{"target": []string{target}}
	if online {
		form.Set("online", "1")
	}
	// migrate returns a UPID, so use the task request directly.
	u := c.baseURL + "/api2/json" + path
	req, err := newFormRequestWithContext(ctx, "POST", u, form, c.apiToken)
	if err != nil {
		return "", err
	}
	return c.doTask(req)
}

func (c *Client) HostRRD(ctx context.Context, node, timeframe, cf string) ([]RRDPoint, error) {
	return c.rrd(ctx, fmt.Sprintf("/nodes/%s/rrddata", url.PathEscape(node)), timeframe, cf)
}

func (c *Client) ResourceRRD(ctx context.Context, node, kind string, vmid int, timeframe, cf string) ([]RRDPoint, error) {
	path := fmt.Sprintf("/nodes/%s/%s/%d/rrddata", url.PathEscape(node), url.PathEscape(kind), vmid)
	return c.rrd(ctx, path, timeframe, cf)
}

func (c *Client) rrd(ctx context.Context, path, timeframe, cf string) ([]RRDPoint, error) {
	if timeframe == "" {
		timeframe = "hour"
	}
	if cf == "" {
		cf = "AVERAGE"
	}
	path += "?timeframe=" + url.QueryEscape(timeframe) + "&cf=" + url.QueryEscape(cf)
	var out []RRDPoint
	if err := c.get(ctx, path, &out); err != nil {
		return nil, err
	}
	return out, nil
}
