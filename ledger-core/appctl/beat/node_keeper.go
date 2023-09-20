package beat

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/appctl/beat.NodeKeeper -s _mock.go -g

type NodeKeeper interface {
	NodeNetwork

	AddExpectedBeat(Beat) error
	AddCommittedBeat(Beat) error
}

