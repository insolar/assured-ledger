// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build !cloud_with_consensus

package launchnet

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
)

// Run starts launchnet before execution of callback function (cb) and stops launchnet after.
// Returns exit code as a result from calling callback function.
func Run(cb func() int) int {
	setupFunc := setup
	if isCloudMode() {
		setupFunc = setupCloud
	}
	teardown, err := setupFunc()
	defer teardown()
	if err != nil {
		fmt.Println("error while setup, skip tests: ", err)
		return 1
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)

	go func() {
		sig := <-c
		fmt.Printf("Got %s signal. Aborting...\n", sig)
		teardown()

		os.Exit(2)
	}()

	pulseWatcher, config := pulseWatcherPath()

	code := cb()

	if code != 0 {
		out, err := exec.Command(pulseWatcher, "-c", config, "-s").CombinedOutput()
		if err != nil {
			fmt.Println("PulseWatcher execution error: ", err)
			return 1
		}
		fmt.Println(string(out))
	}
	return code
}
