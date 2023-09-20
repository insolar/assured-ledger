//go:generate sm-uml-gen -f $GOFILE

package example

import (
	"math/rand"
	"reflect"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

// PlayerSM is an example of state machine to illustrate some basic patterns of use.
// PlayerSM looks for a 2nd player, then chooses a game (strategy pattern) and plays the game.
//
// This SM utilizes:
// - shared data + publish to connect players into pairs
// - sync object to handle limited capacity of rooms (PlayRoomLimiter), a pair of players require a room.
// - async adapter call to get a game to play
// - barge-in to wake up players
// - subroutine SM to actually play the game as PlayerSM's code knows no games

type PlayerSM struct {
	smachine.StateMachineDeclTemplate
	gamesToBePlayed int

	pair SharedPairDataLink

	adapter *GameChooseAdapter
}

var PlayRoomLimiter smachine.SyncLink = smsync.NewConditionalBool(true, "unlimited rooms").SyncLink()

/**** Methods of the State Machine Declaration ****/

// GetStateMachineDeclaration returns a declaration (class) of a state machine.
// Declaration is responsible for various initialization and processing capabilities.
//
// For simplicity - SM declaration could be the same object as SM.
// When implementing SM declaration please embed smachine.StateMachineDeclTemplate to avoid manual implementation of required methods.
func (p *PlayerSM) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return p
}

func (PlayerSM) InjectDependencies(sm smachine.StateMachine, link smachine.SlotLink, dj injector.DependencyInjector) {
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

	// Number of plays for this player before leaving
	p.gamesToBePlayed = 1 + rand.Intn(2)

	// Get to the next step
	return ctx.Jump(p.stepFindPair)
}

// freePlayer is a key to advertise a player without a pair
const freePlayer = "freePlayer"

var sharedPlayerPairType = reflect.TypeOf((*SharedPairData)(nil))

// stepFindPair looks for another player without a pair and creates a pair.
// When there is no player without a pair, this step will publish this player as "freePlayer" and will wait for another player to join.
func (p *PlayerSM) stepFindPair(ctx smachine.ExecutionContext) smachine.StateUpdate {

	// GetPublishedLink checks if there is something is registered under the given key
	if hasCollision, sd := false, ctx.GetPublishedLink(freePlayer); sd.IsOfType(sharedPlayerPairType) {

		// we enter this section of code only when there is another player who looks for a pair

		// set the published SharedDataLink into a typed accessor
		p.pair.sharedData = sd

		// now lets try to access the shared data
		// for this, an access function must be connected to the SharedDataLink with PrepareAccess()
		// then UseShared is invoked, and it returns information if the function was applied to the data under SharedDataLink
		if p.pair.PrepareAccess(func(pp *SharedPairData) (wakeup bool) {

			// this closure will be called only when it is safe access to the shared data
			// inside this closure both SM and the shared data are safely accessible
			switch {
			case pp.players[0].IsZero():
				panic(throw.Impossible())
			case !pp.players[1].IsZero():
				// another player got into the pair already
				hasCollision = true
				return false
			default:
				pp.players[1] = ctx.NewBargeIn().WithWakeUp()
			}

			// by returning true here we can wake up the owner of data
			// hence it is the first player in the pair
			return true
		}).TryUse(ctx).IsAvailable() {
			// access to the shared data call has succeeded
			if hasCollision {
				// lets try on next cycle
				// this result instructs Slot to let all other SMs run
				// then to repeat this step
				return ctx.Yield().ThenRepeat()
			}
			// we have a pair - lets get playing
			return ctx.Jump(p.stepStartTheGame)
		}
	}

	// we enter this section of code only when there is no other player who looks for a pair

	// prepare the shared data
	pd := &SharedPairData{}
	pd.players[0] = ctx.NewBargeIn().WithWakeUp()

	// share the data
	p.pair.sharedData = ctx.Share(pd, 0)
	// and publish it to make accessible to other SMs
	if !ctx.Publish(freePlayer, p.pair.sharedData) {
		// collision has happen - lets repeat after other SMs
		return ctx.Yield().ThenRepeat()
	}

	// the next part is a bit tricky
	// it can be made with separate methods of PlayerSM, but then it will require
	// PlayerSM struct to contain a field for SharedPairData.
	//
	// Instead of it, here we use closures to provide access to (pd) while we need a direct access.
	//
	// NB! As this SM has shared (pd), it is guaranteed that (pd) can be safely accessed by any step of this SM.
	// And we use it below to simplify initialization and avoid excessive use of UseShared()

	// here we gonna sleep until a wake up from the UseShared() above when a second player joined to this pair.
	return ctx.Sleep().ThenJump(func(ctx smachine.ExecutionContext) smachine.StateUpdate {
		// double check
		if pd.players[1].IsZero() {
			return ctx.Sleep().ThenRepeat()
		}
		// Un-publish our pair data to allow the next pair to connect.
		// But! the data remains shared, so it can be used by this pair of players
		ctx.Unpublish(freePlayer) // allow the next pair to connect

		// Lets tell to the world - who are in the pair
		ctx.Log().Trace(struct {
			string
			p0, p1 smachine.SlotID
		}{"pair",
			pd.players[0].StepLink().SlotID(),
			pd.players[1].StepLink().SlotID(),
		})

		// Now we will call an external service to get a game to play
		// To do this we use an adapter, that was injected into this SM.
		//
		// NB! PrepareAsync() call only prepares a call to adapter, but doesn't start it.
		p.adapter.PrepareAsync(ctx, func(svc GameChooseService) smachine.AsyncResultFunc {

			// this closure is executed on side of a service and it is safe to call functions of the service
			//
			// WARNING! It is NOT SAFE to access SM here. You have to store all required values and results into variables
			//
			gameFactory := svc.ChooseGame()
			if gameFactory == nil {
				// any panics here will be passed into SM and re-raised there
				panic(throw.IllegalValue())
			}

			return func(ctx smachine.AsyncResultContext) {
				// This closure is executed on SM, so in here it is SAFE to access SM
				//
				// WARNING! It is NOT SAFE to access the service HERE.
				//
				pd.gameFactory = gameFactory

				// instruct the slot to wake up from sleep
				ctx.WakeUp()
			}
		}).Start() // calling Start() send the async call immediately

		// NB! SM can initiate multiple async calls

		// Here we gonna sleep until a wake up by the async result
		return ctx.Sleep().ThenJump(func(ctx smachine.ExecutionContext) smachine.StateUpdate {
			// check if we've got a result
			if pd.gameFactory == nil {
				// if not then sleep again
				return ctx.Sleep().ThenRepeat()
			}
			// try to acquire the room
			if ctx.Acquire(PlayRoomLimiter).IsNotPassed() {
				// everything is busy
				// but Acquire() has remembered that we need access
				// so we've got a position in a sync queue and can get to sleep
				// wake up will be provided by a sync queue when we've got access
				return ctx.Sleep().ThenRepeat()
			}
			return ctx.Jump(p.stepStartTheGame)
		})
	})
}

