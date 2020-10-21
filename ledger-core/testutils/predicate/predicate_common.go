// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package predicate

import (
	"reflect"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/convlog"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/debuglogger"
)

func BeforeStep(fn smachine.StateFunc) Func {
	return func(event debuglogger.UpdateEvent) bool {
		switch {
		case event.Data.EventType != smachine.StepLoggerUpdate && event.Data.EventType != smachine.StepLoggerMigrate:
		case event.Update.NextStep.Transition == nil:
			// Transition == nil means that the step remains the same
			return testutils.CmpStateFuncs(fn, event.Data.CurrentStep.Transition)
		default:
			return testutils.CmpStateFuncs(fn, event.Update.NextStep.Transition)
		}
		return false
	}
}

func BeforeStepName(name string) Func {
	return func(event debuglogger.UpdateEvent) bool {
		switch event.Data.EventType {
		case smachine.StepLoggerUpdate, smachine.StepLoggerMigrate:
			step := event.Update.NextStep
			if step.Transition == nil {
				step = event.Data.CurrentStep
			}
			if name == step.Name {
				return true
			}
			stepName := convlog.GetStepName(event.Update.NextStep.Transition)
			return name == stepName
		default:
			return false
		}
	}
}

func AfterStep(fn smachine.StateFunc) Func {
	return func(event debuglogger.UpdateEvent) bool {
		return event.Data.EventType == smachine.StepLoggerUpdate && testutils.CmpStateFuncs(fn, event.Data.CurrentStep.Transition)
	}
}

func AfterStepName(name string) Func {
	return func(event debuglogger.UpdateEvent) bool {
		if event.Data.EventType == smachine.StepLoggerUpdate {
			if name == event.Data.CurrentStep.Name {
				return true
			}
			stepName := convlog.GetStepName(event.Data.CurrentStep.Transition)
			return stepName == name
		}
		return false
	}
}

func AfterAnyStep() Func {
	return func(event debuglogger.UpdateEvent) bool {
		return event.Data.EventType == smachine.StepLoggerUpdate
	}
}

func AfterAnyMigrate() Func {
	return func(event debuglogger.UpdateEvent) bool {
		return event.Data.EventType == smachine.StepLoggerMigrate
	}
}

func AfterMigrate(fn smachine.MigrateFunc) Func {
	return func(event debuglogger.UpdateEvent) bool {
		return event.Data.EventType == smachine.StepLoggerMigrate && testutils.CmpStateFuncs(fn, event.Update.AppliedMigrate)
	}
}

func BeforeStepExt(s smachine.SlotStep) Func {
	return func(event debuglogger.UpdateEvent) bool {
		step := event.Update.NextStep
		switch {
		case event.Data.EventType != smachine.StepLoggerUpdate && event.Data.EventType != smachine.StepLoggerMigrate:
		case s.Transition != nil && !testutils.CmpStateFuncs(s.Transition, step.Transition):
		case s.Migration != nil && !testutils.CmpStateFuncs(s.Migration, step.Migration):
		case s.Handler != nil && !testutils.CmpStateFuncs(s.Handler, step.Handler):
		case step.Flags&s.Flags != s.Flags:
		case s.Transition == nil:
			return true
		case step.Transition == nil:
			// Transition == nil means that the step remains the same
			return testutils.CmpStateFuncs(s.Transition, event.Data.CurrentStep.Transition)
		default:
			return testutils.CmpStateFuncs(s.Transition, step.Transition)
		}
		return false
	}
}

func AfterStepExt(s smachine.SlotStep) Func {
	return func(event debuglogger.UpdateEvent) bool {
		step := event.Data.CurrentStep
		switch {
		case event.Data.EventType != smachine.StepLoggerUpdate:
		case s.Transition != nil && !testutils.CmpStateFuncs(s.Transition, step.Transition):
		case s.Migration != nil && !testutils.CmpStateFuncs(s.Migration, step.Migration):
		case s.Handler != nil && !testutils.CmpStateFuncs(s.Handler, step.Handler):
		case step.Flags&s.Flags != s.Flags:
		case s.Transition == nil:
			return true
		default:
			return testutils.CmpStateFuncs(s.Transition, step.Transition)
		}
		return false
	}
}

func AfterTestString(marker string) Func {
	return func(event debuglogger.UpdateEvent) bool {
		if event.Data.EventType == smachine.StepLoggerTrace {
			s, ok := event.CustomEvent.(string)
			return ok && s == marker
		}
		return false
	}
}

func AfterCustomEvent(fn func(interface{}) bool) Func {
	return func(event debuglogger.UpdateEvent) bool {
		return event.Data.EventType.IsEvent() && fn(event.CustomEvent)
	}
}

func AfterCustomEventType(tp reflect.Type) Func {
	return func(event debuglogger.UpdateEvent) bool {
		return event.Data.EventType.IsEvent() && reflect.ValueOf(event.CustomEvent).Type().AssignableTo(tp)
	}
}
