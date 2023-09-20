package api_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/insolar/assured-ledger/ledger-core/application/api"
	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
)

func TestAvailabilityChecker_UpdateStatus(t *testing.T) {
	instestlogger.SetTestOutputWithErrorFilter(t, func(s string) bool {
		expectedError := strings.Contains(s, "no response or bad StatusCode") ||
			strings.Contains(s, "Can't decode body: invalid character '}'") ||
			strings.Contains(s, "connection refused") ||
			strings.Contains(s, "No connection could be made because the target machine actively refused it")
		return !expectedError
	})
	ctx, _ := inslogger.InitNodeLoggerByGlobal("", "")

	defer testutils.LeakTester(t,
		goleak.IgnoreTopFunction("github.com/insolar/assured-ledger/ledger-core/application/api/seedmanager.NewSpecified.func1"))

	counter := 0

	keeper := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch counter {
			case 0:
				w.WriteHeader(http.StatusOK)
				_, err := fmt.Fprintln(w, "{\"available\": false}")
				require.NoError(t, err)
			case 1:
				w.WriteHeader(http.StatusBadRequest)
			case 2:
				w.WriteHeader(http.StatusOK)
				_, err := fmt.Fprintln(w, "{\"test\": }")
				require.NoError(t, err)
			default:
				w.WriteHeader(http.StatusOK)
				_, err := fmt.Fprintln(w, "{\"available\": true}")
				require.NoError(t, err)
			}
			counter += 1
		}))

	var checkPeriod uint = 1
	config := configuration.AvailabilityChecker{
		Enabled:        true,
		KeeperURL:      keeper.URL,
		RequestTimeout: 2,
		CheckPeriod:    checkPeriod,
	}

	nc := api.NewNetworkChecker(config)
	require.False(t, nc.IsAvailable(ctx))

	// counter = 0
	nc.UpdateAvailability(ctx)
	require.False(t, nc.IsAvailable(ctx))

	// counter = 1
	nc.UpdateAvailability(ctx)
	require.False(t, nc.IsAvailable(ctx))

	// counter = 2, bad response body
	nc.UpdateAvailability(ctx)
	require.False(t, nc.IsAvailable(ctx))

	// counter default
	nc.UpdateAvailability(ctx)
	require.True(t, nc.IsAvailable(ctx))

	keeper.Close()
	nc.UpdateAvailability(ctx)
	require.False(t, nc.IsAvailable(ctx))
}
