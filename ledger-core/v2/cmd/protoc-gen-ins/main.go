package main

import (
	"io"
	"io/ioutil"
	"os"

	"github.com/gogo/protobuf/gogoproto"
	"github.com/gogo/protobuf/plugin/compare"
	"github.com/gogo/protobuf/plugin/description"
	"github.com/gogo/protobuf/plugin/embedcheck"
	"github.com/gogo/protobuf/plugin/enumstringer"
	"github.com/gogo/protobuf/plugin/equal"
	"github.com/gogo/protobuf/plugin/face"
	"github.com/gogo/protobuf/plugin/gostring"
	"github.com/gogo/protobuf/plugin/oneofcheck"
	"github.com/gogo/protobuf/plugin/populate"
	"github.com/gogo/protobuf/plugin/stringer"
	"github.com/gogo/protobuf/plugin/union"
	"github.com/gogo/protobuf/plugin/unmarshal"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
	"github.com/gogo/protobuf/vanity"
	"github.com/gogo/protobuf/vanity/command"

	plugin "github.com/gogo/protobuf/protoc-gen-gogo/plugin"

	"github.com/insolar/assured-ledger/ledger-core/v2/cmd/protoc-gen-ins/plugins/defaultcheck"
	"github.com/insolar/assured-ledger/ledger-core/v2/cmd/protoc-gen-ins/plugins/marshalto"
	"github.com/insolar/assured-ledger/ledger-core/v2/cmd/protoc-gen-ins/plugins/sizer"
	"github.com/insolar/assured-ledger/ledger-core/v2/insproto"
)

// use: go install github.com/insolar/assured-ledger/ledger-core/v2/cmd/protoc-gen-ins

func main() {
	resetDefaultPlugins()

	// if CatchInput() != nil {
	// 	return
	// }

	req := &plugin.CodeGeneratorRequest{}
	// req = ReplayInput()
	if len(req.FileToGenerate) == 0 {
		req = command.Read()
	}
	files := req.GetProtoFile()
	files = vanity.FilterFiles(files, vanity.NotGoogleProtobufDescriptorProto)

	vanity.ForEachFile(files, vanity.SetBoolFileOption(gogoproto.E_SizerAll, false))
	vanity.ForEachFile(files, vanity.SetBoolFileOption(gogoproto.E_ProtosizerAll, true))
	vanity.ForEachFile(files, vanity.TurnOnMarshalerAll)
	vanity.ForEachFile(files, vanity.TurnOnStable_MarshalerAll)
	vanity.ForEachFile(files, vanity.TurnOnUnmarshalerAll)
	vanity.ForEachFile(files, vanity.TurnOffGoGettersAll)
	vanity.ForEachFile(files, vanity.TurnOffGoUnrecognizedAll)

	vanity.ForEachFile(files, vanity.TurnOffGoUnrecognizedAll)
	vanity.ForEachFile(files, vanity.TurnOffGoUnkeyedAll)
	vanity.ForEachFile(files, vanity.TurnOffGoSizecacheAll)

	vanity.ForEachFile(files, PropagateDefaultOptions)

	resp := command.Generate(req)
	command.Write(resp)
}

func PropagateDefaultOptions(file *descriptor.FileDescriptorProto) {
	for _, msg := range file.GetMessageType() {
		propagateOptions(file, msg, insproto.IsNotation(file, msg))
	}
}

func propagateOptions(file *descriptor.FileDescriptorProto, msgParent *descriptor.DescriptorProto, parentNotation bool) {
	if !insproto.IsHead(msgParent) {
		// do not touch head

		hasMapping := insproto.IsMappingForMessage(file, msgParent)
		for _, field := range msgParent.GetField() {
			hasMapping = hasMapping || insproto.IsMappingForField(field, msgParent, file)
			vanity.SetBoolFieldOption(gogoproto.E_Nullable, false)(field)
		}

		if hasMapping {
			// this fieldNum is reserved and will not be in use
			mapField := insproto.NewFieldMapDescriptorProto(19999)
			vanity.SetBoolFieldOption(gogoproto.E_Nullable, false)(mapField)
			msgParent.Field = append(msgParent.Field, mapField)
		}
	}

	for _, msg := range msgParent.GetNestedType() {
		notation := parentNotation
		switch {
		case msg.Options != nil:
			switch v, err := proto.GetExtension(msg.Options, insproto.E_Notation); {
			case err != nil:
				panic(err)
			case v != nil:
				notation = *v.(*bool)
			case notation:
				vanity.SetBoolMessageOption(insproto.E_Notation, true)(msg)
			}
		case notation:
			vanity.SetBoolMessageOption(insproto.E_Notation, true)(msg)
		}
		propagateOptions(file, msg, notation)
	}
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
	generator.RegisterPlugin(marshalto.NewMarshal()) // this is custom, also includes "context", "head" and "polymorph"
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

func ReadFrom(r io.Reader) *plugin.CodeGeneratorRequest {
	g := generator.New()
	data, err := ioutil.ReadAll(r)
	if err != nil {
		g.Error(err, "reading input")
	}

	if err := proto.Unmarshal(data, g.Request); err != nil {
		g.Error(err, "parsing input proto")
	}

	if len(g.Request.FileToGenerate) == 0 {
		g.Fail("no files to generate")
	}
	return g.Request
}

func CatchInput() error {
	file, err := os.Create(`E:\protoc-dump.txt`)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	io.Copy(file, os.Stdin)
	return io.EOF
}

func ReplayInput() *plugin.CodeGeneratorRequest {
	file, err := os.Open(`E:\protoc-dump.txt`)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	return ReadFrom(file)
}
