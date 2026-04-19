package heartbeat

import (
	"clustercron/internal/broker"
	"context"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	keyPrefix = "clustercron:heartbeat:"
	interval  = 5 * time.Second
	ttl       = 15 * time.Second
)

type Monitor struct {
	redis  *broker.Redis
	nodeID string
}

func NewMonitor(redis *broker.Redis, nodeID string) *Monitor {
	return &Monitor{
		redis:  redis,
		nodeID: nodeID,
	}
}

func (m *Monitor) Run(ctx context.Context) {
	log.Printf("[heartbeat] node %s starting, interval=%s ttl=%s", m.nodeID, interval, ttl)
	m.beat(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.beat(ctx)
		case <-ctx.Done():
			m.remove()
			log.Printf("[heartbeat] node %s stopped", m.nodeID)
			return
		}
	}
}

func (m *Monitor) beat(ctx context.Context) {
	key := keyPrefix + m.nodeID
	value := fmt.Sprintf("%d", time.Now().Unix())
	if err := m.redis.SetWithExpiry(ctx, key, value, ttl); err != nil {
		log.Printf("[heartbeat] ERROR write: %v", err)
	}
}

func (m *Monitor) remove() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	key := keyPrefix + m.nodeID

	_ = m.redis.Delete(ctx, key)
}

func (m *Monitor) LiveWorkers(ctx context.Context) ([]string, error) {
	keys, err := m.redis.ScanKeys(ctx, keyPrefix+"*")
	if err != nil {
		return nil, fmt.Errorf("scan heartbeats: %w", err)
	}
	now := time.Now().Unix()
	var workers []string

	for _, key := range keys {
		nodeID := strings.TrimPrefix(key, keyPrefix)
		val, err := m.redis.Get(ctx, key)
		if err != nil || val == "" {
			continue
		}
		ts, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			continue
		}

		// Consider stale if older than 2x TTL (very conservative).
		if now-ts > int64(ttl.Seconds())*2 {
			continue
		}

		workers = append(workers, nodeID)
	}
	sort.Strings(workers)

	return workers, nil
}
