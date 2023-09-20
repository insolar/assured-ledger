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