// At this step we have a pair and we have a game to play.
// This step is unified for both players as UseShared() also works for owner of shared data.
func (p *PlayerSM) stepStartTheGame(ctx smachine.ExecutionContext) smachine.StateUpdate {

	var gameSM GameStateMachine

	// get access to the shared state for a pair of players this SM is a part of
	if p.pair.PrepareAccess(func(pp *SharedPairData) (wakeup bool) {
		if pp.gameFactory != nil {
			playerIndex := -1
			thisLink := ctx.SlotLink()

			// find SM's index in the pair
			for i := range pp.players {
				if pp.players[i].StepLink().SlotLink == thisLink {
					playerIndex = i
					break
				}
			}
			if playerIndex < 0 {
				panic(throw.IllegalState())
			}

			// create subroutine SM that knows how to play the game
			// because this SM doesn't know any game ...
			gameSM = pp.gameFactory(GamePlayers{pp.players[:], playerIndex})
			if gameSM == nil {
				panic(throw.IllegalState())
			}
		}
		return false
	}).TryUse(ctx).GetDecision() == smachine.Impossible {
		panic(throw.IllegalState())
	}

	if gameSM == nil {
		// we've been woken up before the first player got a game type
		// lets get to sleep then
		return ctx.Sleep().ThenRepeat()
	}

	// now we'll let the subroutine SM to play for this player
	return ctx.CallSubroutine(gameSM, nil, func(ctx smachine.SubroutineExitContext) smachine.StateUpdate {

		// this closure is called on termination of the subroutine SM
		// both stop and errors are handled and returned through (ctx)

		// also it is safe to access the subroutine SM here
		gameSM.GetGameResult()
		return ctx.Jump(p.stepNextGame)
	})
}

func (p *PlayerSM) stepNextGame(ctx smachine.ExecutionContext) smachine.StateUpdate {
	// release sync object / free up a room
	ctx.ReleaseAll()

	// did we play enough?
	if p.gamesToBePlayed <= 0 {
		return ctx.Stop()
	}

	p.gamesToBePlayed--

	// release shared data
	// it is safe to try to un-share data from another slot - it will be ignored
	ctx.Unshare(p.pair.sharedData)

	// reset local data
	p.pair = SharedPairDataLink{}

	// and take a small break
	sleepUntil := time.Now().Add(time.Millisecond * time.Duration(rand.Intn(500)))

	// here we use closure to avoid putting (sleepUntil) into SM's state
	return ctx.Jump(func(ctx smachine.ExecutionContext) smachine.StateUpdate {
		// This operation is a bit tricky - it combines a few operations together
		// When (sleepUntil) has passed then WaitAnyUntil works as Yield()
		// When there is less than a polling interval before (sleepUntil) then WaitAnyUntil works as WaitAny() and SM will be called on every cycle of SlotMachine.
		// Otherwise WaitAnyUntil works as Poll() and will wake up this SM in a polling interval.
		return ctx.WaitAnyUntil(sleepUntil).ThenRepeatOrJump(p.stepFindPair)
	})
}
