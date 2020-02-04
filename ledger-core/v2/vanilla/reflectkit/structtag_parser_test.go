//
//    Copyright 2020 Insolar Technologies
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

package reflectkit

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseStructTag(t *testing.T) {
	for _, tc := range []struct {
		tag    reflect.StructTag
		findFn func(name, qvalue string) bool
		name   string
		value  string
		found  bool
	}{
		{},
		{
			tag: reflect.StructTag(""),
		},
		{
			tag: reflect.StructTag("   "),
		},
		{
			tag: reflect.StructTag(`tag:""`),
			findFn: func(name, qvalue string) bool {
				return true
			},
			found: true,
			name:  "tag",
		},
		{
			tag: reflect.StructTag(`tag:"value"`),
			findFn: func(name, qvalue string) bool {
				return true
			},
			found: true,
			name:  "tag",
			value: "value",
		},
		{
			tag: reflect.StructTag(`   :"value"`),
			findFn: func(name, qvalue string) bool {
				return true
			},
			found: false,
		},
		{
			tag: reflect.StructTag(`   tag:"value"`),
			findFn: func(name, qvalue string) bool {
				return true
			},
			found: true,
			name:  "tag",
			value: "value",
		},
		{
			tag: reflect.StructTag(`tag:"value`),
			findFn: func(name, qvalue string) bool {
				return true
			},
			found: false,
		},
		{
			tag: reflect.StructTag(`tag:"value\\\"`),
			findFn: func(name, qvalue string) bool {
				return true
			},
			found: false,
		},
		{
			tag: reflect.StructTag(`tag:"value"`),
			findFn: func(name, qvalue string) bool {
				return name == "nottag"
			},
			found: false,
		},
	} {
		name, value, found := ParseStructTag(tc.tag, tc.findFn)
		require.Equal(t, tc.found, found, "tag %s, found", tc.tag)
		require.Equal(t, tc.name, name, "tag %s, name", tc.tag)
		require.Equal(t, tc.value, value, "tag %s, value", tc.tag)
	}
}
