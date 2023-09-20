package rmsreg

const MaxTwoByteField = 2047

const RecordBodyField = 19
const RecordBodyTagSize = 2

const RecordFirstField = 20
const RecordLastField = MaxTwoByteField - 256 // 1791

const MessageRecordPayloadsField = 17 // MUST be immediately after 16
const MessageRecordField = RecordBodyField
const MessageFirstField = 1800
