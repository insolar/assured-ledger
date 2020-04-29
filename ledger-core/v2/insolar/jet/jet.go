// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package jet

import (
	"context"
	"strconv"
	"strings"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/bits"
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/insolar/jet.Coordinator -o ./ -s _mock.go -g

// Coordinator provides methods for calculating Jet affinity
// (e.g. to which Jet a message should be sent).
type Coordinator interface {
	// Me returns current node.
	Me() insolar.Reference

	// QueryRole returns node refs responsible for role bound operations for given object and pulse.
	QueryRole(ctx context.Context, role insolar.DynamicRole, obj insolar.ID, pulse insolar.PulseNumber) ([]insolar.Reference, error)
}

// Parent returns a parent of the jet or jet itself if depth of provided JetID is zero.
func Parent(id insolar.JetID) insolar.JetID {
	depth, prefix := id.Depth(), id.Prefix()
	if depth == 0 {
		return id
	}

	return *insolar.NewJetID(depth-1, bits.ResetBits(prefix, depth-1))
}

// NewIDFromString creates new JetID from string represents binary prefix.
//
// "0"     -> prefix=[0..0], depth=1
// "1"     -> prefix=[1..0], depth=1
// "1010"  -> prefix=[1010..0], depth=4
func NewIDFromString(s string) insolar.JetID {
	id := insolar.NewJetID(uint8(len(s)), parsePrefix(s))
	return *id
}

func parsePrefix(s string) []byte {
	var prefix []byte
	tail := s
	for len(tail) > 0 {
		offset := 8
		if len(tail) < 8 {
			tail += strings.Repeat("0", 8-len(tail))
		}
		parsed, err := strconv.ParseUint(tail[:offset], 2, 8)
		if err != nil {
			panic(err)
		}
		prefix = append(prefix, byte(parsed))
		tail = tail[offset:]
	}
	return prefix
}

// Siblings calculates left and right siblings for provided jet.
func Siblings(id insolar.JetID) (insolar.JetID, insolar.JetID) {
	depth, prefix := id.Depth(), id.Prefix()

	leftPrefix := bits.ResetBits(prefix, depth)
	left := insolar.NewJetID(depth+1, leftPrefix)

	rightPrefix := bits.ResetBits(prefix, depth)
	setBit(rightPrefix, depth)
	right := insolar.NewJetID(depth+1, rightPrefix)

	return *left, *right
}
