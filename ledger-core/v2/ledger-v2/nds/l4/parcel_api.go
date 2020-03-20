// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l4

import "github.com/insolar/assured-ledger/ledger-core/v2/vanilla/synckit"

type DeliveryAddress struct {
}

type ReturnAddress struct {
	returnTo DeliveryAddress
}

type DeliveryParcel struct {
	Head   SizerWriterTo
	Body   SizerWriterTo
	Cancel *synckit.ChainedCancel
	// TTL defines how many pulses this parcel can survive before cancellation
	TTL      uint8
	Policies DeliveryPolicies
}

type ParcelId uint64
