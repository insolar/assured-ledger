package uniserver

import (
	"testing"
)

func TestNewReceiveBuffer(t *testing.T) {
	marshaller := &TestProtocolMarshaller{}
	var dispatcher1 Dispatcher
	dispatcher1.RegisterProtocol(0, TestProtocolDescriptor, marshaller, marshaller)
	rb := NewReceiveBuffer(5, 1, 2, &dispatcher1)
	rb.Close()
}
