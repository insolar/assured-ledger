package watchdog

import (
	"context"
	"time"
)

func DoneOf(ctx context.Context) <-chan struct{} {
	if h := from(ctx); h != nil {
		return h.done(ctx)
	}
	return ctx.Done()
}

func Beat(ctx context.Context) {
	ForcedBeat(ctx, false)
}

func ForcedBeat(ctx context.Context, forced bool) {
	if h := from(ctx); h != nil {
		h.beat(ctx, forced)
	}
}

func WithFrame(ctx context.Context, frameName string) context.Context {
	if h := from(ctx); h != nil {
		return h.root.createSubFrame(ctx, frameName, h)
	}
	return ctx
}

func Call(ctx context.Context, frameName string, fn func(context.Context)) {
	if frame := from(ctx); frame == nil {
		fn(ctx)
	} else {
		frame.root.createSubFrame(ctx, frameName, frame).call(fn)
	}
}

func WithFactory(ctx context.Context, name string, factory HeartbeatGeneratorFactory) context.Context {
	r := frameRoot{factory}
	return r.createSubFrame(ctx, name, nil)
}

func WithoutFactory(ctx context.Context) context.Context {
	return context.WithValue(ctx, watchdogKey, nil) // stop search
}

func FromContext(ctx context.Context) (bool, HeartbeatGeneratorFactory) {
	switch f := from(ctx); {
	case f == nil:
		return false, nil
	case f.root == nil:
		return true, nil
	default:
		return true, f.root.factory
	}
}

func from(ctx context.Context) *frame {
	if h, ok := ctx.Value(watchdogKey).(*frame); ok {
		return h
	}
	return nil
}

var watchdogKey = &struct{}{}

type frameRoot struct {
	factory HeartbeatGeneratorFactory
}

func (r *frameRoot) createSubFrame(ctx context.Context, name string, parent *frame) *frame {
	if parent != nil {
		name = parent.name + "/" + name
	}
	return &frame{r, ctx, r.factory.CreateGenerator(name), name}
}

type frame struct {
	root      *frameRoot
	context   context.Context
	generator *HeartbeatGenerator
	name      string
}

func (h *frame) Deadline() (deadline time.Time, ok bool) {
	h.beat(h.context, false)
	return h.context.Deadline()
}

func (h *frame) Value(key interface{}) interface{} {
	if watchdogKey == key {
		return h
	}
	h.beat(h.context, false)
	return h.context.Value(key)
}

func (h *frame) Err() error {
	err := h.context.Err()
	if err != nil {
		h.generator.Cancel()
	} else {
		h.generator.Heartbeat()
	}
	return err
}

func (h *frame) Done() <-chan struct{} {
	return h.done(h.context)
}

func (h *frame) beat(ctx context.Context, forced bool) {
	if ctx.Err() != nil {
		h.generator.Cancel()
	} else {
		h.generator.ForcedHeartbeat(forced)
	}
}

func (h *frame) start() {
	h.generator.ForcedHeartbeat(true)
}

func (h *frame) cancel() {
	h.generator.Cancel()
}

func (h *frame) done(ctx context.Context) <-chan struct{} {
	ch := ctx.Done()
	select {
	case <-ch:
		h.generator.Cancel()
	default:
		h.generator.ForcedHeartbeat(false)
	}
	return ch
}

func (h *frame) call(fn func(context.Context)) {
	h.start()
	defer h.cancel()
	fn(h)
}
