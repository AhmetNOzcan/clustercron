package hashring

import (
	"fmt"
	"math"
	"testing"
)

func TestGetWorker_EmptyRing(t *testing.T) {
	r := New()
	if got := r.GetWorker("any-key"); got != "" {
		t.Errorf("expected empty string from empty ring, got %q", got)
	}
}

func TestGetWorker_SingleWorker(t *testing.T) {
	r := NewWithMembers([]string{"node-1"})

	// Every key should map to the only worker.
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("job-%d", i)
		if got := r.GetWorker(key); got != "node-1" {
			t.Errorf("key %s: expected node-1, got %s", key, got)
		}
	}
}

func TestGetWorker_Deterministic(t *testing.T) {
	r := NewWithMembers([]string{"node-1", "node-2", "node-3"})

	// Same key always maps to the same worker.
	first := r.GetWorker("job-abc")
	for i := 0; i < 100; i++ {
		if got := r.GetWorker("job-abc"); got != first {
			t.Fatalf("iteration %d: got %s, expected %s", i, got, first)
		}
	}
}

func TestGetWorker_Distribution(t *testing.T) {
	r := NewWithMembers([]string{"node-1", "node-2", "node-3"})

	counts := map[string]int{}
	total := 10000

	for i := 0; i < total; i++ {
		key := fmt.Sprintf("job-%d", i)
		worker := r.GetWorker(key)
		counts[worker]++
	}

	// Each worker should get roughly 33% (±10%).
	expected := float64(total) / 3.0
	tolerance := 0.10

	for worker, count := range counts {
		deviation := math.Abs(float64(count)-expected) / expected
		t.Logf("%s: %d jobs (%.1f%%)", worker, count, float64(count)/float64(total)*100)
		if deviation > tolerance {
			t.Errorf("%s got %d jobs, expected ~%.0f (deviation %.1f%% exceeds %.0f%%)",
				worker, count, expected, deviation*100, tolerance*100)
		}
	}
}

func TestGetWorker_MinimalRemapping(t *testing.T) {
	workers := []string{"node-1", "node-2", "node-3"}
	r := NewWithMembers(workers)

	// Record assignments for 1000 jobs.
	total := 1000
	before := make(map[string]string, total)
	for i := 0; i < total; i++ {
		key := fmt.Sprintf("job-%d", i)
		before[key] = r.GetWorker(key)
	}

	// Remove node-3.
	r.Remove("node-3")

	// Check how many jobs moved.
	moved := 0
	for i := 0; i < total; i++ {
		key := fmt.Sprintf("job-%d", i)
		after := r.GetWorker(key)
		if before[key] != after {
			moved++
			// Jobs that moved should have been on node-3.
			if before[key] != "node-3" {
				t.Errorf("job %s moved from %s to %s (should only move if it was on node-3)",
					key, before[key], after)
			}
		}
	}

	t.Logf("%d/%d jobs moved after removing node-3 (%.1f%%)",
		moved, total, float64(moved)/float64(total)*100)

	// Roughly 1/3 should move (the ones that were on node-3).
	// Allow some tolerance.
	if moved < total/6 || moved > total/2 {
		t.Errorf("expected ~33%% of jobs to move, got %d/%d", moved, total)
	}
}

func TestAddRemove(t *testing.T) {
	r := New()

	r.Add("node-1")
	r.Add("node-2")
	if r.Size() != 2 {
		t.Fatalf("expected size 2, got %d", r.Size())
	}

	// Adding the same worker twice is a no-op.
	r.Add("node-1")
	if r.Size() != 2 {
		t.Fatalf("expected size 2 after duplicate add, got %d", r.Size())
	}

	r.Remove("node-1")
	if r.Size() != 1 {
		t.Fatalf("expected size 1, got %d", r.Size())
	}

	// All keys should go to node-2 now.
	if got := r.GetWorker("anything"); got != "node-2" {
		t.Errorf("expected node-2, got %s", got)
	}

	// Removing a non-existent worker is a no-op.
	r.Remove("node-999")
	if r.Size() != 1 {
		t.Fatalf("expected size 1 after removing non-existent, got %d", r.Size())
	}
}

func TestMembers(t *testing.T) {
	r := NewWithMembers([]string{"node-3", "node-1", "node-2"})

	members := r.Members()
	if len(members) != 3 {
		t.Fatalf("expected 3 members, got %d", len(members))
	}

	// Should be sorted.
	if members[0] != "node-1" || members[1] != "node-2" || members[2] != "node-3" {
		t.Errorf("expected sorted members, got %v", members)
	}
}
