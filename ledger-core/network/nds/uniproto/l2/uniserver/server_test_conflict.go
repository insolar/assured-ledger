package uniserver

import (
	"math"
	"math/rand"
	"strconv"
	"strings"

	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func RetryStartListenForTests(p *UnifiedServer, retryCount int) {
	for i := 0;; i++ {
		err := p.TryStartListen()
		if err == nil {
			return
		}
		switch errStr := err.Error(); {
		case i > retryCount:
		case adjustServerPort(p, i, rand.Intn, errStr):
			continue
		}
		panic(err)
	}
}

const lowPortAdjustBoundary = 1<<11

// adjustServerPort for first 10 attempts provides guaranteed unique port number, below or above the conflicted one.
// Goes purely random after 10th attempt.
func adjustServerPort(p *UnifiedServer, attempt int, randFn func(int) int, errStr string) bool {
	addr := nwapi.NewHostPort(p.config.BindingAddress, true)
	if attempt == 0 && addr.HasPort() {
		return false
	}

	const errMarker = ": bind: An attempt was made to access a socket in a way forbidden by its access permissions."
	markerPos := strings.Index(errStr, errMarker)
	if markerPos <= 0 {
		return false
	}

	conflictPort := uint64(0)
	switch {
	case attempt == 0:
		portStart := strings.LastIndexByte(errStr[:markerPos], ':')
		if portStart < 0 {
			panic(throw.Impossible())
		}
		var err error
		if conflictPort, err = strconv.ParseUint(errStr[portStart+1:markerPos], 10, 16); err != nil {
			panic(err)
		}

		if conflictPort > 1<<15 {
			conflictPort -= 1<<14 + 1<<13
		} else {
			// increment is less because next (odd) attempt will also increment
			conflictPort += 1<<12
		}
		conflictPort += uint64(randFn(1<<9))
	case attempt > 10:
		conflictPort = uint64(uint16(randFn(1<<15) + attempt + lowPortAdjustBoundary))
	default:
		conflictPort = uint64(addr.Port())
		rnd := uint64(randFn(1<<9))
		switch {
		case attempt > 0:
			step := uint64(1)<<12
			step += uint64(1)<<(14-(attempt>>1))
			if attempt&1 == 0 {
				conflictPort -= step
				conflictPort += 66 // 127 - 66 = 61 (prime)
			} else {
				conflictPort += step
				conflictPort += 127
			}
		case conflictPort > 1<<15:
			conflictPort -= 1 << 14 + 1 << 13
		default:
			// increment is less because next (odd) attempt will also increment
			conflictPort += 1 << 12
		}
		conflictPort += rnd
	}

	switch {
	case conflictPort < lowPortAdjustBoundary:
		conflictPort = lowPortAdjustBoundary
	case conflictPort > math.MaxUint16:
		panic(throw.Impossible())
	}

	p.config.BindingAddress = addr.WithPort(uint16(conflictPort)).String()
	return true
}
