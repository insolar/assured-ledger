package calltype

type ContractCallType uint8

const (
	_ ContractCallType = iota

	ContractCallOrdered
	ContractCallUnordered
	ContractCallSaga
)

func (t ContractCallType) String() string {
	switch t {
	case ContractCallOrdered:
		return "Ordered"
	case ContractCallUnordered:
		return "Unordered"
	case ContractCallSaga:
		return "Saga"
	default:
		return "Unknown"
	}
}
