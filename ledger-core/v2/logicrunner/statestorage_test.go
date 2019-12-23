//
// Copyright 2019 Insolar Technologies GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package logicrunner

import (
	"context"
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/suite"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/executionregistry"
)

type StateStorageSuite struct{ suite.Suite }

func TestStateStorage(t *testing.T) { suite.Run(t, new(StateStorageSuite)) }

func (s *StateStorageSuite) generateContext() context.Context {
	return inslogger.TestContext(s.T())
}

func (s *StateStorageSuite) generatePulse() insolar.Pulse {
	return insolar.Pulse{PulseNumber: gen.PulseNumber()}
}

func (s *StateStorageSuite) TestOnPulse() {
	mc := minimock.NewController(s.T())
	defer mc.Finish()

	ctx := s.generateContext()
	pulse := s.generatePulse()
	objectRef := gen.Reference()

	ss := NewStateStorage(nil, nil, nil, nil, nil, nil, nil, nil)
	rawStateStorage := ss.(*stateStorage)

	{ // empty state storage
		msgs := ss.OnPulse(ctx, pulse)
		s.Len(msgs, 0)
		s.Len(rawStateStorage.brokers, 0)
		s.Len(rawStateStorage.registries, 0)
	}

	{ // state storage with empty execution registry
		rawStateStorage.registries[objectRef] = executionregistry.NewExecutionRegistryMock(mc).
			OnPulseMock.Return(nil).
			IsEmptyMock.Return(true)
		msgs := rawStateStorage.OnPulse(ctx, pulse)
		s.Len(msgs, 0)
		s.Len(rawStateStorage.brokers, 0)
		s.Len(rawStateStorage.registries, 0)
	}

	{ // state storage with non-empty execution registry
		rawStateStorage.registries[objectRef] = executionregistry.NewExecutionRegistryMock(mc).
			OnPulseMock.Return([]payload.Payload{&payload.StillExecuting{}}).
			IsEmptyMock.Return(false)
		msgs := rawStateStorage.OnPulse(ctx, pulse)
		s.Len(msgs, 1)
		s.Len(rawStateStorage.brokers, 0)
		s.Len(rawStateStorage.registries, 1)

		delete(rawStateStorage.registries, objectRef)
	}

	{ // state storage with execution registry and execution broker
		rawStateStorage.registries[objectRef] = executionregistry.NewExecutionRegistryMock(mc).
			OnPulseMock.Return(nil).
			IsEmptyMock.Return(true)
		rawStateStorage.brokers[objectRef] = NewExecutionBrokerIMock(mc).
			OnPulseMock.Return([]payload.Payload{&payload.ExecutorResults{}})
		msgs := rawStateStorage.OnPulse(ctx, pulse)
		s.Len(msgs, 1)
		s.Len(rawStateStorage.brokers, 0)
		s.Len(rawStateStorage.registries, 0)
	}

	{ // state storage with multiple objects
		rawStateStorage.brokers[objectRef] = NewExecutionBrokerIMock(mc).
			OnPulseMock.Return([]payload.Payload{&payload.ExecutorResults{}})
		rawStateStorage.registries[objectRef] = executionregistry.NewExecutionRegistryMock(mc).
			OnPulseMock.Return([]payload.Payload{&payload.StillExecuting{}}).
			IsEmptyMock.Return(false)
		msgs := rawStateStorage.OnPulse(ctx, pulse)
		s.Len(msgs[objectRef], 2)
		s.Len(rawStateStorage.brokers, 0)
		s.Len(rawStateStorage.registries, 1)

		delete(rawStateStorage.registries, objectRef)
	}

	{ // state storage with multiple objects
		objectRef1 := gen.Reference()
		objectRef2 := gen.Reference()

		rawStateStorage.brokers[objectRef1] = NewExecutionBrokerIMock(mc).
			OnPulseMock.Return([]payload.Payload{&payload.ExecutorResults{}})
		rawStateStorage.registries[objectRef1] = executionregistry.NewExecutionRegistryMock(mc).
			OnPulseMock.Return(nil).
			IsEmptyMock.Return(true)

		rawStateStorage.brokers[objectRef2] = NewExecutionBrokerIMock(mc).
			OnPulseMock.Return([]payload.Payload{&payload.ExecutorResults{}})
		rawStateStorage.registries[objectRef2] = executionregistry.NewExecutionRegistryMock(mc).
			OnPulseMock.Return([]payload.Payload{&payload.StillExecuting{}}).
			IsEmptyMock.Return(false)

		msgs := rawStateStorage.OnPulse(ctx, pulse)
		s.Len(msgs, 2)
		s.Len(msgs[objectRef1], 1)
		s.Len(msgs[objectRef2], 2)
		s.Len(rawStateStorage.brokers, 0)
		s.Len(rawStateStorage.registries, 1)
		s.NotNil(rawStateStorage.registries[objectRef2])
		s.Nil(rawStateStorage.brokers[objectRef2])

		delete(rawStateStorage.registries, objectRef2)
	}
}
