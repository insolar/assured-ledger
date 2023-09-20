package prettylog

import (
	"fmt"
	"os"
	"strings"

	"github.com/insolar/assured-ledger/ledger-core/log/logoutput"
)

const TimestampFormat = "2006-01-02T15:04:05.000000000Z07:00"

var Defaults = Config{
	NoColor:      true,
	TimeFormat:   TimestampFormat,
	FormatCaller: formatCaller,
	PartsOrder: []string{
		logoutput.TimestampFieldName,
		logoutput.LevelFieldName,
		logoutput.MessageFieldName,
		logoutput.CallerFieldName,
	},
}

type Formatter = func(interface{}) string

type Config struct {
	Enable bool

	// NoColor disables the colorized output.
	NoColor bool

	// TimeFormat specifies the format for timestamp in output.
	TimeFormat string

	// PartsOrder defines the order of parts in output.
	PartsOrder []string

	FormatTimestamp     Formatter
	FormatLevel         Formatter
	FormatCaller        Formatter
	FormatMessage       Formatter
	FormatFieldName     Formatter
	FormatFieldValue    Formatter
	FormatErrFieldName  Formatter
	FormatErrFieldValue Formatter
}

func formatCaller(i interface{}) string {
	var c string
	if cc, ok := i.(string); ok {
		c = cc
	}
	if len(c) > 0 {
		if len(cwd) > 0 {
			c = strings.TrimPrefix(c, cwd)
			c = strings.TrimPrefix(c, "/")
		}
		c = logoutput.CallerFieldName + "=" + c
	}
	return c
}

var cwd string

func init() {
	var err error
	cwd, err = os.Getwd()
	if err != nil {
		cwd = ""
		fmt.Println("couldn't get current working directory: ", err.Error())
	}
}
