///
//    Copyright 2019 Insolar Technologies
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.
///

package inslogger

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/logadapter"
)

func stripPackageName(packageName string) string {
	result := strings.TrimPrefix(packageName, insolarPrefix)
	i := strings.Index(result, ".")
	if result == packageName || i == -1 {
		return result
	}
	return result[:i]
}

// beware to adding lines in this test (test output depend on test code offset!)
func TestLog_getCallInfo(t *testing.T) {
	fileName, funcName, line := logadapter.GetCallInfo(1)
	fileName = fileLineMarshaller(fileName, line)

	assert.Contains(t, fileName, "instrumentation/inslogger/sourceinfo_test.go:")
	assert.Equal(t, "TestLog_getCallInfo", funcName)
	assert.Equal(t, 39, line) // equal of line number where getCallInfo is called
}

func TestLog_stripPackageName(t *testing.T) {
	tests := map[string]struct {
		packageName string
		result      string
	}{
		"insolar":    {"github.com/insolar/assured-ledger/ledger-core/v2/mypackage", "mypackage"},
		"thirdParty": {"github.com/stretchr/testify/assert", "github.com/stretchr/testify/assert"},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.result, stripPackageName(test.packageName))
		})
	}
}
