// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package investigation

import (
	"testing"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
)

func TestHappyPath(t *testing.T) {
	instestlogger.TestContext(t)

	LogCase(t, "C1")
}

func TestHappyPathParallel(t *testing.T) {
	instestlogger.TestContext(t)

	t.Run("parallel 1", func(t *testing.T) {
		t.Parallel()
		LogCase(t, "C2")
	})
	t.Run("parallel 2", func(t *testing.T) {
		t.Parallel()
		LogCase(t, "C3")
	})
}

func TestSkipped(t *testing.T) {
	instestlogger.TestContext(t)

	LogCase(t, "C4")
	LogSkip(t, "jira link 1")
}

func TestParallelSkipped(t *testing.T) {
	instestlogger.TestContext(t)

	t.Run("parallel 1", func(t *testing.T) {
		t.Parallel()
		LogCase(t, "C5")
		LogSkip(t, "jira link 2")
	})
	t.Run("parallel 2", func(t *testing.T) {
		t.Parallel()
		LogCase(t, "C6")
		LogSkip(t, "jira link 3")
	})
}

func TestFail(t *testing.T) {
	instestlogger.TestContext(t)

	LogCase(t, "C7")
	t.Fail()
}

func TestFailNow(t *testing.T) {
	instestlogger.TestContext(t)

	LogCase(t, "C8")
	t.FailNow()
}

func TestParallelFail1(t *testing.T) {
	instestlogger.TestContext(t)

	t.Run("parallel 1", func(t *testing.T) {
		t.Parallel()
		LogCase(t, "C9")
		t.Fail()
	})
	t.Run("parallel 2", func(t *testing.T) {
		t.Parallel()
		LogCase(t, "C10")
	})
}

func TestParallelFail2(t *testing.T) {
	instestlogger.TestContext(t)

	t.Run("parallel 1", func(t *testing.T) {
		t.Parallel()
		LogCase(t, "C11")
		t.Fail()
	})
	t.Run("parallel 2", func(t *testing.T) {
		t.Parallel()
		LogCase(t, "C12")
		t.Fail()
	})
}

func TestParallelSkipAndFail(t *testing.T) {
	instestlogger.TestContext(t)

	t.Run("parallel 1", func(t *testing.T) {
		t.Parallel()
		LogCase(t, "C13")
		t.Fail()
	})
	t.Run("parallel 2", func(t *testing.T) {
		t.Parallel()
		LogCase(t, "C14")
		LogSkip(t, "jira link 4")
	})
}

func TestPanics(t *testing.T) {
	instestlogger.TestContext(t)

	LogCase(t, "C15")
	panic(123)
}

func TestParallelFailAndPanic(t *testing.T) {
	instestlogger.TestContext(t)

	t.Run("parallel 1", func(t *testing.T) {
		t.Parallel()
		LogCase(t, "C16")
		t.Fail()
	})
	t.Run("parallel 2", func(t *testing.T) {
		t.Parallel()
		LogCase(t, "C17")
		panic(123)
	})
}

func TestParallelSkipAndPanic(t *testing.T) {
	instestlogger.TestContext(t)

	t.Run("parallel 1", func(t *testing.T) {
		t.Parallel()
		LogCase(t, "C18")
		panic(123)
	})
	t.Run("parallel 2", func(t *testing.T) {
		t.Parallel()
		LogCase(t, "C19")
		LogSkip(t, "jira link 5")
	})
}

func TestParallelSkipPanicAndFail(t *testing.T) {
	instestlogger.TestContext(t)

	t.Run("parallel 1", func(t *testing.T) {
		t.Parallel()
		LogCase(t, "C20")
		panic(123)
	})
	t.Run("parallel 2", func(t *testing.T) {
		t.Parallel()
		LogCase(t, "C21")
		LogSkip(t, "jira link 6")
	})
	t.Run("parallel 3", func(t *testing.T) {
		t.Parallel()
		LogCase(t, "C22")
		t.Fail()
	})
}

func TestSkipPanicAndFail(t *testing.T) {
	instestlogger.TestContext(t)

	t.Run("parallel 1", func(t *testing.T) {
		LogCase(t, "C23")
		panic(123)
	})
	t.Run("parallel 2", func(t *testing.T) {
		LogCase(t, "C24")
		LogSkip(t, "jira link 7")
	})
	t.Run("parallel 3", func(t *testing.T) {
		LogCase(t, "C25")
		t.Fail()
	})
}

func TestTimeout(t *testing.T) {
	instestlogger.TestContext(t)

	LogCase(t, "C25")
	time.Sleep(10 * time.Second)
}

func nestedTest(t *testing.T) {
	LogCase(t, "C26")
}

func TestNestedHappyPath(t *testing.T) {
	instestlogger.TestContext(t)

	t.Run("nested 1", func(t *testing.T) {
		LogCase(t, "C27")
		t.Run("nested 1-1", func(t *testing.T) {
			LogCase(t, "C28")
			t.Run("nested defined 1-1-1", nestedTest)
		})
		t.Run("nested 1-2", func(t *testing.T) {
			LogCase(t, "C29")
			t.Run("nested 1-2-1", func(t *testing.T) {
				LogCase(t, "C30")
			})
		})
	})
}

func TestNestedFaiedPath(t *testing.T) {
	instestlogger.TestContext(t)

	t.Run("nested 1", func(t *testing.T) {
		LogCase(t, "C27")
		t.Run("nested 1-1", func(t *testing.T) {
			LogCase(t, "C28")
			t.Run("nested defined 1-1-1", nestedTest)
			t.Fail()
		})
		t.Run("nested 1-2", func(t *testing.T) {
			LogCase(t, "C29")
			t.Run("nested 1-2-1", func(t *testing.T) {
				LogCase(t, "C30")
			})
		})
	})
}
