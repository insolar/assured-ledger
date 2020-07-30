// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build convlogtxt

package convlog

func UseTextConvLog() bool {
	return useTextConvLog
}

var useTextConvLog = true

func DisableTextConvLog() {
	useTextConvLog = false
}
