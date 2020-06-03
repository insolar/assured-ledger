// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package resolver

import (
	"fmt"
	"net"
	"net/url"

	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type fixedAddressResolver struct {
	publicAddress string
}

func NewFixedAddressResolver(publicAddress string) PublicAddressResolver {
	return newFixedAddressResolver(publicAddress)
}

func newFixedAddressResolver(publicAddress string) *fixedAddressResolver {
	return &fixedAddressResolver{
		publicAddress: publicAddress,
	}
}

func (r *fixedAddressResolver) Resolve(address string) (string, error) {
	url, err := url.Parse(address)

	var port string
	if err != nil {
		_, port, _ = net.SplitHostPort(address)
	} else {
		port = url.Port()
	}

	if port == "" {
		return "", errors.New("Failed to extract port from uri: " + address)
	}
	return fmt.Sprintf("%s:%s", r.publicAddress, port), nil
}
