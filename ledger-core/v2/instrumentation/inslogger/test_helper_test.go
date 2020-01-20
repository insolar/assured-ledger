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

package inslogger

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/insolar/assured-ledger/ledger-core/v2/log"
)

const checkFails = false // set =true to check if failures are properly reported

func TestRedirectZlog(t *testing.T) {
	suite.Run(t, &suiteLogRedirect{adapter: "zerolog", checkFails: checkFails})
}

func TestRedirectBilog(t *testing.T) {
	suite.Run(t, &suiteLogRedirect{adapter: "bilog", checkFails: checkFails})
}

type suiteLogRedirect struct {
	suite.Suite
	adapter    string
	checkFails bool
}

func (v suiteLogRedirect) logger() log.Logger {
	return NewTestLoggerExt(v.T(), v.adapter)
}

func (v suiteLogRedirect) TestRedirectError() {
	if !v.checkFails {
		v.T().SkipNow()
		return
	}
	v.logger().Error("redirect to t.Log")
}

func (v suiteLogRedirect) TestRedirectPanic() {
	if !v.checkFails {
		v.T().SkipNow()
		return
	}
	require.Panics(v.T(), func() {
		v.logger().Panic("redirect to t.Log")
	})
}

func (v suiteLogRedirect) TestRedirectFatal() {
	if !v.checkFails {
		v.T().SkipNow()
		return
	}
	v.logger().Fatal("redirect to t.Log")
}

func (v suiteLogRedirect) TestRedirectWarn() {
	v.logger().Warn("redirect to t.Log")
}
