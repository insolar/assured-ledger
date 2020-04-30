// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package main

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"

	"github.com/insolar/assured-ledger/ledger-core/v2/certificate"
	"github.com/insolar/assured-ledger/ledger-core/v2/configuration"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/global"
	"github.com/insolar/assured-ledger/ledger-core/v2/server"
	"github.com/insolar/assured-ledger/ledger-core/v2/version"
)

const cmdName = "insolard"

func main() {
	var (
		configPath string
	)
	var rootCmd = &cobra.Command{
		Use:     cmdName,
		Version: version.GetFullVersion(),
	}
	rootCmd.AddCommand(
		nodeCommand(),
		testCommands(),
		configCommands(),
		version.GetCommand(cmdName),
	)

	rootCmd.PersistentFlags().StringVarP(&configPath, configFlag, "c", "", "path to config")
	rootCmd.MarkPersistentFlagRequired("config") // nolint

	err := rootCmd.Execute()
	if err != nil {
		global.Fatal("insolard execution failed:", err)
	}
}

// psAgentLauncher is a stub for gops agent launcher (available with 'debug' build tag)
var psAgentLauncher = func() error { return nil }

// nolint:unparam
func runInsolardServer(configPath, genesisConfigPath, roleString string) {
	jww.SetStdoutThreshold(jww.LevelDebug)

	certRole, err := readRoleFromCertificate(configPath)
	if err != nil {
		global.Fatal(errors.Wrap(err, "readRole failed"))
	}
	role := insolar.GetStaticRoleFromString(roleString)
	if role != certRole {
		global.Fatal("Role from certificate and role from flag must be equal")
	}

	if err := psAgentLauncher(); err != nil {
		global.Warnf("Failed to launch gops agent: %s", err)
	}

	switch role {
	case insolar.StaticRoleVirtual:
		s := server.NewVirtualServer(configPath)
		s.Serve()
	default:
		panic("unknown role")
	}
}

func runHeadlessNetwork(configPath string) {
	jww.SetStdoutThreshold(jww.LevelDebug)

	if err := psAgentLauncher(); err != nil {
		global.Warnf("Failed to launch gops agent: %s", err)
	}

	server.NewHeadlessNetworkNodeServer(configPath).Serve()
}

func readRoleFromCertificate(path string) (insolar.StaticRole, error) {
	var err error
	cfg := configuration.NewHolder(path)

	err = cfg.Load()
	if err != nil {
		return insolar.StaticRoleUnknown, errors.Wrap(err, "failed to load configuration from file")
	}

	data, err := ioutil.ReadFile(filepath.Clean(cfg.Configuration.CertificatePath))
	if err != nil {
		return insolar.StaticRoleUnknown, errors.Wrapf(
			err,
			"failed to read certificate from: %s",
			cfg.Configuration.CertificatePath,
		)
	}
	cert := certificate.AuthorizationCertificate{}
	err = json.Unmarshal(data, &cert)
	if err != nil {
		return insolar.StaticRoleUnknown, errors.Wrap(err, "failed to parse certificate json")
	}
	return cert.GetRole(), nil
}
