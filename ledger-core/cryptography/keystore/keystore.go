package keystore

import (
	"context"
	"crypto"

	"github.com/insolar/component-manager"

	"github.com/insolar/assured-ledger/ledger-core/log/global"
	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/keystore/internal/privatekey"
)

type keyStore struct {
	Loader privatekey.Loader `inject:""`
	file   string
}

func (ks *keyStore) GetPrivateKey(identifier string) (crypto.PrivateKey, error) {
	return ks.Loader.Load(ks.file)
}

func (ks *keyStore) Start(ctx context.Context) error {
	// TODO: ugly hack; do proper checks
	if _, err := ks.GetPrivateKey(""); err != nil {
		return errors.W(err, "[ Start ] Failed to start keyStore")
	}

	return nil
}

type cachedKeyStore struct {
	keyStore cryptography.KeyStore

	privateKey crypto.PrivateKey
}

func (ks *cachedKeyStore) getCachedPrivateKey() crypto.PublicKey {
	if ks.privateKey != nil {
		return ks.privateKey
	}
	return nil
}

func (ks *cachedKeyStore) loadPrivateKey(identifier string) (crypto.PrivateKey, error) {
	privateKey, err := ks.keyStore.GetPrivateKey(identifier)
	if err != nil {
		return nil, errors.W(err, "[ loadPrivateKey ] Can't GetPrivateKey")
	}

	ks.privateKey = privateKey
	return privateKey, nil
}

func (ks *cachedKeyStore) GetPrivateKey(_ string) (crypto.PrivateKey, error) {
	privateKey := ks.getCachedPrivateKey()

	return privateKey, nil
}

func (ks *cachedKeyStore) Start(ctx context.Context) error {
	// TODO: ugly hack; do proper checks
	if _, err := ks.loadPrivateKey(""); err != nil {
		return errors.W(err, "[ Start ] Failed to start keyStore")
	}

	return nil
}

func NewKeyStore(path string) (cryptography.KeyStore, error) {
	keyStore := &keyStore{
		file: path,
	}

	cachedKeyStore := &cachedKeyStore{
		keyStore: keyStore,
	}

	manager := component.NewManager(nil)
	manager.SetLogger(global.Logger())
	manager.Inject(
		cachedKeyStore,
		keyStore,
		privatekey.NewLoader(),
	)

	if err := manager.Start(context.Background()); err != nil {
		return nil, errors.W(err, "[ NewKeyStore ] Failed to create keyStore")
	}

	return cachedKeyStore, nil
}

type inPlaceKeyStore struct {
	privateKey crypto.PrivateKey
}

func (ipks *inPlaceKeyStore) GetPrivateKey(string) (crypto.PrivateKey, error) {
	return ipks.privateKey, nil
}

func NewInplaceKeyStore(privateKey crypto.PrivateKey) cryptography.KeyStore {
	return &inPlaceKeyStore{privateKey: privateKey}
}
