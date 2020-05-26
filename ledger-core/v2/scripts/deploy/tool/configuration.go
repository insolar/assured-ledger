// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package main

<<<<<<< HEAD
type KubeDeployTool struct {
	NodesCount    uint
	ManifestsPath string
=======
type KubeDeployToolConfig struct {
	NodesCount             uint
	Env                    string
	KubeRootPath           string
	ManifestsRelPath       string
	BootstrapConfigRelPath string
>>>>>>> 80ed601376d4994db1df8d7ba49dbd725b61259a
}
