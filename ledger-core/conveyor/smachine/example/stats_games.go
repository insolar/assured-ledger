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

const statsDataKey = "statsDataKey777"

var _ StatsStateMachine = &StatsOfTheGames{}

type StatsOfTheGames struct {
	StatsTemplate
	sharedData smachine.SharedDataLink
	gamesStats GamesStats
}

type SharedStatsState struct {
	nextBetPlayer    int
	highestBetPlayer int
	highestBet       float32
	done             bool
	gamesPlayed		 int
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

var sharedStatsState SharedStatsState

func (stat *StatsOfTheGames) stepSetup(ctx smachine.InitializationContext) smachine.StateUpdate {
	if sdl := ctx.GetPublishedLink(statsDataKey); !sdl.IsZero() {
		stat.sharedData = sdl
		return ctx.Jump(stat.stepUpdateStats)
	}

	sd := &sharedStatsState

	// ShareDataDirect protects (sd) from invalidation by stop of the subroutine.
	// So (sd) will remain available while the slot is available and as long as relevant SharedDataLink is retained.
	sdl := ctx.Share(sd, smachine.ShareDataDirect)
	stat.sharedData = sdl
	if !ctx.Publish(statsDataKey, stat.sharedData) {
		panic(throw.IllegalState())
	}
	return ctx.Jump(stat.stepUpdateStats)
}

func (stats *StatsOfTheGames) stepUpdateStats(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if !stats.accessShared(ctx, func(sd *SharedStatsState) bool {
		sd.gamesPlayed ++
		stats.gamesStats.gamesPlayed = sd.gamesPlayed
		return true
	}) {
		panic(throw.IllegalState())
	}
	return ctx.Stop()
}

func (stat *StatsOfTheGames) accessShared(ctx smachine.ExecutionContext, fn func(state *SharedStatsState) bool) (result bool) {
	if stat.sharedData.PrepareAccess(func(i interface{}) (wakeup bool) {
		if sd, ok := i.(*SharedStatsState); ok {
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
