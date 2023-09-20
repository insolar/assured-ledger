package reflectkit

import (
	"reflect"
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/reflectkit/mocks"
)

func TestMakeAddressable(t *testing.T) {
	i := 10
	require.Equal(t, i, MakeAddressable(reflect.ValueOf(i)).Interface())
	require.Equal(t, &i, MakeAddressable(reflect.ValueOf(&i)).Interface())
}

func newInt(i int) *int {
	return &i
}

func newEmptyInt() *int {
	return nil
}

type testStruct struct {
	I *int
}

// FieldValueGetter(index int, fd reflect.StructField, useAddr bool, baseOffset uintptr) FieldGetterFunc
func TestFieldValueGetter(t *testing.T) {
	for _, tc := range []struct {
		name       string
		index      int
		fd         reflect.StructField
		useAddr    bool
		baseOffset uintptr
		input      reflect.Value
		res        interface{}
	}{
		{
			name:  "plain",
			input: reflect.ValueOf(testStruct{}),
			res:   newEmptyInt(),
		},
		{
			name: "pointer",
			fd: reflect.StructField{
				Name:   "I",
				Type:   reflect.TypeOf(new(int)),
				Offset: 0,
				Index:  []int{},
			},
			input: reflect.ValueOf(testStruct{
				I: newInt(2),
			}),
			useAddr: true,
			res:     newInt(2),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fn := FieldValueGetter(tc.index, tc.fd, tc.useAddr, tc.baseOffset)
			res := fn(MakeAddressable(tc.input))
			if tc.res == nil || (reflect.ValueOf(tc.res).Kind() == reflect.Ptr && reflect.ValueOf(tc.res).IsNil()) {
				require.Nil(t, res.Interface())
				return
			} else {
				require.NotNil(t, res.Interface())
			}

			require.Equal(t, *tc.res.(*int), *res.Interface().(*int))
		})
	}
}

