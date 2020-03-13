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

func main() {
	var (
	// configPath        string
	// genesisConfigPath string
	)

	var rootCmd = &cobra.Command{
		Use: "insolard",
		Run: func(_ *cobra.Command, _ []string) {
			// runInsolardServer(configPath, genesisConfigPath)
			global.Fatal("specify command")
		},
		Version: version.GetFullVersion(),
	}
	//rootCmd.Flags().StringVarP(&genesisConfigPath, "heavy-genesis", "", "", "path to genesis config for heavy node")
	rootCmd.AddCommand(
		fullNodeCommand(),
		appCommand(),
		netCommand(),
		testNetworkCommand(),
		version.GetCommand("insolard"),
	)

	err := rootCmd.Execute()
	if err != nil {
		global.Fatal("insolard execution failed:", err)
	}
}

// psAgentLauncher is a stub for gops agent launcher (available with 'debug' build tag)
var psAgentLauncher = func() error { return nil }

func runInsolardServer(configPath string, genesisConfigPath string) {
	jww.SetStdoutThreshold(jww.LevelDebug)

	role, err := readRole(configPath)
	if err != nil {
		global.Fatal(errors.Wrap(err, "readRole failed"))
	}

	if err := psAgentLauncher(); err != nil {
		global.Warnf("Failed to launch gops agent: %s", err)
	}

	switch role {
	case insolar.StaticRoleHeavyMaterial:
		s := server.NewHeavyServer(configPath, genesisConfigPath)
		s.Serve()
	case insolar.StaticRoleLightMaterial:
		s := server.NewLightServer(configPath)
		s.Serve()
	case insolar.StaticRoleVirtual:
		s := server.NewVirtualServer(configPath)
		s.Serve()
	}
}

func readRole(path string) (insolar.StaticRole, error) {
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
