package censusimpl

import (
	"fmt"
	"strings"

	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/profiles"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
)

func newEvictedPopulation(evicts []*updatableSlot, detectedErrors census.RecoverableErrorTypes) evictedPopulation {

	if len(evicts) == 0 {
		return evictedPopulation{detectedErrors: detectedErrors}
	}
	evictedNodes := make(map[node.ShortNodeID]profiles.EvictedNode, len(evicts))

	for _, s := range evicts {
		id := s.GetNodeID()
		evictedNodes[id] = &evictedSlot{s.StaticProfile, s.verifier, s.mode,
			s.leaveReason}
	}

	return evictedPopulation{evictedNodes, detectedErrors}
}

var _ census.EvictedPopulation = &evictedPopulation{}

type evictedPopulation struct {
	profiles       map[node.ShortNodeID]profiles.EvictedNode
	detectedErrors census.RecoverableErrorTypes
}

func (p evictedPopulation) String() string {
	if p.detectedErrors == 0 && len(p.profiles) == 0 {
		return "[]"
	}

	b := strings.Builder{}
	if p.detectedErrors != 0 {
		b.WriteString(fmt.Sprintf("errors:%v ", p.detectedErrors.String()))
	}
	if len(p.profiles) > 0 {
		b.WriteString(fmt.Sprintf("profiles:%d[", len(p.profiles)))

		if len(p.profiles) < 50 {
			for id := range p.profiles {
				b.WriteString(fmt.Sprintf(" %04d ", id))
			}
		} else {
			b.WriteString("too many")
		}
		b.WriteRune(']')
	}
	return b.String()
}

func (p *evictedPopulation) IsValid() bool {
	return p.detectedErrors != 0
}

func (p *evictedPopulation) GetDetectedErrors() census.RecoverableErrorTypes {
	return p.detectedErrors
}

func (p *evictedPopulation) FindProfile(nodeID node.ShortNodeID) profiles.EvictedNode {
	return p.profiles[nodeID]
}

func (p *evictedPopulation) GetCount() int {
	return len(p.profiles)
}

func (p *evictedPopulation) GetProfiles() []profiles.EvictedNode {
	r := make([]profiles.EvictedNode, len(p.profiles))
	idx := 0
	for _, v := range p.profiles {
		r[idx] = v
		idx++
	}
	return r
}

var _ profiles.EvictedNode = &evictedSlot{}

type evictedSlot struct {
	profiles.StaticProfile
	sf          cryptkit.SignatureVerifier
	mode        member.OpMode
	leaveReason uint32
}

func (p *evictedSlot) GetNodeID() node.ShortNodeID {
	return p.GetStaticNodeID()
}

func (p *evictedSlot) GetStatic() profiles.StaticProfile {
	return p.StaticProfile
}

func (p *evictedSlot) GetSignatureVerifier() cryptkit.SignatureVerifier {
	return p.sf
}

func (p *evictedSlot) GetOpMode() member.OpMode {
	return p.mode
}

func (p *evictedSlot) GetLeaveReason() uint32 {
	if !p.mode.IsEvictedGracefully() {
		return 0
	}
	return p.leaveReason
}
