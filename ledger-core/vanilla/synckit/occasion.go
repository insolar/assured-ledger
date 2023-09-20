package synckit

import "time"

type Occasion interface {
	Deadline() time.Time
	NewTimer() TimerHolder
	NewFunc(fn func()) TimerHolder
	IsExpired() bool
}

type TimerHolder interface {
	Channel() <-chan time.Time
	Stop()
}

func Never() TimerHolder {
	return (*timerWithChan)(nil)
}

func NewTimer(d time.Duration) TimerHolder {
	return &timerWithChan{time.NewTimer(d)}
}

func NewTimerWithFunc(d time.Duration, fn func()) TimerHolder {
	return &timerWithFn{time.AfterFunc(d, fn)}
}

type timerWithChan struct {
	t *time.Timer
}

func (p *timerWithChan) Channel() <-chan time.Time {
	if p == nil || p.t == nil {
		return nil
	}
	return p.t.C
}

func (p *timerWithChan) Stop() {
	if p == nil || p.t == nil {
		return
	}
	p.t.Stop()
}

type timerWithFn struct {
	t *time.Timer
}

func (p *timerWithFn) Channel() <-chan time.Time {
	panic("illegal state")
}

func (p *timerWithFn) Stop() {
	p.t.Stop()
}

func NewOccasion(deadline time.Time) Occasion {
	return factory{deadline}
}

func NewOccasionAfter(d time.Duration) Occasion {
	return factory{time.Now().Add(d)}
}

type factory struct {
	d time.Time
}

func (p factory) IsExpired() bool {
	return p.d.Before(time.Now())
}

func (p factory) Deadline() time.Time {
	return p.d
}

func (p factory) NewTimer() TimerHolder {
	return NewTimer(time.Until(p.d))
}

func (p factory) NewFunc(fn func()) TimerHolder {
	return NewTimerWithFunc(time.Until(p.d), fn)
}

func NeverOccasion() Occasion {
	return factoryNever{}
}

type factoryNever struct{}

func (factoryNever) IsExpired() bool {
	return false
}

func (factoryNever) Deadline() time.Time {
	return time.Time{}
}

func (factoryNever) NewTimer() TimerHolder {
	return Never()
}

func (factoryNever) NewFunc(fn func()) TimerHolder {
	return Never()
}

func EverOccasion() Occasion {
	return factoryEver{}
}

type factoryEver struct {
}

func (factoryEver) IsExpired() bool {
	return true
}

func (factoryEver) Deadline() time.Time {
	return time.Time{}
}

func (factoryEver) NewTimer() TimerHolder {
	return NewTimer(0)
}

func (factoryEver) NewFunc(fn func()) TimerHolder {
	go fn()
	return Never()
}
