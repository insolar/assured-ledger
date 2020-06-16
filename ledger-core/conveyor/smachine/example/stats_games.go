// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package example

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewStatsOfGame() StatsStateMachine {
	return &StatsOfTheGames{StatsTemplate: StatsTemplate{}}
}

const statsKey = "statsKey777"

var _ StatsStateMachine = &StatsOfTheGames{}

type StatsOfTheGames struct {
	StatsTemplate
	sharedData smachine.SharedDataLink
	gamesStats GamesStats
}

type sharedStatsState struct {
	nextBetPlayer    int
	highestBetPlayer int
	highestBet       float32
	done             bool
}

type SharedStatsData struct {
	statsFactory StatsFactoryFunc
}

type SharedStatsDataLink struct {
	sharedData smachine.SharedDataLink
}

func (v SharedStatsDataLink) PrepareAccess(fn func(*SharedStatsData) (wakeup bool)) smachine.SharedDataAccessor {
	if fn == nil {
		panic(throw.IllegalValue())
	}
	return v.sharedData.PrepareAccess(func(i interface{}) (wakeup bool) {
		return fn(i.(*SharedStatsData))
	})
}

func (stat *StatsOfTheGames) stepSetup(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Stop()
	return ctx.Jump(stat.stepUpdateStats)
	if sdl := ctx.GetPublishedLink(statsKey); !sdl.IsZero() {
		stat.sharedData = sdl
		return ctx.Jump(stat.stepUpdateStats)
	}

	sd := &sharedStatsState{}

	// ShareDataDirect protects (sd) from invalidation by stop of the subroutine.
	// So (sd) will remain available while the slot is available and as long as relevant SharedDataLink is retained.
	sdl := ctx.Share(sd, smachine.ShareDataDirect)
	if !ctx.Publish(statsKey, sdl) {
		panic(throw.IllegalState())
	}
	stat.sharedData = sdl
	return ctx.Jump(stat.stepUpdateStats)
}

func (stat *StatsOfTheGames) GetCurrentStats() GamesStats {
	return stat.gamesStats
}

func (stat *StatsOfTheGames) stepGetShared(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if sdl := ctx.GetPublishedLink(statsKey); !sdl.IsZero() {
		stat.sharedData = sdl
		return ctx.Jump(stat.stepUpdateStats)
	}
	return ctx.WaitAny().ThenRepeat()
}

func (stats *StatsOfTheGames) stepUpdateStats(ctx smachine.ExecutionContext) smachine.StateUpdate {
	stats.gamesStats.gamesPlayed ++
	return ctx.Stop()

	if stats.accessShared(ctx, func(sd *sharedStatsState) bool {
		stats.gamesStats.gamesPlayed ++
		return true
	}) {
		return ctx.Stop()
	}
	return ctx.Sleep().ThenRepeat()
}

func (stat *StatsOfTheGames) accessShared(ctx smachine.ExecutionContext, fn func(state *sharedStatsState) bool) (result bool) {
	if stat.sharedData.PrepareAccess(func(i interface{}) (wakeup bool) {
		if sd, ok := i.(*sharedStatsState); ok {
			result = fn(sd)
			return false
		}
		panic(throw.IllegalState())
	}).TryUse(ctx).GetDecision().IsValid() {
		return
	}
	panic(throw.IllegalState())
}

func (stats *StatsOfTheGames) GetStats() GamesStats {
	return stats.gamesStats
}

func (stats *StatsOfTheGames) GetSubroutineInitState(ctx smachine.SubroutineStartContext) smachine.InitFunc {
	ctx.SetSubroutineCleanupMode(smachine.SubroutineCleanupAliasesAndShares)
	return stats.stepSetup
}
