package extra

import (
	"strings"
	"unicode"

	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
)

func importCustomName(customName string, imports generator.PluginImports) string {
	packageName, typ := splitCPackageType(customName)
	if packageName != "" {
		pkg := imports.NewImport(packageName)
		pkg.Use()
		return typ
	}
	return typ
}

func splitCPackageType(ctype string) (packageName string, typ string) {
	lastDot := strings.LastIndexByte(ctype, '.')
	if lastDot < 0 {
		return "", ctype
	}
	packageName = ctype[:lastDot]
	importStr := strings.Map(badToUnderscore, packageName)
	typ = importStr + ctype[lastDot:]
	return packageName, typ
}

func badToUnderscore(r rune) rune {
	if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
		return r
	}
	return '_'
}

/**********************************/

type OrderedFields []*descriptor.FieldDescriptorProto

func (v OrderedFields) Len() int {
	return len(v)
}

func (v OrderedFields) Less(i, j int) bool {
	return v[i].GetNumber() < v[j].GetNumber()
}

func (v OrderedFields) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}
