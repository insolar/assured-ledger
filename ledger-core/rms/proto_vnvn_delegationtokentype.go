package rms

type CallDelegationTokenType uint16

const (
	DelegationTokenTypeUninitialized CallDelegationTokenType = iota
	DelegationTokenTypeCall
	DelegationTokenTypeObjV
	DelegationTokenTypeObjL
	DelegationTokenTypeDrop
)

func (t CallDelegationTokenType) Equal(other CallDelegationTokenType) bool {
	return t == other
}

func (m CallDelegationToken) IsZero() bool {
	return m.TokenTypeAndFlags == DelegationTokenTypeUninitialized
}
