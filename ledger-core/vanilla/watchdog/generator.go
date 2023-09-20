package watchdog

import (
	"math"
	"runtime"
	"sync/atomic"
	"time"
)

type HeartbeatGeneratorFactory interface {
	CreateGenerator(name string) *HeartbeatGenerator
}

func NewHeartbeatGenerator(id HeartbeatID, heartbeatPeriod time.Duration, out chan<- Heartbeat) HeartbeatGenerator {
	return NewHeartbeatGeneratorWithRetries(id, heartbeatPeriod, 0, out)
}

func NewHeartbeatGeneratorWithRetries(id HeartbeatID, heartbeatPeriod time.Duration, retryCount uint8, out chan<- Heartbeat) HeartbeatGenerator {
	attempts := retryCount
	if out == nil {
		attempts = 0
	} else if attempts < math.MaxUint8 {
		attempts++
	}

	period := uint32(0)
	switch {
	case heartbeatPeriod < 0:
		panic("illegal value")
	case heartbeatPeriod == 0:
		break
	case heartbeatPeriod <= time.Millisecond:
		period = 1
	default:
		heartbeatPeriod /= time.Millisecond
		if heartbeatPeriod > math.MaxUint32 {
			period = math.MaxUint32
		} else {
			period = uint32(heartbeatPeriod)
		}
	}

	return HeartbeatGenerator{id: id, heartbeatPeriod: period, sendAttempts: attempts, out: out}
}

type HeartbeatGenerator struct {
	id              HeartbeatID
	heartbeatPeriod uint32
	atomicNano      int64
	sendAttempts    uint8
	//name string
	out chan<- Heartbeat
}

func (g *HeartbeatGenerator) Heartbeat() {
	g.ForcedHeartbeat(false)
}

func (g *HeartbeatGenerator) ForcedHeartbeat(forced bool) {

	lastNano := atomic.LoadInt64(&g.atomicNano)
	if lastNano == DisabledHeartbeat {
		//closed channel or generator
		return
	}

	currentNano := time.Now().UnixNano()
	if lastNano != 0 && !forced && currentNano-lastNano < int64(g.heartbeatPeriod)*int64(time.Millisecond) {
		return
	}

	if !atomic.CompareAndSwapInt64(&g.atomicNano, lastNano, currentNano) {
		return // there is no need to retry in case of contention
	}

	if g.send(Heartbeat{From: g.id, PreviousUnixTime: lastNano, UpdateUnixTime: currentNano}) {
		return
	}

	atomic.CompareAndSwapInt64(&g.atomicNano, currentNano, lastNano) // try to roll-back on failed send
}

func (g *HeartbeatGenerator) Cancel() {
	for {
		lastNano := atomic.LoadInt64(&g.atomicNano)
		if lastNano == DisabledHeartbeat {
			//closed channel or generator
			return
		}
		if atomic.CompareAndSwapInt64(&g.atomicNano, lastNano, DisabledHeartbeat) {
			g.send(Heartbeat{From: g.id, PreviousUnixTime: lastNano, UpdateUnixTime: DisabledHeartbeat})
			return // there is no need to retry in case of contention
		}
	}
}

func (g *HeartbeatGenerator) send(beat Heartbeat) bool {
	defer func() {
		err := recover() // just in case of the closed channel
		if err != nil {
			g.Disable()
		}
	}()

	if g.sendAttempts == 0 {
		return true
	}

	for i := g.sendAttempts; i > 0; i-- {
		select {
		case g.out <- beat:
			return true
		default:
			// avoid lock up
			runtime.Gosched()
		}
	}
	return false
}

func (g *HeartbeatGenerator) Disable() {
	atomic.StoreInt64(&g.atomicNano, DisabledHeartbeat)
}

func (g *HeartbeatGenerator) IsEnabled() bool {
	return atomic.LoadInt64(&g.atomicNano) != DisabledHeartbeat
}
