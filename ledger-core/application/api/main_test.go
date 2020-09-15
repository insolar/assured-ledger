// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package api

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/network/mandates"
)

type MainAPISuite struct {
	suite.Suite
}

func (suite *MainAPISuite) TestNewApiRunnerNilConfig() {
	_, err := NewRunner(nil, global.Logger(), nil, nil, nil, nil, nil, nil, nil)
	suite.Contains(err.Error(), "config is nil")
}

func (suite *MainAPISuite) TestNewApiRunnerNoRequiredParams() {
	cfg := configuration.APIRunner{}
	_, err := NewRunner(&cfg, global.Logger(), nil, nil, nil, nil, nil, nil, nil)
	suite.Contains(err.Error(), "Address must not be empty")

	cfg.Address = "address:100"
	_, err = NewRunner(&cfg, global.Logger(), nil, nil, nil, nil, nil, nil, nil)
	suite.Contains(err.Error(), "RPC must exist")

	cfg.RPC = "test"
	_, err = NewRunner(&cfg, global.Logger(), nil, nil, nil, nil, nil, nil, nil)
	suite.Contains(err.Error(), "Missing openAPI spec file path")

	cfg.SwaggerPath = "spec/api-exported.yaml"
	_, err = NewRunner(&cfg, global.Logger(), nil, nil, nil, nil, nil, nil, nil)
	suite.NoError(err)
}

func TestMainTestSuite(t *testing.T) {
	instestlogger.SetTestOutput(t)

	ctx, _ := inslogger.WithTraceField(context.Background(), "APItests")
	http.DefaultServeMux = new(http.ServeMux)
	cfg := configuration.NewAPIRunner(false)
	cfg.SwaggerPath = "spec/api-exported.yaml"
	api, err := NewRunner(&cfg, global.Logger(), nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "new runner constructor")

	cm := mandates.NewCertificateManager(&mandates.Certificate{})
	api.CertificateManager = cm
	api.Start(ctx)

	suite.Run(t, new(MainAPISuite))

	api.Stop(ctx)
}
