package beat

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/appctl/beat.Dispatcher -o ./ -s _mock.go -g
type Dispatcher interface {
	PrepareBeat(Ack)
	CancelBeat()
	CommitBeat(Beat)
	Process(Message) error
}
