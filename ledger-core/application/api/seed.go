// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat/memstor"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/rpc/v2"
	"github.com/insolar/rpc/v2/json2"

	"github.com/insolar/assured-ledger/ledger-core/application/api/instrumenter"
	"github.com/insolar/assured-ledger/ledger-core/application/api/requester"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
)

// SeedArgs is arguments that Seed service accepts.
type SeedArgs struct{}

// NodeService is a service that provides API for getting new seed and status.
type NodeService struct {
	runner *Runner
}

// NewNodeService creates new Node service instance.
func NewNodeService(runner *Runner) *NodeService {
	return &NodeService{runner: runner}
}

// Get returns new active seed.
//
//	Request structure:
//	{
//		"jsonrpc": "2.0",
//		"method": "node.getSeed",
//		"id": str|int|null
//	}
//
//	Response structure:
//	{
//		"jsonrpc": "2.0",
//		"result": {
//			"seed": str, // correct seed for new Call request
//			"traceID": str // traceID for request
//		},
//		"id": str|int|null // same as in request
//	}
//
func (s *NodeService) getSeed(ctx context.Context, _ *http.Request, _ *SeedArgs, reply *requester.SeedReply) error {
	traceID := instrumenter.GetTraceID(ctx)

	seed, err := s.runner.SeedGenerator.Next()
	if err != nil {
		return err
	}

	p, err := s.runner.PulseAccessor.LatestTimeBeat()
	if err != nil {
		return throw.W(err, "couldn't receive pulse")
	}
	s.runner.SeedManager.Add(*seed, p.PulseNumber)

	reply.Seed = seed[:]
	reply.TraceID = traceID

	return nil
}

func (s *NodeService) GetSeed(r *http.Request, args *SeedArgs, _ *rpc.RequestBody, reply *requester.SeedReply) error {
	ctx, instr := instrumenter.NewMethodInstrument("NodeService.getSeed")
	defer instr.End()

	msg := fmt.Sprint("Incoming request: ", r.RequestURI)
	instr.Annotate(msg)

	logger := inslogger.FromContext(ctx)
	logger.Info("[ NodeService.getSeed ] ", msg)

	if !s.runner.AvailabilityChecker.IsAvailable(ctx) {
		logger.Warn("[ NodeService.getSeed ] API is not available")

		instr.SetError(throw.New(ServiceUnavailableErrorMessage), ServiceUnavailableErrorShort)
		return &json2.Error{
			Code:    ServiceUnavailableError,
			Message: ServiceUnavailableErrorMessage,
			Data: requester.Data{
				TraceID: instr.TraceID(),
			},
		}
	}

	err := s.getSeed(ctx, r, args, reply)
	if err != nil {
		if strings.Contains(err.Error(), memstor.ErrNotFound.Error()) {
			logger.Warn("[ NodeService.getSeed ] failed to execute: ", err.Error())

			instr.SetError(throw.New(ServiceUnavailableErrorMessage), ServiceUnavailableErrorShort)
			return &json2.Error{
				Code:    ServiceUnavailableError,
				Message: ServiceUnavailableErrorMessage,
				Data: requester.Data{
					TraceID: instr.TraceID(),
				},
			}

		}
		logger.Error("[ NodeService.getSeed ] failed to execute: ", err.Error())

		instr.SetError(err, InternalErrorShort)
		return &json2.Error{
			Code:    InternalError,
			Message: InternalErrorMessage,
			Data: requester.Data{
				Trace:   strings.Split(err.Error(), ";\t"),
				TraceID: instr.TraceID(),
			},
		}
	}
	return nil
}
