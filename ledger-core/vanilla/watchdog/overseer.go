package watchdog

//func NewPassiveOverseer(name string) *Overseer {
//	return &Overseer{name: name}
//}
//
//func NewActiveOverseer(name string, heartbeatPeriod time.Duration, workersHint int) *Overseer {
//
//	if heartbeatPeriod <= 0 {
//		panic("illegal value")
//	}
//
//	if workersHint <= 0 {
//		workersHint = 10
//	}
//
//	chanLimit := uint64(1+time.Second/heartbeatPeriod) * uint64(workersHint)
//	if chanLimit > 10000 {
//		chanLimit = 10000 // keep it reasonable
//	}
//
//	return &Overseer{name: name, heartbeatPeriod: heartbeatPeriod,
//		beatChannel: make(chan Heartbeat, chanLimit)}
//}
//
//type monitoringMap map[HeartbeatID]*monitoringEntry
//
//type Overseer struct {
//	name            string
//	beaters         monitoringMap
//	atomicIDCounter uint32
//	heartbeatPeriod time.Duration
//	beatChannel     chan Heartbeat
//}
//
//func (seer *Overseer) StartActive(ctx context.Context) {
//	if seer.beatChannel == nil {
//		panic("illegal state")
//	}
//
//	m := activeMonitor{seer, &seer.beaters, seer.beatChannel}
//	go m.worker(ctx)
//}
//
//func (seer *Overseer) AttachContext(ctx context.Context) context.Context {
//	ok, factory := FromContext(ctx)
//	if ok {
//		if factory == seer {
//			return ctx
//		}
//		panic("context is under supervision")
//	}
//	seer.ensure()
//	return WithFactory(ctx, "", seer)
//}
//
//func (seer *Overseer) ensure() {
//}
//
//func (seer *Overseer) GetNewID() uint32 {
//	for {
//		v := atomic.LoadUint32(&seer.atomicIDCounter)
//		if atomic.CompareAndSwapUint32(&seer.atomicIDCounter, v, v+1) {
//			return v + 1
//		}
//	}
//}
//
//func (seer *Overseer) CreateGenerator(name string) *HeartbeatGenerator {
//	id := seer.GetNewID()
//
//	period := seer.heartbeatPeriod
//	if period == 0 && seer.beatChannel == nil {
//		period = math.MaxInt64 //zero state should not cause excessive attempts
//	}
//
//	entryI, loaded := seer.beaters.LoadOrStore(id, &monitoringEntry{name: name})
//
//	entry := entryI.(*monitoringEntry)
//	if !loaded {
//		newGen := NewHeartbeatGenerator(id, period, seer.beatChannel)
//		entry.generator = &newGen
//	}
//	return entry.generator
//}
//
//func (seer *Overseer) cleanup() *HeartbeatGenerator {
//
//}
//
//type monitoringEntry struct {
//	name      string
//	generator *HeartbeatGenerator
//}
//
//type activeMonitor struct {
//	seer    *Overseer
//	beaters *sync.Map
//
//	beatChannel chan Heartbeat
//}
//
//func (m *activeMonitor) worker(ctx context.Context) {
//	defer close(m.beatChannel)
//
//	var prevRecent map[HeartbeatID]*monitoringEntry
//
//	// tick-tack model to detect stuck items
//	for {
//		recent := make(map[HeartbeatID]*monitoringEntry, len(prevRecent)+1)
//		if !m.workOnMap(ctx, recent, nil) {
//			return
//		}
//		prevRecent = recent
//	}
//}
//
//func (m *activeMonitor) workOnMap(ctx context.Context, recent map[HeartbeatID]*monitoringEntry, expire <-chan time.Time) bool {
//	for {
//		select {
//		case <-ctx.Done():
//			return false
//		case <-expire:
//			return true
//		case beat := <-m.beatChannel:
//			storedGen, ok := m.beaters.Load(beat.From)
//			if !ok {
//				m.missingEntryHeartbeat(beat)
//			}
//			me := storedGen.(*monitoringEntry)
//			recent[beat.From] = me
//			m.applyHeartbeat(beat, storedGen.(*monitoringEntry))
//		}
//	}
//}
//
//func (m *activeMonitor) applyHeartbeat(heartbeat Heartbeat, entry *monitoringEntry) {
//	if heartbeat.IsCancelled() {
//		m.beaters.Delete(heartbeat.From)
//	}
//}
//
//func (m *activeMonitor) missingEntryHeartbeat(heartbeat Heartbeat) {
//}
