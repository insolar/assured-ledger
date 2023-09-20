package conveyor

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/managed"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

// componentManager provides management of conveyor-depended components
type componentManager struct {
	mutex      sync.Mutex
	state      int32
	components []managed.Component
}

func (p *componentManager) _sync(fn func()) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	fn()
}

func (p *componentManager) Add(pc *PulseConveyor, mc managed.Component) {
	isStarted := false
	p._sync(func() {
		switch {
		case mc == nil:
			panic(throw.IllegalValue())
		case p.state < 0:
			panic(throw.IllegalState())
		}
		p.components = append(p.components, mc)
		isStarted = p.state > 0
	})
	mh := conveyorHolder{pc}
	mc.Init(mh)
	if isStarted {
		mc.Start(mh)
	}
}

func (p *componentManager) Stopped(pc *PulseConveyor) {
	var components []managed.Component
	p._sync(func() {
		if p.state < 0 {
			panic(throw.IllegalState())
		}
		p.state = -1
		components = p.components
	})

	mh := conveyorHolder{pc}
	for _, c := range components {
		c.Stop(mh)
	}
}

func (p *componentManager) Started(pc *PulseConveyor) {
	var components []managed.Component
	p._sync(func() {
		if p.state != 0 {
			panic(throw.IllegalState())
		}
		p.state = 1
		components = p.components
	})

	mh := conveyorHolder{pc}
	for _, c := range components {
		c.Start(mh)
	}
}

func (p *componentManager) PulseChanged(pc *PulseConveyor, pr pulse.Range) {
	var components []managed.Component
	p._sync(func() {
		if p.state > 0 {
			components = p.components
		}
	})

	mh := conveyorHolder{pc}
	for _, c := range components {
		if mc, ok := c.(managed.ComponentWithPulse); ok {
			mc.PulseMigration(mh, pr)
		}
	}
}

type conveyorHolder struct {
	c *PulseConveyor
}

func (v conveyorHolder) GetDataManager() managed.DataManager {
	return v.c.GetDataManager()
}

func (v conveyorHolder) AddManagedComponent(c managed.Component) {
	v.c.AddManagedComponent(c)
}

func (v conveyorHolder) AddInputExt(pn pulse.Number, event InputEvent, createDefaults smachine.CreateDefaultValues) error {
	return v.c.AddInputExt(pn, event, createDefaults)
}

func (v conveyorHolder) WakeUpWorker() {
	v.c.WakeUpWorker()
}

func (v conveyorHolder) FindDependency(id string) (interface{}, bool) {
	return v.c.FindDependency(id)
}

func (v conveyorHolder) PutDependency(id string, vi interface{}) {
	v.c.PutDependency(id, vi)
}

func (v conveyorHolder) TryPutDependency(id string, vi interface{}) bool {
	return v.c.TryPutDependency(id, vi)
}

func (v conveyorHolder) AddDependency(vi interface{}) {
	v.c.AddDependency(vi)
}

func (v conveyorHolder) AddInterfaceDependency(vi interface{}) {
	v.c.AddInterfaceDependency(vi)
}

func (v conveyorHolder) GetPublishedGlobalAliasAndBargeIn(key interface{}) (smachine.SlotLink, smachine.BargeInHolder) {
	return v.c.GetPublishedGlobalAliasAndBargeIn(key)
}
