package uniserver

type TransportStreamFormat uint8

const (
	_                   TransportStreamFormat = iota
	DetectByFirstPacket                       // considered as Unrestricted
	BinaryLimitedLength
	BinaryUnlimitedLength
	HTTPLimitedLength
	HTTPUnlimitedLength
)

func (v TransportStreamFormat) IsBinary() bool {
	return v>>1 == 1
}

func (v TransportStreamFormat) IsHTTP() bool {
	return v>>1 == 2
}

func (v TransportStreamFormat) IsUnlimited() bool {
	return v&1 != 0 // includes DetectByFirstPacket
}

func (v TransportStreamFormat) IsDefined() bool {
	return v > DetectByFirstPacket
}

func (v TransportStreamFormat) IsDefinedLimited() bool {
	return v.IsDefined() && !v.IsUnlimited()
}
