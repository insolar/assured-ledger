package injector

import (
	"reflect"
	"strings"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func GetDefaultInjectionID(v interface{}) string {
	return GetDefaultInjectionIDByType(reflect.TypeOf(v))
}

func GetDefaultInjectionIDByType(vt reflect.Type) string {
	return strings.TrimLeft(vt.String(), "*")
}

func NewDependencyInjector(target interface{}, globalParent DependencyRegistry, localParent DependencyRegistry) DependencyInjector {
	resolver := NewDependencyResolver(target, globalParent, localParent, nil)
	return NewDependencyInjectorFor(&resolver)
}

func NewDependencyInjectorFor(resolver *DependencyResolver) DependencyInjector {
	if resolver == nil || resolver.IsZero() {
		panic(throw.IllegalValue())
	}
	return DependencyInjector{resolver}
}

type DependencyInjector struct {
	resolver *DependencyResolver
}

func (u DependencyInjector) IsZero() bool {
	return u.resolver.IsZero()
}

func (u DependencyInjector) IsEmpty() bool {
	return u.resolver.IsEmpty()
}

func (u DependencyInjector) MustInject(varRef interface{}) {
	if err := u.Inject(varRef); err != nil {
		panic(throw.WithStack(err))
	}
}

func (u DependencyInjector) MustInjectByID(id string, varRef interface{}) {
	if err := u.InjectByID(id, varRef); err != nil {
		panic(throw.WithStack(err))
	}
}

func (u DependencyInjector) MustInjectAny(varRef interface{}) {
	if err := u.InjectAny(varRef); err != nil {
		panic(throw.WithStack(err))
	}
}

func (u DependencyInjector) MustInjectAll() {
	if err := u.InjectAll(); err != nil {
		panic(throw.WithStack(err))
	}
}

func (u DependencyInjector) Inject(varRef interface{}) error {
	return u.tryInjectVar("", varRef)
}

func (u DependencyInjector) InjectByID(id string, varRef interface{}) error {
	if id == "" {
		panic(throw.IllegalValue())
	}
	return u.tryInjectVar(id, varRef)
}

func (u DependencyInjector) InjectAny(varRef interface{}) error {
	fv := checkVarRef(varRef)
	return u.injectAny(nil, fv, fv.Type(), "", "")
}

//nolint:interfacer
func (u DependencyInjector) injectAny(tt *lazyInjectName, fv reflect.Value, ft reflect.Type, id string, fieldName string) error {
	switch isNillable, isSet := u.check(fv, ft); {
	case isSet:
		return throw.E("dependency is set", struct { ExpectedType reflect.Type } {ft})
	case id != "":
		if u.resolveNameAndSet(id, fv, ft, isNillable) {
			return nil
		}
	default:
		if u.resolveTypeAndSet(tt.String(), fieldName, fv, ft, isNillable) {
			return nil
		}
	}

	if ft.Kind() == reflect.Interface {
		if found, err := u.resolveImplAndSet(fv, ft); found || err != nil {
			return err
		}
	}

	return throw.E("dependency is missing", struct { ExpectedType reflect.Type } {ft})
}

func (u DependencyInjector) InjectAll() error {
	t := reflect.Indirect(reflect.ValueOf(u.resolver.Target()))
	if t.Kind() != reflect.Struct {
		panic(throw.IllegalValue())
	}
	if !t.CanSet() {
		panic(throw.FailHere("readonly"))
	}
	tt := t.Type()

	lazyName := &lazyInjectName{t:tt}

	for i := 0; i < tt.NumField(); i++ {
		sf := tt.Field(i)
		id, ok := sf.Tag.Lookup("inject")
		if !ok {
			continue
		}

		fv := t.Field(i)
		if err := u.injectAny(lazyName, fv, sf.Type, id, sf.Name); err != nil {
			return throw.WithDetails(err, struct {
				Target reflect.Type
				FieldName string
			}{
				Target: tt,
				FieldName: sf.Name,
			})
		}
	}

	return nil
}

func (u DependencyInjector) tryInjectVar(id string, varRef interface{}) error {
	v := checkVarRef(varRef)
	vt := v.Type()
	isNillable, isSet := u.check(v, vt)

	switch {
	case isSet:
		return throw.E("dependency is set", struct {
			ExpectedType reflect.Type
			ID string
		} {vt, id})
	case id != "":
		if u.resolveNameAndSet(id, v, vt, isNillable) {
			return nil
		}
	case u.resolveTypeAndSet(GetDefaultInjectionIDByType(vt), "", v, vt, isNillable):
		return nil
	}

	return throw.E("dependency is missing", struct {
		ExpectedType reflect.Type
		ID string
	} {vt, id})
}

func checkVarRef(varRef interface{}) reflect.Value {
	if varRef == nil {
		panic(throw.IllegalValue())
	}

	v := reflect.ValueOf(varRef)
	switch {
	case v.Kind() != reflect.Ptr:
		panic(throw.FailHere("not a reference"))
	case v.IsNil():
		panic(throw.FailHere("nil reference"))
	case v.CanSet() || v.CanAddr():
		panic(throw.FailHere("must be a literal reference"))
	}
	return v.Elem()
}

func GetInterfaceTypeAndValue(varRef interface{}) (interface{}, reflect.Type) {
	v := checkVarRef(varRef)
	if v.Kind() != reflect.Interface {
		panic(throw.FailHere("not an interface"))
	}
	vv := v.Interface()
	if vv == nil {
		panic(throw.FailHere("nil interface"))
	}

	vt := v.Type()
	return vv, vt
}

func (u DependencyInjector) check(v reflect.Value, vt reflect.Type) (isNillable, hasValue bool) {
	if !v.CanSet() {
		panic(throw.FailHere("readonly"))
	}

	switch vt.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return true, !v.IsNil()
	default:
		zeroValue := reflect.Zero(vt).Interface()
		return false, v.Interface() != zeroValue
	}
}

func (u DependencyInjector) resolveTypeAndSet(typeName, fieldName string, v reflect.Value, vt reflect.Type, nillable bool) bool {
	if u.resolveNameAndSet(typeName, v, vt, nillable) {
		return true
	}

	idx := strings.LastIndexByte(typeName, '.')
	if idx >= 0 && u.resolveNameAndSet(typeName[idx+1:], v, vt, nillable) {
		return true
	}

	if fieldName == "" {
		return false
	}
	typeName = typeName + "." + fieldName

	if u.resolveNameAndSet(typeName, v, vt, nillable) {
		return true
	}
	if idx >= 0 && u.resolveNameAndSet(typeName[idx+1:], v, vt, nillable) {
		return true
	}
	return false
}

func (u DependencyInjector) resolveNameAndSet(n string, v reflect.Value, vt reflect.Type, nillable bool) bool {
	if len(n) == 0 {
		return false
	}

	switch val, ok := u.resolver.getResolved(n); {
	case !ok:
		return false
	case nillable && val == nil:
		return true
	default:
		dv := reflect.ValueOf(val)
		dt := dv.Type()
		if !dt.AssignableTo(vt) {
			return false
		}
		v.Set(dv)
		return true
	}
}

func (u DependencyInjector) resolveImplAndSet(fv reflect.Value, ft reflect.Type) (bool, error) {
	switch v, err := u.resolver.FindImplementation(ft, true); {
	case err != nil:
		return false, err
	case v == nil:
		return false, nil
	default:
		fv.Set(reflect.ValueOf(v))
		return true, nil
	}
}
