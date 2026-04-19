package heartbeat

import (
	"clustercron/internal/broker"
	"context"
	"os"
	"testing"
	"time"
)

func getTestRedis(t *testing.T) *broker.Redis {
	t.Helper()
	url := os.Getenv("REDIS_URL")
	if url == "" {
		t.Skip("REDIS_URL not set, skipping integration test")
	}
	rdb, err := broker.NewRedis(context.Background(), url)
	if err != nil {
		t.Fatalf("connect redis: %v", err)
	}
	t.Cleanup(func() { rdb.Close() })
	return rdb
}

func TestHeartbeat_RegisterAndDiscover(t *testing.T) {
	rdb := getTestRedis(t)
	ctx, cancel := context.WithCancel(context.Background())

	// Start two heartbeat monitors.
	m1 := NewMonitor(rdb, "test-node-1")
	m2 := NewMonitor(rdb, "test-node-2")

	go m1.Run(ctx)
	go m2.Run(ctx)

	// Give them time to write their first heartbeat.
	time.Sleep(2 * time.Second)

	// Both should be visible.
	workers, err := m1.LiveWorkers(context.Background())
	if err != nil {
		t.Fatalf("list workers: %v", err)
	}

	found := map[string]bool{}
	for _, w := range workers {
		found[w] = true
	}

	if !found["test-node-1"] || !found["test-node-2"] {
		t.Errorf("expected both test nodes, got: %v", workers)
	}

	// Stop both — heartbeats should be cleaned up.
	cancel()
	time.Sleep(1 * time.Second)

	// After cleanup, neither should be visible.
	workers, err = m1.LiveWorkers(context.Background())
	if err != nil {
		t.Fatalf("list workers after stop: %v", err)
	}

	for _, w := range workers {
		if w == "test-node-1" || w == "test-node-2" {
			t.Errorf("node %s should be gone after shutdown, but still visible", w)
		}
	}
}
