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

package unsafekit

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func mModelOf(v interface{}) MemoryMapModel {
	return MemoryModelDependencyOf(reflect.TypeOf(v))
}

func TestMemoryModelDependencyOf(t *testing.T) {
	require.Equal(t, MemoryModelIndependent, mModelOf(byte(0)))
	require.Equal(t, MemoryModelIndependent, mModelOf(struct {
	}{}))

	require.Equal(t, MemoryModelIndependent, mModelOf(struct {
		b byte
		f bool
	}{}))

	require.Equal(t, MemoryModelDepended, mModelOf(int(0)))
	require.Equal(t, MemoryModelDepended, mModelOf(struct {
		b byte
		f int
	}{}))

	require.Equal(t, MemoryMapIncompatible, mModelOf(&struct{}{}))
	require.Equal(t, MemoryMapIncompatible, mModelOf(struct {
		b byte
		f int
		p *struct{}
	}{}))
	require.Equal(t, MemoryMapIncompatible, mModelOf(""))
}
