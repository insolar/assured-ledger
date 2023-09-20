package smachine

type Decider interface {
	GetDecision() Decision
}

var _ Decider = Decision(0)

type Decision uint8

const (
	Unknown Decision = iota
	Impossible
	NotPassed
	Passed
)

func (v Decision) GetDecision() Decision {
	return v
}

func (v Decision) IsPassed() bool {
	return v == Passed
}

func (v Decision) IsNotPassed() bool {
	return v == NotPassed
}

func (v Decision) IsValid() bool {
	return v >= NotPassed
}

func (v Decision) IsZero() bool {
	return v == 0
}

func (v Decision) AsValid() (BoolDecision, bool) {
	switch v {
	case Passed:
		return true, true
	case NotPassed:
		return false, true
	default:
		return false, false
	}
}

type BoolDecision bool

func (v BoolDecision) GetDecision() Decision {
	if v {
		return Passed
	}
	return NotPassed
}

func (v BoolDecision) Bool() bool {
	return bool(v)
}

func (v BoolDecision) IsPassed() bool {
	return bool(v)
}

func (v BoolDecision) IsNotPassed() bool {
	return !bool(v)
}

func (v BoolDecision) IsValid() bool {
	return true
}

func (v BoolDecision) AsValid() (BoolDecision, bool) {
	return v, true
}

func ChooseUpdate(d Decider, valid, invalid, other StateUpdate) StateUpdate {
	if d != nil {
		switch d.GetDecision() {
		case Passed:
			return valid
		case NotPassed:
			return other
		}
	}
	return invalid
}

func RepeatOrJump(ctx ExecutionContext, d Decider, next StateFunc) StateUpdate {
	return RepeatOrJumpElse(ctx, d, next, next)
}

func RepeatOrJumpElse(ctx ExecutionContext, d Decider, valid, invalid StateFunc) StateUpdate {
	return ChooseUpdate(d, ctx.Jump(valid), ctx.Jump(invalid), ctx.Yield().ThenRepeat())
}
