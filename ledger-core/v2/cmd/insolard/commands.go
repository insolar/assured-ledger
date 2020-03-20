// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package main

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/log/global"
	"github.com/spf13/cobra"
)

func fullNodeIsolatedCommand() *cobra.Command {
	var (
		configPath        string
		genesisConfigPath string
	)
	c := &cobra.Command{
		Use:   "isolated",
		Short: "start full node in isolated mode",
		Run: func(cmd *cobra.Command, args []string) {
			// todo: implement mode
			global.Fatal("command isolated is not implemented")
		},
	}
	c.PersistentFlags().StringVarP(
		&configPath, "config", "c", "", "path to config")
	c.Flags().StringVar(
		&genesisConfigPath, "heavy-genesis", "", "path to genesis config for heavy node")

	c.MarkFlagRequired("config")
	return c
}

func fullNodeSingleProcessCommand() *cobra.Command {
	var (
		configPath        string
		genesisConfigPath string
	)
	c := &cobra.Command{
		Use:   "single-process",
		Short: "start full node in single process mode",
		Run: func(cmd *cobra.Command, args []string) {
			global.Info("Starting full-node in single-process mode")
			runInsolardServer(configPath, genesisConfigPath)
		},
	}
	c.Flags().StringVarP(
		&configPath, "config", "c", "", "path to config")
	c.Flags().StringVar(
		&genesisConfigPath, "heavy-genesis", "", "path to genesis config for heavy node")
	return c
}

func fullNodeCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "full-node",
		Short: "description",
	}

	c.AddCommand(fullNodeIsolatedCommand(), fullNodeSingleProcessCommand())
	return c
}

func testAppHeavyCommand() *cobra.Command {
	var (
		configPath        string
		genesisConfigPath string
		bridgePort        uint
	)
	c := &cobra.Command{
		Use:   "heavy",
		Short: "Start only application logic with Heavy role",
		Run: func(cmd *cobra.Command, args []string) {
			// todo: implement mode
			global.Fatal("command test-network is not implemented")
		},
	}
	c.Flags().StringVarP(
		&configPath, "config", "c", "", "path to bootstrap config")
	c.Flags().StringVar(
		&genesisConfigPath, "heavy-genesis", "", "path to genesis config for heavy node")
	c.Flags().UintVar(
		&bridgePort, "bridge-port", 0, "bridge port for Net <-> App connection")
	return c
}

func testAppVirtualCommand() *cobra.Command {
	var (
		configPath string
		bridgePort uint
	)
	c := &cobra.Command{
		Use:   "virtual",
		Short: "Start only application logic with Virtual role",
		Run: func(cmd *cobra.Command, args []string) {
			// todo: implement mode
			global.Fatal("subcommand virtual is not implemented")
		},
	}
	c.Flags().StringVarP(
		&configPath, "config", "c", "", "path to bootstrap config")
	c.Flags().UintVar(
		&bridgePort, "bridge-port", 0, "bridge port for Net <-> App connection")
	return c
}

func appCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "app",
		Short: "Start only application logic, connect to separated network process",
	}
	c.AddCommand(
		testAppHeavyCommand(),
		testAppVirtualCommand(),
	)
	return c
}

func netCommand() *cobra.Command {
	var (
		configPath string
		bridgePort uint
		isHeadless bool
	)
	c := &cobra.Command{
		Use:   "net",
		Short: "Start only network and consensus logic",
		Run: func(cmd *cobra.Command, args []string) {
			// todo: implement mode
			global.Fatal("command net is not implemented")
		},
	}
	c.Flags().StringVarP(
		&configPath, "config", "c", "", "path to bootstrap config")
	c.Flags().BoolVar(
		&isHeadless, "headless", false, "dir with certificate files")
	c.Flags().UintVar(
		&bridgePort, "bridge-port", 0, "bridge port for Net <-> App connection")
	return c
}

func testNetworkCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "test-network",
		Short: "Start multi-role configuration with mocked network for debug and test purpose",
		Run: func(cmd *cobra.Command, args []string) {
			// todo: implement mode
			global.Fatal("command test-network is not implemented")
		},
	}
	return c
}
