// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package bootstrap

import (
	"time"

	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms/legacyhost"
)

const permitTTL = 300

// CreatePermit creates permit as signed protobuf for joiner node to
func CreatePermit(authorityNodeRef reference.Holder, reconnectHost *legacyhost.Host, joinerPublicKey []byte, signer cryptography.Signer) (*rms.Permit, error) {
	payload := rms.PermitPayload{
		AuthorityNodeRef: rms.NewReference(authorityNodeRef),
		ExpireTimestamp:  time.Now().Unix() + permitTTL,
		ReconnectTo:      *reconnectHost,
		JoinerPublicKey:  joinerPublicKey,
	}

	data, err := payload.Marshal()
	if err != nil {
		return nil, errors.W(err, "failed to marshal bootstrap permit")
	}
	signature, err := signer.Sign(data)
	if err != nil {
		return nil, errors.W(err, "failed to sign bootstrap permit")
	}
	return &rms.Permit{Payload: payload, Signature: signature.Bytes()}, nil
}

// ValidatePermit validate granted permit and verifies signature of Authority Node
func ValidatePermit(permit *rms.Permit, cert nodeinfo.Certificate, verifier cryptography.Service) error {
	discovery := network.FindDiscoveryByRef(cert, permit.Payload.AuthorityNodeRef.Get())
	if discovery == nil {
		return errors.New("failed to find a discovery node from reference in permit")
	}

	payload, err := permit.Payload.Marshal()
	if err != nil || payload == nil {
		return errors.New("failed to marshal bootstrap permission payload part")
	}

	verified := verifier.Verify(discovery.GetPublicKey(), cryptography.SignatureFromBytes(permit.Signature), payload)

	if !verified {
		return errors.New("bootstrap permission payload verification failed")
	}
	return nil
}
