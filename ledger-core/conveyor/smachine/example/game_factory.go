// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package example

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

type GameFactoryFunc func(GamePlayers) GameStateMachine

type GameStateMachine interface {
	smachine.SubroutineStateMachine
	GetGameResult() GameResult
}

type GamePlayers struct {
	Players     []smachine.BargeIn
	PlayerIndex int
}

type GameResult struct {
}

type GameTemplate struct {
	smachine.StateMachineDeclTemplate
	gameKey longbits.ByteString
	players GamePlayers
}

func (g *GameTemplate) GetInitStateFor(smachine.StateMachine) smachine.InitFunc {
	return nil // can't be run as a normal StateMachine
}

func (g *GameTemplate) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return g
}
