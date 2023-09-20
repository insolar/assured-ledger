package json

import (
	"encoding/json"
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/pulsewatcher/output"
	"github.com/insolar/assured-ledger/ledger-core/pulsewatcher/status"
)

type Writer struct{}

func NewOutput() output.Printer {
	return Writer{}
}

func (Writer) Print(results []status.Node) {
	type DocumentItem struct {
		URL                string
		NetworkState       string
		ID                 uint32
		NetworkPulseNumber uint32
		PulseNumber        uint32
		ActiveListSize     int
		WorkingListSize    int
		Role               string
		Timestamp          string
		Error              string
	}

	doc := make([]DocumentItem, len(results))

	for i, res := range results {
		doc[i].URL = res.URL
		doc[i].NetworkState = res.Reply.NetworkState
		doc[i].ID = res.Reply.Origin.ID
		doc[i].NetworkPulseNumber = res.Reply.NetworkPulseNumber
		doc[i].PulseNumber = res.Reply.PulseNumber
		doc[i].ActiveListSize = res.Reply.ActiveListSize
		doc[i].WorkingListSize = res.Reply.WorkingListSize
		doc[i].Role = res.Reply.Origin.Role
		doc[i].Timestamp = res.Reply.Timestamp.Format(output.TimeFormat)
		doc[i].Error = res.ErrStr
	}

	jsonDoc, err := json.MarshalIndent(doc, "", "    ")
	if err != nil {
		panic(err) // should never happen
	}
	fmt.Print(string(jsonDoc))
}
