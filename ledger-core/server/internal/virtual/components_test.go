// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package virtual

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/log"
)

func TestComponents(t *testing.T) {
	instestlogger.SetTestOutputWithErrorFilter(t, func(s string) bool {
		return !strings.Contains(s, "Failed to export to Prometheus: cannot register the collector")
	})

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
	cfg.Host.Transport.Address = "0.0.0.0:0"

	bootstrapComponents := initBootstrapComponents(ctx, cfg)
	cert := initCertificateManager(
		ctx,
		cfg,
		bootstrapComponents.CryptographyService,
		bootstrapComponents.KeyProcessor,
	)
	cm, stopWatermill := initComponents(
		ctx,
		cfg,
		bootstrapComponents.CryptographyService,
		bootstrapComponents.PlatformCryptographyScheme,
		bootstrapComponents.KeyStore,
		bootstrapComponents.KeyProcessor,
		cert,
	)
	require.NotNil(t, cm)
	require.NotNil(t, stopWatermill)

	err := cm.Init(ctx)
	require.NoError(t, err)
}
