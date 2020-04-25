// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

func (app *appCtx) valueHexDumpCommand() *cobra.Command {
	var dumpCmd = &cobra.Command{
		Use: "dump",
	}

	var key []byte
	keyArgCheck := func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("requires a argument with hex key value")
		}
		var err error
		key, err = hex.DecodeString(args[0])
		if err != nil {
			return fmt.Errorf("provided argument should be valid hex string: %v", err)
		}
		return nil
	}

	var dumpBinaryCmd = &cobra.Command{
		Use:   "bin",
		Short: "dump binary value by key",
		Args:  keyArgCheck,
		Run: func(_ *cobra.Command, _ []string) {
			db, close := openDB(app.dataDir)
			defer close()
			value, err := readValueByKey(db, key)
			if err != nil {
				fatalf("failed to get key from badger: %v", err)
			}
			_, err = io.Copy(os.Stdout, bytes.NewReader(value))
			if err != nil {
				fatalf("failed copy to stdin: %v", err)
			}
		},
	}

	dumpCmd.AddCommand(
		dumpBinaryCmd,
	)

	return dumpCmd
}
