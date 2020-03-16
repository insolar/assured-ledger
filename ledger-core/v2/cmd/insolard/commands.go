// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package main

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/log/global"
	"github.com/spf13/cobra"
)

func fullNodeCommand() *cobra.Command {
	var (
		configPath        string
		genesisConfigPath string
		isSingleProcess   bool
		isIsolated        bool
	)
	c := &cobra.Command{
		Use:   "full-node",
		Short: "description",
		Run: func(cmd *cobra.Command, args []string) {
			if isSingleProcess && isIsolated {
				global.Fatal("--single-process and --isolated flags could not be set in the same time")
			}

			if isSingleProcess {
				global.Info("Starting full-node in single-process mode")
				runInsolardServer(configPath, genesisConfigPath)
			}

			if isIsolated {
				global.Info("Starting full-node in isolated mode")
				global.Fatal("full-node in isolated mode is not implemented")
			}

			global.Fatal("--single-process or --isolated flags must be set")
		},
	}
	c.Flags().StringVarP(
		&configPath, "config", "c", "", "path to config")
	c.Flags().StringVarP(
		&genesisConfigPath, "heavy-genesis", "", "", "path to genesis config for heavy node")
	c.Flags().BoolVarP(
		&isSingleProcess, "single-process", "", false, "start full node in single process mode")
	c.Flags().BoolVarP(
		&isIsolated, "isolated", "", false, "start full node in isolated mode")
	return c
}

func appCommand() *cobra.Command {
	var (
		configPath        string
		genesisConfigPath string
		networkPort       uint
	)
	c := &cobra.Command{
		Use:   "app",
		Short: "Start only application logic, connect to separated network process",
		Run: func(cmd *cobra.Command, args []string) {
			// todo: implement mode
			global.Fatal("command app is not implemented")
		},
	}
	c.Flags().StringVarP(
		&configPath, "config", "c", "", "path to bootstrap config")
	c.Flags().StringVarP(
		&genesisConfigPath, "heavy-genesis", "", "", "path to genesis config for heavy node")
	c.Flags().UintVarP(
		&networkPort, "network-port", "", 0, "path to bootstrap config")
	return c
}

func netCommand() *cobra.Command {
	var (
		configPath string
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
	c.Flags().BoolVarP(
		&isHeadless, "headless", "", false, "dir with certificate files")
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
