// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rmsreg

const MaxTwoByteField = 2047

const RecordBodyField = 19
const RecordBodyTagSize = 2

const RecordFirstField = 20
const RecordLastField = MaxTwoByteField - 256 // 1791

const MessageRecordPayloadsField = 17 // MUST be immediately after 16
const MessageRecordField = RecordBodyField
const MessageFirstField = 1800
