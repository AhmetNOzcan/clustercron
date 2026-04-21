package hashring

import (
	"fmt"
	"hash/crc32"
	"sort"
	"sync"
)

const defaultReplicas = 150

// Ring implements consistent hashing with virtual nodes.
type Ring struct {
	mu       sync.RWMutex
	replicas int
	points   []uint32          // sorted virtual node positions
	pointMap map[uint32]string // position → worker ID
	members  map[string]bool   // set of real worker IDs
}

// New creates an empty Ring.
func New() *Ring {
	return &Ring{
		replicas: defaultReplicas,
		pointMap: make(map[uint32]string),
		members:  make(map[string]bool),
	}
}

// NewWithMembers creates a Ring pre-populated with the given workers.
func NewWithMembers(workers []string) *Ring {
	r := New()
	for _, w := range workers {
		r.add(w)
	}
	return r
}

// Add places a worker on the ring.
func (r *Ring) Add(worker string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.add(worker)
}

func (r *Ring) add(worker string) {
	if r.members[worker] {
		return // already on the ring
	}

	r.members[worker] = true
	for i := 0; i < r.replicas; i++ {
		key := virtualNodeKey(worker, i)
		hash := hashKey(key)
		r.points = append(r.points, hash)
		r.pointMap[hash] = worker
	}

	sort.Slice(r.points, func(i, j int) bool {
		return r.points[i] < r.points[j]
	})
}

// Remove takes a worker off the ring.
func (r *Ring) Remove(worker string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.members[worker] {
		return
	}

	delete(r.members, worker)

	// Remove all virtual nodes for this worker.
	for i := 0; i < r.replicas; i++ {
		key := virtualNodeKey(worker, i)
		hash := hashKey(key)
		delete(r.pointMap, hash)
	}

	// Rebuild the sorted points list without the removed entries.
	newPoints := make([]uint32, 0, len(r.pointMap))
	for h := range r.pointMap {
		newPoints = append(newPoints, h)
	}
	sort.Slice(newPoints, func(i, j int) bool {
		return newPoints[i] < newPoints[j]
	})
	r.points = newPoints
}

// GetWorker returns the worker responsible for the given key.
// Returns empty string if the ring is empty.
func (r *Ring) GetWorker(key string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.points) == 0 {
		return ""
	}

	hash := hashKey(key)

	// Binary search for the first point >= hash.
	idx := sort.Search(len(r.points), func(i int) bool {
		return r.points[i] >= hash
	})

	// Wrap around the ring.
	if idx == len(r.points) {
		idx = 0
	}

	return r.pointMap[r.points[idx]]
}

// Members returns the current set of workers on the ring.
func (r *Ring) Members() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	members := make([]string, 0, len(r.members))
	for m := range r.members {
		members = append(members, m)
	}
	sort.Strings(members)
	return members
}

// Size returns the number of workers on the ring.
func (r *Ring) Size() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.members)
}

// hashKey hashes a string to a uint32 using CRC-32.
func hashKey(key string) uint32 {
	return crc32.ChecksumIEEE([]byte(key))
}

// virtualNodeKey builds the string that gets hashed for a virtual node.
func virtualNodeKey(worker string, index int) string {
	return fmt.Sprintf("%s-%d", worker, index)
}
