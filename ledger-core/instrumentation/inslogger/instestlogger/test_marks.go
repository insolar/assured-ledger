package instestlogger

import (
	"fmt"
	"io"
	"strings"
	"time"
)

type markerT interface {
	Name() string
	Cleanup(func())
	Failed() bool
	Skipped() bool
}

func emulateTestText(out io.Writer, tLog interface{}, nowFn func() time.Time) {
	t, ok := tLog.(markerT)
	if !ok {
		return
	}

	startedAt := nowFn()
	_, _ = fmt.Fprintln(out, `=== RUN  `, t.Name())
	_emulateTestFinish(out, t, startedAt, nowFn,nil)
}

func emulateTestJSON(out io.Writer, tLog interface{}, nowFn func() time.Time) {
	t, ok := tLog.(markerT)
	if !ok {
		return
	}

	packageName := "" // ? how to get it ?
	const timestampFormat = "2006-01-02T15:04:05.000000000Z"

	startedAt := nowFn()
	_, _ = fmt.Fprintf(out, `{"Time":"%s","Action":"run","Package":"%s","Test":"%s"}`+"\n",
		startedAt.Format(timestampFormat),
		packageName, t.Name())

	_, _ = fmt.Fprintf(out, `{"Time":"%s","Action":"output","Package":"%s","Test":"%s","Output":"=== RUN   %s"}`+"\n",
		nowFn().Format(timestampFormat),
		packageName, t.Name(), t.Name())


	_emulateTestFinish(out, t, startedAt, nowFn, func(d time.Duration, resName, msg string) {
		resName = strings.ToLower(resName)

		_, _ = fmt.Fprintf(out, `{"Time":"%s","Action":"output","Package":"%s","Test":"%s","Output":"%s"}`+"\n",
			nowFn().Format(timestampFormat),
			packageName, t.Name(), msg)

		_, _ = fmt.Fprintf(out, `{"Time":"%s","Action":"%s","Package":"%s","Test":"%s","Elapsed":%.3f}`+"\n",
			nowFn().Format(timestampFormat), resName,
			packageName, t.Name(), d.Seconds())
	})
}

func _emulateTestFinish(out io.Writer, t markerT, startedAt time.Time, nowFn func() time.Time, fn func(d time.Duration, resName, msg string)) {
	t.Cleanup(func() {
		d := nowFn().Sub(startedAt)
		resName := ""
		switch {
		case t.Failed():
			resName = "FAIL"
		case t.Skipped():
			resName = "SKIP"
		default:
			resName = "PASS"
		}

		if fn == nil {
			_, _ = fmt.Fprintf(out, "--- %s: %s (%.2fs)\n", resName, t.Name(), d.Seconds())
			return
		}
		msg := fmt.Sprintf("--- %s: %s (%.2fs)", resName, t.Name(), d.Seconds())
		fn(d, resName, msg)
	})
}
