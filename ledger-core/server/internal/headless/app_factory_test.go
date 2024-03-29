package headless

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insapp"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/log"
)

func TestAppFactory(t *testing.T) {
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

	server := insapp.New(cfg, nil, &AppComponent{})
	initer := server.GetInitializerForTest()

	cm, stopWatermill := initer.StartComponents(ctx, cfg, nil, func(context.Context, configuration.Log, string, string) context.Context {
		return ctx
	})

	require.NotNil(t, cm)
	require.Nil(t, stopWatermill)

	err := cm.Init(ctx)
	require.NoError(t, err)
}
