package readersvc

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jetalloc"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/readersvc/readbundle"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ Service = &serviceImpl{}

func newService(allocStrategy jetalloc.MaterialAllocationStrategy, merklePair cryptkit.PairDigester, provider readbundle.Provider) *serviceImpl {
	switch {
	case allocStrategy == nil:
		panic(throw.IllegalValue())
	case merklePair == nil:
		panic(throw.IllegalValue())
	case provider == nil:
		panic(throw.IllegalValue())
	}
	return &serviceImpl{provider}
}

type serviceImpl struct {
	// allocStrategy jetalloc.MaterialAllocationStrategy
	// merklePair    cryptkit.PairDigester
	provider readbundle.Provider
}

func (p *serviceImpl) FindCabinet(pn pulse.Number) (Cabinet, error) {
	cab, err := p.provider.FindCabinet(pn)
	switch {
	case err != nil:
		return nil, err
	case cab == nil:
		panic(throw.Impossible())
	}
	return readCabinet{cab}, nil
}

func (p *serviceImpl) ReadFromCabinet(cab Cabinet, id jet.DropID, fn func(readbundle.Reader) error) error {
	rCab := cab.(readCabinet)
	pn := rCab.PulseNumber()

	switch {
	case fn == nil:
		panic(throw.IllegalValue())
	case id == 0:
	case id.CreatedAt()	!= pn:
		return throw.E("cabinet and drop pulses are different", struct {
			CabinetPN pulse.Number
			DropID jet.DropID
		}{
		cab.PulseNumber(), id,
		})
	}

	reader := rCab.cab.Reader()
	if reader == nil {
		return throw.E("cabinet is closed")
	}

	return fn(reader)
}

func (p *serviceImpl) NeedsBatching() bool {
	return false
}

type readCabinet struct {
	cab readbundle.ReadCabinet
}

func (v readCabinet) PulseNumber() pulse.Number {
	return v.cab.PulseNumber()
}

func (v readCabinet) Open() error {
	return v.cab.Open()
}

func (v readCabinet) Close() error {
	return v.cab.Close()
}
