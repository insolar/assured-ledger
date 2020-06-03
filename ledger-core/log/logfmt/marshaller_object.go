// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package logfmt

import (
	"reflect"
	"strconv"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/reflectkit"
)

type objectMarshaller struct {
	t *typeMarshaller
	v reflect.Value
}

func (v objectMarshaller) MarshalLogObject(output LogObjectWriter, collector LogObjectMetricCollector) (string, bool) {
	return v.t.printFields(v.v, output, collector)
}

func (v objectMarshaller) MarshalMutedLogObject(collector LogObjectMetricCollector) {
	if collector == nil {
		return
	}
	v.t.reportFields(v.v, collector)
}

type typeMarshaller struct {
	loggerFields    []logField
	metricFields    []reportField
	msgField        logField
	printNeedsAddr  bool
	reportNeedsAddr bool
	hasReports      bool
	defMessage      bool
}

type reportField struct {
	fieldGet reflectkit.FieldGetterFunc
	reportFn FieldReporterFunc
	name     string
}

type logField struct {
	fieldGet reflectkit.FieldGetterFunc
	reportFn FieldReporterFunc
	getterFn reflectkit.ValueToReceiverFunc
	receiver logFieldReceiver
}

type logFieldReceiver struct {
	key    string
	fmtStr string
	fmtTag fmtTagType
}

type fieldDesc struct {
	reflect.StructField
	getterFn   reflectkit.ValueToReceiverFunc
	index      int
	reportFn   FieldReporterFunc
	outputRecv logFieldReceiver
}

func (p *typeMarshaller) getFieldsOf(t reflect.Type, baseOffset uintptr, getReporterFn func(reflect.Type) FieldReporterFunc) bool {
	n := t.NumField()
	var msgGetter fieldDesc
	valueGetters := make([]fieldDesc, 0, n)

	for i := 0; i < n; i++ {
		tf := t.Field(i)
		fieldName := tf.Name

		if tf.Type.Kind() == reflect.Struct && tf.Type.NumField() == 0 && fieldName == "_logignore" {
			// struct with _logignore struct{} field must be ignored
			return false
		}

		k := tf.Type.Kind()
		valueGetterFactory := reflectkit.ValueToReceiverFactory(k, marshallerSpecialTypes)
		if valueGetterFactory == nil {
			continue
		}
		unexported := len(tf.PkgPath) != 0

		fd := fieldDesc{StructField: tf, index: i}

		fd.reportFn = getReporterFn(fd.Type)
		tagType, fmtStr := singleTag(fd.Tag)
		if !tagType.HasStr() {
			fmtStr = ""
		}

		msgField := false
		switch {
		case tagType == fmtTagText && fieldName == "_":
			msgField = true
		case fd.reportFn != nil:
			//
		case tagType == fmtTagSkip:
			continue
		case fieldName == "" || fieldName[0] == '_':
			continue
		case !fd.Anonymous:
			//
		case tagType == fmtTagText:
			msgField = true
		default:
			switch k := fd.Type.Kind(); {
			case k == reflect.String:
				msgField = fieldName == "string"
			case k > reflect.Array: // any other non-literals
				continue
			}
		}

		if !msgField {
			switch fieldName {
			case "msg", "Msg", "message", "Message":
				msgField = true
			}
		}

		var needsAddr bool
		switch {
		case tagType != fmtTagSkip && tagType != fmtTagText:
			needsAddr, fd.getterFn = valueGetterFactory(unexported, tf.Type, tagType.IsOpt())
		case msgField || tagType == fmtTagText:
			fd.getterFn = func(_ reflect.Value, out reflectkit.TypedReceiver) {
				out.ReceiveZero(reflect.String)
			}
		default:
			fd.getterFn = func(reflect.Value, reflectkit.TypedReceiver) {}
		}

		fd.outputRecv = logFieldReceiver{fd.StructField.Name, fmtStr, tagType}

		switch {
		case !msgField:
			valueGetters = append(valueGetters, fd)
		case msgGetter.getterFn == nil:
			msgGetter = fd
		case fieldName == "_":
			fd.outputRecv.key = "_txtTag" + strconv.Itoa(i)
			fallthrough
		default:
			valueGetters = append(valueGetters, fd)
		}

		p.printNeedsAddr = needsAddr || p.printNeedsAddr
		if fd.reportFn != nil {
			p.hasReports = true
			p.reportNeedsAddr = needsAddr || p.reportNeedsAddr
		}
	}

	if p.reportNeedsAddr && !p.printNeedsAddr {
		panic("illegal state")
	}

	if len(valueGetters) == 0 && msgGetter.getterFn == nil {
		return false
	}

	p.loggerFields = make([]logField, 0, len(valueGetters))

	for _, fd := range valueGetters {
		fieldGetter := reflectkit.FieldValueGetter(fd.index, fd.StructField, p.printNeedsAddr, baseOffset)

		if fd.outputRecv.fmtTag != fmtTagSkip {
			p.loggerFields = append(p.loggerFields, logField{
				fieldGetter, fd.reportFn, fd.getterFn, fd.outputRecv})
		}

		if fd.reportFn != nil {
			reportFieldGetter := fieldGetter
			if p.reportNeedsAddr != p.printNeedsAddr {
				reportFieldGetter = reflectkit.FieldValueGetter(fd.index, fd.StructField, p.reportNeedsAddr, baseOffset)
			}
			p.metricFields = append(p.metricFields, reportField{
				reportFieldGetter, fd.reportFn, fd.Name})
		}
	}

	p.defMessage = true
	switch {
	case msgGetter.getterFn != nil:
		p.defMessage = false
	case len(t.PkgPath()) != 0:
		if s := t.String(); len(s) > 0 {
			p.msgField = logField{
				fieldGet: func(_ reflect.Value) reflect.Value {
					return reflect.Value{}
				},
				getterFn: func(value reflect.Value, out reflectkit.TypedReceiver) {
					out.ReceiveString(reflect.String, s)
				},
			}
			return true
		}
		fallthrough
	default:
		p.msgField = logField{}
		return true
	}

	fieldGetter := reflectkit.FieldValueGetter(msgGetter.index, msgGetter.StructField, p.printNeedsAddr, baseOffset)
	p.msgField = logField{
		fieldGetter, msgGetter.reportFn, msgGetter.getterFn, msgGetter.outputRecv}
	p.msgField.receiver.key = ""

	return true
}

