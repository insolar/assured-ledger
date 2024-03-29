package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/network/mandates"
	"github.com/insolar/assured-ledger/ledger-core/server"
	"github.com/insolar/assured-ledger/ledger-core/version"
)

const cmdName = "insolard"

func main() {
	fmt.Println("Version: ", version.GetFullVersion())

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
	certRole, err := readRoleFromCertificate(configPath)
	if err != nil {
		global.Fatal(errors.W(err, "readRole failed"))
	}
	role := member.GetPrimaryRoleFromString(roleString)
	if role != certRole {
		global.Fatal("Role from certificate and role from flag must be equal")
	}

	if err := psAgentLauncher(); err != nil {
		global.Warnf("Failed to launch gops agent: %s", err)
	}

	var s server.Server
	switch role {
	// supported roles
	case member.PrimaryRoleVirtual, member.PrimaryRoleLightMaterial:
	default:
		panic("unknown role")
	}
	cfg := readConfig(configPath)
	fmt.Printf("Starts with configuration:\n%s\n", configuration.ToString(&cfg))

	s = server.NewNode(cfg)
	s.Serve()
}

func runHeadlessNetwork(configPath string) {
	if err := psAgentLauncher(); err != nil {
		global.Warnf("Failed to launch gops agent: %s", err)
	}
	cfg := readConfig(configPath)
	fmt.Printf("Starts with configuration:\n%s\n", configuration.ToString(&cfg))

	server.NewHeadlessNetworkNodeServer(cfg).Serve()
}

func readRoleFromCertificate(path string) (member.PrimaryRole, error) {
	var err error
	cfg := configuration.NewHolder(path)

	err = cfg.Load()
	if err != nil {
		return member.PrimaryRoleUnknown, errors.W(err, "failed to load configuration from file")
	}

	data, err := ioutil.ReadFile(filepath.Clean(cfg.Configuration.CertificatePath))
	if err != nil {
		return member.PrimaryRoleUnknown, errors.Wrapf(
			err,
			"failed to read certificate from: %s",
			cfg.Configuration.CertificatePath,
		)
	}
	cert := mandates.AuthorizationCertificate{}
	err = json.Unmarshal(data, &cert)
	if err != nil {
		return member.PrimaryRoleUnknown, errors.W(err, "failed to parse certificate json")
	}
	return cert.GetRole(), nil
}
