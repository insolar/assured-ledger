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

package handle

import (
	"context"

	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/flow"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger/light/proc"
)

type HotObjects struct {
	dep  *proc.Dependencies
	meta payload.Meta
}

func NewHotObjects(dep *proc.Dependencies, meta payload.Meta) *HotObjects {
	return &HotObjects{
		dep:  dep,
		meta: meta,
	}
}

func (s *HotObjects) Present(ctx context.Context, f flow.Flow) error {
	logger := inslogger.FromContext(ctx)
	logger.Info("start hotObjects msg processing")

	msg, err := payload.Unmarshal(s.meta.Payload)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal message")
	}

	hots, ok := msg.(*payload.HotObjects)
	if !ok {
		return errors.New("received wrong message")
	}
	logger = logger.WithFields(map[string]interface{}{
		"pulse":  hots.Pulse,
		"jet_id": hots.JetID.DebugString(),
	})

	notificationLimit := s.dep.Config().MaxNotificationsPerPulse
	hdProc := proc.NewHotObjects(s.meta, hots.Pulse, hots.JetID, hots.Drop, hots.Indexes, notificationLimit)
	s.dep.HotObjects(hdProc)
	if err := f.Procedure(ctx, hdProc, false); err != nil {
		return err
	}

	logger.Info("finish hotObjects msg processing")
	return nil
}
