package example

import (
	"math/rand"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewGameOfRandom(gameKey longbits.ByteString, players GamePlayers) GameStateMachine {
	return &GameRandom{GameTemplate: GameTemplate{gameKey: gameKey, players: players}}
}

var _ GameStateMachine = &GameRandom{}

type GameRandom struct {
	GameTemplate
	sharedData smachine.SharedDataLink
	prize      GameResult
}

type sharedGameRandom struct {
	nextBetPlayer    int
	highestBetPlayer int
	highestBet       float32
	done             bool
}

func (sd *sharedGameRandom) applyBet(players GamePlayers, myBetFn func() float32) bool {
	thisPlayer := players.PlayerIndex
	switch {
	case sd.nextBetPlayer < thisPlayer:
		return false
	case sd.nextBetPlayer > thisPlayer:
		panic(throw.IllegalState())
	}

	myBet := myBetFn()
	if sd.highestBet < myBet {
		sd.highestBet = myBet
		sd.highestBetPlayer = thisPlayer
	}
	sd.nextBetPlayer++

	if sd.nextBetPlayer >= len(players.Players) {
		sd.done = true
		// wake all players
		for i := range players.Players {
			players.Players[i].Call()
			// can also use ctx.CallBargeIn(players.Players[i]) - it is more efficient
		}
	} else {
		// wake up the next player
		players.Players[sd.nextBetPlayer].Call()
		// can also use ctx.CallBargeIn(players.Players[i]) - it is more efficient
	}
	return true
}

func (sd *sharedGameRandom) getResult() (GameResult, bool) {
	return GameResult{}, sd.done
}

func (g *GameRandom) GetSubroutineInitState(ctx smachine.SubroutineStartContext) smachine.InitFunc {
	ctx.SetSubroutineCleanupMode(smachine.SubroutineCleanupAliasesAndShares)
	return g.stepSetup
}

func (g *GameRandom) GetGameResult() GameResult {
	return g.prize
}

func (g *GameRandom) makeBet() float32 {
	return rand.Float32()
}

func (g *GameRandom) stepSetup(ctx smachine.InitializationContext) smachine.StateUpdate {
	if g.players.PlayerIndex > 0 {
		return ctx.Jump(g.stepGetShared)
	}

	sd := &sharedGameRandom{}

	// ShareDataDirect protects (sd) from invalidation by stop of the subroutine.
	// So (sd) will remain available while the slot is available and as long as relevant SharedDataLink is retained.
	sdl := ctx.Share(sd, smachine.ShareDataDirect)
	if !ctx.Publish(g.gameKey, sdl) {
		panic(throw.IllegalState())
	}
	g.sharedData = sdl

	sd.applyBet(g.players, g.makeBet)
	return ctx.Jump(g.waitForResult)
}

func (g *GameRandom) stepGetShared(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if sdl := ctx.GetPublishedLink(g.gameKey); !sdl.IsZero() {
		g.sharedData = sdl
		return ctx.Jump(g.stepBetRound)
	}
	return ctx.WaitAny().ThenRepeat()
}

// func (g *GameRandom) stepDone(ctx smachine.InitializationContext) smachine.StateUpdate {
// 	return ctx.Stop()
// }

func (g *GameRandom) stepBetRound(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if g.accessShared(ctx, func(sd *sharedGameRandom) bool {
		return sd.applyBet(g.players, g.makeBet)
	}) {
		return ctx.Sleep().ThenJump(g.waitForResult)
	}
	return ctx.Sleep().ThenRepeat()
}

func (g *GameRandom) waitForResult(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if g.accessShared(ctx, func(sd *sharedGameRandom) bool {
		if rs, ok := sd.getResult(); ok {
			g.prize = rs
			return true
		}
		return false
	}) {
		return ctx.Stop()
	}
	return ctx.Sleep().ThenRepeat()
}

func (g *GameRandom) accessShared(ctx smachine.ExecutionContext, fn func(*sharedGameRandom) bool) (result bool) {
	if g.sharedData.PrepareAccess(func(i interface{}) (wakeup bool) {
		if sd, ok := i.(*sharedGameRandom); ok {
			result = fn(sd)
			return false
		}
		panic(throw.IllegalState())
	}).TryUse(ctx).GetDecision().IsValid() {
		return
	}
	panic(throw.IllegalState())
}
