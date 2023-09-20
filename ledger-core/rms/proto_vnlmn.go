package rms

const (
	TypeLVerifyRequestPolymorphID            = TypeLRegisterRequestPolymorphID + 1
)

type (
	LVerifyRequest            = LRegisterRequest
)

func init() {
	RegisterMessageType(TypeLVerifyRequestPolymorphID, "", (*LVerifyRequest)(nil))
}

