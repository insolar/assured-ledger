//
// Copyright 2019 Insolar Technologies GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

// Code generated by insgocc. DO NOT EDIT.
// source template in logicrunner/preprocessor/templates

package {{ .PackageName }}

import (
{{ range $name, $path := .CustomImports }}
	{{ $name }} {{ $path }}
{{- end }}

{{- range $import, $i := .Imports }}
	{{ $import }}
{{- end }}
)

{{ range $typeStruct := .Types }}
	{{- $typeStruct }}
{{ end }}

// ClassReference to class of this contract
// error checking hides in generator
var ClassReference, _ = reference.GlobalFromString("{{ .ClassReference }}")


// {{ .ContractType }} holds proxy type
type {{ .ContractType }} struct {
	Reference reference.Global
	Class reference.Global
	Code reference.Global
}

// ContractConstructorHolder holds logic with object construction
type ContractConstructorHolder struct {
	constructorName string
	argsSerialized []byte
}

// AsChild saves object as child
func (r *ContractConstructorHolder) AsChild(objRef reference.Global) (*{{ .ContractType }}, error) {
	var ph = common.CurrentProxyCtx()
	ret, err := ph.CallConstructor(objRef, ClassReference, r.constructorName, r.argsSerialized)
	if err != nil {
		return nil, err
	}

	var ref reference.Global
	var constructorError *foundation.Error
	resultContainer := foundation.Result{
		Returns: []interface{}{ &ref, &constructorError },
	}
	err = ph.Deserialize(ret, &resultContainer)
	if err != nil {
		return nil, err
	}

	if resultContainer.Error != nil {
		return nil, resultContainer.Error
	}

	if constructorError != nil {
		return nil, constructorError
	}

	return &{{ .ContractType }}{Reference: ref}, nil
}

// GetObject returns proxy object
func GetObject(ref reference.Global) *{{ .ContractType }} {
    if !ref.IsObjectReference() {
        return nil
    }
	return &{{ .ContractType }}{Reference: ref}
}

// GetClass returns reference to the class
func GetClass() reference.Global {
	return ClassReference
}

{{ range $func := .ConstructorsProxies }}
// {{ $func.Name }} is constructor
func {{ $func.Name }}( {{ $func.Arguments }} ) *ContractConstructorHolder {
	{{ $func.InitArgs }}

	var argsSerialized []byte
	err := common.CurrentProxyCtx().Serialize(args, &argsSerialized)
	if err != nil {
		panic(err)
	}

	return &ContractConstructorHolder{constructorName: "{{ $func.Name }}", argsSerialized: argsSerialized}
}
{{ end }}

// GetReference returns reference of the object
func (r *{{ $.ContractType }}) GetReference() reference.Global {
	return r.Reference
}

// GetClass returns reference to the code
func (r *{{ $.ContractType }}) GetClass() (reference.Global, error) {
	var ph = common.CurrentProxyCtx()
	if r.Class.IsEmpty() {
		ret := [2]interface{}{}
		var ret0 reference.Global
		ret[0] = &ret0
		var ret1 *foundation.Error
		ret[1] = &ret1

		res, err := ph.CallMethod(
			r.Reference, XXX_contract.CallIntolerable, XXX_contract.CallValidated, false, "GetClass", make([]byte, 0), ClassReference)
		if err != nil {
			return ret0, err
		}

		err = ph.Deserialize(res, &ret)
		if err != nil {
			return ret0, err
		}

		if ret1 != nil {
			return ret0, ret1
		}

		r.Class = ret0
	}

	return r.Class, nil

}

// GetCode returns reference to the code
func (r *{{ $.ContractType }}) GetCode() (reference.Global, error) {
	var ph = common.CurrentProxyCtx()
	if r.Code.IsEmpty() {
		ret := [2]interface{}{}
		var ret0 reference.Global
		ret[0] = &ret0
		var ret1 *foundation.Error
		ret[1] = &ret1

		res, err := ph.CallMethod(
			r.Reference, XXX_contract.CallIntolerable, XXX_contract.CallValidated, false, "GetCode", make([]byte, 0), ClassReference)
		if err != nil {
			return ret0, err
		}

		err = ph.Deserialize(res, &ret)
		if err != nil {
			return ret0, err
		}

		if ret1 != nil {
			return ret0, ret1
		}

		r.Code = ret0
	}

	return r.Code, nil
}

{{ range $method := .MethodsProxies }}
// {{ $method.Name }} is proxy generated method
func (r *{{ $.ContractType }}) {{ $method.Name }}{{if $method.Immutable}}AsMutable{{end}}( {{ $method.Arguments }} ) ( {{ $method.ResultsTypes }} ) {
	{{ $method.InitArgs }}
	var argsSerialized []byte

	{{ $method.ResultZeroList }}

	var ph = common.CurrentProxyCtx()

	err := ph.Serialize(args, &argsSerialized)
	if err != nil {
		return {{ $method.ResultsWithErr }}
	}

	{{/* Saga call doesn't has a reply (it's `nil`), thus we shouldn't try to deserialize it. */}}
	{{if $method.SagaInfo.IsSaga }}
	_, err = ph.CallMethod(r.Reference, XXX_contract.CallTolerable, XXX_contract.CallDirty, {{ $method.SagaInfo.IsSaga }}, "{{ $method.Name }}", argsSerialized, ClassReference)
	if err != nil {
		return {{ $method.ResultsWithErr }}
	}
	{{else}}
	res, err := ph.CallMethod(r.Reference, XXX_contract.CallTolerable, XXX_contract.CallDirty, {{ $method.SagaInfo.IsSaga }}, "{{ $method.Name }}", argsSerialized, ClassReference)
	if err != nil {
		return {{ $method.ResultsWithErr }}
	}

	resultContainer := foundation.Result{
		Returns: ret,
	}
	err = ph.Deserialize(res, &resultContainer)
	if err != nil {
		return {{ $method.ResultsWithErr }}
	}
	if resultContainer.Error != nil {
		err = resultContainer.Error
		return {{ $method.ResultsWithErr }}
	}
	if {{ $method.ErrorVar }} != nil {
		return {{ $method.Results }}
	}
	{{end -}}

	return {{ $method.ResultsNilError }}
}

{{if not $method.SagaInfo.IsSaga}}

// {{ $method.Name }}AsImmutable is proxy generated method
func (r *{{ $.ContractType }}) {{ $method.Name }}{{if not $method.Immutable}}AsImmutable{{end}}( {{ $method.Arguments }} ) ( {{ $method.ResultsTypes }} ) {
	{{ $method.InitArgs }}
	var argsSerialized []byte

	{{ $method.ResultZeroList }}

	var ph = common.CurrentProxyCtx()

    err := ph.Serialize(args, &argsSerialized)
	if err != nil {
		return {{ $method.ResultsWithErr }}
	}

	res, err := ph.CallMethod(
			r.Reference, XXX_contract.CallIntolerable, XXX_contract.CallValidated, false, "{{ $method.Name }}", argsSerialized, ClassReference)
	if err != nil {
		return {{ $method.ResultsWithErr }}
	}

	resultContainer := foundation.Result{
		Returns: ret,
	}
	err = ph.Deserialize(res, &resultContainer)
	if err != nil {
		return {{ $method.ResultsWithErr }}
	}
	if resultContainer.Error != nil {
		err = resultContainer.Error
		return {{ $method.ResultsWithErr }}
	}
	if {{ $method.ErrorVar }} != nil {
		return {{ $method.Results }}
	}
	return {{ $method.ResultsNilError }}
}
{{ end }}
{{ end }}
