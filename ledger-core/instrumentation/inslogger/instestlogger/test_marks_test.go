package instestlogger

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEmulateTextMarks(t *testing.T) {
	var emuT markerT

	emuT = &mockT{name: "emuTest"}
	emuTime := mockTime{time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}
	out := &bytes.Buffer{}
	emulateTestText(out, emuT, emuTime.Now)

	expected := "=== RUN   emuTest\n--- PASS: emuTest (1.00s)\n"
	require.Equal(t, expected, string(out.Bytes()))

	emuT = &mockT{name: "emuTest", skip: true}
	emuTime = mockTime{time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}
	out = &bytes.Buffer{}

	emulateTestText(out, emuT, emuTime.Now)

	expected = "=== RUN   emuTest\n--- SKIP: emuTest (1.00s)\n"
	require.Equal(t, expected, string(out.Bytes()))

	emuT = &mockT{name: "emuTest", fail: true}
	emuTime = mockTime{time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}
	out = &bytes.Buffer{}
	emulateTestText(out, emuT, emuTime.Now)

	expected = "=== RUN   emuTest\n--- FAIL: emuTest (1.00s)\n"
	require.Equal(t, expected, string(out.Bytes()))
}

func TestEmulateJSONMarks(t *testing.T) {
	var emuT markerT

	emuT = &mockT{name: "emuTest"}
	emuTime := mockTime{time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}
	out := &bytes.Buffer{}
	emulateTestJSON(out, emuT, emuTime.Now)

	expected :=
		`{"Time":"2020-01-01T00:00:00.000000000Z","Action":"run","Package":"","Test":"emuTest"}` + "\n" +
		`{"Time":"2020-01-01T00:00:01.000000000Z","Action":"output","Package":"","Test":"emuTest","Output":"=== RUN   emuTest"}` + "\n" +
		`{"Time":"2020-01-01T00:00:03.000000000Z","Action":"output","Package":"","Test":"emuTest","Output":"--- PASS: emuTest (2.00s)"}` + "\n" +
		`{"Time":"2020-01-01T00:00:04.000000000Z","Action":"pass","Package":"","Test":"emuTest","Elapsed":2.000}` + "\n"
	require.Equal(t, expected, string(out.Bytes()))

	emuT = &mockT{name: "emuTest", skip: true}
	emuTime = mockTime{time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}
	out = &bytes.Buffer{}

	emulateTestJSON(out, emuT, emuTime.Now)

	expected =
		`{"Time":"2020-01-01T00:00:00.000000000Z","Action":"run","Package":"","Test":"emuTest"}` + "\n" +
		`{"Time":"2020-01-01T00:00:01.000000000Z","Action":"output","Package":"","Test":"emuTest","Output":"=== RUN   emuTest"}` + "\n" +
		`{"Time":"2020-01-01T00:00:03.000000000Z","Action":"output","Package":"","Test":"emuTest","Output":"--- SKIP: emuTest (2.00s)"}` + "\n" +
		`{"Time":"2020-01-01T00:00:04.000000000Z","Action":"skip","Package":"","Test":"emuTest","Elapsed":2.000}` + "\n"
	require.Equal(t, expected, string(out.Bytes()))

	emuT = &mockT{name: "emuTest", fail: true}
	emuTime = mockTime{time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}
	out = &bytes.Buffer{}

	emulateTestJSON(out, emuT, emuTime.Now)

	expected =
		`{"Time":"2020-01-01T00:00:00.000000000Z","Action":"run","Package":"","Test":"emuTest"}` + "\n" +
		`{"Time":"2020-01-01T00:00:01.000000000Z","Action":"output","Package":"","Test":"emuTest","Output":"=== RUN   emuTest"}` + "\n" +
		`{"Time":"2020-01-01T00:00:03.000000000Z","Action":"output","Package":"","Test":"emuTest","Output":"--- FAIL: emuTest (2.00s)"}` + "\n" +
		`{"Time":"2020-01-01T00:00:04.000000000Z","Action":"fail","Package":"","Test":"emuTest","Elapsed":2.000}` + "\n"
	require.Equal(t, expected, string(out.Bytes()))
}

type mockTime struct {
	t time.Time
}

func (p *mockTime) Now() time.Time {
	r := p.t
	p.t = p.t.Add(time.Second)
	return r
}

type mockT struct {
	name string
	fail bool
	skip bool
}

func (p *mockT) Name() string {
	return p.name
}

func (p *mockT) Cleanup(f func()) {
	f()
}

func (p *mockT) Failed() bool {
	return p.fail
}

func (p *mockT) Skipped() bool {
	return p.skip
}
