package future

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/packet/types"
	"github.com/insolar/assured-ledger/ledger-core/rms"
)

type futureManager struct {
	mutex   sync.RWMutex
	futures map[types.RequestID]Future
}

func NewManager() Manager {
	return &futureManager{
		futures: make(map[types.RequestID]Future),
	}
}

func (fm *futureManager) Create(packet *rms.Packet) Future {
	// TODO: replace wrapping with own types in protobuf
	future := NewFuture(types.RequestID(packet.RequestID), packet.Receiver, packet, fm.canceler)

	fm.mutex.Lock()
	defer fm.mutex.Unlock()

	fm.futures[types.RequestID(packet.RequestID)] = future

	return future
}

func (fm *futureManager) Get(packet *rms.Packet) Future {
	// TODO: replace wrapping with own types in protobuf
	fm.mutex.RLock()
	defer fm.mutex.RUnlock()

	return fm.futures[types.RequestID(packet.RequestID)]
}

func (fm *futureManager) canceler(f Future) {
	fm.mutex.Lock()
	defer fm.mutex.Unlock()

	delete(fm.futures, f.ID())
}
