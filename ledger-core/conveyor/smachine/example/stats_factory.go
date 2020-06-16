// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package example

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
)

type StatsFactoryFunc func() StatsStateMachine

type StatsStateMachine interface {
	smachine.SubroutineStateMachine
	GetStats() GamesStats
}

type GamesStats struct {
	gamesPlayed int
	highestBet float32
	highestBetPlayer int
}

type StatsTemplate struct {
	smachine.StateMachineDeclTemplate
}

func (g *StatsTemplate) GetInitStateFor(smachine.StateMachine) smachine.InitFunc {
	return nil // can't be run as a normal StateMachine
}

func (g *StatsTemplate) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return g
}
