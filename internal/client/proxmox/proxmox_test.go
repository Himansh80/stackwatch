package proxmox

import (
	"context"
	"errors"
	"testing"
)

func TestNewClient_Ping(t *testing.T) {
	// Real Proxmox at .107
	c := NewClient("https://192.168.0.107:8006",
		"PVEAPIToken=root@pam!monitor=613a1a19-718c-4534-93ad-9efb679ffb58",
		false)
	if err := c.Ping(context.Background()); err != nil {
		t.Fatalf("ping: %v", err)
	}
}

func TestNewClient_ListNodes(t *testing.T) {
	c := NewClient("https://192.168.0.107:8006",
		"PVEAPIToken=root@pam!monitor=613a1a19-718c-4534-93ad-9efb679ffb58",
		false)
	nodes, err := c.ListNodes(context.Background())
	if err != nil {
		t.Fatalf("list nodes: %v", err)
	}
	if len(nodes) == 0 {
		t.Fatal("expected at least 1 node")
	}
	t.Logf("found %d node(s):", len(nodes))
	for _, n := range nodes {
		t.Logf("  %s status=%s cpu=%v mem=%d/%d disk=%d/%d",
			n.Node, n.Status, n.CPU, n.Mem, n.MaxMem, n.Disk, n.MaxDisk)
	}
}

func TestNewClient_ListVMs(t *testing.T) {
	c := NewClient("https://192.168.0.107:8006",
		"PVEAPIToken=root@pam!monitor=613a1a19-718c-4534-93ad-9efb679ffb58",
		false)
	vms, err := c.ListVMs(context.Background())
	if err != nil {
		t.Fatalf("list vms: %v", err)
	}
	t.Logf("found %d vm/lxc:", len(vms))
	for _, v := range vms {
		t.Logf("  %s id=%s name=%s status=%s cpu=%v mem=%d/%d",
			v.Type, v.ID, v.Name, v.Status, v.CPU, v.Mem, v.MaxMem)
	}
}

func TestNewClient_BadToken(t *testing.T) {
	c := NewClient("https://192.168.0.107:8006",
		"PVEAPIToken=root@pam!monitor=bad-token",
		false)
	err := c.Ping(context.Background())
	if err == nil {
		t.Fatal("expected error with bad token")
	}
	if !errors.Is(err, err) { // any error
		// not a typed error, just check it's not nil
	}
	t.Logf("got expected error: %v", err)
}
