// Code generated by insgocc. DO NOT EDIT.
// source template in application/preprocessor/templates

package {{ .Package }}

import (
{{ range $name, $path := .CustomImports }}
	{{ $name }} {{ $path }}
{{- end }}

{{- range $import, $i := .Imports }}
	{{ $import }}
{{- end }}
)

const PanicIsLogicalError = {{ .PanicIsLogicalError }}

func INS_META_INFO() ([] map[string]string) {
	result := make([]map[string] string, 0)
	{{ range $method := .Methods }}
		{{ if $method.SagaInfo.IsSaga }}
		{
		info := make(map[string] string, 3)
		info["Type"] = "SagaInfo"
		info["MethodName"] = "{{ $method.Name }}"
		info["RollbackMethodName"] = "{{ $method.SagaInfo.RollbackMethodName }}"
		result = append(result, info)
		}
		{{end}}
	{{end}}
	return result
}

func INSMETHOD_GetCode(object []byte, data []byte, ph XXX_contract.ProxyHelper) ([]byte, []byte, error) {
	self := new({{ $.ContractType }})

	if len(object) == 0 {
		return nil, nil, &foundation.Error{S: "[ Fake GetCode ] ( Generated Method ) Object is nil"}
	}

	err := ph.Deserialize(object, self)
	if err != nil {
		e := &foundation.Error{ S: "[ Fake GetCode ] ( Generated Method ) Can't deserialize args.Data: " + err.Error() }
		return nil, nil, e
	}

	state := []byte{}
	err = ph.Serialize(self, &state)
	if err != nil {
		return nil, nil, err
	}

	ret := []byte{}
	err = ph.Serialize([]interface{} { self.GetCode().AsBytes() }, &ret)

	return state, ret, err
}

func INSMETHOD_GetClass(object []byte, data []byte, ph XXX_contract.ProxyHelper) ([]byte, []byte, error) {
	self := new({{ $.ContractType }})

	if len(object) == 0 {
		return nil, nil, &foundation.Error{ S: "[ Fake GetClass ] ( Generated Method ) Object is nil"}
	}

	err := ph.Deserialize(object, self)
	if err != nil {
		e := &foundation.Error{ S: "[ Fake GetClass ] ( Generated Method ) Can't deserialize args.Data: " + err.Error() }
		return nil, nil, e
	}

	state := []byte{}
	err = ph.Serialize(self, &state)
	if err != nil {
		return nil, nil, err
	}

	ret := []byte{}
	err = ph.Serialize([]interface{} { self.GetClass().AsBytes() }, &ret)

	return state, ret, err
}

{{ range $method := .Methods }}
func INSMETHOD_{{ $method.Name }}(object []byte, data []byte, ph XXX_contract.ProxyHelper) (newState []byte, result []byte, err error) {
	ph.SetSystemError(nil)

	self := new({{ $.ContractType }})

	if len(object) == 0 {
		err = &foundation.Error{ S: "[ Fake{{ $method.Name }} ] ( INSMETHOD_* ) ( Generated Method ) Object is nil"}
		return
	}

	err = ph.Deserialize(object, self)
	if err != nil {
		err = &foundation.Error{ S: "[ Fake{{ $method.Name }} ] ( INSMETHOD_* ) ( Generated Method ) Can't deserialize args.Data: " + err.Error() }
		return
	}

	{{ $method.ArgumentsZeroList }}
	err = ph.Deserialize(data, &args)
	if err != nil {
		err = &foundation.Error{ S: "[ Fake{{ $method.Name }} ] ( INSMETHOD_* ) ( Generated Method ) Can't deserialize args.Arguments: " + err.Error() }
		return
	}

    // Set Foundation since it will be required for outgoing calls
	self.InitFoundation(ph)

	{{ $method.ResultDefinitions }}

	serializeResults := func() error {
		return ph.Serialize(
			foundation.Result{Returns:[]interface{}{ {{ $method.Results }} }},
			&result,
		)
	}

	needRecover := true
	defer func() {
		if !needRecover {
			return
		}
		if recoveredError := throw.RW(recover(), nil, "Failed to execute method (panic)"); recoveredError != nil {
			recoveredError = ph.MakeErrorSerializable(recoveredError)

			if PanicIsLogicalError {
				ret{{ $method.LastErrorInRes }} = recoveredError

				newState = object
				err = serializeResults()
				if err == nil {
				    newState = object
				}
			} else {
				err = recoveredError
			}
		}
	}()

	{{ $method.Results }} = self.{{ $method.Name }}( {{ $method.Arguments }} )

    // Nullify Foundation since we don't need to store it with contract
    // It must be done after method call and before serialization of new state
	self.ResetFoundation()

	needRecover = false

	if ph.GetSystemError() != nil {
		return nil, nil, ph.GetSystemError()
	}

	err = ph.Serialize(self, &newState)
	if err != nil {
		return nil, nil, err
	}

{{ range $i := $method.ErrorInterfaceInRes }}
	ret{{ $i }} = ph.MakeErrorSerializable(ret{{ $i }})
{{ end }}

	err = serializeResults()
	if err != nil {
		return
	}

	return
}
{{ end }}


