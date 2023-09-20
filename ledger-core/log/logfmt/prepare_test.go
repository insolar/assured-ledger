package logfmt

import (
	"reflect"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/args"
)

func TestLookupOrder(t *testing.T) {
	s, vt, b := prepareValue(newMixedType(func() string { return "funcString" }))
	require.Equal(t, "LogString", s)
	require.Equal(t, reflect.Interface, vt)
	require.False(t, b)
}

func TestTypeLookup(t *testing.T) {
	sampleValues, _, sampleFns := dataTypeLookup()

	for i, fn := range sampleFns {
		if i&3 != 3 {
			require.NotNil(t, fn, i)
			v := sampleValues[i]
			s, vt, b := prepareValue(v)
			sf, st, bf := fn(v)
			require.Equal(t, s, sf, i)
			require.Equal(t, b, bf, i)
			require.Equal(t, vt, st, i)
		} else {
			require.Nil(t, fn, i)
		}
	}
}

func dataTypeLookup() (sampleValues []interface{}, sampleTypes []reflect.Type, sampleFns []valuePrepareFn) {
	sampleValues = make([]interface{}, 1, 9)
	sampleValues[0] = newMixedType(func() string { return "funcString" })

	for len(sampleValues) < cap(sampleValues) {
		var v interface{}
		switch len(sampleValues) & 3 {
		case 0:
			v = args.LazyFmt("x")
		case 1:
			v = func() string { return "" }
		case 2:
			v = reflect.String
		case 3:
			v = sampleValues
		default:
			panic("unexpected")
		}

		sampleValues = append(sampleValues, v)
	}
	sampleTypes = make([]reflect.Type, len(sampleValues))
	for i, v := range sampleValues {
		sampleTypes[i] = reflect.TypeOf(v)
	}

	sampleFns = make([]valuePrepareFn, len(sampleValues))
	for i, vt := range sampleTypes {
		sampleFns[i] = findPrepareValueFn(vt)
	}

	return
}

func BenchmarkTypeLookup(b *testing.B) {
	sampleValues, sampleTypes, sampleFns := dataTypeLookup()

	b.Run("byType", func(b *testing.B) {
		for j := b.N; j > 0; j-- {
			for _, s := range sampleTypes {
				fn := findPrepareValueFn(s)
				runtime.KeepAlive(fn)
			}
		}
	})

	b.Run("byFn", func(b *testing.B) {
		for j := b.N; j > 0; j-- {
			for i, fn := range sampleFns {
				if fn == nil {
					continue
				}
				r, _, _ := fn(sampleValues[i])
				runtime.KeepAlive(r)
			}
		}
	})

	b.Run("byValue", func(b *testing.B) {
		for j := b.N; j > 0; j-- {
			for _, s := range sampleValues {
				r, _, _ := prepareValue(s)
				runtime.KeepAlive(r)
			}
		}
	})
}

type mixedType func() string

func (mixedType) LogString() string {
	return "LogString"
}

func (mixedType) String() string {
	return "String"
}

func newMixedType(fn func() string) mixedType {
	return mixedType(fn)
}
