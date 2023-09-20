package logfmt

import (
	"fmt"
	"time"
)

func GetDefaultLogMsgFormatter() MsgFormatConfig {
	return MsgFormatConfig{
		Sformat:  fmt.Sprint,
		Sformatf: fmt.Sprintf,
		MFactory: GetMarshallerFactory(),
		TimeFmt:  time.RFC3339,
	}
}
