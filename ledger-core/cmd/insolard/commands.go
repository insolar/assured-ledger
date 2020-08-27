// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package main

import (
	"github.com/spf13/cobra"

	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
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
		Short: "Run application process only, in a two-processes mode",
		Long: `Two-processes mode implies two processes: first process contains time-critical network and consensus components
and runs with a higher priority and resource quota; second process contains all other components.`,
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
		Short: "Run node in production, in the two-processes  mode",
		Long: `Two-processes mode implies two processes: first process contains time-critical network and consensus components
and runs with a higher priority and resource quota; second process contains all other components.
Use the --passive flag for the first process if you want to run the second process manually, for example, in Kubernetes deployment.
If the first process is not run with this flag, it starts the second process automatically and passes all flags.`,
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
		&genesisConfigPath, heavyGenesisFlag, "", "path to genesis config for the Heavy node")

	c.PersistentFlags().UintVar(
		&pipelinePort, pipelinePortFlag, 0, "port for connection between node processes")

	c.LocalFlags().BoolVar(&passive, passiveFlag, false,
		`If this flag is not set, the first process runs the second process with the app-process command and passes all flags.
If not set explicitly, the pipeline-port flag will be added automatically, with a random value.`)

	c.MarkFlagRequired(roleFlag) // nolint

	c.AddCommand(appProcessCommand())
	return c
}

func testCloudCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "cloud",
		Short: "Run single process in the single-process-cloud",
		Long: `Single-process-cloud mode implies no consensus and no pulsar service (pulses are still generated).
It is used for development and debugging of smart contracts`,
		Run: func(cmd *cobra.Command, args []string) {
			// todo: implement mode
			runInsolardCloud(cmd.Flag(configFlag).Value.String())
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
		Short: "Run node in the single-process mode",
		Long: `Single-process mode implies that all components run in the same process: 
both the time-critical network and consensus components and app components.
Test- and development-use only! NEVER use in production!`,
		Run: func(cmd *cobra.Command, args []string) {
			global.Info("Starting node in single-process mode")
			runInsolardServer(cmd.Flag(configFlag).Value.String(), genesisConfigPath, role)
		},
	}

	c.PersistentFlags().StringVarP(
		&role, roleFlag, "r", "", "set static role")
	c.MarkFlagRequired(roleFlag) // nolint

	c.PersistentFlags().StringVar(
		&genesisConfigPath, heavyGenesisFlag, "", "path to genesis config for the Heavy node")

	return c
}

func testHeadlessCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "headless",
		Short: "Run node in the headless mode",
		Long: `Headless mode implies network communication and consensus but no running of smart contracts.
		It is used for testing of consensus and network algorithms.`,
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
		Short: "Group of subcommands for running nodes in the development and testing modes.",
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
		Short: "Generate configuration for a node in the selected static role and run mode.",
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
		Short: "Validate configuration for a node in the selected static role and run mode.",
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
