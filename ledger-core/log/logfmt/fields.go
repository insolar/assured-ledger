package logfmt

import "fmt"

type LogFields struct {
	Msg    string
	Fields map[string]interface{}
}

func (v LogFields) MarshalLogObject(w LogObjectWriter, _ LogObjectMetricCollector) (string, bool) {
	for k, v := range v.Fields {
		w.AddIntfField(k, v, LogFieldFormat{})
	}
	return v.Msg, false
}

func (v LogFields) MarshalLogFields(w LogObjectWriter) {
	FieldMapMarshaller(v.Fields).MarshalLogFields(w)
}

type FieldMapMarshaller map[string]interface{}

func (v FieldMapMarshaller) MarshalLogFields(w LogObjectWriter) {
	for k, v := range v {
		w.AddIntfField(k, v, LogFieldFormat{})
	}
}

type LogObjectFields struct {
	Object LogObjectMarshaller
}

func (v LogObjectFields) MarshalLogObject(w LogObjectWriter, c LogObjectMetricCollector) (string, bool) {
	return v.Object.MarshalLogObject(w, c)
}

func (v LogObjectFields) MarshalLogFields(w LogObjectWriter) {
	if v.Object == nil {
		return
	}
	s, _ := v.Object.MarshalLogObject(w, nil)
	if s != "" {
		w.AddStrField(fmt.Sprintf("Message_%T", v.Object), s, LogFieldFormat{})
	}
}

type LogField struct {
	Name string
	Value interface{}
}

func (v LogField) MarshalLogObject(w LogObjectWriter, _ LogObjectMetricCollector) (string, bool) {
	v.MarshalLogFields(w)
	return "", false
}

func (v LogField) MarshalLogFields(w LogObjectWriter) {
	w.AddIntfField(v.Name, v.Value, LogFieldFormat{})
}

func JoinFields(a []LogFieldMarshaller, b ...LogFieldMarshaller) []LogFieldMarshaller {
	n := len(a) + len(b)
	if n == 0 {
		return nil
	}
	r := make([]LogFieldMarshaller, len(a), n)
	copy(r, a)
	for _, bb := range b {
		if bb != nil {
			r = append(r, bb)
		}
	}
	return r
}

