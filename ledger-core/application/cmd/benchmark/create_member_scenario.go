package main

import (
	"github.com/insolar/assured-ledger/ledger-core/application/api/sdk"
)

type createMemberScenario struct {
	insSDK *sdk.SDK
}

func (s *createMemberScenario) canBeStarted() error {
	return nil
}

func (s *createMemberScenario) prepare(repetition int) {}

func (s *createMemberScenario) start(concurrentIndex int, repetitionIndex int) (string, error) {
	_, traceID, err := s.insSDK.CreateMember()
	return traceID, err
}

func (s *createMemberScenario) getBalanceCheckMembers() []sdk.Member {
	return nil
}
