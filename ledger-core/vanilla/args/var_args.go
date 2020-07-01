// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package args

func AppendIntfArgs(args []interface{}, moreArgs ...interface{}) []interface{} {
	switch {
	case len(moreArgs) == 0:
		return args
	case len(args) == 0:
		return moreArgs
	default:
		return append(append(make([]interface{}, 0, len(args) + len(moreArgs)), args...), moreArgs...)
	}
}
