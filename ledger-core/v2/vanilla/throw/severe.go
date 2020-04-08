// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package throw

import "errors"

type Severity uint8

const (
	_ Severity = iota
	NormalSeverity
	BlameSeverity
	ViolationSeverity
	FraudSeverity
	RemoteBreachSeverity
	LocalBreachSeverity
	FatalSeverity
)

func (v Severity) IsFatal() bool {
	return v >= LocalBreachSeverity
}

func (v Severity) IsError() bool {
	return v > RemoteBreachSeverity
}

func (v Severity) IsWarn() bool {
	return v > NormalSeverity
}

func (v Severity) IsDeadCanary() bool {
	return v >= LocalBreachSeverity
}

func (v Severity) IsCompromised() bool {
	return v >= RemoteBreachSeverity
}

func (v Severity) IsFraudOrWorse() bool {
	return v >= FraudSeverity
}

func Blame(msg string, description ...interface{}) error {
	return Severe(BlameSeverity, msg, description...)
}

func Violation(msg string, description ...interface{}) error {
	return Severe(ViolationSeverity, msg, description...)
}

func Fraud(msg string, description ...interface{}) error {
	return Severe(FraudSeverity, msg, description...)
}

func RemoteBreach(msg string, description ...interface{}) error {
	return Severe(RemoteBreachSeverity, msg, description...)
}

func LocalBreach(msg string, description ...interface{}) error {
	return Severe(LocalBreachSeverity, msg, description...)
}

func DeadCanary(msg string, description ...interface{}) error {
	return Severe(LocalBreachSeverity, msg, description...)
}

func Fatal(msg string, description ...interface{}) error {
	return Severe(FatalSeverity, msg, description...)
}

func WithSeverity(err error, s Severity) error {
	switch {
	case s == 0:
		return err
	case err == nil:
		return nil
	default:
		return severityWrap{err, s}
	}
}

func WithDefaultSeverity(err error, s Severity) error {
	switch {
	case s == 0:
		return err
	case err == nil:
		return nil
	default:
		if _, ok := GetSeverity(err); ok {
			return err
		}
		return severityWrap{err, s}
	}
}

func SeverityOf(errChain error) Severity {
	s, _ := GetSeverity(errChain)
	return s
}

func GetSeverity(errChain error) (Severity, bool) {
	for errChain != nil {
		var s Severity
		switch v := errChain.(type) {
		case fmtWrap:
			s = v.severity
		case panicWrap:
			s = v.severity
		//case detailsWrap:
		//	s = v.details.severity
		case severityWrap:
			return v.severity, true
		}
		if s != 0 {
			return s, true
		}
		errChain = errors.Unwrap(errChain)
	}
	return NormalSeverity, false
}
