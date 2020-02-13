// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build slowtest

package heavy

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/application"
	"github.com/insolar/assured-ledger/ledger-core/v2/configuration"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/log"
)

func TestComponents(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "heavy-")
	defer os.RemoveAll(tmpdir)
	require.NoError(t, err)

	ctx := inslogger.UpdateLogger(context.Background(), func(logger log.Logger) (log.Logger, error) {
		return logger.Copy().WithBuffer(100, false).Build()
	})
	cfg := configuration.NewConfiguration()
	cfg.KeysPath = "testdata/bootstrap_keys.json"
	cfg.CertificatePath = "testdata/certificate.json"
	cfg.Metrics.ListenAddress = "0.0.0.0:0"
	cfg.APIRunner.Address = "0.0.0.0:0"
	cfg.AdminAPIRunner.Address = "0.0.0.0:0"
	cfg.APIRunner.SwaggerPath = "../../../application/api/spec/api-exported.yaml"
	cfg.AdminAPIRunner.SwaggerPath = "../../../application/api/spec/api-exported.yaml"
	cfg.Ledger.Storage.DataDirectory = tmpdir
	cfg.Exporter.Addr = ":0"

	_, err = newComponents(ctx, cfg, application.GenesisHeavyConfig{Skip: true})
	require.NoError(t, err)
}
