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
const DefaultInsolarImage = "insolar/assured-ledger:latest"

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

func (m *InsolarNetManager) checkDependencies() error {
	// check local images
	out, err := exec.Command(
		"docker", "images", DefaultInsolarImage, "-q",
	).CombinedOutput()
	if err != nil {
		return fmt.Errorf("check images failed: %s %w", string(out), err)
	}
	if len(out) == 0 {
		return fmt.Errorf("docker image %s not found", DefaultInsolarImage)
	}

	// check ingress installation
	out, err = exec.Command(
		Kubectl,
		"-n",
		"kube-system",
		"rollout",
		"status",
		"deploy/traefik-ingress-controller",
	).CombinedOutput()
	if err != nil {
		return fmt.Errorf("check ingress failed: %s %w", string(out), err)
	}
	return nil
}

func (m *InsolarNetManager) start(netParams NetParams) error {
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

	bootstrapFinished := make(chan bool, 1)
	go func() {
		for {
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
				fmt.Printf("bootstrap check failed: %s %s\n", string(out), err.Error())
				break
			}
			if string(out) == "\"Succeeded\"" {
				bootstrapFinished <- true
				break
			}
			time.Sleep(1 * time.Second)
		}
	}()
	select {
	case <-bootstrapFinished:
		fmt.Println("bootstrap finished")
	case <-time.After(netParams.WaitBootstrap):
		fmt.Println("bootstrap timed out")
	}

	netReady := make(chan bool, 1)
	go func() {
		for {
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
				fmt.Printf("insolar ready check failed: %s %s", string(out), err.Error())
				break
			}
			if strings.Contains(string(out), "READY") && !strings.Contains(string(out), "NOT") {
				netReady <- true
				break
			}
			time.Sleep(1 * time.Second)
		}
	}()
	select {
	case <-netReady:
		fmt.Println("insolar has been started")
		m.callReady(netParams, eventTiming{
			startedAt: startedAt,
			stoppedAt: time.Now(),
		})
		return nil
	case <-time.After(netParams.WaitBootstrap):
		fmt.Println("insolar ready wait timed out")
	}

	return fmt.Errorf("insolar has not been started")
}

func (m *InsolarNetManager) stop(netParams NetParams) error {
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

type PrometheusManager struct {
	kubeParams KubeParams
}

func NewPrometheusManager(kubeParams KubeParams) *PrometheusManager {
	return &PrometheusManager{kubeParams: kubeParams}
}

func (m *PrometheusManager) start() error {
	pathToKustomize := getExecutablePath() + m.kubeParams.KubeRootPath + m.kubeParams.Prometheus.ManifestsRelPath
	out, err := exec.Command(Kubectl, "apply", "-k", pathToKustomize).CombinedOutput()
	if err != nil {
		return fmt.Errorf("prometheus start failed: %s %w", string(out), err)
	}
	return nil
}

func (m *PrometheusManager) stop() error {
	pathToKustomize := getExecutablePath() + m.kubeParams.KubeRootPath + m.kubeParams.Prometheus.ManifestsRelPath
	out, err := exec.Command(Kubectl, "delete", "-k", pathToKustomize).CombinedOutput()
	if err != nil {
		return fmt.Errorf("prometheus stop failed: %s %w", string(out), err)
	}
	return nil
}
