//
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
//

package args

import (
	"fmt"
	"time"
)

func LazyStr(fn func() string) fmt.Stringer {
	return &lazyStringer{fn}
}

func LazyFmt(format string, a ...interface{}) fmt.Stringer {
	return &lazyStringer{func() string {
		return fmt.Sprintf(format, a...)
	}}
}

func LazyTimeFmt(format string, t time.Time) fmt.Stringer {
	return &lazyStringer{func() string {
		return t.Format(format)
	}}
}

type lazyStringer struct {
	fn func() string
}

func (v *lazyStringer) String() string {
	return v.fn()
}
