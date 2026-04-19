package leader

import (
	"context"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func getTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestElection_SingleNode(t *testing.T) {
	pool := getTestPool(t)

	var elected atomic.Bool
	e := NewElection(pool, "test-node")
	e.interval = 1 * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go e.Run(ctx, func(leaderCtx context.Context) {
		elected.Store(true)
		<-leaderCtx.Done()
	})

	// Give it time to acquire the lock.
	time.Sleep(2 * time.Second)

	if !elected.Load() {
		t.Fatal("expected node to become leader")
	}
	if !e.IsLeader() {
		t.Fatal("IsLeader() should be true")
	}
}

func TestElection_TwoNodes(t *testing.T) {
	pool := getTestPool(t)

	var leader1Elected, leader2Elected atomic.Bool

	e1 := NewElection(pool, "node-1")
	e1.interval = 1 * time.Second
	e2 := NewElection(pool, "node-2")
	e2.interval = 1 * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	go e1.Run(ctx, func(leaderCtx context.Context) {
		leader1Elected.Store(true)
		<-leaderCtx.Done()
	})

	go e2.Run(ctx, func(leaderCtx context.Context) {
		leader2Elected.Store(true)
		<-leaderCtx.Done()
	})

	time.Sleep(3 * time.Second)

	// Exactly one should be the leader.
	l1 := leader1Elected.Load()
	l2 := leader2Elected.Load()

	if l1 == l2 {
		t.Fatalf("expected exactly one leader, got node-1=%v node-2=%v", l1, l2)
	}
}
