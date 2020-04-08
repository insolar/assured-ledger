// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l2

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/apinetwork"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type PeerMap struct {
	peers       []*Peer
	unusedCount uint32
	unusedMin   uint32
	aliases     map[apinetwork.Address]uint32
}

func (p *PeerMap) ensureEmpty() {
	if len(p.aliases) > 0 {
		panic(throw.IllegalState())
	}
}

func (p *PeerMap) get(a apinetwork.Address) (uint32, *Peer) {
	if idx, ok := p.aliases[mapId(a)]; ok {
		return idx, p.peers[idx]
	}
	return 0, nil
}

func mapId(a apinetwork.Address) apinetwork.Address {
	if a.IsLoopback() {
		return a
	}
	return a.HostOnly()
}

func (p *PeerMap) checkAliases(peer *Peer, peerIndex uint32, aliases []apinetwork.Address) ([]apinetwork.Address, error) {
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
			var primary apinetwork.Address
			if peer != nil {
				primary = peer.GetPrimary()
			}
			return nil, throw.E("alias conflict", struct {
				Address, Alias, ConflictWith apinetwork.Address
			}{primary, a, p.peers[conflictIndex].GetPrimary()})
		}
	}
	return aliases[:j], nil
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

func (p *PeerMap) removeAlias(a apinetwork.Address) bool {
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

func (p *PeerMap) addAliases(peerIndex uint32, aliases []apinetwork.Address) error {
	peer := p.peers[peerIndex]
	switch aliases, err := p.checkAliases(peer, peerIndex, aliases); {
	case err != nil:
		return err
	case len(aliases) == 0:
		//
	default:
		peer.transport.AddAliases(aliases)
		if p.aliases == nil {
			p.aliases = make(map[apinetwork.Address]uint32)
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
		p.aliases = make(map[apinetwork.Address]uint32)
	}
	for _, a := range peer.transport.aliases {
		p.aliases[mapId(a)] = peerIndex
	}
	return peerIndex
}
