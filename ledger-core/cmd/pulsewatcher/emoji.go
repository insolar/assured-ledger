// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package main

import (
	"github.com/insolar/assured-ledger/ledger-core/application/api/requester"

	"sync"
)

type Emojer interface {
	RegisterNode(string, requester.Node)
	GetEmoji(n requester.Node) string
}

type NoEmoji struct{}

func (e *NoEmoji) RegisterNode(string, requester.Node) {}
func (e *NoEmoji) GetEmoji(requester.Node) string      { return "" }

type Emoji struct {
	mu        sync.RWMutex
	registred map[uint32]string
	light     []string
	virtual   []string
}

func NewEmoji() *Emoji {
	return &Emoji{
		registred: make(map[uint32]string),
		light:     []string{"😀"},
		// todo refactor this, remove node count limitations
		virtual: []string{
			"😀", "😆", "😎", "😭", "😴", "♈️", "♉️", "♊️️", "♋️", "♌️", "♍️", "♎️", "♏️", "♐️", "♑️", "️♒️", "♓️", "⚽", "🏀", "🏈", "🏐", "🏉", "🚗", "🚕", "🚙", "🚌", "🚒", "🚛", "🚜", "🚑️", "🚎", "🏎", "🚐", "🚚", "😀", "😆", "😎", "😭", "😴", "♈️", "♉️", "♊️️", "♋️", "♌️", "♍️", "♎️", "♏️", "♐️", "♑️", "️♒️", "♓️", "😀", "😆", "😎", "😭", "😴", "♈️", "♉️", "♊️️", "♋️", "♌️", "♍️", "♎️", "♏️", "♐️", "♑️", "️♒️", "♓️", "😀", "😆", "😎", "😭", "😴", "♈️", "♉️", "♊️️", "♋️", "♌️", "♍️", "♎️", "♏️", "♐️", "♑️", "️♒️", "♓️",
			"😀", "😆", "😎", "😭", "😴", "♈️", "♉️", "♊️️", "♋️", "♌️", "♍️", "♎️", "♏️", "♐️", "♑️", "️♒️", "♓️", "⚽", "🏀", "🏈", "🏐", "🏉", "🚗", "🚕", "🚙", "🚌", "🚒", "🚛", "🚜", "🚑️", "🚎", "🏎", "🚐", "🚚", "😀", "😆", "😎", "😭", "😴", "♈️", "♉️", "♊️️", "♋️", "♌️", "♍️", "♎️", "♏️", "♐️", "♑️", "️♒️", "♓️", "😀", "😆", "😎", "😭", "😴", "♈️", "♉️", "♊️️", "♋️", "♌️", "♍️", "♎️", "♏️", "♐️", "♑️", "️♒️", "♓️", "😀", "😆", "😎", "😭", "😴", "♈️", "♉️", "♊️️", "♋️", "♌️", "♍️", "♎️", "♏️", "♐️", "♑️", "️♒️", "♓️",
			"😀", "😆", "😎", "😭", "😴", "♈️", "♉️", "♊️️", "♋️", "♌️", "♍️", "♎️", "♏️", "♐️", "♑️", "️♒️", "♓️", "⚽", "🏀", "🏈", "🏐", "🏉", "🚗", "🚕", "🚙", "🚌", "🚒", "🚛", "🚜", "🚑️", "🚎", "🏎", "🚐", "🚚", "😀", "😆", "😎", "😭", "😴", "♈️", "♉️", "♊️️", "♋️", "♌️", "♍️", "♎️", "♏️", "♐️", "♑️", "️♒️", "♓️", "😀", "😆", "😎", "😭", "😴", "♈️", "♉️", "♊️️", "♋️", "♌️", "♍️", "♎️", "♏️", "♐️", "♑️", "️♒️", "♓️", "😀", "😆", "😎", "😭", "😴", "♈️", "♉️", "♊️️", "♋️", "♌️", "♍️", "♎️", "♏️", "♐️", "♑️", "️♒️", "♓️",
			"😀", "😆", "😎", "😭", "😴", "♈️", "♉️", "♊️️", "♋️", "♌️", "♍️", "♎️", "♏️", "♐️", "♑️", "️♒️", "♓️", "⚽", "🏀", "🏈", "🏐", "🏉", "🚗", "🚕", "🚙", "🚌", "🚒", "🚛", "🚜", "🚑️", "🚎", "🏎", "🚐", "🚚", "😀", "😆", "😎", "😭", "😴", "♈️", "♉️", "♊️️", "♋️", "♌️", "♍️", "♎️", "♏️", "♐️", "♑️", "️♒️", "♓️", "😀", "😆", "😎", "😭", "😴", "♈️", "♉️", "♊️️", "♋️", "♌️", "♍️", "♎️", "♏️", "♐️", "♑️", "️♒️", "♓️", "😀", "😆", "😎", "😭", "😴", "♈️", "♉️", "♊️️", "♋️", "♌️", "♍️", "♎️", "♏️", "♐️", "♑️", "️♒️", "♓️",
		},
	}
}

//todo: one url has many shortISs if node restart
func (e *Emoji) RegisterNode(_ string, n requester.Node) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, ok := e.registred[n.ID]; ok {
		return
	}

	var x string
	switch n.Role {
	case "heavy_material":
		e.registred[n.ID] = "😈"
	case "light_material":
		// pop front
		x, e.light = e.light[0], e.light[1:]
		e.registred[n.ID] = x
	case "virtual":
		x, e.virtual = e.virtual[0], e.virtual[1:]
		e.registred[n.ID] = x
	}
}

func (e *Emoji) GetEmoji(n requester.Node) string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if res, ok := e.registred[n.ID]; ok {
		return res
	}
	return "️⛔️"
}
