// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const Kubectl = "kubectl"
const DefaultInsolarImage = "insolar/assured-ledger:latest"
const logFileTemplate = "%d-nodes-virtual-%d.log"

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
	stopWaitingBootstrap := make(chan bool, 1)
	defer func() { stopWaitingBootstrap <- true }()

	go func() {
	loop:
		for {
			select {
			case <-stopWaitingBootstrap:
				break loop
			case <-time.After(time.Second):
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
			}
		}
	}()
	select {
	case <-bootstrapFinished:
		fmt.Println("bootstrap finished")
	case <-time.After(netParams.WaitBootstrap):
		fmt.Printf("bootstrap timed out after %s\n", netParams.WaitBootstrap.String())
	}

	netReady := make(chan bool, 1)
	stopWaitingReady := make(chan bool, 1)
	defer func() { stopWaitingReady <- true }()

	go func() {
	loop:
		for {
			select {
			case <-stopWaitingReady:
				break loop
			case <-time.After(time.Second):
				ready, err := m.checkReady()
				if err != nil {
					fmt.Printf("insolar ready check failed: %s\n", err.Error())
				}
				if ready {
					netReady <- true
				}
			}

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
	case <-time.After(netParams.WaitReady):
		fmt.Printf("ready waiting timed out after %s\n", netParams.WaitReady)
	}

	return fmt.Errorf("insolar has not been started")
}

func (m *InsolarNetManager) stop(netParams NetParams) error {
	fmt.Printf("stopping insolar\n")
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

func (m *InsolarNetManager) collectLogs(netParams NetParams) error {
	fmt.Println("start collecting pod logs")
	for i := 0; i < int(netParams.NodesCount); i++ {
		podName := "virtual-" + strconv.Itoa(i)
		out, err := exec.Command(
			Kubectl,
			"-n",
			"insolar",
			"logs",
			podName,
		).CombinedOutput()
		if err != nil {
			return fmt.Errorf("collect log failed: %s %w", string(out), err)
		}

		fileName := fmt.Sprintf(m.kubeParams.LogCollector.PathToSave+logFileTemplate, netParams.NodesCount, i)
		err = ioutil.WriteFile(fileName, out, 0644)
		if err != nil {
			return fmt.Errorf("write log failed: %w", err)
		}
	}
	return nil
}

func (m *InsolarNetManager) cleanLogDir() error {
	err := os.RemoveAll(m.kubeParams.LogCollector.PathToSave)
	if err != nil {
		return fmt.Errorf("clearing log dir failed: %w", err)
	}
	err = os.MkdirAll(m.kubeParams.LogCollector.PathToSave, 0755)
	if err != nil {
		return fmt.Errorf("creating log dir failed: %w", err)
	}
	fmt.Println("logs dir cleaned")
	return nil
}

func (m *InsolarNetManager) checkReady() (bool, error) {
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
		return false, fmt.Errorf("%s %w", string(out), err)
	}
	if strings.Contains(string(out), "READY") && !strings.Contains(string(out), "NOT") {
		return true, nil
	}
	return false, nil
}

func (m *InsolarNetManager) waitInReady(net NetParams) error {
	fmt.Printf("waiting in ready state for %s\n", net.WaitInReadyState)
	waitTimeout := time.After(net.WaitInReadyState)
	for {
		select {
		case <-waitTimeout:
			return nil
		default:
			ready, err := m.checkReady()
			if err != nil {
				fmt.Printf("insolar ready check failed: %s\n", err.Error())
			}
			if !ready {
				return fmt.Errorf("insolar ready check returned 'false' during waiting in ready state")
			}
			time.Sleep(time.Second)
		}
	}
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
	fmt.Println("prometheus started")
	return nil
}

func (m *PrometheusManager) stop() error {
	pathToKustomize := getExecutablePath() + m.kubeParams.KubeRootPath + m.kubeParams.Prometheus.ManifestsRelPath
	out, err := exec.Command(Kubectl, "delete", "-k", pathToKustomize).CombinedOutput()
	if err != nil {
		return fmt.Errorf("prometheus stop failed: %s %w", string(out), err)
	}
	fmt.Println("prometheus stopped")
	return nil
}
