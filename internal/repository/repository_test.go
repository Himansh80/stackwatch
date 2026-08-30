package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestMockServerLifecycle checks the mock repo boilerplate.
func TestMockServerLifecycle(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()
	tid := uuid.New()

	// create
	s, err := repo.CreateServer(ctx, tid, CreateServerInput{
		Name:     "Test",
		Hostname: "test-host",
	})
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "Test" {
		t.Fatalf("wrong name: %q", s.Name)
	}

	// get
	got, err := repo.GetServer(ctx, tid, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Hostname != "test-host" {
		t.Fatalf("wrong hostname: %q", got.Hostname)
	}

	// lookup by hostname
	gotByHost, err := repo.GetServerByHostname(ctx, "test-host")
	if err != nil {
		t.Fatal(err)
	}
	if gotByHost.ID != s.ID {
		t.Fatal("hostname lookup returned different server")
	}

	// update
	newName := "Renamed"
	res, err := repo.UpdateServer(ctx, tid, s.ID, UpdateServerInput{Name: &newName})
	if err != nil {
		t.Fatal(err)
	}
	if res != 1 {
		t.Fatalf("expected 1 row affected, got %d", res)
	}

	// list
	lst, err := repo.ListServers(ctx, tid)
	if err != nil {
		t.Fatal(err)
	}
	if len(lst) != 1 {
		t.Fatalf("expected 1 server, got %d", len(lst))
	}

	// delete
	res, err = repo.DeleteServer(ctx, tid, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if res != 1 {
		t.Fatal("expected delete to affect 1 row")
	}
}

// TestMockIngestHeartbeat pieces are on the caller side; this just proves
// mock + auto-register path executes without crashing.
func TestMockIngestAutoRegister(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()

	s, err := repo.AutoRegisterServer(ctx, "new-device")
	if err != nil {
		t.Fatal(err)
	}
	if s.Hostname != "new-device" {
		t.Fatal("hostname not preserved")
	}

	// Insert one metric via QueryMetrics hitting 0 rows
	_ = s
	start := time.Now().Add(-1 * time.Minute)
	end := time.Now().Add(10 * time.Second)
	points, err := repo.QueryMetrics(ctx, s.TenantID, []uuid.UUID{s.ID}, []string{"cpu.usage"}, start, end, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 0 {
		t.Fatalf("expected 0 points, got %d", len(points))
	}
}
