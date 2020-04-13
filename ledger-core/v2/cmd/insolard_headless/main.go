// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package main

import (
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/global"
	"github.com/insolar/assured-ledger/ledger-core/v2/server"
	"github.com/insolar/assured-ledger/ledger-core/v2/version"
)

func main() {
	var (
		configPath        string
		genesisConfigPath string
	)

	var rootCmd = &cobra.Command{
		Use: "insolard",
		Run: func(_ *cobra.Command, _ []string) {
			runInsolardServer(configPath, genesisConfigPath)
		},
	}
	rootCmd.Flags().StringVarP(&configPath, "config", "c", "", "path to config file")
	rootCmd.Flags().StringVarP(&genesisConfigPath, "heavy-genesis", "", "", "path to genesis config for heavy node")
	rootCmd.AddCommand(version.GetCommand("insolard"))
	err := rootCmd.Execute()
	if err != nil {
		global.Fatal("insolard execution failed:", err)
	}
}

func runInsolardServer(configPath string, _ string) {
	jww.SetStdoutThreshold(jww.LevelDebug)

	s := server.NewHeadlessNetworkNodeServer(configPath)
	s.Serve()
}
