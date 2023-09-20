package uniserver

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto/l1"
)

var _ l1.BasicOutTransport = &outFacade{}

type outFacade struct {
	delegate l1.BasicOutTransport
	hadError bool
	sentData bool
}

func (p *outFacade) Send(payload io.WriterTo) error {
	p.sentData = true
	if err := p.delegate.Send(payload); err != nil {
		p.hadError = true
		return err
	}
	return nil
}

func (p *outFacade) SendBytes(b []byte) error {
	p.sentData = true
	if err := p.delegate.SendBytes(b); err != nil {
		p.hadError = true
		return err
	}
	return nil
}
