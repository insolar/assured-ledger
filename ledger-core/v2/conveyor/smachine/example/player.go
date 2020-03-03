/*
 * Copyright 2020 Insolar Network Ltd.
 * All rights reserved.
 * This material is licensed under the Insolar License version 1.0,
 * available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package example

import (
	"math/rand"
	"reflect"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

// PlayerSM is an example of state machine to illustrate some basic patterns of use.
// PlayerSM looks for a 2nd player, then chooses a game (strategy pattern) and plays the game.
type PlayerSM struct {
	smachine.StateMachineDeclTemplate
	gamesToBePlayed int

	pair SharedPairDataLink

	adapter *GameChooseAdapter
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

func (PlayerSM) InjectDependencies(sm smachine.StateMachine, link smachine.SlotLink, dj *injector.DependencyInjector) {
	dj.MustInject(&sm.(*PlayerSM).adapter)
}

// GetInitStateFor is a method of SM declaration that provides an init step for the given SM
// This method is invoked after injections and can rely on fields of SM.
// Nil returned by GetInitStateFor will cause an error for a caller.
func (PlayerSM) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {

	// Return the step of (sm) to start from.
	return sm.(*PlayerSM).stepInit
}

/**** Methods of the State Machine ****/

// stepInit receives InitializationContext parameter hence represent an initialization step of a state machine.
// At such step, SM can't wait or call adapters.
// Use this step to setup default handlers, like migration and error handlers.
// Also you can use this step together with InitChild() call by a parent to ensure that some actions are made before the parent will proceed,
// e.g. publish shared data or acquire a sync object.
func (p *PlayerSM) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {

	p.gamesToBePlayed = 1 + rand.Intn(16)
	// Make this player available from outside, e.g. for player rank table.
	// ctx.PublishGlobalAlias(ctx.SlotLink().SlotID())

	return ctx.Jump(p.stepFindPair)
}

// freePlayer is a key to advertise a player without a pair
const freePlayer = "freePlayer"

var sharedPlayerPairType = reflect.TypeOf((*SharedPairData)(nil))

// stepFindPair looks for another player without a pair and creates a pair.
// When there is no player without a pair, this step will publish this player as "freePlayer" and will wait for another player to join.
func (p *PlayerSM) stepFindPair(ctx smachine.ExecutionContext) smachine.StateUpdate {

	hasCollision := false
	// GetPublishedLink checks if there is something is registered under the given key
	if sd := ctx.GetPublishedLink(freePlayer); sd.IsOfType(sharedPlayerPairType) {
		p.pair.sharedData = sd
		// and uses it to create a pair
		if ctx.UseShared(p.pair.PrepareAccess(func(pp *SharedPairData) (wakeup bool) {
			switch {
			case pp.players[0].IsZero():
				panic(throw.Impossible())
			case !pp.players[1].IsZero():
				hasCollision = true
				return false
			default:
				pp.players[1] = ctx.NewBargeIn().WithWakeUp()
			}
			return true // wake up the first player
		})).IsAvailable() {
			if hasCollision {
				return ctx.WaitAny().ThenRepeat()
			}
			return ctx.Jump(p.stepStartTheGame)
		}
	}

	pd := &SharedPairData{}
	pd.players[0] = ctx.NewBargeIn().WithWakeUp()
	p.pair.sharedData = ctx.Share(pd, smachine.ShareDataWakesUpAfterUse)
	if !ctx.Publish(freePlayer, p.pair.sharedData) {
		return ctx.Yield().ThenRepeat()
	}

	return ctx.Sleep().ThenJump(func(ctx smachine.ExecutionContext) smachine.StateUpdate {
		if pd.players[1].IsZero() {
			return ctx.Sleep().ThenRepeat()
		}
		ctx.Unpublish(freePlayer) // allow the next pair to connect

		ctx.Log().Trace(struct {
			string
			p0, p1 smachine.SlotID
		}{"pair",
			pd.players[0].StepLink().SlotID(),
			pd.players[1].StepLink().SlotID(),
		})

		p.adapter.PrepareAsync(ctx, func(svc GameChooseService) smachine.AsyncResultFunc {
			gameFactory := svc.ChooseGame()
			if gameFactory == nil {
				panic(throw.IllegalValue())
			}

			return func(ctx smachine.AsyncResultContext) {
				pd.gameFactory = gameFactory
				ctx.WakeUp()
			}
		}).Start()

		return ctx.Sleep().ThenJump(func(ctx smachine.ExecutionContext) smachine.StateUpdate {
			if pd.gameFactory == nil {
				return ctx.Sleep().ThenRepeat()
			}
			return ctx.Jump(p.stepStartTheGame)
		})
	})
}

func (p *PlayerSM) stepStartTheGame(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var gameSM GameStateMachine

	if ctx.UseShared(p.pair.PrepareAccess(func(pp *SharedPairData) (wakeup bool) {
		if pp.gameFactory != nil {
			playerIndex := -1
			thisLink := ctx.SlotLink()
			for i := range pp.players {
				if pp.players[i].StepLink().SlotLink == thisLink {
					playerIndex = i
					break
				}
			}
			if playerIndex < 0 {
				panic(throw.IllegalState())
			}
			gameSM = pp.gameFactory(GamePlayers{pp.players[:], playerIndex})
			if gameSM == nil {
				panic(throw.IllegalState())
			}
		}
		return false
	})).GetDecision() == smachine.Impossible {
		panic(throw.IllegalState())
	}
	if gameSM == nil {
		return ctx.Sleep().ThenRepeat()
	}

	return ctx.CallSubroutine(gameSM, nil, func(ctx smachine.SubroutineExitContext) smachine.StateUpdate {
		gameSM.GetGameResult()
		return ctx.Jump(p.stepNextGame)
	})
}

func (p *PlayerSM) stepNextGame(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if p.gamesToBePlayed <= 0 {
		return ctx.Stop()
	}
	p.gamesToBePlayed--
	ctx.Unshare(p.pair.sharedData) // it is safe to try to un-share data from another slot - it will be ignored
	p.pair = SharedPairDataLink{}

	sleepUntil := time.Now().Add(time.Millisecond * time.Duration(rand.Intn(500)))
	return ctx.Jump(func(ctx smachine.ExecutionContext) smachine.StateUpdate {
		return ctx.WaitAnyUntil(sleepUntil).ThenRepeatOrJump(p.stepFindPair)
	})
}
