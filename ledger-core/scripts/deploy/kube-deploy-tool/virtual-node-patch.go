// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package main

type VirtualNodePatch struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string
	Metadata   Metadata
	Spec       Spec
}

type Metadata struct {
	Name string
}

type Spec struct {
	Selector Selector
	Replicas int
}

type Selector struct {
	MatchLabels MatchLabels `yaml:"matchLabels"`
}

type MatchLabels struct {
	App string
}
