package utils

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/gojuno/minimock/v3"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/testutils/insrail"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/mock/publisher/checker"
)

var duplicateTestRailID = make(map[string]string)

func assureTestRailID(id string, t *testing.T) {
	if id != "" {
		if ok, err := regexp.Match("^C[0-9]{4,}$", []byte(id)); err != nil {
			panic("failed to compile regexp")
		} else if !ok {
			panic(fmt.Sprintf("bad TestRailID, expected C****, got %s", id))
		}

		if val, ok := duplicateTestRailID[id]; ok && val != t.Name() {
			panic(fmt.Sprintf("duplicate testRailID for %s", id))
		} else if !ok {
			duplicateTestRailID[id] = t.Name()
		}
	}
}

type TestCase struct {
	Name       string
	TestRailID string
	Skipped    string

	parallel    bool
	errorFilter func(s string) bool

	Minimock     *minimock.Controller
	Server       *Server
	Context      context.Context
	Runner       *logicless.ServiceMock
	TypedChecker *checker.Typed
}

func NewTestCase(name string) *TestCase {
	return &TestCase{Name: name}
}

func NewSkippedTestCase(name string, jiraLink string) *TestCase {
	if jiraLink == "" {
		return NewTestCase(name)
	}
	return &TestCase{Name: name, Skipped: jiraLink}
}

func (c *TestCase) WithErrorFilter(fn func(string) bool) *TestCase {
	c.errorFilter = fn
	return c
}

func (c *TestCase) WithSkipped(skipped string) *TestCase {
	c.Skipped = skipped
	return c
}

func (c *TestCase) WithTestRailID(id string) *TestCase {
	c.TestRailID = id
	return c
}

func (c *TestCase) init(t *testing.T) {
	c.Minimock = minimock.NewController(t)

	ctx := instestlogger.TestContext(t)
	c.Server, c.Context = NewUninitializedServerWithErrorFilter(ctx, t, c.errorFilter)

	c.Runner = logicless.NewServiceMock(c.Context, c.Minimock, nil)
	c.Server.ReplaceRunner(c.Runner)
	c.TypedChecker = c.Server.PublisherMock.SetTypedCheckerWithLightStubs(c.Context, c.Minimock, c.Server)

	c.Server.Init(c.Context)
}

func (c *TestCase) stop(_ *testing.T) {
	c.Server.Stop()
}

func (c *TestCase) finish(_ *testing.T) {
	c.Minimock.Finish()
}

func (c *TestCase) Run(t *testing.T, fn func(t *testing.T)) {
	assureTestRailID(c.TestRailID, t)

	if c.Name == "" {
		panic(throw.IllegalValue())
	}

	t.Run(c.Name, func(t *testing.T) {
		if c.parallel {
			t.Parallel()
		}
		if c.TestRailID != "" {
			if c.Skipped != "" {
				insrail.LogSkipCase(t, c.TestRailID, c.Skipped)
			} else {
				insrail.LogCase(t, c.TestRailID)
			}
		} else if c.Skipped != "" {
			t.Skip(c.Skipped)
		}

		c.init(t)
		defer c.stop(t)

		fn(t)

		c.finish(t)
	})
}

func (c *TestCase) ResetTestRailID() {
	c.TestRailID = ""
}

func (c *TestCase) SetParallel(parallel bool) {
	c.parallel = parallel
}

type Suite struct {
	Skipped    string
	Parallel   bool
	TestRailID string

	Cases []TestRunner
}

func (suite Suite) Run(t *testing.T) {
	assureTestRailID(suite.TestRailID, t)

	if suite.TestRailID != "" {
		if suite.Skipped != "" {
			insrail.LogSkipCase(t, suite.TestRailID, suite.Skipped)
		} else {
			insrail.LogCase(t, suite.TestRailID)
		}
	} else if suite.Skipped != "" {
		t.Skip(suite.Skipped)
	}

	for _, testCase := range suite.Cases {
		testCase.SetParallel(suite.Parallel)

		if suite.TestRailID != "" {
			testCase.ResetTestRailID()
		}

		testCase.TestRun(t)
	}
}

type TestRunner interface {
	TestRun(t *testing.T)
	SetParallel(parallel bool)
	ResetTestRailID()
}
