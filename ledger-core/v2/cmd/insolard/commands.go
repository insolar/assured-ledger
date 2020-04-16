// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package main

import (
	"github.com/spf13/cobra"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/global"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

const (
	configFlag       = "config"
	roleFlag         = "role"
	heavyGenesisFlag = "heavy-genesis"
)

func appProcessCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "app-process",
		Short: "Start only application logic, connect to separated network process",
		Run: func(cmd *cobra.Command, args []string) {
			// todo: implement mode
			global.Fatalm(throw.NotImplemented())
		},
	}

	return c
}

func nodeCommand() *cobra.Command {
	var (
		role              string
		genesisConfigPath string
		passive           bool
		pipelinePort      uint
	)
	c := &cobra.Command{
		Use:   "node",
		Short: "Run node in production two-processes  mode",
		Run: func(cmd *cobra.Command, args []string) {
			// todo: implement mode

			global.Info(cmd.Flag(configFlag).Value.String())
			global.Fatalm(throw.NotImplemented())
		},
	}

	c.PersistentFlags().StringVarP(
		&role, roleFlag, "r", "", "set static role")

	c.PersistentFlags().StringVar(
		&genesisConfigPath, heavyGenesisFlag, "", "path to genesis config for heavy node")

	c.PersistentFlags().UintVar(
		&pipelinePort, "pipeline-port", 0, "port for connection between node processes")

	c.LocalFlags().BoolVar(
		&passive, "passive", false, "passive help text")

	c.MarkFlagRequired(roleFlag) // nolint

	c.AddCommand(appProcessCommand())
	return c
}

func testCloudCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "cloud",
		Short: "",
		Run: func(cmd *cobra.Command, args []string) {
			// todo: implement mode
			global.Fatalm(throw.NotImplemented())
		},
	}
	return c
}

func testNodeCommand() *cobra.Command {
	var (
		role              string
		genesisConfigPath string
	)
	c := &cobra.Command{
		Use:   "node",
		Short: "Run node in single-process mode",
		Run: func(cmd *cobra.Command, args []string) {
			global.Info("Starting node in single-process mode")
			runInsolardServer(cmd.Flag(configFlag).Value.String(), genesisConfigPath, role)
		},
	}

	c.PersistentFlags().StringVarP(
		&role, roleFlag, "r", "", "set static role")
	c.MarkFlagRequired(roleFlag) // nolint

	c.PersistentFlags().StringVar(
		&genesisConfigPath, heavyGenesisFlag, "", "path to genesis config for heavy node")

	return c
}

func testHeadlessCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "headless",
		Short: "Run node in headless mode",
		Run: func(cmd *cobra.Command, args []string) {
			runHeadlessNetwork(cmd.Flag(configFlag).Value.String())
		},
	}
	c.MarkFlagRequired("config") // nolint

	return c
}

func testCommands() *cobra.Command {
	c := &cobra.Command{
		Use: "test",
	}
	c.AddCommand(
		testCloudCommand(),
		testNodeCommand(),
		testHeadlessCommand(),
	)
	return c
}

func configGenerateCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "generate",
		Short: "",
		Run: func(cmd *cobra.Command, args []string) {
			// todo: implement mode
			global.Fatalm(throw.NotImplemented())
		},
	}
	return c
}

func configValidateCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "validate",
		Short: "",
		Run: func(cmd *cobra.Command, args []string) {
			// todo: implement mode
			global.Fatalm(throw.NotImplemented())
		},
	}
	return c
}

func configCommands() *cobra.Command {
	c := &cobra.Command{
		Use: "config",
	}
	c.AddCommand(
		configGenerateCommand(),
		configValidateCommand(),
	)
	return c
}
