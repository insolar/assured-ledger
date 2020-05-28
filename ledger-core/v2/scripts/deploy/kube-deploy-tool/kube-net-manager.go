// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const Kubectl = "kubectl"

type eventTiming struct {
	startedAt time.Time
	stoppedAt time.Time
}

type InsolarNetManager struct {
	kubeParams  KubeParams
	callStarted func(params NetParams, timing eventTiming)
	callReady   func(params NetParams, timing eventTiming)
	callStopped func(params NetParams, timing eventTiming)
}

func NewInsolarNetManager(
	netConfig KubeParams,
	callbackStart func(params NetParams, timing eventTiming),
	callbackReady func(params NetParams, timing eventTiming),
	callbackStop func(params NetParams, timing eventTiming),
) *InsolarNetManager {
	return &InsolarNetManager{
		kubeParams:  netConfig,
		callStarted: callbackStart,
		callReady:   callbackReady,
		callStopped: callbackStop,
	}
}

func (m *InsolarNetManager) startNetwork(netParams NetParams) error {
	startedAt := time.Now()
	pathToKustomize := getExecutablePath() + m.kubeParams.KubeRootPath + m.kubeParams.Env + "/"
	out, err := exec.Command(Kubectl, "apply", "-k", pathToKustomize).CombinedOutput()
	if err != nil {
		return fmt.Errorf("run failed: %s %w", string(out), err)
	}
	m.callStarted(netParams, eventTiming{
		startedAt: startedAt,
		stoppedAt: time.Now(),
	})
	return nil
}

func (m *InsolarNetManager) waitForReady(netParams NetParams) error {
	startedAt := time.Now()
	for i := 0; i < 60; i++ {
		args := []string{
			"-n",
			"insolar",
			"get",
			"po",
			"bootstrap",
			"-o",
			"jsonpath=\"{.status.phase}\"",
		}
		cmd := exec.Command(Kubectl, args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("bootstrap failed: %s %w", string(out), err)
		}
		if string(out) == "\"Succeeded\"" {
			fmt.Println("bootstrap finished")
			break
		}
		time.Sleep(1 * time.Second)
	}

	for i := 0; i < 60; i++ {
		args := []string{
			"-n",
			"insolar",
			"exec",
			"-i",
			"deploy/pulsewatcher",
			"--",
			`pulsewatcher`,
			"-c",
			"/etc/pulsewatcher/pulsewatcher.yaml",
			`-s`,
		}
		cmd := exec.Command(Kubectl, args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("insolar ready check failed: %s %w", string(out), err)
		}
		if strings.Contains(string(out), "READY") && !strings.Contains(string(out), "NOT") {
			fmt.Println("insolar has been started")
			m.callReady(netParams, eventTiming{
				startedAt: startedAt,
				stoppedAt: time.Now(),
			})
			return nil
		}
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("insolar has not been started")
}

func (m *InsolarNetManager) stopNetwork(netParams NetParams) error {
	startedAt := time.Now()
	pathToKustomize := getExecutablePath() + m.kubeParams.KubeRootPath + m.kubeParams.Env + "/"
	out, err := exec.Command(Kubectl, "delete", "-k", pathToKustomize).CombinedOutput()
	if err != nil {
		return fmt.Errorf("stop failed: %s %w", string(out), err)
	}
	m.callStopped(netParams, eventTiming{
		startedAt: startedAt,
		stoppedAt: time.Now(),
	})
	return nil
}
