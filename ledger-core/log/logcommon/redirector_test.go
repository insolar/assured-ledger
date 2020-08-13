// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package logcommon

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type outMock struct {
	closed bool
	flushed bool
	written []byte
	err error
}

func (p *outMock) Close() error {
	p.closed = true
	return p.err
}

func (p *outMock) Flush() error {
	p.flushed = true
	return p.err
}

func (p *outMock) Write(b []byte) (int, error) {
	p.written = b
	return len(b), p.err
}

type tMock struct {
	log string
	err string
	ftl string
}

func (p *tMock) Helper() {}

func (p *tMock) Log(i ...interface{}) {
	p.log = fmt.Sprint(i...)
}

func (p *tMock) Error(i ...interface{}) {
	p.err = fmt.Sprint(i...)
}

func (p *tMock) Fatal(i ...interface{}) {
	p.ftl = fmt.Sprint(i...)
}

func TestTestingLoggerOutputClose(t *testing.T) {
	to := &TestingLoggerOutput{}
	require.NoError(t, to.Close())

	m1 := &outMock{}
	to.Output = m1
	require.NoError(t, to.Close())
	require.True(t, m1.closed)

	m1.closed = false
	m2 := &outMock{}
	to.EchoTo = m2
	require.NoError(t, to.Close())
	require.True(t, m1.closed)
	require.True(t, m2.closed)

	m2.err = errors.New("err2")
	require.NoError(t, to.Close())

	m1.err = errors.New("err1")
	err := to.Close()
	require.Error(t, err)
	require.Equal(t, "err1", err.Error())
}

func TestTestingLoggerOutputFlush(t *testing.T) {
	to := &TestingLoggerOutput{}
	require.NoError(t, to.Flush())

	m1 := &outMock{}
	to.Output = m1
	require.NoError(t, to.Flush())
	require.True(t, m1.flushed)

	m1.flushed = false
	m2 := &outMock{}
	to.EchoTo = m2
	require.NoError(t, to.Flush())
	require.True(t, m1.flushed)
	require.True(t, m2.flushed)

	m2.err = errors.New("err2")
	require.NoError(t, to.Flush())

	m1.err = errors.New("err1")
	err := to.Flush()
	require.Error(t, err)
	require.Equal(t, "err1", err.Error())
}

func TestTestingLoggerOutputWrite(t *testing.T) {
	sample := []byte("testSample")
	to := &TestingLoggerOutput{}
	n, err := to.Write(sample)
	require.NoError(t, err)
	require.Equal(t, len(sample), n)

	m1 := &outMock{}
	to.Output = m1
	n, err = to.Write(sample)
	require.NoError(t, err)
	require.Equal(t, len(sample), n)
	require.Equal(t, sample, m1.written)

	m1.written = nil
	n, err = to.LogLevelWrite(NoLevel, sample)
	require.NoError(t, err)
	require.Equal(t, len(sample), n)
	require.Equal(t, sample, m1.written)


	m2 := &outMock{}
	to.EchoTo = m2
	n, err = to.Write(sample)
	require.NoError(t, err)
	require.Equal(t, len(sample), n)
	require.Equal(t, sample, m1.written)
	require.Equal(t, sample, m2.written)

	m1.written = nil
	m2.written = nil
	n, err = to.LogLevelWrite(NoLevel, sample)
	require.NoError(t, err)
	require.Equal(t, len(sample), n)
	require.Equal(t, sample, m1.written)
	require.Equal(t, sample, m2.written)

	m2.err = errors.New("err2")
	_, err = to.Write(sample)
	require.NoError(t, err)

	m1.err = errors.New("err1")
	_, err = to.Write(sample)
	require.Error(t, err)
	require.Equal(t, "err1", err.Error())
}

func TestTestingLoggerOutputErrorOrPanic(t *testing.T) {
	sample := []byte("testSample")

	for _, level := range []Level{ErrorLevel, PanicLevel} {
		to := &TestingLoggerOutput{LogFiltered: true}
		m1 := &outMock{}
		mt := &tMock{}
		to.Output = m1
		to.Testing = mt

		n, err := to.LogLevelWrite(level, sample)
		require.Equal(t, len(sample), n)
		require.NoError(t, err)
		require.Equal(t, sample, m1.written)
		require.Equal(t, "", mt.log)
		require.Equal(t, "testSample", mt.err)
		require.Equal(t, "", mt.ftl)

		m1.written = nil
		mt.err = ""

		to.ErrorFilterFn = func(string) bool { return false }
		n, err = to.LogLevelWrite(level, sample)
		require.Equal(t, len(sample), n)
		require.NoError(t, err)
		require.Equal(t, sample, m1.written)
		require.Equal(t, "testSample", mt.log)
		require.Equal(t, "", mt.err)
		require.Equal(t, "", mt.ftl)
	}
}

func TestTestingLoggerOutputFatal(t *testing.T) {
	sample := []byte("testSample")

	to := &TestingLoggerOutput{}
	m1 := &outMock{}
	mt := &tMock{}
	to.Output = m1
	to.Testing = mt

	n, err := to.LogLevelWrite(FatalLevel, sample)
	require.Equal(t, len(sample), n)
	require.NoError(t, err)
	require.Equal(t, sample, m1.written)
	require.Equal(t, "", mt.log)
	require.Equal(t, "", mt.err)
	require.Equal(t, "testSample", mt.ftl)

	m1.written = nil
	mt.ftl = ""

	to.ErrorFilterFn = func(string) bool { return false }
	n, err = to.LogLevelWrite(FatalLevel, sample)
	require.Equal(t, len(sample), n)
	require.NoError(t, err)
	require.Equal(t, sample, m1.written)
	require.Equal(t, "", mt.log)
	require.Equal(t, "", mt.err)
	require.Equal(t, "testSample", mt.ftl)

	intercept := false
	to.InterceptFatal = func(b []byte) bool {
		require.Equal(t, sample, b)
		return intercept
	}

	m1.written = nil
	mt.ftl = ""
	n, err = to.LogLevelWrite(FatalLevel, sample)
	require.Equal(t, len(sample), n)
	require.NoError(t, err)
	require.Equal(t, sample, m1.written)
	require.Equal(t, "", mt.log)
	require.Equal(t, "", mt.err)
	require.Equal(t, "testSample", mt.ftl)

	intercept = true

	m1.written = nil
	mt.ftl = ""

	to.ErrorFilterFn = func(string) bool { return false } // to.SuppressTestError = true does NOT apply to intercepted panics
	n, err = to.LogLevelWrite(FatalLevel, sample)
	require.Equal(t, len(sample), n)
	require.NoError(t, err)
	require.Equal(t, sample, m1.written)
	require.Equal(t, "", mt.log)
	require.Equal(t, "testSample", mt.err)
	require.Equal(t, "", mt.ftl)
}
