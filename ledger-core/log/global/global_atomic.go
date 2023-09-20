package global

import (
	"sync/atomic"
	"unsafe"

	"github.com/insolar/assured-ledger/ledger-core/log/logcommon"
)

var adapter *logcommon.GlobalLogAdapter // atomic

func _globalLogAdapterRef() *unsafe.Pointer {
	return (*unsafe.Pointer)(unsafe.Pointer(&adapter))
}

func getGlobalLogAdapter() logcommon.GlobalLogAdapter {
	p := (*logcommon.GlobalLogAdapter)(atomic.LoadPointer(_globalLogAdapterRef()))
	if p == nil {
		return nil
	}
	return *p
}

func setGlobalLogAdapter(adapter logcommon.GlobalLogAdapter) {
	if adapter == nil {
		atomic.StorePointer(_globalLogAdapterRef(), nil)
		return
	}

	adapterRef := unsafe.Pointer(&adapter)
	for {
		p := atomic.LoadPointer(_globalLogAdapterRef())
		if p != nil && *(*logcommon.GlobalLogAdapter)(p) == adapter {
			return
		}
		if atomic.CompareAndSwapPointer(_globalLogAdapterRef(), p, adapterRef) {
			return
		}
	}
}
