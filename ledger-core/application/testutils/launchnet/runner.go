// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build !cloud_with_consensus

package launchnet

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
)

func isCloudMode() bool {
	cloudMode := os.Getenv("CLOUD_MODE")
	v, err := strconv.ParseBool(cloudMode)
	return err == nil && v
}

// Run starts launchnet before execution of callback function (cb) and stops launchnet after.
// Returns exit code as a result from calling callback function.
func Run(cb func() int) int {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	ctx, abort := context.WithCancel(context.Background())
	defer abort()

	setupFunc := setup
	if isCloudMode() {
		cr := CloudRunner{}
		cr.PrepareConfig()
		setupFunc = cr.SetupCloud
	}
	teardown, err := setupFunc(ctx)
	defer teardown()
	if err != nil {
		fmt.Println("error while setup, skip tests: ", err)
		return 1
	}

	go func() {
		sig := <-c
		abort()
		fmt.Printf("Got %s signal. Aborting...\n", sig)
		teardown()

		os.Exit(2)
	}()

	pulseWatcher, config := pulseWatcherPath()

	code := cb()

	if code != 0 {
		pulseWatcherCmd := exec.Command(pulseWatcher, "--config", config)

		pulseWatcherCmd.Env = append(pulseWatcherCmd.Env, fmt.Sprintf("PULSEWATCHER_ONESHOT=TRUE"))
		out, err := pulseWatcherCmd.CombinedOutput()
		if err != nil {
			fmt.Println("PulseWatcher execution error: ", err)
			return 1
		}
		fmt.Println(string(out))
	}
	return code
}
