package rms

type Marshaler interface {
	Marshal() ([]byte, error)
	// Temporary commented to look like insolar.Payload
	//MarshalHead() ([]byte, error)
	//MarshalContent() ([]byte, error)
}
