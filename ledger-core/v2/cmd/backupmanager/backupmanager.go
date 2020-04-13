// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package main

import (
	"context"
	"math"

	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/insolar/assured-ledger/ledger-core/v2/configuration"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/jet"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/store"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger/heavy/executor"
	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/global"
)

func exitWithError(err error) {
	global.Error(err.Error())
	Exit(1)
}

func exit(err error) {
	if err == nil {
		Exit(0)
	}
	exitWithError(err)
}

func stopDB(ctx context.Context, bdb *store.BadgerDB, originalError error) {
	closeError := bdb.Stop(ctx)
	if closeError != nil {
		err := errors.Wrap(originalError, "failed to stop store.BadgerDB")
		global.Error(err.Error())
	}
	if originalError != nil {
		global.Error(originalError.Error())
	}
	if originalError != nil || closeError != nil {
		Exit(1)
	}
}

type BadgerLogger struct {
	log.Logger
}

func (b BadgerLogger) Warningf(fmt string, args ...interface{}) {
	b.Warnf(fmt, args...)
}

var (
	badgerLogger BadgerLogger
)

func finalizeLastPulse(ctx context.Context, bdb *store.BadgerDB) (insolar.PulseNumber, error) {
	pulsesDB := pulse.NewDB(bdb)

	jetKeeper := executor.NewJetKeeper(jet.NewDBStore(bdb), bdb, pulsesDB)
	global.Info("Current top sync pulse: ", jetKeeper.TopSyncPulse().String())

	it := bdb.NewIterator(executor.BackupStartKey(math.MaxUint32), true)
	if !it.Next() {
		return 0, errors.New("no backup start keys")
	}

	pulseNumber := insolar.NewPulseNumber(it.Key())
	global.Info("Found last backup start key: ", pulseNumber.String())

	if pulseNumber < jetKeeper.TopSyncPulse() {
		return 0, errors.New("Found last backup start key must be grater or equal to top sync pulse")
	}

	if !jetKeeper.HasAllJetConfirms(ctx, pulseNumber) {
		return 0, errors.New("data is inconsistent. pulse " + pulseNumber.String() + " must have all confirms")
	}

	global.Info("All jets confirmed for pulse: ", pulseNumber.String())
	err := jetKeeper.AddBackupConfirmation(ctx, pulseNumber)
	if err != nil {
		return 0, errors.Wrap(err, "failed to add backup confirmation for pulse "+pulseNumber.String())
	}

	if jetKeeper.TopSyncPulse() != pulseNumber {
		return 0, errors.New("new top sync pulse must be equal to last backuped")

	}

	return jetKeeper.TopSyncPulse(), nil
}

// prepareBackup does:
// 1. finalize last pulse, since it comes not finalized ( since we set finalization after success of backup )
// 2. gets last backuped version
// 3. write 2. to file
func prepareBackup(dbDir string) {
	global.Info("prepareBackup. dbDir: ", dbDir)

	ops := badger.DefaultOptions(dbDir)
	ops.Logger = badgerLogger
	bdb, err := store.NewBadgerDB(ops)
	if err != nil {
		err := errors.Wrap(err, "failed to open DB")
		exitWithError(err)
	}
	global.Info("DB is opened")
	ctx := context.Background()

	topSyncPulse, err := finalizeLastPulse(ctx, bdb)
	if err != nil {
		err = errors.Wrap(err, "failed to finalizeLastPulse")
		stopDB(ctx, bdb, err)
	}

	stopDB(ctx, bdb, nil)
	global.Info("New top sync pulse: ", topSyncPulse.String())
}

func parsePrepareBackupParams() *cobra.Command {
	var (
		dbDir string
	)
	var prepareBackupCmd = &cobra.Command{
		Use:   "prepare_backup",
		Short: "prepare backup for usage",
		Run: func(cmd *cobra.Command, args []string) {
			prepareBackup(dbDir)
		},
	}

	dbDirFlagName := "db-dir"
	prepareBackupCmd.Flags().StringVarP(
		&dbDir, dbDirFlagName, "d", "", "directory where new DB will be created (required)")

	err := cobra.MarkFlagRequired(prepareBackupCmd.Flags(), dbDirFlagName)
	if err != nil {
		err = errors.Wrap(err, "failed to set required param: "+dbDirFlagName)
		exitWithError(err)
	}

	return prepareBackupCmd
}

func parseInputParams() {
	var rootCmd = &cobra.Command{
		Use:   "backupmanager",
		Short: "backupmanager is the command line client for managing backups",
	}

	rootCmd.AddCommand(parsePrepareBackupParams())

	exit(rootCmd.Execute())
}

func initLogger() context.Context {
	global.SetLevel(log.DebugLevel)

	cfg := configuration.NewLog()
	cfg.Level = "Debug"
	cfg.Formatter = "text"

	ctx, logger := inslogger.InitNodeLogger(context.Background(), cfg, "", "backuper")
	badgerLogger.Logger = logger.WithField("component", "badger")

	return ctx
}

func initExit(ctx context.Context) {
	InitExitContext(inslogger.FromContext(ctx))
	AtExit("logger-flusher", func() error {
		global.Flush()
		return nil
	})
}

func main() {
	ctx := initLogger()
	initExit(ctx)
	parseInputParams()
}
