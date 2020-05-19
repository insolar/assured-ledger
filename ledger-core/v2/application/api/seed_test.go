// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package api

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/fortytw2/leaktest"
	"github.com/gojuno/minimock/v3"
	"github.com/insolar/rpc/v2"
	"github.com/insolar/rpc/v2/json2"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/application/api/requester"
	"github.com/insolar/assured-ledger/ledger-core/v2/application/api/seedmanager"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulse"
)

func TestNodeService_GetSeed(t *testing.T) {
	defer leaktest.Check(t)()

	availableFlag := false
	mc := minimock.NewController(t)
	checker := NewAvailabilityCheckerMock(mc)
	checker = checker.IsAvailableMock.Set(func(ctx context.Context) (b1 bool) {
		return availableFlag
	})

	// 0 = false, 1 = pulse.ErrNotFound, 2 = another error
	pulseError := 0
	accessor := mockPulseAccessor(t)
	accessor = accessor.LatestMock.Set(func(ctx context.Context) (p1 pulse.Pulse, err error) {
		switch pulseError {
		case 1:
			return pulse.Pulse{}, pulse.ErrNotFound
		case 2:
			return pulse.Pulse{}, errors.New("fake error")
		default:
			return *pulse.GenesisPulse, nil
		}
	})

	runner := Runner{
		AvailabilityChecker: checker,
		PulseAccessor:       accessor,
		SeedManager:         seedmanager.New(),
		SeedGenerator:       seedmanager.SeedGenerator{},
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
		pulseError = 1

		err := s.GetSeed(&http.Request{}, &SeedArgs{}, &body, &requester.SeedReply{})
		require.Error(t, err)
		require.Equal(t, ServiceUnavailableErrorMessage, err.Error())
	})
	t.Run("pulse internal error", func(t *testing.T) {
		availableFlag = true
		pulseError = 2

		err := s.GetSeed(&http.Request{}, &SeedArgs{}, &body, &requester.SeedReply{})
		require.Error(t, err)
		require.Equal(t, InternalErrorMessage, err.Error())

		res, ok := err.(*json2.Error)
		require.True(t, ok)

		data, ok := res.Data.(requester.Data)
		require.True(t, ok)

		require.Equal(t, []string{"couldn't receive pulse", "fake error"}, data.Trace)
	})
}