func TestValueToReceiverFactory(t *testing.T) {
	t.Run("simple types", func(t *testing.T) {
		type TestStruct struct {
			B bool
			b bool
			I int64
			i int64
			U uint8
			u uint8
			F float64
			f float32
			C complex64
			c complex128
			S string
			s string
		}

		input := TestStruct{
			B: true,
			b: false,
			I: 10,
			i: -1,
			U: 0,
			u: 10,
			F: 0.1,
			f: -0.1,
			C: complex(0.1, 1),
			c: complex(-0.1, 0),
			S: "test",
			s: "",
		}
		st := reflect.TypeOf(input)
		for i := 0; i < st.NumField(); i++ {
			field := st.Field(i)
			t.Run(field.Type.String(), func(t *testing.T) {
				val := ValueToReceiverFactory(field.Type.Kind(), nil)
				t.Run("zero", func(t *testing.T) {
					mc := minimock.NewController(t)
					mockReceiver := mocks.NewTypedReceiverMock(mc).ReceiveZeroMock.Set(func(k1 reflect.Kind) {
						assert.Equal(t, field.Type.Kind(), k1)
					})
					withZeroUnexported, withZero := val(true, field.Type, true)
					withZero(reflect.Zero(field.Type), mockReceiver)
					assert.False(t, withZeroUnexported)
					mc.Finish()
				})

				t.Run("nonzero", func(t *testing.T) {
					mc := minimock.NewController(t)
					nonZeroValFn := FieldValueGetter(i, field, true, 0)
					value := nonZeroValFn(MakeAddressable(reflect.ValueOf(input)))
					mockReceiver := newMockByKind(mc, field.Type, value.Interface())
					nonZeroUnexported, nonZero := val(true, field.Type, false)
					nonZero(value, mockReceiver)
					assert.False(t, nonZeroUnexported)
					mc.Finish()
				})
			})
		}

	})
	t.Run("nillable types fully defined at compile time", func(t *testing.T) {
		type myFunc func()
		type TestStruct struct {
			f myFunc
			F myFunc
			s []byte
			S []byte
		}

		input := TestStruct{
			f: myFunc(func() {}),
			F: myFunc(func() {}),
			s: []byte{},
			S: []byte{},
		}
		st := reflect.TypeOf(input)
		for i := 0; i < st.NumField(); i++ {
			field := st.Field(i)
			exported := reflect.ValueOf(input).Field(i).CanInterface()
			t.Run("with custom", func(t *testing.T) {
				t.Run("zero", func(t *testing.T) {
					mc := minimock.NewController(t)
					custom := newCustomByKind()
					val := ValueToReceiverFactory(field.Type.Kind(), custom)
					mockReceiver := mocks.NewTypedReceiverMock(mc).ReceiveNilMock.Set(func(k1 reflect.Kind) {
						assert.Equal(t, field.Type.Kind(), k1)
					})
					withZeroUnexported, withZero := val(!exported, field.Type, true)
					withZero(reflect.Zero(field.Type), mockReceiver)
					assert.Equal(t, !exported, withZeroUnexported)
					mc.Finish()
				})
				t.Run("nonzero", func(t *testing.T) {
					mc := minimock.NewController(t)
					nonZeroValFn := FieldValueGetter(i, field, true, 0)
					value := nonZeroValFn(MakeAddressable(reflect.ValueOf(input)))

					custom := newCustomByKind()
					val := ValueToReceiverFactory(field.Type.Kind(), custom)

					mockReceiver := mocks.NewTypedReceiverMock(mc)
					if !value.IsValid() || IsZero(value) {
						mockReceiver.ReceiveZeroMock.Set(func(tKind reflect.Kind) {
							assert.Equal(t, field.Type, tKind, "expected %s, got %s", field.Type.String(), tKind.String())
						})
					} else {
						mockReceiver.ReceiveElseMock.Set(func(tKind reflect.Kind, v interface{}, isZero bool) {
						})
					}
					nonZeroUnexported, nonZero := val(!exported, field.Type, true)
					nonZero(value, mockReceiver)
					assert.Equal(t, !exported, nonZeroUnexported)
					mc.Finish()
				})
			})
			t.Run("without custom", func(t *testing.T) {
				t.Run("zero", func(t *testing.T) {
					mc := minimock.NewController(t)
					mockReceiver := mocks.NewTypedReceiverMock(mc).ReceiveNilMock.Set(func(k1 reflect.Kind) {
						assert.Equal(t, field.Type.Kind(), k1)
					})
					val := ValueToReceiverFactory(field.Type.Kind(), nil)
					withZeroUnexported, withZero := val(!exported, field.Type, true)
					withZero(reflect.Zero(field.Type), mockReceiver)
					assert.Equal(t, !exported, withZeroUnexported)
					mc.Finish()
				})
				t.Run("nonzero", func(t *testing.T) {
					val := ValueToReceiverFactory(field.Type.Kind(), nil)
					mc := minimock.NewController(t)
					nonZeroValFn := FieldValueGetter(i, field, true, 0)
					value := nonZeroValFn(MakeAddressable(reflect.ValueOf(input)))
					mockReceiver := newMockByKind(mc, field.Type, value.Interface())
					withoutZeroUnexported, nonZero := val(!exported, field.Type, false)
					nonZero(value, mockReceiver)
					assert.Equal(t, !exported, withoutZeroUnexported)
					mc.Finish()
				})
			})
		}
	})
	t.Run("interface", func(t *testing.T) {
		type myFunc func()
		type TestStruct struct {
			f interface{}
			F interface{}
		}

		input := TestStruct{
			f: myFunc(func() {}),
			F: myFunc(func() {}),
		}
		st := reflect.TypeOf(input)
		for i := 0; i < st.NumField(); i++ {
			field := st.Field(i)
			exported := reflect.ValueOf(input).Field(i).CanInterface()
			t.Run("with custom", func(t *testing.T) {
				t.Run("zero", func(t *testing.T) {
					mc := minimock.NewController(t)
					custom := newCustomByKind()
					val := ValueToReceiverFactory(field.Type.Kind(), custom)
					mockReceiver := mocks.NewTypedReceiverMock(mc).ReceiveNilMock.Set(func(k1 reflect.Kind) {
						assert.Equal(t, reflect.Invalid, k1)
					})
					withZeroUnexported, withZero := val(!exported, field.Type, true)
					withZero(reflect.Zero(field.Type), mockReceiver)
					assert.Equal(t, !exported, withZeroUnexported)
					mc.Finish()
				})
				t.Run("nonzero", func(t *testing.T) {
					mc := minimock.NewController(t)
					nonZeroValFn := FieldValueGetter(i, field, true, 0)
					value := nonZeroValFn(MakeAddressable(reflect.ValueOf(input)))

					custom := newCustomByKind()
					val := ValueToReceiverFactory(field.Type.Kind(), custom)

					mockReceiver := mocks.NewTypedReceiverMock(mc)
					if !value.IsValid() || IsZero(value) {
						mockReceiver.ReceiveZeroMock.Set(func(tKind reflect.Kind) {
							assert.Equal(t, field.Type, tKind, "expected %s, got %s", field.Type.String(), tKind.String())
						})
					} else {
						mockReceiver.ReceiveElseMock.Set(func(tKind reflect.Kind, v interface{}, isZero bool) {
						})
					}
					nonZeroUnexported, nonZero := val(!exported, field.Type, true)
					nonZero(value, mockReceiver)
					assert.Equal(t, !exported, nonZeroUnexported)
					mc.Finish()
				})
			})
			t.Run("without custom", func(t *testing.T) {
				t.Run("zero", func(t *testing.T) {
					mc := minimock.NewController(t)
					mockReceiver := mocks.NewTypedReceiverMock(mc).ReceiveNilMock.Set(func(k1 reflect.Kind) {
						assert.Equal(t, field.Type.Kind(), k1)
					})
					val := ValueToReceiverFactory(field.Type.Kind(), nil)
					withZeroUnexported, withZero := val(!exported, field.Type, true)
					withZero(reflect.Zero(field.Type), mockReceiver)
					assert.Equal(t, !exported, withZeroUnexported)
					mc.Finish()
				})
				t.Run("nonzero", func(t *testing.T) {
					val := ValueToReceiverFactory(field.Type.Kind(), nil)
					mc := minimock.NewController(t)
					nonZeroValFn := FieldValueGetter(i, field, true, 0)
					value := nonZeroValFn(MakeAddressable(reflect.ValueOf(input)))
					mockReceiver := newMockByKind(mc, field.Type, value.Interface())
					withoutZeroUnexported, nonZero := val(!exported, field.Type, false)
					nonZero(value, mockReceiver)
					assert.Equal(t, !exported, withoutZeroUnexported)
					mc.Finish()
				})
			})
		}
	})

	t.Run("Excluded", func(t *testing.T) {
		require.Nil(t, ValueToReceiverFactory(reflect.UnsafePointer, nil))
	})
}

