// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package defaults

import (
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

type tCase struct {
	env    map[string]string
	defFn  func() string
	expect string
}

var cases = []tCase{

	// ArtifactsDir checks
	{
		defFn:  ArtifactsDir,
		expect: ".artifacts",
	},
	{
		env: map[string]string{
			"INSOLAR_ARTIFACTS_DIR": "blah/bla",
		},
		defFn:  ArtifactsDir,
		expect: "blah/bla",
	},

	// LaunchnetDir checks
	{
		defFn:  LaunchnetDir,
		expect: ".artifacts/launchnet",
	},
	{
		env: map[string]string{
			"INSOLAR_ARTIFACTS_DIR": "blah/bla",
		},
		defFn:  LaunchnetDir,
		expect: "blah/bla/launchnet",
	},
	{
		env: map[string]string{
			"LAUNCHNET_BASE_DIR": "blah/bla",
		},
		defFn:  LaunchnetDir,
		expect: "blah/bla",
	},
}

func TestDefaults(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Patch separator differs on Windows")
	}

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
