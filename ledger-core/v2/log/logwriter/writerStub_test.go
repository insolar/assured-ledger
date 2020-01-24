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

package logwriter

import (
	"errors"
	"strings"
)

type TestWriterStub struct {
	strings.Builder
	CloseCount int
	FlushCount int
	NoFlush    bool
}

func (w *TestWriterStub) Close() error {
	w.CloseCount++
	if w.CloseCount > 1 {
		return errClosed
	}
	return nil
}

func (w *TestWriterStub) Flush() error {
	w.FlushCount++
	if w.CloseCount >= 1 {
		return errClosed
	}
	if w.NoFlush {
		return errors.New("unsupported")
	}
	return nil
}