func newMockByKind(mc *minimock.Controller, t reflect.Type, val interface{}) *mocks.TypedReceiverMock {
	res := mocks.NewTypedReceiverMock(mc)
	switch t.Kind() {
	case reflect.Bool:
		res.ReceiveBoolMock.Set(func(k1 reflect.Kind, b1 bool) {
			assert.Equal(mc, t.Kind(), k1)
			assert.Equal(mc, val.(bool), b1)
		})
	case reflect.Int64, reflect.Int32:
		res.ReceiveIntMock.Set(func(k1 reflect.Kind, i1 int64) {
			assert.Equal(mc, t.Kind(), k1)
		})
	case reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8:
		res.ReceiveUintMock.Set(func(k1 reflect.Kind, i1 uint64) {
			assert.Equal(mc, t.Kind(), k1)
		})
	case reflect.Float64, reflect.Float32:
		res.ReceiveFloatMock.Set(func(k1 reflect.Kind, i1 float64) {
			assert.Equal(mc, t.Kind(), k1)
		})
	case reflect.String:
		res.ReceiveStringMock.Set(func(k1 reflect.Kind, s1 string) {
			assert.Equal(mc, t.Kind(), k1)
			assert.Equal(mc, val.(string), s1)
		})
	case reflect.Complex128, reflect.Complex64:
		res.ReceiveComplexMock.Set(func(k1 reflect.Kind, c1 complex128) {
			assert.Equal(mc, t.Kind(), k1)
		})
	case reflect.Func, reflect.Ptr, reflect.Map, reflect.Slice, reflect.Chan:
		res.ReceiveElseMock.Set(func(t reflect.Kind, v interface{}, isZero bool) {
		})
	case reflect.Interface:
		res.ReceiveElseMock.Set(func(t reflect.Kind, v interface{}, isZero bool) {
		})
	default:
		assert.Fail(mc, "unexpected type")
	}
	return res
}

func newCustomByKind() IfaceToReceiverFactoryFunc {
	return func(t reflect.Type, checkZero bool) IfaceToReceiverFunc {
		return func(value interface{}, kind reflect.Kind, out TypedReceiver) {
			val := reflect.ValueOf(value)
			if checkZero {
				switch {
				case !val.IsValid() || val.IsNil():
					out.ReceiveNil(kind)
					return
				case IsZero(val):
					out.ReceiveZero(kind)
					return
				}
			}

			switch val.Kind() {
			case reflect.Func, reflect.Ptr, reflect.Map, reflect.Slice, reflect.Chan:
				out.ReceiveElse(val.Kind(), value, false)
			default:
				panic("unexpected input")
			}

		}
	}
}

// IsZero could be removed after go 1.13, in 1.13 it is in reflect package
func IsZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return v.IsNil()
	}
	return false
}
