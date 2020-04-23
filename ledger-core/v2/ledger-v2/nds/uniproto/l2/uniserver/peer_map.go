// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniserver

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type PeerMap struct {
	peers       []*Peer
	unusedCount uint32
	unusedMin   uint32
	aliases     map[nwapi.Address]uint32
}

func (p *PeerMap) ensureEmpty() {
	if !p.isEmpty() {
		panic(throw.IllegalState())
	}
}

func (p *PeerMap) isEmpty() bool {
	return len(p.aliases) == 0
}

func (p *PeerMap) get(a nwapi.Address) (uint32, *Peer) {
	if idx, ok := p.aliases[mapId(a)]; ok {
		return idx, p.peers[idx]
	}
	return 0, nil
}

func mapId(a nwapi.Address) nwapi.Address {
	if a.IsLoopback() {
		return a.WithoutName()
	}
	return a.HostIdentity()
}

func (p *PeerMap) checkAliases(peer *Peer, peerIndex uint32, aliases []nwapi.Address) ([]nwapi.Address, error) {
	j := 0
	for i, a := range aliases {
		switch conflictIndex, hasConflict := p.aliases[mapId(a)]; {
		case !hasConflict:
			if i != j {
				aliases[j] = a
			}
			j++
		case conflictIndex == peerIndex && peer != nil:
			//
		default:
			var primary nwapi.Address
			if peer != nil {
				primary = peer.GetPrimary()
			}
			return nil, throw.E("alias conflict", struct {
				Address, Alias, ConflictWith nwapi.Address
			}{primary, a, p.peers[conflictIndex].GetPrimary()})
		}
	}

	aliases = removeDuplicates(aliases[:j])
	return aliases, nil
}

func removeDuplicates(aliases []nwapi.Address) []nwapi.Address {
	j := 1
outer:
	for i := 1; i < len(aliases); i++ {
		a := aliases[i]
		for k := 0; k < j; k++ {
			if a == aliases[k] {
				continue outer
			}
		}
		if i != j {
			aliases[j] = aliases[i]
		}
		j++
	}
	return aliases[:j]
}

func (p *PeerMap) remove(idx uint32) {
	peer := p.peers[idx]
	for _, a := range peer.onRemoved() {
		delete(p.aliases, mapId(a))
	}
	p.peers[idx] = nil
	p.unusedCount++
	if p.unusedMin > idx || p.unusedCount == 1 {
		p.unusedMin = idx
	}
}

func (p *PeerMap) removeAlias(a nwapi.Address) bool {
	_, peer := p.get(a)
	if peer == nil {
		return false
	}

	if peer.removeAlias(a) {
		delete(p.aliases, mapId(a))
		return true
	}
	return false
}

func (p *PeerMap) addAliases(peerIndex uint32, aliases []nwapi.Address) error {
	peer := p.peers[peerIndex]
	switch aliases, err := p.checkAliases(peer, peerIndex, aliases); {
	case err != nil:
		return err
	case len(aliases) == 0:
		//
	default:
		peer.transport.addAliases(aliases)
		if p.aliases == nil {
			p.aliases = make(map[nwapi.Address]uint32)
		}
		for _, a := range aliases {
			p.aliases[mapId(a)] = peerIndex
		}
	}
	return nil
}

func (p *PeerMap) cleanup() []*Peer {
	p.aliases = nil
	peers := p.peers
	p.peers = nil
	return peers
}

func (p *PeerMap) _addPeer(peer *Peer) uint32 {
	peerIndex := uint32(len(p.peers))
	if p.unusedCount > 0 {
		for ; p.unusedMin < peerIndex; p.unusedMin++ {
			if p.peers[p.unusedMin] == nil {
				peerIndex = p.unusedMin
				p.peers[peerIndex] = peer
				p.unusedMin++
				p.unusedCount--
				return peerIndex
			}
		}
		panic(throw.Impossible())
	}
	p.peers = append(p.peers, peer)
	return peerIndex
}

func (p *PeerMap) addPeer(peer *Peer) uint32 {
	peerIndex := p._addPeer(peer)
	if p.aliases == nil {
		p.aliases = make(map[nwapi.Address]uint32)
	}
	for _, a := range peer.transport.aliases {
		p.aliases[mapId(a)] = peerIndex
	}
	return peerIndex
}
