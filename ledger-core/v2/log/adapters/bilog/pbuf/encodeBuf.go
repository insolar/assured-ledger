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

package pbuf

import "io"

var _ io.Writer = &encodeBuf{}
var _ io.ByteWriter = &encodeBuf{}

type encodeBuf struct {
	dst []byte
}

func (e *encodeBuf) WriteByte(c byte) error {
	e.dst = append(e.dst, c)
	return nil
}

func (e *encodeBuf) Write(p []byte) (n int, err error) {
	e.dst = append(e.dst, p...)
	return len(p), nil
}
