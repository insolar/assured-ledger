package insproto

// `go:generate` command below is here just for information. It should not be used by go generate.
// Actual generation is in the Makefile.
// go:generate protoc -I=. -I=$GOPATH/src --gogofaster_out=Mgoogle/protobuf/descriptor.proto=github.com/gogo/protobuf/protoc-gen-gogo/descriptor:./ ins.proto
