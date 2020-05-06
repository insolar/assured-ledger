// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package foundation

import (
	"encoding/binary"
	"errors"
	"math/rand"
	"strings"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
)

// GetPulseNumber returns current pulse from context.
func GetPulseNumber() (insolar.PulseNumber, error) {
	req := GetLogicalContext().Request
	if req.IsEmpty() {
		return insolar.PulseNumber(0), errors.New("request from LogicCallContext is nil, get pulse is failed")
	}
	return req.GetLocal().Pulse(), nil
}

// GetRequestReference - Returns request reference from context.
func GetRequestReference() (reference.Global, error) {
	ctx := GetLogicalContext()
	if ctx.Request.IsEmpty() {
		return reference.Global{}, errors.New("request from LogicCallContext is nil, get pulse is failed")
	}
	return ctx.Request, nil
}

// NewSource returns source initialized with entropy from pulse.
func NewSource() rand.Source {
	randNum := binary.LittleEndian.Uint64(GetLogicalContext().Pulse.Entropy[:])
	return rand.NewSource(int64(randNum))
}

// GetObject creates proxy by address
// unimplemented
func GetObject(ref reference.Global) ProxyInterface {
	panic("not implemented")
}

// TrimPublicKey trims public key
func TrimPublicKey(publicKey string) string {
	return strings.Join(strings.Split(strings.TrimSpace(between(publicKey, "KEY-----", "-----END")), "\n"), "")
}

// TrimAddress trims address
func TrimAddress(address string) string {
	return strings.ToLower(strings.Join(strings.Split(strings.TrimSpace(address), "\n"), ""))
}

func between(value string, a string, b string) string {
	// Get substring between two strings.
	pos := strings.Index(value, a)
	if pos == -1 {
		return value
	}
	posLast := strings.Index(value, b)
	if posLast == -1 {
		return value
	}
	posFirst := pos + len(a)
	if posFirst >= posLast {
		return value
	}
	return value[posFirst:posLast]
}
