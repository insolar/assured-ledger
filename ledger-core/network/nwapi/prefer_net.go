// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package nwapi

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type Preference uint8

const (
	_ Preference = iota
	PreferV4
	PreferV6
)

func (v Preference) suffux() string {
	switch v {
	case PreferV4:
		return "4"
	case PreferV6:
		return "6"
	default:
		return ""
	}
}

func (v Preference) Default(name string) string {
	if v == 0 {
		return name
	}
	switch name {
	case "udp", "tcp", "ip":
		return name + v.suffux()
	default:
		return name
	}
}

func (v Preference) Override(name string) string {
	switch name {
	case "udp", "tcp", "ip":
		if v != 0 {
			return name + v.suffux()
		}

	case "udp4", "tcp4", "ip4":
		if v != PreferV4 {
			return name[:len(name)-1] + v.suffux()
		}

	case "udp6", "tcp6", "ip6":
		if v != PreferV6 {
			return name[:len(name)-1] + v.suffux()
		}
	}
	return name
}

func (v Preference) ChooseOne(a []Address) Address {
	switch n := len(a); {
	case n == 0:
		panic(throw.IllegalValue())
	case n == 1:
		//
	case v == PreferV6:
		for _, aa := range a {
			if aa.isIPv6() {
				return aa
			}
		}
	case v == PreferV4:
		for _, aa := range a {
			if aa.isIPv4() {
				return aa
			}
		}
	}
	return a[0]
}

func (v Preference) String() string {
	if v == 0 {
		return "noPrefer"
	}
	return "preferV" + v.suffux()
}
