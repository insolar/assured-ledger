package cryptkit

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit.CertificateHolder -o . -s _mock.go -g

type CertificateHolder interface {
	GetPublicKey() SigningKeyHolder
	IsValidForHostAddress(HostAddress string) bool
}
