package defaults

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

type tCase struct {
	env    map[string]string
	defFn  func() string
	expect string
}

var BlahBla = func() string {
	// Path separator differs on Linux and Windows
	return filepath.Join("blah", "bla")
}()

var cases = []tCase{

	// ArtifactsDir checks
	{
		defFn:  ArtifactsDir,
		expect: ".artifacts",
	},
	{
		env: map[string]string{
			"INSOLAR_ARTIFACTS_DIR": BlahBla,
		},
		defFn:  ArtifactsDir,
		expect: BlahBla,
	},

	// LaunchnetDir checks
	{
		defFn:  LaunchnetDir,
		expect: func() string {
			return filepath.Join(".artifacts", "launchnet")
		}(),
	},
	{
		env: map[string]string{
			"INSOLAR_ARTIFACTS_DIR": BlahBla,
		},
		defFn:  LaunchnetDir,
		expect: func() string {
			return filepath.Join("blah","bla","launchnet")
		}(),
	},
	{
		env: map[string]string{
			"LAUNCHNET_BASE_DIR": BlahBla,
		},
		defFn:  LaunchnetDir,
		expect: BlahBla,
	},
}

func TestDefaults(t *testing.T) {
	for _, tc := range cases {
		for name, value := range tc.env {
			os.Setenv(name, value)
		}

		assert.Equal(t, tc.defFn(), tc.expect)

		for name := range tc.env {
			os.Setenv(name, "")
		}
	}
}
