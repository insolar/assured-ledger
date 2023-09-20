package seedmanager

import (
	"sync"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

// DefaultTTL is default time period for deleting expired seeds
const DefaultTTL = 5 * time.Second

// DefaultCleanPeriod default time period for launching cleaning goroutine
const DefaultCleanPeriod = 5 * time.Second

type storedSeed struct {
	ts    time.Time
	pulse pulse.Number
}

// SeedManager manages working with seed pool
// It's thread safe
type SeedManager struct {
	mutex    sync.Mutex
	seedPool map[Seed]storedSeed
	ttl      time.Duration
	stopped  chan struct{}
}

// New creates new seed manager with default params
func New() *SeedManager {
	return NewSpecified(DefaultTTL, DefaultCleanPeriod)
}

// NewSpecified creates new seed manager with custom params
func NewSpecified(ttl time.Duration, cleanPeriod time.Duration) *SeedManager {
	sm := SeedManager{
		seedPool: make(map[Seed]storedSeed),
		ttl:      ttl,
		stopped:  make(chan struct{}),
	}

	ticker := time.NewTicker(cleanPeriod)
	defer ticker.Stop()

	go func() {
		var stop = false
		for !stop {
			select {
			case <-ticker.C:
				sm.deleteExpired()
			case <-sm.stopped:
				stop = true
			}
		}
		sm.stopped <- struct{}{}
	}()

	return &sm
}

func (sm *SeedManager) Stop() {
	sm.stopped <- struct{}{}
	<-sm.stopped
}

// Add adds seed to pool
func (sm *SeedManager) Add(seed Seed, pulse pulse.Number) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.seedPool[seed] = storedSeed{
		ts:    time.Now(),
		pulse: pulse,
	}

}

func (sm *SeedManager) isExpired(ts time.Time) bool {
	return time.Since(ts) > sm.ttl
}

// Pop deletes and returns seed from the pool
func (sm *SeedManager) Pop(seed Seed) (pulse.Number, bool) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	stored, ok := sm.seedPool[seed]

	if ok && !sm.isExpired(stored.ts) {
		delete(sm.seedPool, seed)
		return stored.pulse, true
	}

	return 0, false
}

func (sm *SeedManager) deleteExpired() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	for seed, stored := range sm.seedPool {
		if sm.isExpired(stored.ts) {
			delete(sm.seedPool, seed)
		}
	}
}

// SeedFromBytes converts slice of bytes to Seed. Returns nil if slice's size is not equal to SeedSize
func SeedFromBytes(slice []byte) *Seed {
	if len(slice) != int(SeedSize) {
		return nil
	}
	var result Seed
	copy(result[:], slice[:SeedSize])
	return &result
}
