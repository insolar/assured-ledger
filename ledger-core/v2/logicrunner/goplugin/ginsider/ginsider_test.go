// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build slowtest
// +build !race

// TODO test failed in race test call. added build tag to ignore this test
package ginsider

import (
	"fmt"
	"net"
	"net/rpc"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils"
)

var binaryPath string

func TestHealthCheck(t *testing.T) {
	t.Skip()
	protocol := "unix"
	socket := os.TempDir() + "/" + testutils.RandomString() + ".sock"

	tmpDir := insolar.ContractBuildTmpDir("ginsidertest-")
	defer os.RemoveAll(tmpDir)

	currentPath, err := os.Getwd()
	require.NoError(t, err)

	insgoccPath := binaryPath + "/insgocc"
	healthcheckPath := binaryPath + "/healthcheck"

	fmt.Println(insgoccPath)
	if _, err = os.Stat(healthcheckPath); err != nil {
		assert.Failf(t, "Binary file %s is not found, please run make build", healthcheckPath)
	}

	if !strings.HasPrefix(tmpDir, "/") {
		tmpDir, err = filepath.Rel(currentPath, tmpDir)
		require.NoError(t, err, "failed to compose relative path")
	}

	args := []string{
		"compile-genesis-plugins",
		"--no-proxy",
		"--sources-dir", currentPath,
		"-o", tmpDir,
		"healthcheck",
	}

	fmt.Println(insgoccPath, strings.Join(args, " "))
	gocc := exec.Command(insgoccPath, args...)
	gocc.Stderr = os.Stderr
	gocc.Stdout = os.Stdout
	err = gocc.Run()
	require.NoError(t, err, "failed to compile contract")

	// start GoInsider
	gi := NewGoInsider(tmpDir, protocol, socket)

	refString := "insolar:1MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI"
	ref, err := insolar.NewReferenceFromString(refString)
	require.NoError(t, err)

	healthcheckSoFile := path.Join(tmpDir, "healthcheck.so")
	err = gi.AddPlugin(*ref, healthcheckSoFile)
	require.NoError(t, err, "failed to add plugin by path "+healthcheckSoFile)

	prepareGoInsider(t, gi, protocol, socket)

	healthcheckArgs := []string{
		"-a", socket,
		"-p", protocol,
		"-r", refString,
	}

	cmd := exec.Command(healthcheckPath, healthcheckArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()

	assert.NoError(t, err)
}

func prepareGoInsider(t *testing.T, gi *GoInsider, protocol, socket string) {
	err := rpc.Register(&RPC{GI: gi})
	require.NoError(t, err, "can't register gi as rpc")
	listener, err := net.Listen(protocol, socket)
	require.NoError(t, err, "can't start listener")
	go rpc.Accept(listener)
}

func init() {
	var ok bool

	binaryPath, ok = os.LookupEnv("BIN_DIR")
	if !ok {
		wd, err := os.Getwd()
		binaryPath = filepath.Join(wd, "..", "..", "..", "bin")

		if err != nil {
			panic(err.Error())
		}
	}
}
