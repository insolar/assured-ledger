// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package injector

import (
	"reflect"
)

type lazyInjectName struct {
	t reflect.Type
	s string
}

func (p *lazyInjectName) String() string {
	if p == nil || p.t == nil {
		return ""
	}
	if p.s == "" {
		p.s = GetDefaultInjectionIDByType(p.t)
	}
	return p.s
}
