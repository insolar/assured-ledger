package watchdog

import "math"

const DisabledHeartbeat = math.MinInt64

type HeartbeatID uint32

type Heartbeat struct {
	From             HeartbeatID
	PreviousUnixTime int64
	UpdateUnixTime   int64
}

func (h Heartbeat) IsCancelled() bool {
	return h.UpdateUnixTime == DisabledHeartbeat
}

func (h Heartbeat) IsFirst() bool {
	return h.PreviousUnixTime == 0
}
