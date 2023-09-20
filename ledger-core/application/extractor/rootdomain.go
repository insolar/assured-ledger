package extractor

import (
	"encoding/json"

	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/runner/executor/common/foundation"
)

// InfoResponse represents response from Info() method of RootDomain contract
type Info struct {
	RootDomain              string   `json:"rootDomain"`
	RootMember              string   `json:"rootMember"`
	MigrationDaemonsMembers []string `json:"migrationDaemonMembers"`
	MigrationAdminMember    string   `json:"migrationAdminMember"`
	NodeDomain              string   `json:"nodeDomain"`
}

// InfoResponse returns response from Info() method of RootDomain contract
func InfoResponse(data []byte) (*Info, error) {
	var infoMap interface{}
	var contractErr *foundation.Error
	err := foundation.UnmarshalMethodResultSimplified(data, &infoMap, &contractErr)
	if err != nil {
		return nil, errors.W(err, "[ InfoResponse ] Can't unmarshal")
	}
	if contractErr != nil {
		return nil, errors.W(contractErr, "[ InfoResponse ] Has error in response")
	}

	var info Info
	data = infoMap.([]byte)
	err = json.Unmarshal(data, &info)
	if err != nil {
		return nil, errors.W(err, "[ InfoResponse ] Can't unmarshal response ")
	}

	return &info, nil
}
