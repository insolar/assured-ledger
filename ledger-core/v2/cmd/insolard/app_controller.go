// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package main

import (
	"github.com/spf13/cobra"
)

func fullNodeCommand() *cobra.Command {
	var (
		configPath      string
		isSingleProcess bool
		isIsolated      bool
	)
	c := &cobra.Command{
		Use:   "full-node",
		Short: "description",
		Run: func(cmd *cobra.Command, args []string) {
			// TODO:
			//gen, err := bootstrap.NewGenerator(configPath, certificatesOutDir)
			//if err != nil {
			//	global.Fatal("insolard execution failed:", err)
			//}
			//
			//err = gen.Run(context.Background())
			//if err != nil {
			//	global.Fatal("insolard execution failed:", err)
			//}
		},
	}
	c.Flags().StringVarP(
		&configPath, "config", "c", "", "path to bootstrap config")
	c.Flags().BoolVarP(
		&isSingleProcess, "single-process", "", false, "dir with certificate files")
	c.Flags().BoolVarP(
		&isIsolated, "isolated", "", false, "dir with certificate files")
	return c
}

func appCommand() *cobra.Command {
	var (
		configPath  string
		networkPort uint
	)
	c := &cobra.Command{
		Use:   "app",
		Short: "description",
		Run: func(cmd *cobra.Command, args []string) {
			// todo:
			panic("command app is not implmented")
		},
	}
	c.Flags().StringVarP(
		&configPath, "config", "c", "", "path to bootstrap config")
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
		Short: "description",
		Run: func(cmd *cobra.Command, args []string) {
			// todo:
			panic("command net is not implmented")
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
		Short: "description",
		Run: func(cmd *cobra.Command, args []string) {
			// todo:
			panic("command test-network is not implmented")
		},
	}
	return c
}
