package platformpolicy

import (
	"crypto"

	"github.com/insolar/component-manager"

	"github.com/insolar/assured-ledger/ledger-core/log/global"
	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/keystore"
)

type NodeCryptographyService struct {
	KeyStore                   cryptography.KeyStore                   `inject:""`
	PlatformCryptographyScheme cryptography.PlatformCryptographyScheme `inject:""`
	KeyProcessor               cryptography.KeyProcessor               `inject:""`
}

func (cs *NodeCryptographyService) GetPublicKey() (crypto.PublicKey, error) {
	privateKey, err := cs.KeyStore.GetPrivateKey("")
	if err != nil {
		return nil, errors.W(err, "[ Sign ] Failed to get private privateKey")
	}

	return cs.KeyProcessor.ExtractPublicKey(privateKey), nil
}

func (cs *NodeCryptographyService) Sign(payload []byte) (*cryptography.Signature, error) {
	privateKey, err := cs.KeyStore.GetPrivateKey("")
	if err != nil {
		return nil, errors.W(err, "[ Sign ] Failed to get private privateKey")
	}

	signer := cs.PlatformCryptographyScheme.DataSigner(privateKey, cs.PlatformCryptographyScheme.IntegrityHasher())
	signature, err := signer.Sign(payload)
	if err != nil {
		return nil, errors.W(err, "[ Sign ] Failed to sign payload")
	}

	return signature, nil
}

func (cs *NodeCryptographyService) Verify(publicKey crypto.PublicKey, signature cryptography.Signature, payload []byte) bool {
	return cs.PlatformCryptographyScheme.DataVerifier(publicKey, cs.PlatformCryptographyScheme.IntegrityHasher()).Verify(signature, payload)
}

func NewCryptographyService() cryptography.Service {
	return &NodeCryptographyService{}
}

func NewKeyBoundCryptographyService(privateKey crypto.PrivateKey) cryptography.Service {
	platformCryptographyScheme := NewPlatformCryptographyScheme()
	keyStore := keystore.NewInplaceKeyStore(privateKey)
	keyProcessor := NewKeyProcessor()
	cryptographyService := NewCryptographyService()

	cm := component.NewManager(nil)
	cm.SetLogger(global.Logger())

	cm.Register(platformCryptographyScheme)
	cm.Inject(keyStore, cryptographyService, keyProcessor)
	return cryptographyService
}

func NewStorageBoundCryptographyService(path string) (cryptography.Service, error) {
	platformCryptographyScheme := NewPlatformCryptographyScheme()
	keyStore, err := keystore.NewKeyStore(path)
	if err != nil {
		return nil, errors.W(err, "[ NewStorageBoundCryptographyService ] Failed to create KeyStore")
	}
	keyProcessor := NewKeyProcessor()
	cryptographyService := NewCryptographyService()

	cm := component.NewManager(nil)
	cm.SetLogger(global.Logger())

	cm.Register(platformCryptographyScheme, keyStore)
	cm.Inject(cryptographyService, keyProcessor)
	return cryptographyService, nil
}
