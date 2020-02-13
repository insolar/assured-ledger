// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

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
