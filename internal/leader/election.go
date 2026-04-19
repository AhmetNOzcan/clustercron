package leader

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const lockKey = 123456789

type Election struct {
	pool     *pgxpool.Pool
	nodeID   string
	interval time.Duration // how often to try acquiring the lock

	mu       sync.Mutex
	isLeader bool
}

func NewElection(pool *pgxpool.Pool, nodeID string) *Election {
	return &Election{
		pool:     pool,
		nodeID:   nodeID,
		interval: 10 * time.Second,
	}
}

func (e *Election) IsLeader() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.isLeader
}

func (e *Election) setLeader(v bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.isLeader = v
}

func (e *Election) Run(ctx context.Context, onElected func(ctx context.Context)) {
	log.Printf("[election] node %s joining election, trying every %s", e.nodeID, e.interval)
	for {
		if ctx.Err() != nil {
			log.Printf("[election] node %s stopped", e.nodeID)
			return
		}

		e.tryLead(ctx, onElected)

		select {
		case <-time.After(e.interval):
		case <-ctx.Done():
			return
		}
	}
}

func (e *Election) tryLead(ctx context.Context, onElected func(context.Context)) {
	conn, err := e.pool.Acquire(ctx)
	if err != nil {
		log.Printf("[election] ERROR acquire conn: %v", err)
		return
	}

	var acquired bool
	err = conn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", lockKey).Scan(&acquired)
	if err != nil {
		conn.Release()
		log.Printf("[election] ERROR try lock: %v", err)
		return
	}

	if !acquired {
		// Someone else is the leader. Release the connection and try later.
		conn.Release()
		return
	}

	log.Printf("[election] node %s became LEADER", e.nodeID)
	e.setLeader(true)

	leaderCtx, cancelLeader := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		onElected(leaderCtx)
	}()

	e.holdLock(ctx, conn)

	log.Printf("[election] node %s lost leadership", e.nodeID)
	e.setLeader(false)
	cancelLeader()

	<-done

	_, _ = conn.Exec(ctx, "SELECT pg_advisory_unlock($1)", lockKey)
	conn.Release()

	log.Printf("[election] node %s released lock", e.nodeID)
}

func (e *Election) holdLock(ctx context.Context, conn *pgxpool.Conn) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			err := conn.Ping(pingCtx)
			cancel()
			if err != nil {
				log.Printf("[election] lock connection lost: %v", err)
				return
			}
		case <-ctx.Done():
			return
		}
	}
}
