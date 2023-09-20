package rms

func GetSenderDelegationToken(msg interface{}) (CallDelegationToken, bool) {
	type tokenHolder interface {
		GetDelegationSpec() CallDelegationToken
	}

	if th, ok := msg.(tokenHolder); ok {
		return th.GetDelegationSpec(), true
	}
	return CallDelegationToken{}, false
}
