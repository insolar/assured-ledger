package inslogger

import (
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/log/logoutput"
)

func stripPackageName(packageName string) string {
	result := strings.TrimPrefix(packageName, insolarPrefix)
	i := strings.Index(result, ".")
	if result == packageName || i == -1 {
		return result
	}
	return result[:i]
}

// Beware to adding lines in this test (test output depend on test code offset!)
func TestLog_getCallInfo(t *testing.T) {
	_, _, expectedLine, ok := runtime.Caller(0)
	fileName, funcName, line := logoutput.GetCallerInfo(0)
	fileName = fileLineMarshaller(fileName, line)

	require.True(t, ok)
	expectedLine += 1 // expectedLine must point to the line where getCallerInfo is called

	assert.Contains(t, fileName, "instrumentation/inslogger/sourceinfo_test.go:")
	assert.Equal(t, "TestLog_getCallInfo", funcName)
	assert.Equal(t, expectedLine, line) // equal of line number where getCallInfo is called
}

func TestLog_stripPackageName(t *testing.T) {
	tests := map[string]struct {
		packageName string
		result      string
	}{
		"insolar":    {"github.com/insolar/assured-ledger/ledger-core/mypackage", "mypackage"},
		"thirdParty": {"github.com/stretchr/testify/assert", "github.com/stretchr/testify/assert"},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.result, stripPackageName(test.packageName))
		})
	}
}
