// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/insolar/rpc/v2"
	"github.com/insolar/rpc/v2/json2"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/application/api/requester"
	"github.com/insolar/assured-ledger/ledger-core/application/api/seedmanager"
	"github.com/insolar/assured-ledger/ledger-core/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
)

func TestNodeService_GetSeed(t *testing.T) {
	defer testutils.LeakTester(t,
		goleak.IgnoreTopFunction("github.com/insolar/assured-ledger/ledger-core/application/api/seedmanager.NewSpecified.func1"))

	instestlogger.SetTestOutputWithErrorFilter(t, func(s string) bool {
		return !strings.Contains(s, "another error")
	})

	availableFlag := false
	mc := minimock.NewController(t)
	checker := NewAvailabilityCheckerMock(mc)
	checker = checker.IsAvailableMock.Set(func(ctx context.Context) (b1 bool) {
		return availableFlag
	})

	accessor := beat.NewAppenderMock(t)
	accessor.LatestTimeBeatMock.Return(pulsestor.GenesisPulse, nil)

	runner := Runner{
		AvailabilityChecker: checker,
		PulseAccessor:       accessor,
		SeedManager:         seedmanager.New(),
		SeedGenerator:       seedmanager.RandomSeedGenerator,
	}
	s := NewNodeService(&runner)
	defer runner.SeedManager.Stop()

	body := rpc.RequestBody{Raw: []byte(`{}`)}

	t.Run("success", func(t *testing.T) {
		availableFlag = true
		reply := requester.SeedReply{}

		err := s.GetSeed(&http.Request{}, &SeedArgs{}, &body, &reply)
		require.Nil(t, err)
		require.NotEmpty(t, reply.Seed)
	})
	t.Run("service not available", func(t *testing.T) {
		availableFlag = false

		err := s.GetSeed(&http.Request{}, &SeedArgs{}, &body, &requester.SeedReply{})
		require.Error(t, err)
		require.Equal(t, ServiceUnavailableErrorMessage, err.Error())
	})
	t.Run("pulse not found", func(t *testing.T) {
		availableFlag = true
		accessor.LatestTimeBeatMock.Return(beat.Beat{}, errors.New("some error"))

		err := s.GetSeed(&http.Request{}, &SeedArgs{}, &body, &requester.SeedReply{})
		require.Error(t, err)
		require.Equal(t, ServiceUnavailableErrorMessage, err.Error())
	})

	t.Run("pulse internal error", func(t *testing.T) {
		availableFlag = true
		runner.SeedGenerator = func() (seedmanager.Seed, error) {
			return seedmanager.Seed{}, errors.New("another error")
		}

		err := s.GetSeed(&http.Request{}, &SeedArgs{}, &body, &requester.SeedReply{})
		require.Error(t, err)
		require.Equal(t, InternalErrorMessage, err.Error())

		res, ok := err.(*json2.Error)
		require.True(t, ok)

		data, ok := res.Data.(requester.Data)
		require.True(t, ok)

		require.Equal(t, []string{"another error"}, data.Trace)
	})
}
