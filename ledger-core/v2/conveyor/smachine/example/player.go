/*
 * Copyright 2020 Insolar Network Ltd.
 * All rights reserved.
 * This material is licensed under the Insolar License version 1.0,
 * available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package example

import (
	"reflect"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

// PlayerSM is an example of state machine to illustrate some basic patterns of use.
// PlayerSM looks for a 2nd player, then chooses a game (strategy pattern) and plays the game.
type PlayerSM struct {
	smachine.StateMachineDeclTemplate
	pair    SharedPairDataLink
	adapter GameChooseAdapter
}

/**** Methods of the State Machine Declaration ****/

// GetStateMachineDeclaration returns a declaration (class) of a state machine.
// Declaration is responsible for various initialization and processing capabilities.
//
// For simplicity - SM declaration could be the same object as SM.
// When implementing SM declaration please embed smachine.StateMachineDeclTemplate to avoid manual implementation of required methods.
func (p *PlayerSM) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return p
}

// GetInitStateFor is a method of SM declaration that provides an init step for the given SM
// This method is invoked after injections and can rely on fields of SM.
// Nil returned by GetInitStateFor will cause an error for a caller.
func (PlayerSM) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	return sm.(*PlayerSM).stepInit
}

/**** Methods of the State Machine ****/

// stepInit receives InitializationContext parameter hence represent an initialization step of a state machine.
// At such step, SM can't wait or call adapters.
// Use this step to setup default handlers, like migration and error handlers.
// Also you can use this step together with InitChild() call by a parent to ensure that some actions are made before the parent will proceed,
// e.g. publish shared data or acquire a sync object.
func (p *PlayerSM) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {

	// Make this player available from outside, e.g. for player rank table.
	ctx.PublishGlobalAlias(ctx.SlotLink().SlotID())

	return ctx.Jump(p.stepFindPair)
}

// freePlayer is a key to advertise a player without a pair
const freePlayer = "freePlayer"

var sharedPlayerPairType = reflect.TypeOf((*SharedPairData)(nil))

// stepFindPair looks for another player without a pair and creates a pair.
// When there is no player without a pair, this step will publish this player as "freePlayer" and will wait for another player to join.
func (p *PlayerSM) stepFindPair(ctx smachine.ExecutionContext) smachine.StateUpdate {

	// GetPublishedLink checks if there is something is registered under the given key
	if sd := ctx.GetPublishedLink(freePlayer); sd.IsOfType(sharedPlayerPairType) {
		p.pair.sharedData = sd
		// and uses it to create a pair
		sharedAccessFn := p.pair.PrepareAccess(func(pp *SharedPairData) (wakeup bool) {
			switch {
			case pp.players[0].IsZero():
				panic(throw.Impossible())
			case !pp.players[1].IsZero():
				panic(throw.Impossible())
			default:
				pp.players[1] = ctx.SlotLink()
			}
			return false
		})
		if ctx.UseShared(sharedAccessFn).IsAvailable() {
			return ctx.Jump(p.stepChooseGame)
		}
	}

	pd := SharedPairData{players: [2]smachine.SlotLink{ctx.SlotLink()}}
	p.pair.sharedData = ctx.Share(&pd, smachine.ShareDataWakesUpAfterUse)
	if !ctx.Publish(freePlayer, p.pair.sharedData) {
		return ctx.Repeat(1)
	}

	return ctx.Sleep().ThenJump(func(ctx smachine.ExecutionContext) smachine.StateUpdate {
		if pd.game == nil {
			return ctx.Sleep().ThenRepeat()
		}
		return ctx.Jump(p.stepStartTheGame)
	})
}

func (p *PlayerSM) stepChooseGame(ctx smachine.ExecutionContext) smachine.StateUpdate {
	p.adapter.PrepareAsync(ctx, func(svc GameChooseService) smachine.AsyncResultFunc {
		game := svc.ChooseGame()
		return func(ctx smachine.AsyncResultContext) {

			ctx.WakeUp()
		}
	}).Start()

	return ctx.Sleep().ThenRepeat()
}

func (p *PlayerSM) stepStartTheGame(ctx smachine.ExecutionContext) smachine.StateUpdate {
}
