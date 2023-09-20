package configuration

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfiguration_Load_Default(t *testing.T) {
	holder := NewHolder("testdata/default.yml")
	err := holder.Load()
	require.NoError(t, err)

	cfg := NewConfiguration()
	// fmt.Println(ToString(cfg))
	require.Equal(t, cfg, *holder.Configuration)
}

func TestConfiguration_Load_Changed(t *testing.T) {
	holder := NewHolder("testdata/changed.yml")
	err := holder.Load()
	require.NoError(t, err)

	cfg := NewConfiguration()
	require.NotEqual(t, cfg, holder.Configuration)

	cfg.Log.Level = "Debug"
	require.Equal(t, cfg, *holder.Configuration)
}

func TestConfiguration_Load_Invalid(t *testing.T) {
	holder := NewHolder("testdata/invalid.yml")
	err := holder.Load()
	require.Error(t, err)
}

func TestConfiguration_LoadEnv(t *testing.T) {
	holder := NewHolder("testdata/default.yml")

	require.NoError(t, os.Setenv("INSOLAR_HOST_TRANSPORT_ADDRESS", "127.0.0.2:5555"))
	err := holder.Load()
	require.NoError(t, err)
	require.NoError(t, os.Unsetenv("INSOLAR_HOST_TRANSPORT_ADDRESS"))

	require.NoError(t, err)
	require.Equal(t, "127.0.0.2:5555", holder.Configuration.Host.Transport.Address)

	defaultCfg := NewConfiguration()
	require.Equal(t, "127.0.0.1:0", defaultCfg.Host.Transport.Address)
}

func TestMain(m *testing.M) {
	// backup and delete INSOLAR_ env variables, that may interfere with tests
	variablesBackup := make(map[string]string)
	for _, varPair := range os.Environ() {
		varPairSlice := strings.SplitN(varPair, "=", 2)
		varName, varValue := varPairSlice[0], varPairSlice[1]

		if strings.HasPrefix(varName, "INSOLAR_") {
			variablesBackup[varName] = varValue
			if err := os.Unsetenv(varName); err != nil {
				fmt.Printf("Failed to unset env variable '%s': %s\n",
					varName, err.Error())
			}
		}
	}

	// run tests
	exitCode := m.Run()

	// restore back variables
	for varName, varValue := range variablesBackup {
		if err := os.Setenv(varName, varValue); err != nil {
			fmt.Printf("Failed to unset env variable '%s' with '%s': %s\n",
				varName, varValue, err.Error())
		}

	}
	os.Exit(exitCode)
}
