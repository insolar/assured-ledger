// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

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