{{ range $f := .Functions }}
func INSCONSTRUCTOR_{{ $f.Name }}(ref reference.Global, data []byte, ph XXX_contract.ProxyHelper) (state []byte, result []byte, err error) {
	ph.SetSystemError(nil)

	{{ $f.ArgumentsZeroList }}
	err = ph.Deserialize(data, &args)
	if err != nil {
		err = &foundation.Error{ S: "[ Fake{{ $f.Name }} ] ( INSCONSTRUCTOR_* ) ( Generated Method ) Can't deserialize args.Arguments: " + err.Error() }
		return
	}

	{{ $f.ResultDefinitions }}

	serializeResults := func() error {
		return ph.Serialize(
			foundation.Result{Returns:[]interface{}{ ref, ret1 }},
			&result,
		)
	}

	needRecover := true
	defer func() {
		if !needRecover {
			return
		}
		if recoveredError := throw.RW(recover(), nil, "Failed to execute constructor (panic)"); recoveredError != nil {
			recoveredError = ph.MakeErrorSerializable(recoveredError)

			if PanicIsLogicalError {
				ret1 = recoveredError

				err = serializeResults()
				if err== nil {
				    state = data
				}
			} else {
				err = recoveredError
			}
		}
	}()

	{{ $f.Results }} = {{ $f.Name }}( {{ $f.Arguments }} )

	needRecover = false

	ret1 = ph.MakeErrorSerializable(ret1)
	if ret0 == nil && ret1 == nil {
		ret1 = &foundation.Error{ S: "constructor returned nil" }
	}

	if ph.GetSystemError() != nil {
		err = ph.GetSystemError()
		return
	}

	err = serializeResults()
	if err != nil {
		return
	}

	if ret1 != nil {
		// logical error, the result should be registered with type SideEffectNone
		state = nil
		return
	}

	err = ph.Serialize(ret0, &state)
	if err != nil {
		return
	}

	return
}
{{ end }}

{{ if $.GenerateInitialize -}}
func Initialize() XXX_contract.Wrapper {
	return XXX_contract.Wrapper{
		GetCode: INSMETHOD_GetCode,
		GetClass: INSMETHOD_GetClass,
		Methods: XXX_contract.Methods{
			{{ range $method := .Methods -}}
					"{{ $method.Name }}": XXX_contract.Method{
					    Func: INSMETHOD_{{ $method.Name }},
						Isolation: XXX_contract.MethodIsolation {
							Interference: {{ $method.Interference }},
							State: {{ $method.State }},
						},
					},
			{{ end }}
		},
		Constructors: XXX_contract.Constructors{
			{{ range $f := .Functions -}}
					"{{ $f.Name }}": INSCONSTRUCTOR_{{ $f.Name }},
			{{ end }}
		},
	}
}
{{- end }}
