// +build convlogtxt

package convlog

func UseTextConvLog() bool {
	return useTextConvLog
}

var useTextConvLog = true

func DisableTextConvLog() {
	useTextConvLog = false
}
