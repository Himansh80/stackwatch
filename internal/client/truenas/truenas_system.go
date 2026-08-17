package truenas

// SystemInfo is system.info result (mapped map).
type SystemInfo map[string]any

// GetSystemInfo returns host, uptime, cores, model, license etc.
func (c *Client) GetSystemInfo() (SystemInfo, error) {
	res, err := c.Call("system.info", []any{})
	if err != nil {
		return nil, err
	}
	return SystemInfo(res.(map[string]any)), nil
}

// Hostname returns just the hostname.
func (c *Client) Hostname() (string, error) {
	info, err := c.GetSystemInfo()
	if err != nil {
		return "", err
	}
	if h, ok := info["hostname"].(string); ok {
		return h, nil
	}
	return "", nil
}

// ProductName returns the system manufacturer + product (best-effort).
func (c *Client) ProductName() (string, error) {
	info, err := c.GetSystemInfo()
	if err != nil {
		return "", err
	}
	mfg, _ := info["system_manufacturer"].(string)
	product, _ := info["system_product"].(string)
	if mfg == "" {
		mfg = "Unknown"
	}
	return mfg + " " + product, nil
}

// -- Boot environment --

// BootEnvironment is one boot slot (A | B).
type BootEnvironment map[string]any

// ListBootEnvironments returns all boot slots.
func (c *Client) ListBootEnvironments() ([]BootEnvironment, error) {
	res, err := c.Query("boot.environment.query", nil)
	if err != nil {
		return nil, err
	}
	out := make([]BootEnvironment, len(res))
	for i, r := range res {
		out[i] = BootEnvironment(r.(map[string]any))
	}
	return out, nil
}

// ActivateBootEnvironment marks a slot as the next boot target.
func (c *Client) ActivateBootEnvironment(name string) error {
	_, err := c.Call("boot.environment.activate", []any{name})
	return err
}

// DestroyBootEnvironment removes a slot (active cannot be destroyed).
func (c *Client) DestroyBootEnvironment(name string) error {
	_, err := c.Call("boot.environment.destroy", []any{name})
	return err
}

// CloneBootEnvironment duplicates the running slot into a new one.
func (c *Client) CloneBootEnvironment(srcName, dstName string) error {
	args := map[string]any{"name": dstName, "source": srcName}
	_, err := c.Call("boot.environment.clone", []any{args})
	return err
}

// -- Updates --

// UpdateStatus is update.status result.
type UpdateStatus map[string]any

// CheckUpdate returns whether a new version is available.
func (c *Client) CheckUpdate() (UpdateStatus, error) {
	res, err := c.Call("update.status", []any{})
	if err != nil {
		return nil, err
	}
	return UpdateStatus(res.(map[string]any)), nil
}

// ApplyUpdate starts an upgrade to the desired train.
func (c *Client) ApplyUpdate(train string, reboot bool) error {
	opts := map[string]any{
		"train_name": train,
		"reboot":     reboot,
	}
	_, err := c.Call("update.update", []any{opts})
	return err
}

// DownloadUpdate downloads (does not apply) the new version.
func (c *Client) DownloadUpdate(train string) error {
	opts := map[string]any{"train_name": train}
	_, err := c.Call("update.download", []any{opts})
	return err
}

// -- Services --

// Service is one row from service.query.
type Service map[string]any

// ListServices returns all services (smbd, nfsd, sshd, etc).
func (c *Client) ListServices() ([]Service, error) {
	res, err := c.Query("service.query", nil)
	if err != nil {
		return nil, err
	}
	out := make([]Service, len(res))
	for i, r := range res {
		out[i] = Service(r.(map[string]any))
	}
	return out, nil
}

// StartService starts a service.
func (c *Client) StartService(name string) error {
	opts := map[string]any{"service": name}
	_, err := c.Call("service.start", []any{opts})
	return err
}

// StopService stops a service.
func (c *Client) StopService(name string) error {
	opts := map[string]any{"service": name}
	_, err := c.Call("service.stop", []any{opts})
	return err
}

// RestartService restarts a service.
func (c *Client) RestartService(name string) error {
	opts := map[string]any{"service": name}
	_, err := c.Call("service.restart", []any{opts})
	return err
}

// ReloadService reloads a service config without restarting.
func (c *Client) ReloadService(name string) error {
	opts := map[string]any{"service": name}
	_, err := c.Call("service.reload", []any{opts})
	return err
}

// UpdateService toggles autostart or other props.
type ServiceUpdate struct {
	Enable bool `json:"enable,omitempty"`
}

// UpdateService changes service properties.
func (c *Client) UpdateService(name string, u ServiceUpdate) error {
	_, err := c.Call("service.update", []any{name, u})
	return err
}
