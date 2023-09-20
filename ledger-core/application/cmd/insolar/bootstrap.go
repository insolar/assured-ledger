package main

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/insolar/assured-ledger/ledger-core/application/bootstrap"
)

func bootstrapCommand() *cobra.Command {
	var (
		configPath         string
		certificatesOutDir string
		properNames        bool
	)
	c := &cobra.Command{
		Use:   "bootstrap",
		Short: "creates files required for new network (keys, genesis config)",
		Run: func(cmd *cobra.Command, args []string) {
			gen, err := bootstrap.NewGenerator(configPath, certificatesOutDir)
			check("bootstrap failed to start", err)

			err = gen.Run(context.Background(), properNames)
			check("bootstrap failed", err)
		},
	}
	c.Flags().StringVarP(
		&configPath, "config", "c", "bootstrap.yaml", "path to bootstrap config")
	c.Flags().StringVarP(
		&certificatesOutDir, "certificates-out-dir", "o", "", "dir with certificate files")
	c.Flags().BoolVarP(
		&properNames, "propernames", "p", false, "Generate proper file names for kube deployment")
	return c
}
