package main

import (
	"github.com/gogo/protobuf/gogoproto"
	"github.com/gogo/protobuf/plugin/compare"
	"github.com/gogo/protobuf/plugin/description"
	"github.com/gogo/protobuf/plugin/enumstringer"
	"github.com/gogo/protobuf/plugin/equal"
	"github.com/gogo/protobuf/plugin/face"
	"github.com/gogo/protobuf/plugin/gostring"
	"github.com/gogo/protobuf/plugin/oneofcheck"
	"github.com/gogo/protobuf/plugin/populate"
	"github.com/gogo/protobuf/plugin/stringer"
	"github.com/gogo/protobuf/plugin/union"
	"github.com/gogo/protobuf/plugin/unmarshal"
	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
	"github.com/gogo/protobuf/vanity"
	"github.com/gogo/protobuf/vanity/command"

	"github.com/insolar/assured-ledger/ledger-core/v2/cmd/protoc-gen-ins/plugins/defaultcheck"
	"github.com/insolar/assured-ledger/ledger-core/v2/cmd/protoc-gen-ins/plugins/embedcheck"
	"github.com/insolar/assured-ledger/ledger-core/v2/cmd/protoc-gen-ins/plugins/marshalto"
	"github.com/insolar/assured-ledger/ledger-core/v2/cmd/protoc-gen-ins/plugins/sizer"
)

// use: go install github.com/insolar/assured-ledger/ledger-core/v2/cmd/protoc-gen-ins

func main() {
	resetDefaultPlugins()

	req := command.Read()
	files := req.GetProtoFile()
	files = vanity.FilterFiles(files, vanity.NotGoogleProtobufDescriptorProto)

	vanity.ForEachFile(files, vanity.SetBoolFileOption(gogoproto.E_SizerAll, false))
	vanity.ForEachFile(files, vanity.SetBoolFileOption(gogoproto.E_ProtosizerAll, true))
	vanity.ForEachFile(files, vanity.TurnOnMarshalerAll)
	vanity.ForEachFile(files, vanity.TurnOnStable_MarshalerAll)
	vanity.ForEachFile(files, vanity.TurnOnUnmarshalerAll)
	vanity.ForEachFile(files, vanity.TurnOffGoGettersAll)
	vanity.ForEachFile(files, vanity.TurnOffGoUnrecognizedAll)

	vanity.ForEachFieldInFilesExcludingExtensions(files, TurnOffNullableAll)
	vanity.ForEachFile(files, vanity.TurnOffGoUnrecognizedAll)
	vanity.ForEachFile(files, vanity.TurnOffGoUnkeyedAll)
	vanity.ForEachFile(files, vanity.TurnOffGoSizecacheAll)

	resp := command.Generate(req)
	command.Write(resp)
}

func TurnOffNullableAll(field *descriptor.FieldDescriptorProto) {
	if field.IsRepeated() {
		return
	}
	vanity.SetBoolFieldOption(gogoproto.E_Nullable, false)(field)
}

func resetDefaultPlugins() {
	// This code replaces generator.plugins without producing any output
	// It is necessary to override some of pre-existing plugins
	g := generator.New()
	g.GeneratePlugin(stubPlugin{})

	// And now we can register plugins with some replacements
	generator.RegisterPlugin(compare.NewPlugin())
	generator.RegisterPlugin(defaultcheck.NewPlugin()) // this is custom
	generator.RegisterPlugin(description.NewPlugin())
	generator.RegisterPlugin(embedcheck.NewPlugin()) // this is custom
	generator.RegisterPlugin(enumstringer.NewEnumStringer())
	generator.RegisterPlugin(equal.NewPlugin())
	generator.RegisterPlugin(face.NewPlugin())
	generator.RegisterPlugin(gostring.NewGoString())
	generator.RegisterPlugin(marshalto.NewMarshal()) // this is custom
	generator.RegisterPlugin(oneofcheck.NewPlugin())
	generator.RegisterPlugin(populate.NewPlugin())
	generator.RegisterPlugin(sizer.NewSize())
	generator.RegisterPlugin(stringer.NewStringer())
	// NB! testgen can't be reused as it is unexported
	generator.RegisterPlugin(union.NewUnion())
	generator.RegisterPlugin(unmarshal.NewUnmarshal())
}

type stubPlugin struct{}

func (stubPlugin) Name() string {
	return "stubPlugin"
}

func (s stubPlugin) Init(*generator.Generator) {}

func (s stubPlugin) Generate(*generator.FileDescriptor) {}

func (s stubPlugin) GenerateImports(*generator.FileDescriptor) {}
