package smachine

import (
	"errors"
)

func (s *Slot) handleError(worker DetachableSlotWorker, stateUpdate StateUpdate, isPanic bool, err error) (ok, wakeup bool, errRet error) {
	area := StepArea

	slotError := SlotPanicError{}
	if errors.As(err, &slotError) && slotError.Area != 0 {
		area = slotError.Area
	}

	for isCallerSM := false; ; isCallerSM = true {
		action := ErrorHandlerDefault
		if eh := s.getErrorHandler(); eh != nil {
			fc := failureContext{
				isPanic: isPanic,
				canRecover: area.CanRecoverByHandler(),
				area: area,
				err: err,
				callerSM: isCallerSM,
			}

			ok := false
			if ok, action, err = fc.executeFailure(eh); !ok {
				area = ErrorHandlerArea
				action = ErrorHandlerDefault
			}
		}

		switch action {
		case ErrorHandlerRecover, ErrorHandlerRecoverAndWakeUp:
			switch {
			case !area.CanRecoverByHandler():
				s.logStepError(errorHandlerRecoverDenied, stateUpdate, area, err)
			default:
				s.logStepError(ErrorHandlerRecover, stateUpdate, area, err)
				return true, action == ErrorHandlerRecoverAndWakeUp, nil
			}
		case ErrorHandlerMute:
			s.logStepError(ErrorHandlerMute, stateUpdate, area, err)
		default:
			s.logStepError(ErrorHandlerDefault, stateUpdate, area, err)
		}

		if !s.hasSubroutine() {
			err = s.callFinalizeOnce(worker, err)
			return false, false, err
		}

		s.applySubroutineStop(err, worker)

		if area.CanRecoverBySubroutineExit() {
			return true, true, nil
		}
	}
}

func (s *Slot) callFinalizeOnce(worker DetachableSlotWorker, err error) error {
	finalizeFn := s.defFinalize
	if finalizeFn == nil {
		return err
	}
	s.defFinalize = nil

	// TODO logging
	fc := finalizeContext{ slotContext{s: s, w: worker}, err}
	return fc.executeFinalize(finalizeFn)
}


