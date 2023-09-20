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
