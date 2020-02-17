// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package logfmt

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

func TestErrorMarshaller_MarshalLogObject_struct(t *testing.T) {
	reportMsg := "report"
	err := throw.E("start", struct {
		string
		f0 int
		f1 string
	}{"main", 1, "ABC"})
	err = throw.WithDetails(err, struct {
		string
		f2 int
	}{"ext", 2})
	err = throw.WithStack(err)
	err = throw.WithDetails(err, io.EOF)
	err = throw.WithDetails(err, throw.E(reportMsg, struct{ f3 uint }{3}))
	err = throw.WithStack(err) // repeated stack trace capture should not pollute the output

	s, o := fmtError(t, err)
	assert.Equal(t, reportMsg, s)
	assert.Contains(t, o, "f3:3:uint,errorMsg:EOF,f2:2:int,errorMsg:ext,f0:1:int,f1:ABC:string,errorMsg:main,errorMsg:start,errorStack:")
	assert.Equal(t, 1, strings.Count(o, "TestErrorMarshaller_MarshalLogObject"))
}

func fmtError(t *testing.T, err error) (string, string) {
	m, sp := fmtLogStruct(err, GetMarshallerFactory(), false, false)
	require.Nil(t, sp)
	require.NotNil(t, m)

	o := output{}
	s := m.MarshalLogObject(&o, nil)
	return s, o.buf.String()
}

func TestErrorMarshaller_MarshalLogObject_simple(t *testing.T) {
	s, o := fmtError(t, io.EOF)
	assert.Equal(t, io.EOF.Error(), s)
	assert.Equal(t, "", o)

	err := fmt.Errorf("wrapper %w", io.EOF)
	s, o = fmtError(t, err)
	assert.Equal(t, "wrapper %w", s)
	assert.Equal(t, "errorMsg:EOF,", o)

	s, o = fmtError(t, throw.WithStack(err))
	assert.Equal(t, "wrapper %w", s)
	assert.Contains(t, o, "errorMsg:EOF,errorStack:")
	assert.Equal(t, 1, strings.Count(o, "TestErrorMarshaller_MarshalLogObject_simple"))

	s, o = fmtError(t, throw.WithDetails(err, struct{ x int }{99}))
	assert.Equal(t, "wrapper %w", s)
	assert.Equal(t, "x:99:int,errorMsg:EOF,", o)
}

func TestErrorMarshaller_MarshalLogObject_mixed(t *testing.T) {
	err := throw.E("start", struct {
		string
		f0 int
		f1 string
	}{"main", 1, "ABC"})
	err = throw.WithDetails(err, struct { // same message "wrapper" here must not be deduplicated with "wrapper" of fmt.Errorf
		string
		f2 int
	}{"ext", 2})
	err = throw.WithStack(err)

	err = fmt.Errorf("wrapper %w", err) // mess up the chain - get stack and other parts to be blended-in
	assert.Equal(t, 1, strings.Count(err.Error(), "TestErrorMarshaller_MarshalLogObject_mixed"))

	err = throw.WithDetails(err, throw.E("panicMsg", struct{ f3 uint }{3}))
	err = throw.WithStack(err) // repeated stack trace capture should not pollute the output

	s, o := fmtError(t, err)
	assert.Equal(t, "panicMsg", s)
	assert.Contains(t, o, "f3:3:uint,errorMsg:wrapper %w,f2:2:int,errorMsg:ext,f0:1:int,f1:ABC:string,errorMsg:main,errorMsg:start,errorStack:")
	assert.Equal(t, 1, strings.Count(o, "TestErrorMarshaller_MarshalLogObject_mixed"))
	assert.Equal(t, 1, strings.Count(err.Error(), "TestErrorMarshaller_MarshalLogObject_mixed"))
}

func TestErrorMarshaller_MarshalLogObject_repeated(t *testing.T) {
	err := throw.E("m", struct{}{})
	err = throw.WithDetails(err, struct{ string }{"mm"})
	err = throw.WithDetails(err, struct{ string }{"mm"})
	err = throw.WithDetails(err, throw.E("mm", struct{ string }{"mm"}))
	err = throw.WithDetails(err, throw.E("mm", struct{ string }{"mm"}))
	err = throw.WithStack(err)
	err = fmt.Errorf("mmm %w", err)
	err = throw.WithDetails(err, struct{ string }{"mm"})
	err = throw.WithStack(err)
	err = fmt.Errorf("mmmm %w", err)
	err = fmt.Errorf("mmmm %w", err)
	s, o := fmtError(t, err)
	assert.Equal(t, "mmmm %w", s)
	assert.Contains(t, o, "errorMsg:mmmm %w,errorMsg:mm,errorMsg:mmm %w,errorMsg:mm,errorMsg:mm,errorMsg:mm,errorMsg:mm,errorMsg:mm,errorMsg:mm,errorMsg:m,errorStack:")
	assert.Equal(t, 1, strings.Count(o, "TestErrorMarshaller_MarshalLogObject_repeated"))
	assert.Equal(t, 1, strings.Count(err.Error(), "TestErrorMarshaller_MarshalLogObject_repeated"))
}

func TestErrorMarshaller_MarshalLogObject_skipTop(t *testing.T) {
	err := throw.IllegalState()
	err = throw.WithStack(err)
	s, o := fmtError(t, err)
	assert.Equal(t, "illegal state", s)
	assert.True(t, strings.HasPrefix(o, "errorStack:"))
	assert.Equal(t, 1, strings.Count(o, "TestErrorMarshaller_MarshalLogObject_skipTop"))
	assert.Equal(t, 1, strings.Count(err.Error(), "TestErrorMarshaller_MarshalLogObject_skipTop"))
}

func TestErrorMarshaller_MarshalLogObject_stackTop(t *testing.T) {
	err := throw.IllegalState()
	s, o := fmtError(t, err)
	assert.Equal(t, "illegal state", s)
	assert.True(t, strings.HasPrefix(o, "errorStack:"))
	assert.Equal(t, 1, strings.Count(o, "TestErrorMarshaller_MarshalLogObject_stackTop"))
	assert.Equal(t, 1, strings.Count(err.Error(), "TestErrorMarshaller_MarshalLogObject_stackTop"))

	err = throw.WithDetails(err, struct{ string }{"mm"})
	err = throw.WithStack(err)
	s, o = fmtError(t, err)
	assert.Equal(t, "illegal state", s)
	assert.Contains(t, o, "errorMsg:mm,errorStack:")
	//fmt.Println(o)
	assert.Equal(t, 1, strings.Count(o, "TestErrorMarshaller_MarshalLogObject_stackTop"))
	assert.Equal(t, 1, strings.Count(err.Error(), "TestErrorMarshaller_MarshalLogObject_stackTop"))
}