func (p *typeMarshaller) prepareValue(value reflect.Value) reflect.Value {
	return p._prepareValue(value, p.printNeedsAddr)
}

func (p *typeMarshaller) _prepareValue(value reflect.Value, needsAddr bool) reflect.Value {
	if !needsAddr {
		return value
	}
	return reflectkit.MakeAddressable(value)
}

func printAndReportField(v reflect.Value, receiver fieldFmtReceiver, c LogObjectMetricCollector) {
	fieldValue := receiver.fieldGet(v)
	if receiver.reportFn != nil {
		receiver.reportFn(c, receiver.receiver.key, fieldValue.Interface())
	}
	receiver.getterFn(fieldValue, receiver)
}

func printField(v reflect.Value, receiver fieldFmtReceiver) {
	fieldValue := receiver.fieldGet(v)
	receiver.getterFn(fieldValue, receiver)
}

func (p *typeMarshaller) printFields(value reflect.Value, writer LogObjectWriter, collector LogObjectMetricCollector) (string, bool) {
	value = p._prepareValue(value, p.printNeedsAddr) // double check

	receiver := fieldFmtReceiver{w: writer}

	doReports := collector != nil && p.hasReports
	if doReports {
		for i := range p.loggerFields {
			receiver.logField = &p.loggerFields[i]
			printAndReportField(value, receiver, collector)
		}
	} else {
		for i := range p.loggerFields {
			receiver.logField = &p.loggerFields[i]
			printField(value, receiver)
		}
	}

	if p.msgField.getterFn == nil {
		return "", true
	}

	sc := stringCapturer{}
	receiver.logField = &p.msgField
	receiver.w = &sc
	if doReports {
		printAndReportField(value, receiver, collector)
	} else {
		printField(value, receiver)
	}
	return sc.v, p.defMessage
}

func (p *typeMarshaller) reportFields(value reflect.Value, collector LogObjectMetricCollector) {
	if len(p.metricFields) == 0 {
		return
	}

	value = p._prepareValue(value, p.reportNeedsAddr) // double check

	for _, field := range p.metricFields {
		fieldValue := field.fieldGet(value)
		field.reportFn(collector, field.name, fieldValue.Interface())
	}
}

type fmtTagType uint8

const (
	fmtTagDefault fmtTagType = iota
	fmtTagOptional

	fmtTagText
	fmtTagSkip // + opt

	fmtTagFormatRaw
	fmtTagFormatRawOpt // + opt

	fmtTagFormatValue
	fmtTagFormatValueOpt // + opt
)

func (v fmtTagType) IsOpt() bool {
	return v&fmtTagOptional != 0
}

func (v fmtTagType) IsRaw() bool {
	return v&^1 == fmtTagFormatRaw
}

func (v fmtTagType) HasFmt() bool {
	return v >= fmtTagFormatRaw
}

func (v fmtTagType) HasStr() bool {
	return v == fmtTagText || v.HasFmt()
}

func singleTag(tag reflect.StructTag) (fmtTagType, string) {
	tagType := fmtTagDefault
	if _, v, ok := reflectkit.ParseStructTag(tag, func(name, _ string) bool {
		switch name {
		case "fmt+opt", "opt+fmt":
			tagType = fmtTagFormatValueOpt
		case "fmt":
			tagType = fmtTagFormatValue
		case "raw+opt", "opt+raw":
			tagType = fmtTagFormatRawOpt
		case "raw":
			tagType = fmtTagFormatRaw
		case "skip":
			tagType = fmtTagSkip
		case "txt":
			tagType = fmtTagText
		case "opt":
			tagType = fmtTagOptional
		default:
			return false
		}
		return true
	}); ok {
		return tagType, v
	}
	return fmtTagDefault, ""
}

func marshallerSpecialTypes(t reflect.Type, checkZero bool) reflectkit.IfaceToReceiverFunc {
	var prepFn valuePrepareFn

	switch kind := t.Kind(); kind {
	case reflect.Interface:
		prepFn = prepareValue

	case reflect.Ptr:
		if te := t.Elem(); te.Kind() == reflect.String {
			return func(value interface{}, _ reflect.Kind, out reflectkit.TypedReceiver) {
				if vv := value.(*string); vv == nil {
					out.ReceiveNil(reflect.String)
				} else if v := *vv; !checkZero || v != "" {
					out.ReceiveString(reflect.String, v)
				} else {
					out.ReceiveZero(reflect.String)
				}
			}
		}
		fallthrough

	default:
		prepFn = findPrepareValueFn(t)
		if prepFn == nil {
			return nil
		}
	}

	return func(value interface{}, kind reflect.Kind, out reflectkit.TypedReceiver) {
		switch s, k, isNil := prepFn(value); {
		case k == reflect.Invalid:
			out.ReceiveElse(kind, value, false)
		case isNil:
			out.ReceiveNil(kind)
		case !checkZero || s != "":
			out.ReceiveString(kind, s)
		default:
			out.ReceiveZero(kind)
		}
	}
}
