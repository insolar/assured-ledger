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
	modeFlag         = "mode"
	heavyGenesisFlag = "heavy-genesis"
	pipelinePortFlag = "pipeline-port"
	passiveFlag      = "passive"
)

func appProcessCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "app-process",
		Short: "Run only application process in two-processes mode",
		Run: func(cmd *cobra.Command, args []string) {
			// todo: implement mode
			global.Infof("TODO: run app-process with flags config=%s role=%s port=%s",
				cmd.Flag(configFlag).Value.String(),
				cmd.Flag(roleFlag).Value.String(),
				cmd.Flag(pipelinePortFlag).Value.String(),
			)
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
		Long: `Two-processes mode is the main mode for production  production mode.
Time critical network and consensus components runs in separated process.
Use --passive flag for running the second process manually, for example in Kubernetes deployment`,
		Run: func(cmd *cobra.Command, args []string) {
			// todo: implement mode

			global.Infof("TODO: run node with flags config=%s role=%s port=%s",
				cmd.Flag(configFlag).Value.String(),
				cmd.Flag(roleFlag).Value.String(),
				cmd.Flag(pipelinePortFlag).Value.String(),
			)
			global.Fatalm(throw.NotImplemented())
		},
	}

	c.PersistentFlags().StringVarP(
		&role, roleFlag, "r", "", "set static role")

	c.PersistentFlags().StringVar(
		&genesisConfigPath, heavyGenesisFlag, "", "path to genesis config for heavy node")

	c.PersistentFlags().UintVar(
		&pipelinePort, pipelinePortFlag, 0, "port for connection between node processes")

	c.LocalFlags().BoolVar(&passive, passiveFlag, false,
		`If the flag is not set, then the first process runs the second process with app-process command and passes flags.
The pipeline-port flag will be added if it didn't set explicitly`)

	c.MarkFlagRequired(roleFlag) // nolint

	c.AddCommand(appProcessCommand())
	return c
}

func testCloudCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "cloud",
		Short: "Run single process cloud",
		Long:  "Single process cloud mode id needed for development and debug smart-contracts",
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
		Long: `In Single-process mode the time critical components runs in the same process.
Only for test purpose, Don't use in production`,
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
		Long:  "Headless mode is needed for testing consensus and network algorithms",
		Run: func(cmd *cobra.Command, args []string) {
			runHeadlessNetwork(cmd.Flag(configFlag).Value.String())
		},
	}
	c.MarkFlagRequired(configFlag) // nolint

	return c
}

func testCommands() *cobra.Command {
	c := &cobra.Command{
		Use:   "test",
		Short: "Group of subcommands for run nodes in development and testing modes",
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
		Short: "Generate configuration for a node in selected static role and run mode",
		Run: func(cmd *cobra.Command, args []string) {
			// todo: implement mode
			global.Infof("TODO: generate configuration with config=%s role=%s mode=%s",
				cmd.Flag(configFlag).Value.String(),
				cmd.Flag(roleFlag).Value.String(),
				cmd.Flag(modeFlag).Value.String(),
			)
			global.Fatalm(throw.NotImplemented())
		},
	}
	return c
}

func configValidateCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration for a node in selected static role and run mode",
		Run: func(cmd *cobra.Command, args []string) {
			// todo: implement mode
			global.Infof("TODO: validate configuration with config=%s role=%s mode=%s",
				cmd.Flag(configFlag).Value.String(),
				cmd.Flag(roleFlag).Value.String(),
				cmd.Flag(modeFlag).Value.String(),
			)
			global.Fatalm(throw.NotImplemented())
		},
	}
	return c
}

func configCommands() *cobra.Command {
	var (
		role string
		mode string
	)
	c := &cobra.Command{
		Use:   "config",
		Short: "Manage node configuration",
	}
	c.AddCommand(
		configGenerateCommand(),
		configValidateCommand(),
	)

	c.PersistentFlags().StringVarP(&role, roleFlag, "r", "", "set static role")
	c.PersistentFlags().StringVarP(&mode, modeFlag, "m", "", "set run mode")

	return c
}
