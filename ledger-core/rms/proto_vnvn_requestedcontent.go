package rms

type StateRequestContentFlags uint32

// nolint:unused
const (
	RequestLatestValidatedState StateRequestContentFlags = 1 << iota
	RequestLatestDirtyState
	RequestLatestValidatedCode
	RequestLatestDirtyCode
	RequestOrderedQueue
	RequestUnorderedQueue

	maxRequestedContentByte = iota
)

// Equal required by protobuf custom type
func (f StateRequestContentFlags) Equal(other StateRequestContentFlags) bool {
	return f == other
}

func (f StateRequestContentFlags) Contains(other StateRequestContentFlags) bool {
	return f&other != 0
}

func (f *StateRequestContentFlags) Set(flags ...StateRequestContentFlags) {
	for _, flag := range flags {
		*f = StateRequestContentFlags(uint32(*f) | uint32(flag))
	}
}

func (f *StateRequestContentFlags) Unset(flags ...StateRequestContentFlags) {
	for _, flag := range flags {
		*f = StateRequestContentFlags(uint32(*f) & ^uint32(flag))
	}
}

func (f StateRequestContentFlags) IsValid() bool {
	for i := 0; i < maxRequestedContentByte; i++ {
		f.Unset(1 << i)
	}

	return f == 0
}
