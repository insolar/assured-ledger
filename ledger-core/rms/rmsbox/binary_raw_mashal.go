package rmsbox

type rawBinaryMarshal struct {
	rawBinary
}

func (p rawBinaryMarshal) ProtoSize() int {
	return p.rawBinary.protoSize()
}

func (p rawBinaryMarshal) MarshalTo(b []byte) (int, error) {
	return p.rawBinary.marshalTo(b)
}

func (p rawBinaryMarshal) MarshalToSizedBuffer(b []byte) (int, error) {
	return p.rawBinary.marshalToSizedBuffer(b)
}

func (p rawBinaryMarshal) Unmarshal(b []byte) error {
	return p.rawBinary.unmarshal(b)
}
